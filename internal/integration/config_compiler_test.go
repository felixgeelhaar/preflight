//go:build integration

// Package integration provides integration tests for preflight.
package integration

import (
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/felixgeelhaar/preflight/internal/provider/brew"
	"github.com/felixgeelhaar/preflight/internal/provider/files"
	"github.com/felixgeelhaar/preflight/internal/provider/git"
	"github.com/felixgeelhaar/preflight/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigToCompiler_MinimalBrewConfig tests loading a minimal config
// with brew packages and compiling it to a step graph.
func TestConfigToCompiler_MinimalBrewConfig(t *testing.T) {
	t.Parallel()

	// Setup temp directory
	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	// Create manifest
	manifest := `
targets:
  default:
    - base
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	// Create layers directory and base layer
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
	require.NotNil(t, merged)

	// Compile to step graph
	cmdRunner := command.NewRealRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify step graph contains expected steps
	steps := graph.Steps()
	assert.GreaterOrEqual(t, len(steps), 1, "should have at least one step")

	// Find brew:formula:git step
	var foundGit bool
	for _, step := range steps {
		if step.ID().String() == "brew:formula:git" {
			foundGit = true
			break
		}
	}
	assert.True(t, foundGit, "should have brew:formula:git step")
}

// TestConfigToCompiler_GitConfig tests loading git configuration
// and compiling it to appropriate steps.
func TestConfigToCompiler_GitConfig(t *testing.T) {
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
  core:
    editor: nvim
  alias:
    st: status
    co: checkout
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load config
	loader := config.NewLoader()
	target, err := config.NewTargetName("default")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	// Compile
	fs := filesystem.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(git.NewProvider(fs))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Verify git:config step exists
	steps := graph.Steps()
	var foundGitConfig bool
	for _, step := range steps {
		if step.ID().String() == "git:config" {
			foundGitConfig = true
			break
		}
	}
	assert.True(t, foundGitConfig, "should have git:config step")
}

// TestConfigToCompiler_MultiLayer tests loading and merging multiple layers.
func TestConfigToCompiler_MultiLayer(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  work:
    - base
    - work
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)

	layersDir := testutil.WriteTempDir(t, tmpDir, "layers")

	// Base layer with git defaults
	baseLayer := `
name: base
git:
  core:
    editor: vim
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Work layer overrides editor
	workLayer := `
name: work
git:
  user:
    name: Work User
    email: work@company.com
  core:
    editor: nvim
`
	testutil.WriteTempFile(t, layersDir, "work.yaml", workLayer)

	// Load config
	loader := config.NewLoader()
	target, err := config.NewTargetName("work")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	// Verify merge: work layer should override base
	raw := merged.Raw()
	gitConfig, ok := raw["git"].(map[string]interface{})
	require.True(t, ok, "git config should exist")

	coreConfig, ok := gitConfig["core"].(map[string]interface{})
	require.True(t, ok, "git.core should exist")

	// Editor should be nvim from work layer (last wins)
	editor, ok := coreConfig["editor"].(string)
	require.True(t, ok)
	assert.Equal(t, "nvim", editor, "work layer should override base editor")

	// User should come from work layer
	userConfig, ok := gitConfig["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Work User", userConfig["name"])
}

// TestConfigToCompiler_FilesProvider tests files provider compilation.
func TestConfigToCompiler_FilesProvider(t *testing.T) {
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
  - path: ~/.bashrc
    mode: generated
  - path: ~/.vimrc
    mode: generated
`
	testutil.WriteTempFile(t, layersDir, "base.yaml", baseLayer)

	// Load config
	loader := config.NewLoader()
	target, err := config.NewTargetName("default")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	// Compile
	fs := filesystem.NewRealFileSystem()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(files.NewProvider(fs))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)

	// Verify files steps exist
	steps := graph.Steps()
	assert.GreaterOrEqual(t, len(steps), 2, "should have at least 2 file steps")
}

// TestConfigToCompiler_InvalidConfig tests handling of invalid configurations.
func TestConfigToCompiler_InvalidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	// Invalid YAML
	invalidYAML := `
targets: [invalid yaml
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", invalidYAML)

	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")

	_, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	assert.Error(t, err, "should error on invalid YAML")
}

// TestConfigToCompiler_MissingLayer tests handling of missing layer files.
func TestConfigToCompiler_MissingLayer(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

	manifest := `
targets:
  default:
    - nonexistent
`
	testutil.WriteTempFile(t, tmpDir, "preflight.yaml", manifest)
	testutil.WriteTempDir(t, tmpDir, "layers")

	loader := config.NewLoader()
	target, _ := config.NewTargetName("default")

	_, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	assert.Error(t, err, "should error on missing layer")
}

// TestConfigToCompiler_EmptyTarget tests handling of empty target.
func TestConfigToCompiler_EmptyTarget(t *testing.T) {
	t.Parallel()

	tmpDir, cleanup := testutil.TempConfigDir(t)
	defer cleanup()

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

	// Compile should still work with empty config
	comp := compiler.NewCompiler()
	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)
	assert.Empty(t, graph.Steps(), "empty config should produce empty graph")
}

// TestConfigToCompiler_ProviderContextAccess tests that providers receive proper context.
func TestConfigToCompiler_ProviderContextAccess(t *testing.T) {
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

	loader := config.NewLoader()
	target, err := config.NewTargetName("default")
	require.NoError(t, err)

	merged, err := loader.Load(filepath.Join(tmpDir, "preflight.yaml"), target)
	require.NoError(t, err)

	cmdRunner := command.NewRealRunner()
	comp := compiler.NewCompiler()
	comp.RegisterProvider(brew.NewProvider(cmdRunner))

	graph, err := comp.Compile(merged.Raw())
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify all step types
	stepIDs := make(map[string]bool)
	for _, step := range graph.Steps() {
		stepIDs[step.ID().String()] = true
	}

	assert.True(t, stepIDs["brew:tap:homebrew/cask"], "should have tap step")
	assert.True(t, stepIDs["brew:formula:git"], "should have formula step")
	assert.True(t, stepIDs["brew:cask:visual-studio-code"], "should have cask step")
}
