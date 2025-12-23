package vscode_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionStep_ID(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)

	// Dots are replaced with underscores in step ID
	assert.Equal(t, "vscode:extension:ms-python_python", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)

	assert.Empty(t, step.DependsOn())
}

func TestExtensionStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	// Empty list response means extension not installed
	runner.AddResult("code", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "golang.go\n",
	})
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Check_Installed(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	runner.AddResult("code", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ms-python.python\ngolang.go\n",
	})
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Contains(t, diff.Summary(), "ms-python.python")
}

func TestExtensionStep_Apply(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	runner.AddResult("code", []string{"--install-extension", "ms-python.python", "--force"}, ports.CommandResult{
		ExitCode: 0,
	})
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	runner := ports.NewMockCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	assert.NotEmpty(t, explanation.Summary())
	assert.Contains(t, explanation.Detail(), "ms-python.python")
}

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)

	assert.Equal(t, "vscode:settings", step.ID().String())
}

func TestSettingsStep_Check_NotExists(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestKeybindingsStep_ID(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)

	assert.Equal(t, "vscode:keybindings", step.ID().String())
}

func TestKeybindingsStep_Check_NotExists(t *testing.T) {
	t.Parallel()

	fs := ports.NewMockFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}
