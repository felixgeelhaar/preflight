package tmux_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/tmux"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tmuxConfigPath returns the best-practice tmux config path relative to homeDir.
// This matches the Discovery.BestPracticePath() behavior (XDG location).
func tmuxConfigPath(homeDir string) string {
	return filepath.Join(homeDir, ".config", "tmux", "tmux.conf")
}

// =============================================================================
// TPMStep Tests
// =============================================================================

func TestTPMStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	assert.Equal(t, "tmux:tpm", step.ID().String())
}

func TestTPMStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestTPMStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create TPM directory
	tpmPath := filepath.Join(tmpDir, ".tmux", "plugins", "tpm")
	err := os.MkdirAll(tpmPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestTPMStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestTPMStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "tpm", diff.Resource())
	assert.Equal(t, "tmux-plugins/tpm", diff.Name())
}

func TestTPMStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install TPM", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "Tmux Plugin Manager")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.Contains(t, explanation.DocLinks()[0], "tpm")
	assert.NotEmpty(t, explanation.Tradeoffs())
}

func TestTPMStep_Apply_Success(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	tpmPath := filepath.Join(tmpDir, ".tmux", "plugins", "tpm")
	runner.AddResult("git", []string{"clone", "https://github.com/tmux-plugins/tpm", tpmPath}, ports.CommandResult{
		ExitCode: 0,
	})

	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestTPMStep_Apply_GitFails(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	tpmPath := filepath.Join(tmpDir, ".tmux", "plugins", "tpm")
	runner.AddResult("git", []string{"clone", "https://github.com/tmux-plugins/tpm", tpmPath}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "clone failed",
	})

	step := tmux.NewTPMStep(runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clone failed")
}

// =============================================================================
// PluginStep Tests
// =============================================================================

func TestPluginStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	assert.Equal(t, "tmux:plugin:tmux-plugins/tmux-sensible", step.ID().String())
}

func TestPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "tmux:tpm", deps[0].String())
}

func TestPluginStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config with plugin at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g @plugin 'tmux-plugins/tmux-sensible'\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPluginStep_Check_NeedsApply_NoConfig(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Check_NeedsApply_PluginNotInConfig(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config without the plugin at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g @plugin 'tmux-plugins/tpm'\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "plugin", diff.Resource())
	assert.Equal(t, "tmux-plugins/tmux-sensible", diff.Name())
}

func TestPluginStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure tmux Plugin", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "tmux-plugins/tmux-sensible")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

func TestPluginStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify plugin was added (using XDG best-practice path)
	configPath := tmuxConfigPath(tmpDir)
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tmux-plugins/tmux-sensible")
}

func TestPluginStep_Apply_AppendToExisting(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing config at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g @plugin 'tmux-plugins/tpm'\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	tpmID := compiler.MustNewStepID("tmux:tpm")
	step := tmux.NewPluginStep("tmux-plugins/tmux-sensible", tpmID, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify both plugins are in config
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tmux-plugins/tpm")
	assert.Contains(t, string(data), "tmux-plugins/tmux-sensible")
}

// =============================================================================
// ConfigStep Tests
// =============================================================================

func TestConfigStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]string{"prefix": "C-a"}
	step := tmux.NewConfigStep(settings, "", runner)

	assert.Equal(t, "tmux:config", step.ID().String())
}

func TestConfigStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]string{"prefix": "C-a"}
	step := tmux.NewConfigStep(settings, "", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestConfigStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config with settings at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g prefix C-a\nset -g mouse on\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a",
		"mouse":  "on",
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigStep_Check_NeedsApply_NoConfig(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]string{"prefix": "C-a"}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Check_NeedsApply_SettingMissing(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config with only one setting at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g prefix C-a\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a",
		"mouse":  "on", // This setting is missing
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a",
		"mouse":  "on",
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "config", diff.Resource())
	assert.Equal(t, "tmux.conf", diff.Name())
	assert.Contains(t, diff.NewValue(), "2 settings")
}

func TestConfigStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a",
		"mouse":  "on",
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure tmux", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 tmux settings")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

func TestConfigStep_Apply_NewConfig(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a",
		"mouse":  "on",
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were written (using XDG best-practice path)
	configPath := tmuxConfigPath(tmpDir)
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "set -g prefix C-a")
	assert.Contains(t, string(data), "set -g mouse on")
}

func TestConfigStep_Apply_UpdateExisting(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing config at XDG best-practice path
	configPath := tmuxConfigPath(tmpDir)
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(configPath, []byte("set -g prefix C-b\nset -g status on\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	settings := map[string]string{
		"prefix": "C-a", // Update existing
		"mouse":  "on",  // Add new
	}
	step := tmux.NewConfigStep(settings, "", runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were updated
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "set -g prefix C-a")
	assert.Contains(t, string(data), "set -g mouse on")
	assert.Contains(t, string(data), "set -g status on") // Preserved
}
