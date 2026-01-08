package zed_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/zed"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zedConfigDir returns the platform-appropriate Zed config directory relative to homeDir.
// This matches the Discovery.BestPracticePath() behavior for tests.
func zedConfigDir(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Zed")
	default:
		return filepath.Join(homeDir, ".config", "zed")
	}
}

// =============================================================================
// ExtensionStep Tests
// =============================================================================

func TestExtensionStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	assert.Equal(t, "zed:extension:python", step.ID().String())
}

func TestExtensionStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestExtensionStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create extension directory using platform-appropriate path
	extPath := filepath.Join(zedConfigDir(tmpDir), "extensions", "installed", "python")
	err := os.MkdirAll(extPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestExtensionStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestExtensionStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeAdd, diff.Type())
	assert.Equal(t, "extension", diff.Resource())
	assert.Equal(t, "python", diff.Name())
}

func TestExtensionStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify extension was added to auto_install_extensions
	settingsPath := filepath.Join(zedConfigDir(tmpDir), "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	err = json.Unmarshal(data, &settings)
	require.NoError(t, err)

	extensions, ok := settings["auto_install_extensions"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, extensions, "python")
}

func TestExtensionStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewExtensionStep("python", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Install Zed Extension", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "python")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// SettingsStep Tests
// =============================================================================

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"tab_size": 4}
	step := zed.NewSettingsStep(settings, runner)

	assert.Equal(t, "zed:settings", step.ID().String())
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"tab_size": 4}
	step := zed.NewSettingsStep(settings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing settings file using platform-appropriate path
	configPath := zedConfigDir(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"tab_size": 4}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := zed.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSettingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"tab_size": 4}
	step := zed.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"tab_size": 4,
		"vim_mode": true,
	}
	step := zed.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "settings", diff.Resource())
	assert.Contains(t, diff.NewValue(), "2 settings")
}

func TestSettingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"tab_size": 4,
		"vim_mode": true,
	}
	step := zed.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were written using platform-appropriate path
	settingsPath := filepath.Join(zedConfigDir(tmpDir), "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var written map[string]interface{}
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.InDelta(t, float64(4), written["tab_size"], 0.001)
	assert.Equal(t, true, written["vim_mode"])
}

func TestSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"tab_size": 4,
		"vim_mode": true,
	}
	step := zed.NewSettingsStep(settings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Zed Settings", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 settings")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// KeymapStep Tests
// =============================================================================

func TestKeymapStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}}}
	step := zed.NewKeymapStep(keymap, runner)

	assert.Equal(t, "zed:keymap", step.ID().String())
}

func TestKeymapStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}}}
	step := zed.NewKeymapStep(keymap, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestKeymapStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}}}
	step := zed.NewKeymapStep(keymap, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeymapStep_Check_NeedsApply_FileExists(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing keymap file using platform-appropriate path
	configPath := zedConfigDir(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(configPath, "keymap.json"), []byte("[]"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}}}
	step := zed.NewKeymapStep(keymap, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeymapStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{
		{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}},
		{Context: "Workspace", Bindings: map[string]string{"ctrl-q": "zed::Quit"}},
	}
	step := zed.NewKeymapStep(keymap, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "keymap", diff.Resource())
	assert.Contains(t, diff.NewValue(), "2 contexts")
}

func TestKeymapStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{
		{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}},
		{Bindings: map[string]string{"ctrl-q": "zed::Quit"}},
	}
	step := zed.NewKeymapStep(keymap, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify keymap was written using platform-appropriate path
	keymapPath := filepath.Join(zedConfigDir(tmpDir), "keymap.json")
	data, err := os.ReadFile(keymapPath)
	require.NoError(t, err)

	var written []map[string]interface{}
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.Len(t, written, 2)
	assert.Equal(t, "Editor", written[0]["context"])
	_, hasContext := written[1]["context"]
	assert.False(t, hasContext) // Second binding has no context
}

func TestKeymapStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keymap := []zed.KeyBinding{
		{Context: "Editor", Bindings: map[string]string{"ctrl-p": "file_finder::Toggle"}},
		{Context: "Workspace", Bindings: map[string]string{"ctrl-q": "zed::Quit"}},
	}
	step := zed.NewKeymapStep(keymap, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Zed Keymap", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "2 contexts")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ThemeStep Tests
// =============================================================================

func TestThemeStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	assert.Equal(t, "zed:theme:one-dark", step.ID().String())
}

func TestThemeStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestThemeStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing settings with theme using platform-appropriate path
	configPath := zedConfigDir(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"theme": "one-dark"}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestThemeStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestThemeStep_Check_NeedsApply_DifferentTheme(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create existing settings with different theme using platform-appropriate path
	configPath := zedConfigDir(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"theme": "solarized-light"}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "settings.json"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestThemeStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "theme", diff.Resource())
	assert.Equal(t, "one-dark", diff.NewValue())
}

func TestThemeStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify theme was written using platform-appropriate path
	settingsPath := filepath.Join(zedConfigDir(tmpDir), "settings.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	var settings map[string]interface{}
	err = json.Unmarshal(data, &settings)
	require.NoError(t, err)
	assert.Equal(t, "one-dark", settings["theme"])
}

func TestThemeStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := zed.NewThemeStep("one-dark", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set Zed Theme", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "one-dark")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}
