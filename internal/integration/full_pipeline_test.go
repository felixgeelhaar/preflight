//go:build integration

package integration

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/files"
	"github.com/felixgeelhaar/preflight/internal/provider/git"
	"github.com/felixgeelhaar/preflight/internal/provider/ssh"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullPipeline_LoadCompilePlan tests the complete pipeline from config to plan.
func TestFullPipeline_LoadCompilePlan(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	// Create a complete config structure
	manifest := `
targets:
  default:
    - base
    - development
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
      - curl
git:
  user:
    name: Developer
    email: dev@example.com
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	devLayer := `
name: development
packages:
  brew:
    formulae:
      - jq
      - ripgrep
`
	testutil.WriteTempFile(t, layersDir, "development.yaml", devLayer)

	// Phase 1: Load and merge config
	loader := config.NewLoader()
	target, err := config.NewTargetName("default")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)
	require.NotNil(t, merged)

	// Phase 2: Compile to step graph
	cmdRunner := command.NewRealRunner()
	fs := filesystem.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))
	comp.RegisterProvider(git.NewProvider(fs))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify expected steps
	steps := graph.Steps()
	stepIDs := make(map[string]bool)
	for _, step := range steps {
		stepIDs[step.ID().String()] = true
	}

	assert.True(t, stepIDs["brew:tap:homebrew/cask"], "should have tap step")
	assert.True(t, stepIDs["brew:formula:git"], "should have git formula step")
	assert.True(t, stepIDs["brew:formula:curl"], "should have curl formula step")
	assert.True(t, stepIDs["brew:formula:jq"], "should have jq formula step from dev layer")
	assert.True(t, stepIDs["git:config"], "should have git config step")

	// Phase 3: Generate execution plan
	planner := execution.NewPlanner()
	ctx := context.Background()
	plan, err := planner.Plan(ctx, graph)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Verify plan
	summary := plan.Summary()
	assert.GreaterOrEqual(t, summary.Total, 5, "should have at least 5 steps")
}

// TestFullPipeline_DryRunApply tests the full pipeline with dry-run execution.
func TestFullPipeline_DryRunApply(t *testing.T) {
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

	cmdRunner := command.NewRealRunner()
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

	// All results should be valid (dry run should not fail)
	for _, result := range results {
		assert.NotEqual(t, compiler.StatusFailed, result.Status(),
			"dry run should not fail for step: %s", result.StepID().String())
	}
}

// TestFullPipeline_AppLayer tests the app layer facade.
func TestFullPipeline_AppLayer(t *testing.T) {
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

	// Use the app layer
	var buf bytes.Buffer
	pf := app.New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, filepath.Join(tmpDir, "preflight.yaml"), "default")
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Print plan should produce output
	pf.PrintPlan(plan)
	output := buf.String()
	assert.Contains(t, output, "Preflight Plan", "should print plan header")
}

// TestFullPipeline_MultiProvider tests using multiple providers together.
func TestFullPipeline_MultiProvider(t *testing.T) {
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
git:
  user:
    name: Developer
    email: dev@example.com
ssh:
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_ed25519
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Register multiple providers
	cmdRunner := command.NewRealRunner()
	fs := filesystem.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))
	comp.RegisterProvider(git.NewProvider(fs))
	comp.RegisterProvider(ssh.NewProvider(fs))

	// Load and compile
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Verify steps from all providers
	stepIDs := make(map[string]bool)
	for _, step := range graph.Steps() {
		stepIDs[step.ID().String()] = true
	}

	assert.True(t, stepIDs["brew:formula:git"], "should have brew step")
	assert.True(t, stepIDs["git:config"], "should have git config step")
	assert.True(t, stepIDs["ssh:config"], "should have ssh config step")
}

// TestFullPipeline_FilesProvider tests the files provider in full pipeline.
func TestFullPipeline_FilesProvider(t *testing.T) {
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

	// Use the config domain's FileDeclaration format
	baseLayer := `
name: base
files:
  - path: ~/.testrc
    mode: generated
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Full pipeline
	fs := filesystem.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(files.NewProvider(fs))

	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Verify files step exists
	steps := graph.Steps()
	assert.GreaterOrEqual(t, len(steps), 1, "should have file step")
}

