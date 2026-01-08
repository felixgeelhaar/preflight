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
// Discovery Tests
// =============================================================================

func TestNewDiscoveryWithOS(t *testing.T) {
	t.Parallel()

	d := sublime.NewDiscoveryWithOS("darwin")
	assert.NotNil(t, d)

	dLinux := sublime.NewDiscoveryWithOS("linux")
	assert.NotNil(t, dLinux)

	dWindows := sublime.NewDiscoveryWithOS("windows")
	assert.NotNil(t, dWindows)
}

func TestSearchOpts(t *testing.T) {
	t.Parallel()

	opts := sublime.SearchOpts()
	assert.Equal(t, "SUBLIME_DATA", opts.EnvVar)
	assert.Contains(t, opts.ConfigFileName, "Preferences.sublime-settings")
	assert.NotEmpty(t, opts.MacOSPaths)
	assert.NotEmpty(t, opts.LinuxPaths)
	assert.NotEmpty(t, opts.WindowsPaths)
}

func TestDiscovery_BestPracticePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Test darwin
	d := sublime.NewDiscoveryWithOS("darwin")
	path := d.BestPracticePath()
	assert.Contains(t, path, "Sublime Text")
	assert.Contains(t, path, "Library/Application Support")

	// Test linux
	dLinux := sublime.NewDiscoveryWithOS("linux")
	pathLinux := dLinux.BestPracticePath()
	assert.Contains(t, pathLinux, "sublime-text")
	assert.Contains(t, pathLinux, ".config")

	// Test windows
	t.Setenv("APPDATA", tmpDir)
	dWindows := sublime.NewDiscoveryWithOS("windows")
	pathWindows := dWindows.BestPracticePath()
	assert.Contains(t, pathWindows, "Sublime Text")

	// Test windows with empty APPDATA
	t.Setenv("APPDATA", "")
	pathWindowsNoAppData := dWindows.BestPracticePath()
	assert.Contains(t, pathWindowsNoAppData, "Sublime Text")

	// Test unknown OS
	dUnknown := sublime.NewDiscoveryWithOS("freebsd")
	pathUnknown := dUnknown.BestPracticePath()
	assert.Contains(t, pathUnknown, "sublime-text")
}

func TestDiscovery_FindConfigDir_WithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config via env var path
	sublimeDataDir := filepath.Join(tmpDir, "sublime-data")
	userDir := filepath.Join(sublimeDataDir, "Packages", "User")
	err := os.MkdirAll(userDir, 0o755)
	require.NoError(t, err)

	t.Setenv("SUBLIME_DATA", sublimeDataDir)
	t.Setenv("HOME", tmpDir)

	d := sublime.NewDiscovery()
	path := d.FindConfigDir()
	assert.Equal(t, userDir, path)
}

func TestDiscovery_FindConfigDir_Linux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	// Create sublime-text directory
	sublimeDir := filepath.Join(tmpDir, ".config", "sublime-text", "Packages", "User")
	err := os.MkdirAll(sublimeDir, 0o755)
	require.NoError(t, err)

	d := sublime.NewDiscoveryWithOS("linux")
	path := d.FindConfigDir()
	assert.Equal(t, sublimeDir, path)
}

func TestDiscovery_FindConfigDir_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	// Create sublime-text directory
	sublimeDir := filepath.Join(tmpDir, "Sublime Text", "Packages", "User")
	err := os.MkdirAll(sublimeDir, 0o755)
	require.NoError(t, err)

	d := sublime.NewDiscoveryWithOS("windows")
	path := d.FindConfigDir()
	assert.Equal(t, sublimeDir, path)
}

func TestDiscovery_FindConfigDir_WindowsNoAppData(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", "")
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	d := sublime.NewDiscoveryWithOS("windows")
	path := d.FindConfigDir()
	// Should fall back to AppData/Roaming path
	assert.Contains(t, path, "Sublime Text")
}

func TestDiscovery_FindConfigDir_UnknownOS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	d := sublime.NewDiscoveryWithOS("freebsd")
	path := d.FindConfigDir()
	assert.Contains(t, path, "sublime-text")
}

func TestDiscovery_FindKeybindingsPath_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("APPDATA", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	d := sublime.NewDiscoveryWithOS("windows")
	path := d.FindKeybindingsPath()
	assert.Contains(t, path, "Default (Windows).sublime-keymap")
}

func TestDiscovery_FindKeybindingsPath_UnknownOS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SUBLIME_DATA", "")

	d := sublime.NewDiscoveryWithOS("freebsd")
	path := d.FindKeybindingsPath()
	assert.Contains(t, path, "Default (Linux).sublime-keymap")
}

func TestDiscovery_GetCandidatePaths(t *testing.T) {
	t.Parallel()

	d := sublime.NewDiscovery()
	paths := d.GetCandidatePaths()
	assert.NotEmpty(t, paths)
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
