package sublime_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/sublime"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PackagesStep Tests
// =============================================================================

func TestPackagesStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control"}, runner)

	assert.Equal(t, "sublime:packages", step.ID().String())
}

func TestPackagesStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control"}, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestPackagesStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control", "SublimeLinter"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPackagesStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	// Create Package Control settings with packages already installed
	configPath := getSublimeConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	pkgControlConfig := map[string]interface{}{
		"installed_packages": []string{"Package Control", "SublimeLinter"},
	}
	data, _ := json.Marshal(pkgControlConfig)
	err = os.WriteFile(filepath.Join(configPath, "Package Control.sublime-settings"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control", "SublimeLinter"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPackagesStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control", "SublimeLinter"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "packages", diff.Resource())
	assert.Contains(t, diff.NewValue(), "2 packages")
}

func TestPackagesStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control", "SublimeLinter"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify packages were written
	configPath := getSublimeConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "Package Control.sublime-settings"))
	require.NoError(t, err)

	var written map[string]interface{}
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)

	installedPkgs, ok := written["installed_packages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, installedPkgs, 2)
}

func TestPackagesStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := sublime.NewPackagesStep([]string{"Package Control"}, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Sublime Text Packages", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// SettingsStep Tests
// =============================================================================

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"font_size": 14}
	step := sublime.NewSettingsStep(settings, "", "", runner)

	assert.Equal(t, "sublime:settings", step.ID().String())
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"font_size": 14}
	step := sublime.NewSettingsStep(settings, "", "", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"font_size": 14}
	step := sublime.NewSettingsStep(settings, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	// Create existing settings
	configPath := getSublimeConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	settings := map[string]interface{}{"font_size": 14}
	data, _ := json.Marshal(settings)
	err = os.WriteFile(filepath.Join(configPath, "Preferences.sublime-settings"), data, 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := sublime.NewSettingsStep(settings, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"font_size": 14,
		"tab_size":  4,
	}
	step := sublime.NewSettingsStep(settings, "Adaptive.sublime-theme", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "settings", diff.Resource())
	assert.Contains(t, diff.NewValue(), "3 settings") // 2 settings + 1 theme
}

func TestSettingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{
		"font_size": 14,
		"tab_size":  4,
	}
	step := sublime.NewSettingsStep(settings, "Adaptive.sublime-theme", "Packages/Catppuccin/Mocha.tmTheme", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings were written
	configPath := getSublimeConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "Preferences.sublime-settings"))
	require.NoError(t, err)

	var written map[string]interface{}
	err = json.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.InDelta(t, float64(14), written["font_size"], 0.001)
	assert.Equal(t, "Adaptive.sublime-theme", written["theme"])
	assert.Equal(t, "Packages/Catppuccin/Mocha.tmTheme", written["color_scheme"])
}

func TestSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"font_size": 14}
	step := sublime.NewSettingsStep(settings, "", "", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Sublime Text Settings", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// KeybindingsStep Tests
// =============================================================================

func TestKeybindingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{{Keys: []string{"ctrl+p"}, Command: "show_overlay"}}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	assert.Equal(t, "sublime:keybindings", step.ID().String())
}

func TestKeybindingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{{Keys: []string{"ctrl+p"}, Command: "show_overlay"}}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestKeybindingsStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{{Keys: []string{"ctrl+p"}, Command: "show_overlay"}}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKeybindingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{
		{Keys: []string{"ctrl+p"}, Command: "show_overlay"},
		{Keys: []string{"ctrl+shift+p"}, Command: "show_overlay"},
	}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "keybindings", diff.Resource())
	assert.Contains(t, diff.NewValue(), "2 keybindings")
}

func TestKeybindingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupSublimeConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{
		{
			Keys:    []string{"ctrl+p"},
			Command: "show_overlay",
			Args:    map[string]interface{}{"overlay": "goto", "show_files": true},
		},
		{
			Keys:    []string{"ctrl+shift+p"},
			Command: "show_overlay",
			Args:    map[string]interface{}{"overlay": "command_palette"},
			Context: []sublime.KeyContext{
				{Key: "setting.is_widget", Operator: "equal", Operand: false},
			},
		},
	}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify keybindings were written - check if file exists
	configPath := getSublimeConfigPathForTest(tmpDir)
	files, err := os.ReadDir(configPath)
	require.NoError(t, err)

	// Should have at least one keymap file
	var foundKeymap bool
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sublime-keymap" {
			foundKeymap = true
			data, err := os.ReadFile(filepath.Join(configPath, f.Name()))
			require.NoError(t, err)

			var written []map[string]interface{}
			err = json.Unmarshal(data, &written)
			require.NoError(t, err)
			assert.Len(t, written, 2)
			break
		}
	}
	assert.True(t, foundKeymap, "should have created a keymap file")
}

func TestKeybindingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	keybindings := []sublime.Keybinding{{Keys: []string{"ctrl+p"}, Command: "show_overlay"}}
	step := sublime.NewKeybindingsStep(keybindings, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Sublime Text Keybindings", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupSublimeConfigPath(t *testing.T, tmpDir string) {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", tmpDir)
	case "linux":
		t.Setenv("HOME", tmpDir)
	default: // windows
		t.Setenv("APPDATA", tmpDir)
	}
	t.Setenv("SUBLIME_DATA", "")
}

func getSublimeConfigPathForTest(tmpDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(tmpDir, "Library", "Application Support", "Sublime Text", "Packages", "User")
	case "linux":
		return filepath.Join(tmpDir, ".config", "sublime-text", "Packages", "User")
	default: // windows
		return filepath.Join(tmpDir, "Sublime Text", "Packages", "User")
	}
}
