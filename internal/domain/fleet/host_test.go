package fleet

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    HostID
		wantErr bool
	}{
		{
			name:  "simple id",
			input: "server01",
			want:  HostID("server01"),
		},
		{
			name:  "id with hyphen",
			input: "server-01",
			want:  HostID("server-01"),
		},
		{
			name:  "id with dot",
			input: "server.prod",
			want:  HostID("server.prod"),
		},
		{
			name:  "id with underscore",
			input: "server_01",
			want:  HostID("server_01"),
		},
		{
			name:  "whitespace trimmed",
			input: "  server01  ",
			want:  HostID("server01"),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with number",
			input:   "01server",
			wantErr: true,
		},
		{
			name:    "contains space",
			input:   "server 01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewHostID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHostID_String(t *testing.T) {
	t.Parallel()
	id := HostID("server01")
	assert.Equal(t, "server01", id.String())
}

func TestSSHConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		cfg := SSHConfig{Hostname: "example.com", Port: 22}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("missing hostname", func(t *testing.T) {
		t.Parallel()
		cfg := SSHConfig{Port: 22}
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid port negative", func(t *testing.T) {
		t.Parallel()
		cfg := SSHConfig{Hostname: "example.com", Port: -1}
		assert.Error(t, cfg.Validate())
	})

	t.Run("invalid port too high", func(t *testing.T) {
		t.Parallel()
		cfg := SSHConfig{Hostname: "example.com", Port: 65536}
		assert.Error(t, cfg.Validate())
	})
}

func TestSSHConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	cfg := SSHConfig{Hostname: "example.com"}
	cfg = cfg.WithDefaults()

	assert.Equal(t, 22, cfg.Port)
	assert.Equal(t, "root", cfg.User)
	assert.NotZero(t, cfg.ConnectTimeout)
}

func TestNewHost(t *testing.T) {
	t.Parallel()

	t.Run("creates host with valid config", func(t *testing.T) {
		t.Parallel()
		id, _ := NewHostID("server01")
		ssh := SSHConfig{Hostname: "10.0.0.1", User: "admin", Port: 22}

		host, err := NewHost(id, ssh)
		require.NoError(t, err)

		assert.Equal(t, id, host.ID())
		assert.Equal(t, "10.0.0.1", host.SSH().Hostname)
		assert.Equal(t, "admin", host.SSH().User)
		assert.Equal(t, 22, host.SSH().Port)
		assert.Equal(t, HostStatusUnknown, host.Status())
	})

	t.Run("applies defaults", func(t *testing.T) {
		t.Parallel()
		id, _ := NewHostID("server01")
		ssh := SSHConfig{Hostname: "10.0.0.1"}

		host, err := NewHost(id, ssh)
		require.NoError(t, err)

		assert.Equal(t, 22, host.SSH().Port)
		assert.Equal(t, "root", host.SSH().User)
	})

	t.Run("returns error for invalid SSH config", func(t *testing.T) {
		t.Parallel()
		id, _ := NewHostID("server01")
		ssh := SSHConfig{} // Missing hostname

		_, err := NewHost(id, ssh)
		assert.Error(t, err)
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestHost_Tags(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1"})

	t.Run("set tags", func(t *testing.T) {
		tags, _ := NewTags("darwin", "production")
		host.SetTags(tags)
		assert.Equal(t, tags, host.Tags())
	})

	t.Run("add tag", func(t *testing.T) {
		tag, _ := NewTag("arm64")
		host.AddTag(tag)
		assert.True(t, host.HasTag(tag))
	})

	t.Run("add duplicate tag", func(t *testing.T) {
		tag, _ := NewTag("darwin")
		initialLen := len(host.Tags())
		host.AddTag(tag)
		assert.Len(t, host.Tags(), initialLen)
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestHost_Groups(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1"})

	t.Run("set groups", func(t *testing.T) {
		host.SetGroups([]string{"production", "web"})
		groups := host.Groups()
		assert.Contains(t, groups, "production")
		assert.Contains(t, groups, "web")
	})

	t.Run("add group", func(t *testing.T) {
		host.AddGroup("database")
		assert.True(t, host.InGroup("database"))
	})

	t.Run("add duplicate group", func(t *testing.T) {
		initialLen := len(host.Groups())
		host.AddGroup("production")
		assert.Len(t, host.Groups(), initialLen)
	})

	t.Run("in group check", func(t *testing.T) {
		assert.True(t, host.InGroup("production"))
		assert.False(t, host.InGroup("nonexistent"))
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestHost_Status(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1"})

	t.Run("mark online", func(t *testing.T) {
		host.MarkOnline()
		assert.Equal(t, HostStatusOnline, host.Status())
		assert.NotZero(t, host.LastSeen())
		assert.NoError(t, host.LastError())
	})

	t.Run("mark offline", func(t *testing.T) {
		host.MarkOffline()
		assert.Equal(t, HostStatusOffline, host.Status())
	})

	t.Run("mark error", func(t *testing.T) {
		err := errors.New("connection refused")
		host.MarkError(err)
		assert.Equal(t, HostStatusError, host.Status())
		assert.Equal(t, err, host.LastError())
	})
}

func TestHost_Metadata(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1"})

	host.SetMetadata("os", "darwin")
	host.SetMetadata("arch", "arm64")

	meta := host.Metadata()
	assert.Equal(t, "darwin", meta["os"])
	assert.Equal(t, "arm64", meta["arch"])

	// Verify it's a copy
	meta["os"] = "linux"
	assert.Equal(t, "darwin", host.Metadata()["os"])
}

func TestHost_HasTags(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1"})
	tags, _ := NewTags("darwin", "production", "arm64")
	host.SetTags(tags)

	t.Run("has tag", func(t *testing.T) {
		t.Parallel()
		assert.True(t, host.HasTag(Tag("darwin")))
		assert.False(t, host.HasTag(Tag("linux")))
	})

	t.Run("has any tag", func(t *testing.T) {
		t.Parallel()
		check, _ := NewTags("linux", "darwin")
		assert.True(t, host.HasAnyTag(check))

		noMatch, _ := NewTags("linux", "windows")
		assert.False(t, host.HasAnyTag(noMatch))
	})

	t.Run("has all tags", func(t *testing.T) {
		t.Parallel()
		check, _ := NewTags("darwin", "production")
		assert.True(t, host.HasAllTags(check))

		partial, _ := NewTags("darwin", "linux")
		assert.False(t, host.HasAllTags(partial))
	})
}

func TestHost_Summary(t *testing.T) {
	t.Parallel()

	id, _ := NewHostID("server01")
	host, _ := NewHost(id, SSHConfig{Hostname: "10.0.0.1", User: "admin", Port: 2222})

	tags, _ := NewTags("darwin", "production")
	host.SetTags(tags)
	host.SetGroups([]string{"web"})
	host.SetMetadata("region", "us-west")
	host.MarkOnline()

	summary := host.Summary()

	assert.Equal(t, id, summary.ID)
	assert.Equal(t, "10.0.0.1", summary.Hostname)
	assert.Equal(t, "admin", summary.User)
	assert.Equal(t, 2222, summary.Port)
	assert.Contains(t, summary.Tags, "darwin")
	assert.Contains(t, summary.Tags, "production")
	assert.Contains(t, summary.Groups, "web")
	assert.Equal(t, HostStatusOnline, summary.Status)
	assert.Equal(t, "us-west", summary.Metadata["region"])
}
