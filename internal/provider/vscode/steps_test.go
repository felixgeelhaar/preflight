package vscode_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/vscode"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)

	// Dots are replaced with underscores in step ID
	assert.Equal(t, "vscode:extension:ms-python_python", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)

	assert.Empty(t, step.DependsOn())
}

func TestExtensionStep_Check_NotInstalled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
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

	runner := mocks.NewCommandRunner()
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

	runner := mocks.NewCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Contains(t, diff.Summary(), "ms-python.python")
}

func TestExtensionStep_Apply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
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

	runner := mocks.NewCommandRunner()
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	assert.NotEmpty(t, explanation.Summary())
	assert.Contains(t, explanation.Detail(), "ms-python.python")
}

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)

	assert.Equal(t, "vscode:settings", step.ID().String())
}

func TestSettingsStep_Check_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestKeybindingsStep_ID(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)

	assert.Equal(t, "vscode:keybindings", step.ID().String())
}

func TestKeybindingsStep_Check_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Check_Error(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// Don't add any result - the mock will return an error
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestExtensionStep_Apply_Error(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// Don't add any result - the mock will return an error
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
}

func TestExtensionStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("code", []string{"--install-extension", "ms-python.python", "--force"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "Extension not found",
	})
	step := vscode.NewExtensionStep("ms-python.python", runner)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)

	assert.Empty(t, step.DependsOn())
}

func TestSettingsStep_Check_Exists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Add existing settings file using ExpandPath equivalent
	fs.AddFile("/Users/felixgeelhaar/.config/Code/User/settings.json", `{"editor.fontSize": 12}`)
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	// Since settings exists but we need to apply changes
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Apply_NewFile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := vscode.NewSettingsStep(settings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestSettingsStep_Apply_MergeExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Add existing settings
	fs.AddFile("/Users/felixgeelhaar/.config/Code/User/settings.json", `{"editor.tabSize": 4}`)
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := vscode.NewSettingsStep(settings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := vscode.NewSettingsStep(map[string]interface{}{"editor.fontSize": 14}, fs)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	assert.NotEmpty(t, explanation.Summary())
	assert.Contains(t, explanation.Detail(), "settings")
}

func TestKeybindingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)

	assert.Empty(t, step.DependsOn())
}

func TestKeybindingsStep_Check_Exists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	fs.AddFile("/Users/felixgeelhaar/.config/Code/User/keybindings.json", `[]`)
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Plan(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestKeybindingsStep_Apply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewRunContext(context.TODO())

	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestKeybindingsStep_Explain(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	keybindings := []vscode.Keybinding{
		{Key: "ctrl+shift+p", Command: "workbench.action.showCommands"},
	}
	step := vscode.NewKeybindingsStep(keybindings, fs)
	ctx := compiler.NewExplainContext()

	explanation := step.Explain(ctx)
	assert.NotEmpty(t, explanation.Summary())
	assert.Contains(t, explanation.Detail(), "keybindings")
}
