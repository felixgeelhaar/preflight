package targeting

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestInventory(t *testing.T) *fleet.Inventory {
	t.Helper()
	inv := fleet.NewInventory()

	// Create hosts
	hosts := []struct {
		id     string
		host   string
		tags   []string
		groups []string
	}{
		{"web-01", "10.0.0.1", []string{"darwin", "production"}, []string{"web", "production"}},
		{"web-02", "10.0.0.2", []string{"linux", "production"}, []string{"web", "production"}},
		{"api-01", "10.0.1.1", []string{"linux", "production"}, []string{"api", "production"}},
		{"db-01", "10.0.2.1", []string{"linux", "staging"}, []string{"database", "staging"}},
		{"cache-01", "10.0.3.1", []string{"linux", "production"}, []string{"cache", "production"}},
	}

	for _, h := range hosts {
		hostID, _ := fleet.NewHostID(h.id)
		host, _ := fleet.NewHost(hostID, fleet.SSHConfig{Hostname: h.host})
		tags, _ := fleet.NewTags(h.tags...)
		host.SetTags(tags)
		host.SetGroups(h.groups)
		_ = inv.AddHost(host)
	}

	// Create groups
	groups := []struct {
		name     string
		patterns []string
	}{
		{"web", []string{"web-*"}},
		{"production", []string{}},     // Uses direct membership
		{"database", []string{"db-*"}}, // Pattern-based
		{"staging", []string{}},        // Direct membership only
	}

	for _, g := range groups {
		name, _ := fleet.NewGroupName(g.name)
		group := fleet.NewGroup(name)
		for _, p := range g.patterns {
			group.AddHostPattern(p)
		}
		_ = inv.AddGroup(group)
	}

	return inv
}

func TestNewSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantType SelectorType
		wantVal  string
		negated  bool
		wantErr  bool
	}{
		{
			name:     "all with @",
			input:    "@all",
			wantType: SelectorTypeAll,
			wantVal:  "all",
		},
		{
			name:     "all with asterisk",
			input:    "*",
			wantType: SelectorTypeAll,
			wantVal:  "all",
		},
		{
			name:     "group selector",
			input:    "@production",
			wantType: SelectorTypeGroup,
			wantVal:  "production",
		},
		{
			name:     "tag selector",
			input:    "tag:darwin",
			wantType: SelectorTypeTag,
			wantVal:  "darwin",
		},
		{
			name:     "pattern selector",
			input:    "web-*",
			wantType: SelectorTypePattern,
			wantVal:  "web-*",
		},
		{
			name:     "regex pattern",
			input:    "~web-\\d+",
			wantType: SelectorTypePattern,
			wantVal:  "~web-\\d+",
		},
		{
			name:     "host selector",
			input:    "web-01",
			wantType: SelectorTypeHost,
			wantVal:  "web-01",
		},
		{
			name:     "negated group",
			input:    "!@staging",
			wantType: SelectorTypeGroup,
			wantVal:  "staging",
			negated:  true,
		},
		{
			name:     "negated tag",
			input:    "!tag:linux",
			wantType: SelectorTypeTag,
			wantVal:  "linux",
			negated:  true,
		},
		{
			name:    "empty selector",
			input:   "",
			wantErr: true,
		},
		{
			name:    "empty group",
			input:   "@",
			wantErr: true,
		},
		{
			name:    "empty tag",
			input:   "tag:",
			wantErr: true,
		},
		{
			name:    "invalid pattern",
			input:   "~[invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, err := NewSelector(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, s.Type())
			assert.Equal(t, tt.wantVal, s.Value())
			assert.Equal(t, tt.negated, s.IsNegated())
		})
	}
}

func TestSelector_Select_All(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	s, _ := NewSelector("@all")
	hosts := s.Select(inv)

	assert.Len(t, hosts, 5)
}

func TestSelector_Select_Group(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	s, _ := NewSelector("@web")
	hosts := s.Select(inv)

	assert.Len(t, hosts, 2)
	for _, h := range hosts {
		assert.True(t, h.InGroup("web") || hasIDPrefix(h, "web-"))
	}
}

func TestSelector_Select_Tag(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	s, _ := NewSelector("tag:darwin")
	hosts := s.Select(inv)

	assert.Len(t, hosts, 1)
	assert.Equal(t, fleet.HostID("web-01"), hosts[0].ID())
}

