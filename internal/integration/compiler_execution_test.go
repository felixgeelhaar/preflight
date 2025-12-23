//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/git"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompilerToExecution_PlanGeneration tests that a compiled step graph
// can be converted into an execution plan.
func TestCompilerToExecution_PlanGeneration(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
      - curl
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load config
	loader := config.NewLoader()
	target, err := config.NewTargetName("default")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	// Compile
	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Plan
	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Verify plan has entries
	entries := plan.Entries()
	assert.GreaterOrEqual(t, len(entries), 1, "plan should have at least one entry")
}

// TestCompilerToExecution_DryRun tests dry-run execution.
func TestCompilerToExecution_DryRun(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load -> Compile -> Plan
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	// Execute with dry run
	executor := execution.NewExecutor().WithDryRun(true)
	results, err := executor.Execute(ctx, plan)
	require.NoError(t, err)

	// Should have results for all plan entries
	assert.Len(t, results, len(plan.Entries()), "should have result for each plan entry")

	// Dry run should not cause failures (nothing actually applied)
	for _, result := range results {
		assert.NotEqual(t, compiler.StatusFailed, result.Status(),
			"dry run should not fail: %s", result.StepID().String())
	}
}

// TestCompilerToExecution_PlanSummary tests plan summary generation.
func TestCompilerToExecution_PlanSummary(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
git:
  user:
    name: Test User
    email: test@example.com
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load -> Compile -> Plan
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	fs := ports.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(git.NewProvider(fs))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	// Verify summary
	summary := plan.Summary()
	assert.GreaterOrEqual(t, summary.Total, 1, "should have at least one step in summary")
	assert.GreaterOrEqual(t, summary.Total, summary.Satisfied+summary.NeedsApply,
		"total should be >= satisfied + needs_apply")
}

// TestCompilerToExecution_StepOrder tests that steps are executed in correct order.
func TestCompilerToExecution_StepOrder(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
packages:
  brew:
    taps:
      - homebrew/cask
    formulae:
      - git
    casks:
      - visual-studio-code
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load -> Compile
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Get topological order from graph
	sortedSteps, err := graph.TopologicalSort()
	require.NoError(t, err)

	// Verify dependencies come before dependents
	stepIndex := make(map[string]int)
	for i, step := range sortedSteps {
		stepIndex[step.ID().String()] = i
	}

	for _, step := range sortedSteps {
		myIndex := stepIndex[step.ID().String()]
		for _, depID := range step.DependsOn() {
			depIndex, exists := stepIndex[depID.String()]
			if exists {
				assert.Less(t, depIndex, myIndex,
					"dependency %s should come before %s",
					depID.String(), step.ID().String())
			}
		}
	}
}

// TestCompilerToExecution_ContextCancellation tests cancellation during execution.
func TestCompilerToExecution_ContextCancellation(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
      - curl
      - jq
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load -> Compile -> Plan
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	// Create a cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Execute should handle cancellation gracefully
	executor := execution.NewExecutor().WithDryRun(true)
	results, err := executor.Execute(cancelCtx, plan)

	// Should return context error or empty/partial results
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled, "should return context.Canceled error")
	} else {
		// If no error, results should be <= plan entries (might be partial)
		assert.LessOrEqual(t, len(results), len(plan.Entries()))
	}
}

// TestCompilerToExecution_HasChanges tests the HasChanges method on plans.
func TestCompilerToExecution_HasChanges(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	// Empty config should have no changes
	manifest := `
targets:
  default: []
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)
	testutil.WriteTempDir(t, tmpDir, "layers")

	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	comp := compiler.NewCompiler()
	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	// Empty plan should have no changes
	assert.False(t, plan.HasChanges(), "empty plan should have no changes")
}

// TestCompilerToExecution_MultipleDependencies tests steps with multiple dependencies.
func TestCompilerToExecution_MultipleDependencies(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	// Brew taps typically come before formulae and casks
	baseLayer := `
name: base
packages:
  brew:
    taps:
      - homebrew/cask
      - homebrew/core
    formulae:
      - git
      - curl
    casks:
      - firefox
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load -> Compile
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Plan
	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	// Verify we have multiple entries
	entries := plan.Entries()
	assert.GreaterOrEqual(t, len(entries), 4, "should have tap, formula, and cask entries")
}

// TestCompilerToExecution_ResultStatus tests that results have correct statuses.
func TestCompilerToExecution_ResultStatus(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Full pipeline
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := ports.NewRealCommandRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)

	executor := execution.NewExecutor().WithDryRun(true)
	results, err := executor.Execute(ctx, plan)
	require.NoError(t, err)

	// Each result should have a valid status
	for _, result := range results {
		status := result.Status()
		assert.True(t,
			status == compiler.StatusSatisfied ||
				status == compiler.StatusNeedsApply ||
				status == compiler.StatusSkipped ||
				status == compiler.StatusFailed ||
				status == compiler.StatusUnknown,
			"result should have valid status: %s", result.StepID().String())

		// All results should have a step ID
		assert.NotEmpty(t, result.StepID().String(), "result should have step ID")
	}
}
