package fleet

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockInventorySource is a test double for InventorySource.
type mockInventorySource struct {
	name      string
	available bool
	hosts     []*Host
	err       error
}

func (m *mockInventorySource) Name() string { return m.name }

func (m *mockInventorySource) Discover(_ context.Context) ([]*Host, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hosts, nil
}

func (m *mockInventorySource) Available() bool { return m.available }

func newMockSource(t *testing.T, name string, available bool, hostnames ...string) *mockInventorySource {
	t.Helper()
	hosts := make([]*Host, 0, len(hostnames))
	for i, hostname := range hostnames {
		id, err := NewHostID(fmt.Sprintf("%s-host%d", name, i+1))
		require.NoError(t, err)
		host, err := NewHost(id, SSHConfig{Hostname: hostname})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return &mockInventorySource{
		name:      name,
		available: available,
		hosts:     hosts,
	}
}

func TestInventorySource_Interface(t *testing.T) {
	t.Parallel()

	var source InventorySource = &mockInventorySource{
		name:      "test",
		available: true,
	}

	assert.Equal(t, "test", source.Name())
	assert.True(t, source.Available())

	hosts, err := source.Discover(context.Background())
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func TestInventorySource_DiscoverReturnsHosts(t *testing.T) {
	t.Parallel()

	source := newMockSource(t, "cloud", true, "10.0.0.1", "10.0.0.2")

	hosts, err := source.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestInventorySource_DiscoverReturnsError(t *testing.T) {
	t.Parallel()

	source := &mockInventorySource{
		name:      "failing",
		available: true,
		err:       fmt.Errorf("API error"),
	}

	hosts, err := source.Discover(context.Background())
	assert.Error(t, err)
	assert.Nil(t, hosts)
}

func TestInventorySource_NotAvailable(t *testing.T) {
	t.Parallel()

	source := &mockInventorySource{
		name:      "unconfigured",
		available: false,
	}

	assert.False(t, source.Available())
}
