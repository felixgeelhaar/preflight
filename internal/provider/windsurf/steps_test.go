package windsurf_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/windsurf"
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
	step := windsurf.NewExtensionStep("golang.go", runner)

	assert.Equal(t, "windsurf:extension:golang_go", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := windsurf.NewExtensionStep("golang.go", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestExtensionStep_Check_Satisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("windsurf", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "golang.go\nrust-lang.rust-analyzer\n",
	})

	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestExtensionStep_Check_NeedsApply(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("windsurf", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "rust-lang.rust-analyzer\n",
	})

	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Check_CommandError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("windsurf", []string{"--list-extensions"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "windsurf not found",
	})

	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "extension", diff.Resource())
	assert.Equal(t, "golang.go", diff.Name())
}

func TestExtensionStep_Apply_Success(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("windsurf", []string{"--install-extension", "golang.go"}, ports.CommandResult{
		ExitCode: 0,
	})

	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)
}

func TestExtensionStep_Apply_Failure(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("windsurf", []string{"--install-extension", "golang.go"}, ports.CommandResult{
		ExitCode: 1,
		Stderr:   "extension not found",
	})

	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extension not found")
}

func TestExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := windsurf.NewExtensionStep("golang.go", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install Windsurf Extension", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "golang.go")
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
	step := windsurf.NewSettingsStep(settings, runner)

	assert.Equal(t, "windsurf:settings", step.ID().String())
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := windsurf.NewSettingsStep(settings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupWindsurfConfigPath(t, tmpDir)

	// Create existing settings file
	configPath := getWindsurfConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"editor.fontSize": 14}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := windsurf.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSettingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupWindsurfConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := windsurf.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_NeedsApply_DifferentValue(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupWindsurfConfigPath(t, tmpDir)

	// Create existing settings with different value
	configPath := getWindsurfConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	existingSettings := map[string]interface{}{"editor.fontSize": 12}
	data, _ := json.Marshal(existingSettings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := windsurf.NewSettingsStep(settings, runner)

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
	step := windsurf.NewSettingsStep(settings, runner)

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
	setupWindsurfConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.tabSize":  4,
	}
	step := windsurf.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were written
	configPath := getWindsurfConfigPathForTest(tmpDir)
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
	step := windsurf.NewSettingsStep(settings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Windsurf Settings", explanation.Summary())
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
	keybindings := []windsurf.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	assert.Equal(t, "windsurf:keybindings", step.ID().String())
}

func TestKeybindingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []windsurf.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestKeybindingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupWindsurfConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []windsurf.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Check_NeedsApply_FileExists(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupWindsurfConfigPath(t, tmpDir)

	// Create existing keybindings file
	configPath := getWindsurfConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configPath, "keybindings.json"), []byte("[]"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	keybindings := []windsurf.Keybinding{{Key: "ctrl+p", Command: "quickOpen"}}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// Always returns NeedsApply when keybindings are defined
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []windsurf.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands"},
	}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

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
	setupWindsurfConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []windsurf.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands", When: "editorFocus"},
	}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify keybindings were written
	configPath := getWindsurfConfigPathForTest(tmpDir)
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
	keybindings := []windsurf.Keybinding{
		{Key: "ctrl+p", Command: "quickOpen"},
		{Key: "ctrl+shift+p", Command: "showCommands"},
	}
	step := windsurf.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Windsurf Keybindings", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 custom keybindings")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupWindsurfConfigPath(t *testing.T, tmpDir string) {
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

func getWindsurfConfigPathForTest(tmpDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(tmpDir, "Library", "Application Support", "Windsurf", "User")
	case "linux":
		return filepath.Join(tmpDir, ".config", "Windsurf", "User")
	default: // windows
		return filepath.Join(tmpDir, "Windsurf", "User")
	}
}
