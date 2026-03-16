//go:build !noaws

package cloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ fleet.InventorySource = (*AWSSource)(nil)

// mockAWSClient is a test double for AWSClient.
type mockAWSClient struct {
	instances []EC2Instance
	err       error
	filters   map[string]string // captured filters from last call
}

func (m *mockAWSClient) DescribeInstances(_ context.Context, filters map[string]string) ([]EC2Instance, error) {
	m.filters = filters
	if m.err != nil {
		return nil, m.err
	}
	return m.instances, nil
}

func TestNewAWSSource(t *testing.T) {
	t.Parallel()

	client := &mockAWSClient{}
	source := NewAWSSource(client, "us-east-1")

	assert.NotNil(t, source)
	assert.Equal(t, "aws", source.Name())
}

func TestAWSSource_Name(t *testing.T) {
	t.Parallel()

	source := NewAWSSource(&mockAWSClient{}, "eu-west-1")
	assert.Equal(t, "aws", source.Name())
}

func TestAWSSource_Available(t *testing.T) {
	t.Parallel()

	t.Run("with client", func(t *testing.T) {
		t.Parallel()
		source := NewAWSSource(&mockAWSClient{}, "us-east-1")
		assert.True(t, source.Available())
	})

	t.Run("without client", func(t *testing.T) {
		t.Parallel()
		source := NewAWSSource(nil, "us-east-1")
		assert.False(t, source.Available())
	})
}

func TestAWSSource_Discover(t *testing.T) {
	t.Parallel()

	t.Run("discovers running instances", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID:       "i-abc123",
					PrivateIP:        "10.0.1.10",
					PublicIP:         "54.1.2.3",
					State:            "running",
					Tags:             map[string]string{"Name": "web-server", "env": "production"},
					KeyName:          "my-key",
					AvailabilityZone: "us-east-1a",
					InstanceType:     "t3.micro",
				},
				{
					InstanceID:       "i-def456",
					PrivateIP:        "10.0.1.20",
					State:            "running",
					Tags:             map[string]string{"Name": "db-server"},
					KeyName:          "db-key",
					AvailabilityZone: "us-east-1b",
					InstanceType:     "r5.large",
				},
			},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 2)

		// First host
		assert.Equal(t, fleet.HostID("i-abc123"), hosts[0].ID())
		assert.Equal(t, "10.0.1.10", hosts[0].SSH().Hostname)

		// Second host
		assert.Equal(t, fleet.HostID("i-def456"), hosts[1].ID())
		assert.Equal(t, "10.0.1.20", hosts[1].SSH().Hostname)
	})

	t.Run("skips non-running instances", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-running",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{},
				},
				{
					InstanceID: "i-stopped",
					PrivateIP:  "10.0.1.20",
					State:      "stopped",
					Tags:       map[string]string{},
				},
				{
					InstanceID: "i-terminated",
					PrivateIP:  "10.0.1.30",
					State:      "terminated",
					Tags:       map[string]string{},
				},
			},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		assert.Len(t, hosts, 1)
		assert.Equal(t, fleet.HostID("i-running"), hosts[0].ID())
	})

	t.Run("maps EC2 tags to fleet tags", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-tagged",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{"env": "staging", "role": "web"},
				},
			},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)

		tags := hosts[0].Tags()
		// Should have ec2 tags + source tag
		assert.GreaterOrEqual(t, len(tags), 2, "should have at least the EC2 tags")
	})

	t.Run("sets metadata on hosts", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID:       "i-meta",
					PrivateIP:        "10.0.1.10",
					PublicIP:         "54.1.2.3",
					State:            "running",
					Tags:             map[string]string{},
					KeyName:          "my-key",
					AvailabilityZone: "us-east-1a",
					InstanceType:     "t3.micro",
				},
			},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)

		meta := hosts[0].Metadata()
		assert.Equal(t, "us-east-1", meta["aws:region"])
		assert.Equal(t, "us-east-1a", meta["aws:az"])
		assert.Equal(t, "t3.micro", meta["aws:instance-type"])
		assert.Equal(t, "54.1.2.3", meta["aws:public-ip"])
		assert.Equal(t, "my-key", meta["aws:key-name"])
	})

	t.Run("adds host to aws group", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-grouped",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{},
				},
			},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.True(t, hosts[0].InGroup("aws"))
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			err: fmt.Errorf("access denied"),
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
		assert.Nil(t, hosts)
	})

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{},
		}

		source := NewAWSSource(client, "us-east-1")
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		assert.Empty(t, hosts)
	})
}

func TestAWSSource_WithOptions(t *testing.T) {
	t.Parallel()

	t.Run("with SSH user", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-user",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{},
				},
			},
		}

		source := NewAWSSource(client, "us-east-1", WithSSHUser("ec2-user"))
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "ec2-user", hosts[0].SSH().User)
	})

	t.Run("with SSH key", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-key",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{},
				},
			},
		}

		source := NewAWSSource(client, "us-east-1", WithSSHKey("/home/user/.ssh/aws.pem"))
		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "/home/user/.ssh/aws.pem", hosts[0].SSH().IdentityFile)
	})

	t.Run("with filters", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{},
		}

		filters := map[string]string{"tag:env": "production"}
		source := NewAWSSource(client, "us-east-1", WithFilters(filters))
		_, _ = source.Discover(context.Background())

		assert.Equal(t, filters, client.filters)
	})

	t.Run("with multiple options", func(t *testing.T) {
		t.Parallel()

		client := &mockAWSClient{
			instances: []EC2Instance{
				{
					InstanceID: "i-multi",
					PrivateIP:  "10.0.1.10",
					State:      "running",
					Tags:       map[string]string{},
				},
			},
		}

		source := NewAWSSource(client, "us-west-2",
			WithSSHUser("ubuntu"),
			WithSSHKey("/path/to/key.pem"),
			WithFilters(map[string]string{"tag:env": "staging"}),
		)

		hosts, err := source.Discover(context.Background())

		require.NoError(t, err)
		require.Len(t, hosts, 1)
		assert.Equal(t, "ubuntu", hosts[0].SSH().User)
		assert.Equal(t, "/path/to/key.pem", hosts[0].SSH().IdentityFile)
	})
}

func TestAWSSource_SkipsInstancesWithNoPrivateIP(t *testing.T) {
	t.Parallel()

	client := &mockAWSClient{
		instances: []EC2Instance{
			{
				InstanceID: "i-noip",
				PrivateIP:  "",
				State:      "running",
				Tags:       map[string]string{},
			},
			{
				InstanceID: "i-hasip",
				PrivateIP:  "10.0.1.10",
				State:      "running",
				Tags:       map[string]string{},
			},
		},
	}

	source := NewAWSSource(client, "us-east-1")
	hosts, err := source.Discover(context.Background())

	require.NoError(t, err)
	assert.Len(t, hosts, 1)
	assert.Equal(t, fleet.HostID("i-hasip"), hosts[0].ID())
}
