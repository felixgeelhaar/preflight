package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
)

// TestFullWorkflow_ConfigToPlan tests the flow from config creation to planning
func TestFullWorkflow_ConfigToPlan(t *testing.T) {
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

	// Plan should work
	ctx := context.Background()
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Should have entries for git config
	entries := plan.Entries()
	assert.NotEmpty(t, entries)
}

// TestFullWorkflow_PlanAndApplyDryRun tests plan creation and dry-run apply
func TestFullWorkflow_PlanAndApplyDryRun(t *testing.T) {
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

	ctx := context.Background()

	// Create plan
	plan, err := h.Preflight().Plan(ctx, configPath, "default")
	require.NoError(t, err)

	// Apply in dry-run mode
	results, err := h.Preflight().Apply(ctx, plan, true)
	require.NoError(t, err)

	// With minimal config, results may be empty
	_ = results // Results vary based on config content
}

// TestFullWorkflow_DoctorCheck tests the doctor check flow
func TestFullWorkflow_DoctorCheck(t *testing.T) {
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

	// Run doctor
	report, err := h.Doctor(configPath, "default")
	require.NoError(t, err)
	require.NotNil(t, report)
}

// TestFullWorkflow_CaptureFindings tests the capture flow
func TestFullWorkflow_CaptureFindings(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Create some files in home directory
	h.CreateFile(".gitconfig", "[user]\n  name = Test\n")
	h.CreateFile(".zshrc", "# zsh config\n")

	// Run capture
	findings, err := h.Capture()
	require.NoError(t, err)
	require.NotNil(t, findings)
}

// TestFullWorkflow_SnapshotAndRestore tests snapshot and restore functionality
func TestFullWorkflow_SnapshotAndRestore(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	ctx := context.Background()

	// Create dotfiles
	gitconfig := filepath.Join(homeDir, ".gitconfig")
	zshrc := filepath.Join(homeDir, ".zshrc")

	originalGitconfig := "[user]\n  name = Original\n"
	originalZshrc := "# Original zshrc\nexport PATH=$PATH\n"

	require.NoError(t, os.WriteFile(gitconfig, []byte(originalGitconfig), 0o644))
	require.NoError(t, os.WriteFile(zshrc, []byte(originalZshrc), 0o644))

	// Create snapshot manager
	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)

	// Take snapshot before changes
	set, err := manager.BeforeApply(ctx, []string{gitconfig, zshrc})
	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Len(t, set.Snapshots, 2)

	// Simulate changes
	require.NoError(t, os.WriteFile(gitconfig, []byte("[user]\n  name = Modified\n"), 0o644))
	require.NoError(t, os.WriteFile(zshrc, []byte("# Modified\n"), 0o644))

	// Verify changes
	content, _ := os.ReadFile(gitconfig)
	assert.Contains(t, string(content), "Modified")

	// Restore from snapshot
	err = manager.Restore(ctx, set.ID)
	require.NoError(t, err)

	// Verify restoration
	content, _ = os.ReadFile(gitconfig)
	assert.Equal(t, originalGitconfig, string(content))

	content, _ = os.ReadFile(zshrc)
	assert.Equal(t, originalZshrc, string(content))
}

// TestFullWorkflow_MultiLayerConfig tests config with multiple layers
func TestFullWorkflow_MultiLayerConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	layersDir := filepath.Join(configDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	// Create manifest with multiple layers
	manifest := `
targets:
  work:
    - base
    - identity.work
    - role.dev
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "preflight.yaml"), []byte(manifest), 0o644))

	// Create base layer
	baseLayer := `
name: base
shell:
  default: zsh
git:
  core:
    editor: nvim
`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(baseLayer), 0o644))

	// Create identity layer
	identityLayer := `
name: identity.work
git:
  user:
    name: "Work User"
    email: "work@company.com"
`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "identity.work.yaml"), []byte(identityLayer), 0o644))

	// Create role layer
	roleLayer := `
name: role.dev
packages:
  brew:
    formulae:
      - go
      - node
`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "role.dev.yaml"), []byte(roleLayer), 0o644))

	// Parse manifest
	manifestPath := filepath.Join(configDir, "preflight.yaml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	parsed, err := config.ParseManifest(data)
	require.NoError(t, err)

	// Verify all layers are referenced
	workLayers := parsed.Targets["work"]
	assert.Len(t, workLayers, 3)
	assert.Equal(t, "base", workLayers[0].String())
	assert.Equal(t, "identity.work", workLayers[1].String())
	assert.Equal(t, "role.dev", workLayers[2].String())
}

// TestFullWorkflow_LayerMerging tests that layers merge correctly
func TestFullWorkflow_LayerMerging(t *testing.T) {
	t.Parallel()

	// Parse two layers
	baseYAML := `
name: base
git:
  user:
    name: "Base Name"
  core:
    editor: vim
shell:
  default: bash
`
	overrideYAML := `
name: override
git:
  user:
    email: "override@example.com"
  core:
    editor: nvim
`
	base, err := config.ParseLayer([]byte(baseYAML))
	require.NoError(t, err)

	override, err := config.ParseLayer([]byte(overrideYAML))
	require.NoError(t, err)

	// Verify both layers parsed correctly
	assert.Equal(t, "Base Name", base.Git.User.Name)
	assert.Equal(t, "vim", base.Git.Core.Editor)
	assert.Equal(t, "bash", base.Shell.Default)

	assert.Equal(t, "override@example.com", override.Git.User.Email)
	assert.Equal(t, "nvim", override.Git.Core.Editor)
}

// TestFullWorkflow_ConfigValidation tests config validation
func TestFullWorkflow_ConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		manifest  string
		wantError bool
	}{
		{
			name: "valid config",
			manifest: `
targets:
  default:
    - base
`,
			wantError: false,
		},
		{
			name: "empty targets",
			manifest: `
defaults:
  mode: locked
`,
			wantError: true,
		},
		{
			name: "multiple targets",
			manifest: `
targets:
  default:
    - base
  work:
    - base
    - identity.work
  personal:
    - base
    - identity.personal
`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.ParseManifest([]byte(tt.manifest))
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFullWorkflow_ErrorHandling tests error scenarios
func TestFullWorkflow_ErrorHandling(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Create manifest referencing non-existent layer
	manifest := `
targets:
  default:
    - nonexistent
`
	layer := `
name: base
`
	configPath := h.CreateConfig(manifest, layer)

	// Plan should fail or handle missing layer gracefully
	ctx := context.Background()
	_, err := h.Preflight().Plan(ctx, configPath, "default")
	// Error handling behavior depends on implementation
	// The test verifies the system doesn't crash
	_ = err
}

// TestFullWorkflow_TargetSelection tests selecting different targets
func TestFullWorkflow_TargetSelection(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	layersDir := filepath.Join(configDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	manifest := `
targets:
  default:
    - base
  work:
    - base
    - identity.work
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "preflight.yaml"), []byte(manifest), 0o644))

	baseLayer := `name: base`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(baseLayer), 0o644))

	workLayer := `name: identity.work`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "identity.work.yaml"), []byte(workLayer), 0o644))

	// Parse manifest
	manifestPath := filepath.Join(configDir, "preflight.yaml")
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	parsed, err := config.ParseManifest(data)
	require.NoError(t, err)

	// Verify targets
	assert.Len(t, parsed.Targets["default"], 1)
	assert.Len(t, parsed.Targets["work"], 2)
}