// TestFullPipeline_EmptyConfig tests handling of empty configuration.
func TestFullPipeline_EmptyConfig(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default: []
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)
	testutil.WriteTempDir(t, tmpDir, "layers")

	var buf bytes.Buffer
	pf := app.New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, filepath.Join(tmpDir, "preflight.yaml"), "default")
	require.NoError(t, err)

	// Empty config should produce valid (but empty) plan
	assert.False(t, plan.HasChanges(), "empty config should have no changes")
}

// TestFullPipeline_LayerMerging tests that layers merge correctly.
func TestFullPipeline_LayerMerging(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - base
    - override
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")

	baseLayer := `
name: base
git:
  user:
    name: Base User
    email: base@example.com
  core:
    editor: vim
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	overrideLayer := `
name: override
git:
  user:
    name: Override User
  core:
    editor: nvim
`
	testutil.WriteTempFile(t, layersDir, "override.yaml", overrideLayer)

	// Load and verify merge
	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	raw := merged.Raw()
	gitConfig, ok := raw["git"].(map[string]interface{})
	require.True(t, ok)

	userConfig, ok := gitConfig["user"].(map[string]interface{})
	require.True(t, ok)

	// Name should be overridden
	assert.Equal(t, "Override User", userConfig["name"], "name should be overridden")
	// Email should be preserved from base
	assert.Equal(t, "base@example.com", userConfig["email"], "email should be preserved from base")

	coreConfig, ok := gitConfig["core"].(map[string]interface{})
	require.True(t, ok)
	// Editor should be overridden
	assert.Equal(t, "nvim", coreConfig["editor"], "editor should be overridden")
}

// TestFullPipeline_PlanOutput tests that plan output is formatted correctly.
func TestFullPipeline_PlanOutput(t *testing.T) {
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

	var buf bytes.Buffer
	pf := app.New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, filepath.Join(tmpDir, "preflight.yaml"), "default")
	require.NoError(t, err)

	pf.PrintPlan(plan)
	output := buf.String()

	// Verify output format
	assert.Contains(t, output, "Preflight Plan", "should have header")
	assert.Contains(t, output, "=", "should have separator")
}

// TestFullPipeline_ResultOutput tests that execution result output is formatted correctly.
func TestFullPipeline_ResultOutput(t *testing.T) {
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

	var buf bytes.Buffer
	pf := app.New(&buf)

	ctx := context.Background()
	plan, err := pf.Plan(ctx, filepath.Join(tmpDir, "preflight.yaml"), "default")
	require.NoError(t, err)

	// Apply with dry run
	results, err := pf.Apply(ctx, plan, true)
	require.NoError(t, err)

	pf.PrintResults(results)
	output := buf.String()

	// Verify output has results section
	assert.Contains(t, output, "Execution Results", "should have results header")
	assert.Contains(t, output, "Summary", "should have summary")
}

// TestFullPipeline_StepDependencies tests step dependency resolution.
func TestFullPipeline_StepDependencies(t *testing.T) {
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
	// Taps must come before formulae that need them
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

	cmdRunner := command.NewRealRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")
	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Get topological order
	sortedSteps, err := graph.TopologicalSort()
	require.NoError(t, err)

	// Build index for verification
	stepIndex := make(map[string]int)
	for i, step := range sortedSteps {
		stepIndex[step.ID().String()] = i
	}

	// Tap should come before cask
	tapIndex, tapExists := stepIndex["brew:tap:homebrew/cask"]
	caskIndex, caskExists := stepIndex["brew:cask:visual-studio-code"]

	if tapExists && caskExists {
		assert.Less(t, tapIndex, caskIndex,
			"tap should come before cask in execution order")
	}
}
