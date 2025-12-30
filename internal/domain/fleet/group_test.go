package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroupName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    GroupName
		wantErr bool
	}{
		{
			name:  "simple name",
			input: "production",
			want:  GroupName("production"),
		},
		{
			name:  "name with hyphen",
			input: "prod-west",
			want:  GroupName("prod-west"),
		},
		{
			name:  "name with underscore",
			input: "prod_west",
			want:  GroupName("prod_west"),
		},
		{
			name:  "name with numbers",
			input: "tier1",
			want:  GroupName("tier1"),
		},
		{
			name:  "whitespace trimmed",
			input: "  production  ",
			want:  GroupName("production"),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "starts with number",
			input:   "1production",
			wantErr: true,
		},
		{
			name:    "contains space",
			input:   "prod west",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewGroupName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGroupName_String(t *testing.T) {
	t.Parallel()
	name := GroupName("production")
	assert.Equal(t, "production", name.String())
}

func TestNewGroup(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("production")
	group := NewGroup(name)

	assert.Equal(t, name, group.Name())
	assert.Empty(t, group.Description())
	assert.Empty(t, group.HostPatterns())
	assert.Empty(t, group.Policies())
	assert.Empty(t, group.Inherit())
}

func TestGroup_Description(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("production")
	group := NewGroup(name)

	group.SetDescription("Production servers")
	assert.Equal(t, "Production servers", group.Description())
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestGroup_HostPatterns(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("web")
	group := NewGroup(name)

	t.Run("add pattern", func(t *testing.T) {
		group.AddHostPattern("web-*")
		assert.Contains(t, group.HostPatterns(), "web-*")
	})

	t.Run("add duplicate pattern", func(t *testing.T) {
		initialLen := len(group.HostPatterns())
		group.AddHostPattern("web-*")
		assert.Len(t, group.HostPatterns(), initialLen)
	})

	t.Run("set patterns", func(t *testing.T) {
		group.SetHostPatterns([]string{"server-*", "node-*"})
		patterns := group.HostPatterns()
		assert.Len(t, patterns, 2)
		assert.Contains(t, patterns, "server-*")
		assert.Contains(t, patterns, "node-*")
	})

	t.Run("patterns are copied", func(t *testing.T) {
		original := []string{"test-*"}
		group.SetHostPatterns(original)
		original[0] = "modified"
		assert.Equal(t, "test-*", group.HostPatterns()[0])
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestGroup_Policies(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("production")
	group := NewGroup(name)

	t.Run("add policy", func(t *testing.T) {
		group.AddPolicy("require-approval")
		assert.Contains(t, group.Policies(), "require-approval")
		assert.True(t, group.HasPolicy("require-approval"))
	})

	t.Run("add duplicate policy", func(t *testing.T) {
		initialLen := len(group.Policies())
		group.AddPolicy("require-approval")
		assert.Len(t, group.Policies(), initialLen)
	})

	t.Run("has policy", func(t *testing.T) {
		assert.True(t, group.HasPolicy("require-approval"))
		assert.False(t, group.HasPolicy("nonexistent"))
	})

	t.Run("set policies", func(t *testing.T) {
		group.SetPolicies([]string{"maintenance-window", "notify-slack"})
		policies := group.Policies()
		assert.Len(t, policies, 2)
		assert.Contains(t, policies, "maintenance-window")
		assert.Contains(t, policies, "notify-slack")
	})
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestGroup_Inherit(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("production-web")
	group := NewGroup(name)

	parent1, _ := NewGroupName("production")
	parent2, _ := NewGroupName("web")

	t.Run("add inherit", func(t *testing.T) {
		group.AddInherit(parent1)
		assert.Contains(t, group.Inherit(), parent1)
	})

	t.Run("add duplicate inherit", func(t *testing.T) {
		initialLen := len(group.Inherit())
		group.AddInherit(parent1)
		assert.Len(t, group.Inherit(), initialLen)
	})

	t.Run("set inherit", func(t *testing.T) {
		group.SetInherit([]GroupName{parent1, parent2})
		inherit := group.Inherit()
		assert.Len(t, inherit, 2)
		assert.Contains(t, inherit, parent1)
		assert.Contains(t, inherit, parent2)
	})
}

func TestGroup_Summary(t *testing.T) {
	t.Parallel()

	name, _ := NewGroupName("production")
	group := NewGroup(name)
	group.SetDescription("Production environment")
	group.SetHostPatterns([]string{"prod-*"})
	group.SetPolicies([]string{"require-approval"})

	parent, _ := NewGroupName("base")
	group.AddInherit(parent)

	summary := group.Summary()

	assert.Equal(t, "production", summary.Name)
	assert.Equal(t, "Production environment", summary.Description)
	assert.Contains(t, summary.HostPatterns, "prod-*")
	assert.Contains(t, summary.Policies, "require-approval")
	assert.Contains(t, summary.Inherit, "base")
}
