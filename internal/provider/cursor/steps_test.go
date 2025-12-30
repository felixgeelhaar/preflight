package cursor_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/cursor"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ExtensionStep Tests
// =============================================================================

func TestExtensionStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := cursor.NewExtensionStep("ms-python.python", runner)

	assert.Equal(t, "cursor:extension:ms-python.python", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := cursor.NewExtensionStep("ms-python.python", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestExtensionStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("cursor", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "ms-python.python\ngolang.go\n",
	})

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestExtensionStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("cursor", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "golang.go\n",
	})

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Check_CommandError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("cursor", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "cursor not found",
	})

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "extension", diff.Resource())
	assert.Equal(t, "ms-python.python", diff.Name())
}

func TestExtensionStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("cursor", []string{"--install-extension", "ms-python.python"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestExtensionStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("cursor", []string{"--install-extension", "ms-python.python"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "extension not found",
	})

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension not found")
}

func TestExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install Cursor Extension", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "ms-python.python")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// SettingsStep Tests
// =============================================================================

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	assert.Equal(t, "cursor:settings", step.ID().String())
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	// Create existing settings file
	configPath := getCursorConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"editor.fontSize": 14}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSettingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_NeedsApply_DifferentValue(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	// Create existing settings with different value
	configPath := getCursorConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	existingSettings := map[string]interface{}{"editor.fontSize": 12}
	data, _ := json.Marshal(existingSettings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.tabSize":  4,
	}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "settings", diff.Resource())
	assert.Equal(t, "settings.json", diff.Name())
	assert.Contains(t, diff.NewValue(), "2 settings")
}

func TestSettingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.tabSize":  4,
	}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were written
	configPath := getCursorConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "settings.json"))
	require.NoError(t, err)

	var written map[string]interface{}
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.InDelta(t, float64(14), written["editor.fontSize"], 0.001)
}

func TestSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.tabSize":  4,
	}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Cursor Settings", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 settings")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// KeybindingsStep Tests
// =============================================================================

func TestKeybindingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	assert.Equal(t, "cursor:keybindings", step.ID().String())
}

func TestKeybindingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestKeybindingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Check_NeedsApply_FileExists(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	// Create existing keybindings file
	configPath := getCursorConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configPath, "keybindings.json"), []byte("[]"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// Always returns NeedsApply when keybindings are defined
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands"},
	}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "keybindings", diff.Resource())
	assert.Equal(t, "keybindings.json", diff.Name())
	assert.Contains(t, diff.NewValue(), "2 keybindings")
}

func TestKeybindingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands", When: "editorFocus"},
	}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify keybindings were written
	configPath := getCursorConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "keybindings.json"))
	require.NoError(t, err)

	var written []map[string]string
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.Len(t, written, 2)
	assert.Equal(t, "ctrl+p", written[0]["key"])
	assert.Equal(t, "editorFocus", written[1]["when"])
}

func TestKeybindingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []cursor.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands"},
	}
	step := cursor.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Cursor Keybindings", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 custom keybindings")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupCursorConfigPath(t *testing.T, tmpDir string) {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", tmpDir)
	case "linux":
		t.Setenv("HOME", tmpDir)
	default: // windows
		t.Setenv("APPDATA", tmpDir)
	}
}

func getCursorConfigPathForTest(tmpDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(tmpDir, "Library", "Application Support", "Cursor", "User")
	case "linux":
		return filepath.Join(tmpDir, ".config", "Cursor", "User")
	default: // windows
		return filepath.Join(tmpDir, "Cursor", "User")
	}
}