func TestSelector_Select_Pattern(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	s, _ := NewSelector("web-*")
	hosts := s.Select(inv)

	assert.Len(t, hosts, 2)
	for _, h := range hosts {
		assert.True(t, hasIDPrefix(h, "web-"))
	}
}

func TestSelector_Select_Host(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	t.Run("existing host", func(t *testing.T) {
		t.Parallel()
		s, _ := NewSelector("web-01")
		hosts := s.Select(inv)
		assert.Len(t, hosts, 1)
		assert.Equal(t, fleet.HostID("web-01"), hosts[0].ID())
	})

	t.Run("nonexistent host", func(t *testing.T) {
		t.Parallel()
		s, _ := NewSelector("nonexistent")
		hosts := s.Select(inv)
		assert.Empty(t, hosts)
	})
}

func TestSelector_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"@all", "@all"},
		{"*", "@all"},
		{"@production", "@production"},
		{"tag:darwin", "tag:darwin"},
		{"web-*", "web-*"},
		{"web-01", "web-01"},
		{"!@staging", "!@staging"},
		{"!tag:linux", "!tag:linux"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			s, _ := NewSelector(tt.input)
			assert.Equal(t, tt.want, s.String())
		})
	}
}

func TestNewTarget(t *testing.T) {
	t.Parallel()

	t.Run("single include", func(t *testing.T) {
		t.Parallel()
		target, err := NewTarget("@production")
		require.NoError(t, err)
		assert.Len(t, target.Includes(), 1)
		assert.Empty(t, target.Excludes())
	})

	t.Run("multiple includes", func(t *testing.T) {
		t.Parallel()
		target, err := NewTarget("@web", "@api")
		require.NoError(t, err)
		assert.Len(t, target.Includes(), 2)
	})

	t.Run("with exclusion", func(t *testing.T) {
		t.Parallel()
		target, err := NewTarget("@production", "!tag:staging")
		require.NoError(t, err)
		assert.Len(t, target.Includes(), 1)
		assert.Len(t, target.Excludes(), 1)
	})

	t.Run("only exclusion defaults to all", func(t *testing.T) {
		t.Parallel()
		target, err := NewTarget("!@staging")
		require.NoError(t, err)
		assert.Len(t, target.Includes(), 1)
		assert.Equal(t, SelectorTypeAll, target.Includes()[0].Type())
		assert.Len(t, target.Excludes(), 1)
	})

	t.Run("invalid selector", func(t *testing.T) {
		t.Parallel()
		_, err := NewTarget("@production", "~[invalid")
		assert.Error(t, err)
	})
}

func TestTarget_Select(t *testing.T) {
	t.Parallel()
	inv := createTestInventory(t)

	t.Run("single group", func(t *testing.T) {
		t.Parallel()
		target, _ := NewTarget("@production")
		hosts := target.Select(inv)
		// web-01, web-02, api-01, cache-01 are in production
		assert.Len(t, hosts, 4)
	})

	t.Run("union of groups", func(t *testing.T) {
		t.Parallel()
		target, _ := NewTarget("@web", "@database")
		hosts := target.Select(inv)
		// web-01, web-02, db-01
		assert.Len(t, hosts, 3)
	})

	t.Run("with exclusion", func(t *testing.T) {
		t.Parallel()
		target, _ := NewTarget("@production", "!tag:darwin")
		hosts := target.Select(inv)
		// production minus darwin (web-01)
		assert.Len(t, hosts, 3)
		for _, h := range hosts {
			assert.NotEqual(t, fleet.HostID("web-01"), h.ID())
		}
	})

	t.Run("exclude by pattern", func(t *testing.T) {
		t.Parallel()
		target, _ := NewTarget("@all", "!web-*")
		hosts := target.Select(inv)
		// all minus web-*
		assert.Len(t, hosts, 3)
		for _, h := range hosts {
			assert.False(t, hasIDPrefix(h, "web-"))
		}
	})
}

func TestTarget_String(t *testing.T) {
	t.Parallel()

	target, _ := NewTarget("@production", "!tag:staging")
	assert.Equal(t, "@production !tag:staging", target.String())
}

func hasIDPrefix(h *fleet.Host, prefix string) bool {
	return len(h.ID()) >= len(prefix) && string(h.ID())[:len(prefix)] == prefix
}
