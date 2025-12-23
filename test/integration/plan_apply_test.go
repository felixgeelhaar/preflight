package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlan_ShowsExpectedChanges(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Run plan
	ctx := context.Background()
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, plan)
}

func TestPlan_WithEmptyConfig(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Run plan
	ctx := context.Background()
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)

	// Empty config should have no changes
	assert.False(t, plan.HasChanges())
}

func TestPlan_WithGitConfig(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
git:
  user:
    name: "Test User"
    email: "test@example.com"
`

	configPath := h.CreateConfig(manifest, layer)

	// Run plan
	ctx := context.Background()
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Should have entries for git config
	entries := plan.Entries()
	assert.NotEmpty(t, entries, "should have plan entries for git config")
}

func TestApply_ExecutesSteps(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	manifest := `
targets:
  default:
    - base
`
	layer := `
name: base
`

	configPath := h.CreateConfig(manifest, layer)

	// Create plan
	ctx := context.Background()
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)

	// Run apply (dry run to avoid system changes)
	results, err := h.Preflight().Apply(ctx, plan, true)
	require.NoError(t, err)

	// With empty config, there should be no results
	assert.Empty(t, results)
}
