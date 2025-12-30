package starship_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/starship"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- InstallStep Tests ---

func TestInstallStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := starship.NewInstallStep(runner)

	assert.Equal(t, "starship:install", step.ID().String())
}

func TestInstallStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := starship.NewInstallStep(runner)

	assert.Empty(t, step.DependsOn())
}

func TestInstallStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("starship", []string{"--version"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "starship 1.0.0",
	})

	step := starship.NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestInstallStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered = error

	step := starship.NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestInstallStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := starship.NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "starship", diff.Resource())
}

func TestInstallStep_Apply_BrewSuccess(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "starship"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := starship.NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestInstallStep_Apply_FallbackToCurl(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("brew", []string{"install", "starship"}, ports.CommandResult{
		ExitCode: 1,
	})
	runner.AddResult("sh", []string{"-c", "curl -sS https://starship.rs/install.sh | sh -s -- -y"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := starship.NewInstallStep(runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestInstallStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := starship.NewInstallStep(runner)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Equal(t, "Install Starship", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "Starship")
	assert.NotEmpty(t, explanation.DocLinks())
}

// --- ConfigStep Tests ---

func TestConfigStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "", installDep, runner)

	assert.Equal(t, "starship:config", step.ID().String())
}

func TestConfigStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "", installDep, runner)

	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "starship:install", deps[0].String())
}

func TestConfigStep_Check_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Check_ConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing config
	configDir := filepath.Join(tmpDir, ".config")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configDir, "starship.toml"), []byte("add_newline = false"), 0644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigStep_Check_WithPresetAlwaysNeedsApply(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing config
	configDir := filepath.Join(tmpDir, ".config")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configDir, "starship.toml"), []byte("add_newline = false"), 0644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "nerd-font-symbols", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Plan_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	settings := map[string]interface{}{"add_newline": false, "format": "$all"}
	step := starship.NewConfigStep(settings, "", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "2 settings")
}

func TestConfigStep_Plan_WithPreset(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "tokyo-night", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "preset: tokyo-night")
}

func TestConfigStep_Apply_WithSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	settings := map[string]interface{}{"add_newline": false}
	step := starship.NewConfigStep(settings, "", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify config was written
	configPath := filepath.Join(tmpDir, ".config", "starship.toml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "add_newline = false")
}

func TestConfigStep_Apply_WithPreset(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	configPath := filepath.Join(configDir, "starship.toml")
	runner.AddResult("starship", []string{"preset", "nerd-font-symbols", "-o", configPath}, ports.CommandResult{
		ExitCode: 0,
	})

	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "nerd-font-symbols", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err = step.Apply(ctx)

	require.NoError(t, err)
}

func TestConfigStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewConfigStep(nil, "", installDep, runner)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Starship", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
}

// --- ShellIntegrationStep Tests ---

func TestShellIntegrationStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)

	assert.Equal(t, "starship:shell:zsh", step.ID().String())
}

func TestShellIntegrationStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("bash", installDep, runner)

	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "starship:install", deps[0].String())
}

func TestShellIntegrationStep_Check_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestShellIntegrationStep_Check_AlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create zshrc with starship init
	err := os.WriteFile(filepath.Join(tmpDir, ".zshrc"), []byte(`# My zshrc
eval "$(starship init zsh)"
`), 0644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestShellIntegrationStep_Check_UnsupportedShell(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("powershell", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestShellIntegrationStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("bash", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "integration", diff.Resource())
	assert.Equal(t, "bash", diff.Name())
	assert.Contains(t, diff.NewValue(), "starship init")
}

func TestShellIntegrationStep_Apply_Zsh(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing zshrc
	err := os.WriteFile(filepath.Join(tmpDir, ".zshrc"), []byte("# My zshrc\n"), 0644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify init was added
	data, err := os.ReadFile(filepath.Join(tmpDir, ".zshrc"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `eval "$(starship init zsh)"`)
}

func TestShellIntegrationStep_Apply_Bash(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("bash", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify init was added
	data, err := os.ReadFile(filepath.Join(tmpDir, ".bashrc"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `eval "$(starship init bash)"`)
}

func TestShellIntegrationStep_Apply_Fish(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("fish", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify init was added
	data, err := os.ReadFile(filepath.Join(tmpDir, ".config", "fish", "config.fish"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "starship init fish | source")
}

func TestShellIntegrationStep_Apply_AlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	original := `# My zshrc
eval "$(starship init zsh)"
`
	err := os.WriteFile(filepath.Join(tmpDir, ".zshrc"), []byte(original), 0644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify not duplicated
	data, err := os.ReadFile(filepath.Join(tmpDir, ".zshrc"))
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestShellIntegrationStep_Apply_UnsupportedShell(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("powershell", installDep, runner)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

func TestShellIntegrationStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	installDep := compiler.MustNewStepID("starship:install")
	step := starship.NewShellIntegrationStep("zsh", installDep, runner)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Shell Integration", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "zsh")
	assert.NotEmpty(t, explanation.DocLinks())
}
