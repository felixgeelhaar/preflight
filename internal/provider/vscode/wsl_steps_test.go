package vscode

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RemoteWSLSetupStep Tests ---

func TestRemoteWSLSetupStep_ID(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSetupStep(nil, nil)
	assert.Equal(t, "vscode:wsl:setup", step.ID().String())
}

func TestRemoteWSLSetupStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSetupStep(nil, nil)
	assert.Empty(t, step.DependsOn())
}

func TestRemoteWSLSetupStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--list-extensions"}, ports.CommandResult{
		Stdout:   "ms-vscode-remote.remote-wsl\nms-python.python\n",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestRemoteWSLSetupStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--list-extensions"}, ports.CommandResult{
		Stdout:   "ms-python.python\n",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestRemoteWSLSetupStep_Check_WSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// When in WSL, uses code.exe to talk to Windows VS Code
	runner.AddResult("code.exe", []string{"--list-extensions"}, ports.CommandResult{
		Stdout:   "ms-vscode-remote.remote-wsl\n",
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestRemoteWSLSetupStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSetupStep(nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Contains(t, diff.Summary(), "ms-vscode-remote.remote-wsl")
}

func TestRemoteWSLSetupStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--install-extension", "ms-vscode-remote.remote-wsl", "--force"}, ports.CommandResult{
		Stdout:   "Extension installed successfully.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestRemoteWSLSetupStep_Apply_WSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// When in WSL, uses code.exe
	runner.AddResult("code.exe", []string{"--install-extension", "ms-vscode-remote.remote-wsl", "--force"}, ports.CommandResult{
		Stdout:   "Extension installed successfully.",
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestRemoteWSLSetupStep_Apply_Failed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--install-extension", "ms-vscode-remote.remote-wsl", "--force"}, ports.CommandResult{
		Stderr:   "Installation failed.",
		ExitCode: 1,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLSetupStep(runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install Remote-WSL extension")
}

func TestRemoteWSLSetupStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSetupStep(nil, nil)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Equal(t, "Install Remote-WSL Extension", exp.Summary())
	assert.Contains(t, exp.Detail(), "Remote-WSL")
	assert.NotEmpty(t, exp.DocLinks())
	assert.NotEmpty(t, exp.Tradeoffs())
}

// --- RemoteWSLExtensionStep Tests ---

func TestRemoteWSLExtensionStep_ID(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLExtensionStep("golang.go", "", nil, nil)
	assert.Equal(t, "vscode:wsl:extension:golang_go", step.ID().String())
}

func TestRemoteWSLExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLExtensionStep("golang.go", "", nil, nil)
	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "vscode:wsl:setup", deps[0].String())
}

func TestRemoteWSLExtensionStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl", "--list-extensions"}, ports.CommandResult{
		Stdout:   "golang.go\nms-python.python\n",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("golang.go", "", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestRemoteWSLExtensionStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl", "--list-extensions"}, ports.CommandResult{
		Stdout:   "ms-python.python\n",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("golang.go", "", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestRemoteWSLExtensionStep_Check_WithDistro(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl+Ubuntu-22.04", "--list-extensions"}, ports.CommandResult{
		Stdout:   "golang.go\n",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("golang.go", "Ubuntu-22.04", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestRemoteWSLExtensionStep_Check_WSL(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// When in WSL, uses code.exe and the WSL distro from platform
	runner.AddResult("code.exe", []string{"--remote", "wsl+Ubuntu", "--list-extensions"}, ports.CommandResult{
		Stdout:   "golang.go\n",
		ExitCode: 0,
	})

	plat := platform.NewWSL(platform.EnvWSL2, "Ubuntu", "/mnt/c")
	step := NewRemoteWSLExtensionStep("golang.go", "", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestRemoteWSLExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLExtensionStep("golang.go", "", nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Contains(t, diff.Summary(), "golang.go")
}

func TestRemoteWSLExtensionStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl", "--install-extension", "golang.go", "--force"}, ports.CommandResult{
		Stdout:   "Extension installed.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("golang.go", "", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestRemoteWSLExtensionStep_Apply_WithDistro(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl+Ubuntu-22.04", "--install-extension", "golang.go", "--force"}, ports.CommandResult{
		Stdout:   "Extension installed.",
		ExitCode: 0,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("golang.go", "Ubuntu-22.04", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestRemoteWSLExtensionStep_Apply_Failed(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--remote", "wsl", "--install-extension", "invalid-ext", "--force"}, ports.CommandResult{
		Stderr:   "Extension not found.",
		ExitCode: 1,
	})

	plat := platform.New(platform.OSWindows, "amd64", platform.EnvNative)
	step := NewRemoteWSLExtensionStep("invalid-ext", "", runner, plat)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install extension")
}

func TestRemoteWSLExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLExtensionStep("golang.go", "", nil, nil)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Equal(t, "Install WSL Remote Extension", exp.Summary())
	assert.Contains(t, exp.Detail(), "golang.go")
	assert.NotEmpty(t, exp.DocLinks())
	assert.NotEmpty(t, exp.Tradeoffs())
}

// --- RemoteWSLSettingsStep Tests ---

func TestRemoteWSLSettingsStep_ID(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSettingsStep(nil, "", nil, nil)
	assert.Equal(t, "vscode:wsl:settings", step.ID().String())
}

func TestRemoteWSLSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSettingsStep(nil, "", nil, nil)
	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "vscode:wsl:setup", deps[0].String())
}

func TestRemoteWSLSettingsStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	settings := map[string]interface{}{
		"terminal.integrated.shell.linux": "/bin/bash",
	}
	step := NewRemoteWSLSettingsStep(settings, "", fs, nil)
	ctx := compiler.NewRunContext(context.Background())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestRemoteWSLSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	settings := map[string]interface{}{
		"terminal.integrated.shell.linux": "/bin/bash",
	}
	step := NewRemoteWSLSettingsStep(settings, "", nil, nil)
	ctx := compiler.NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestRemoteWSLSettingsStep_Apply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	settings := map[string]interface{}{
		"terminal.integrated.shell.linux": "/bin/bash",
	}
	step := NewRemoteWSLSettingsStep(settings, "", fs, nil)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify settings were written
	content, err := fs.ReadFile(step.getSettingsPath())
	require.NoError(t, err)
	assert.Contains(t, string(content), "terminal.integrated.shell.linux")
}

func TestRemoteWSLSettingsStep_Apply_MergesWithExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Write existing settings
	existingSettings := `{
		"editor.fontSize": 14
	}`
	_ = fs.WriteFile(ports.ExpandPath("~/.config/Code/User/settings.json"), []byte(existingSettings), 0o644)

	newSettings := map[string]interface{}{
		"terminal.integrated.shell.linux": "/bin/bash",
	}
	step := NewRemoteWSLSettingsStep(newSettings, "", fs, nil)
	ctx := compiler.NewRunContext(context.Background())

	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify both settings are present
	content, err := fs.ReadFile(step.getSettingsPath())
	require.NoError(t, err)
	assert.Contains(t, string(content), "editor.fontSize")
	assert.Contains(t, string(content), "terminal.integrated.shell.linux")
}

func TestRemoteWSLSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	step := NewRemoteWSLSettingsStep(nil, "", nil, nil)
	ctx := compiler.NewExplainContext()

	exp := step.Explain(ctx)
	assert.Equal(t, "Configure WSL Remote Settings", exp.Summary())
	assert.Contains(t, exp.Detail(), "WSL")
	assert.NotEmpty(t, exp.DocLinks())
	assert.NotEmpty(t, exp.Tradeoffs())
}
