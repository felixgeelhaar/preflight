package jetbrains_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/jetbrains"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PluginStep Tests
// =============================================================================

func TestPluginStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"plugin1"}, runner)

	assert.Equal(t, "jetbrains:goland:plugins", step.ID().String())
}

func TestPluginStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"plugin1"}, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestPluginStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"com.wakatime.intellij"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestPluginStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"plugin1", "plugin2"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "plugins", diff.Resource())
	assert.Contains(t, diff.NewValue(), "2 plugins")
}

func TestPluginStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"com.wakatime.intellij", "rainbow-brackets"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify plugins file was created
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEGoLand)
	optionsPath := filepath.Join(configPath, "options")
	pluginsPath := filepath.Join(optionsPath, "preflight-plugins.txt")

	data, err := os.ReadFile(pluginsPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "com.wakatime.intellij")
	assert.Contains(t, string(data), "rainbow-brackets")
}

func TestPluginStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"plugin1"}, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Summary(), "GoLand")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// SettingsStep Tests
// =============================================================================

func TestSettingsStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "VSCode", "", runner)

	assert.Equal(t, "jetbrains:intellijidea:settings", step.ID().String())
}

func TestSettingsStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "", "", runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"fontSize": 14}
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, settings, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"fontSize": 14}
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, settings, "VSCode", "Google", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "3 settings") // 1 setting + keymap + code style
}

func TestSettingsStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"fontSize": 14}
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, settings, "VSCode", "GoogleStyle", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify config files were created
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	optionsPath := filepath.Join(configPath, "options")

	// Check keymap.xml
	keymapPath := filepath.Join(optionsPath, "keymap.xml")
	keymapData, err := os.ReadFile(keymapPath)
	require.NoError(t, err)
	assert.Contains(t, string(keymapData), "VSCode")

	// Check code.style.schemes.xml
	codeStylePath := filepath.Join(optionsPath, "code.style.schemes.xml")
	codeStyleData, err := os.ReadFile(codeStylePath)
	require.NoError(t, err)
	assert.Contains(t, string(codeStyleData), "GoogleStyle")

	// Check preflight.xml
	settingsPath := filepath.Join(optionsPath, "preflight.xml")
	settingsData, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	assert.Contains(t, string(settingsData), "fontSize")
}

func TestSettingsStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "", "", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Summary(), "IntelliJIdea")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// SettingsSyncStep Tests
// =============================================================================

func TestSettingsSyncStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	assert.Equal(t, "jetbrains:webstorm:settingssync", step.ID().String())
}

func TestSettingsSyncStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestSettingsSyncStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsSyncStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "enabled")
}

func TestSettingsSyncStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{
		Enabled:        true,
		SyncPlugins:    true,
		SyncUI:         true,
		SyncCodeStyles: true,
		SyncKeymaps:    false,
	}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify settings sync file was created
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEWebStorm)
	settingsSyncPath := filepath.Join(configPath, "options", "settingsSync.xml")

	data, err := os.ReadFile(settingsSyncPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "syncEnabled")
	assert.Contains(t, content, "true")
}

func TestSettingsSyncStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Summary(), "WebStorm")
	assert.Contains(t, explanation.Summary(), "Settings Sync")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Additional PluginStep Tests
// =============================================================================

func TestPluginStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	// Create plugins directory with installed plugins (under config path)
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEGoLand)
	pluginsDir := filepath.Join(configPath, "plugins")
	err := os.MkdirAll(filepath.Join(pluginsDir, "com.wakatime.intellij"), 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"wakatime"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// Plugin matching is case-insensitive and partial
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestPluginStep_Check_NeedsApply_NoPluginsDir(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewPluginStep(jetbrains.IDEGoLand, []string{"com.wakatime.intellij"}, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// =============================================================================
// Additional SettingsStep Tests
// =============================================================================

func TestSettingsStep_Check_Satisfied_NoSettings(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	// Create options directory
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	optionsPath := filepath.Join(configPath, "options")
	err := os.MkdirAll(optionsPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	// No settings, keymap, or codeStyle
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestSettingsStep_Check_NeedsApply_NoOptionsDir(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"fontSize": 14}
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, settings, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_NeedsApply_WithKeymap(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	// Create options directory
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	optionsPath := filepath.Join(configPath, "options")
	err := os.MkdirAll(optionsPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "VSCode", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_NeedsApply_WithCodeStyle(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	// Create options directory
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	optionsPath := filepath.Join(configPath, "options")
	err := os.MkdirAll(optionsPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "", "GoogleStyle", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Apply_OnlyKeymap(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "VSCode", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify keymap.xml was created
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	keymapPath := filepath.Join(configPath, "options", "keymap.xml")
	data, err := os.ReadFile(keymapPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "VSCode")
}

func TestSettingsStep_Apply_OnlyCodeStyle(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, nil, "", "GoogleStyle", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify code.style.schemes.xml was created
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	codeStylePath := filepath.Join(configPath, "options", "code.style.schemes.xml")
	data, err := os.ReadFile(codeStylePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "GoogleStyle")
}

func TestSettingsStep_Apply_OnlySettings(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"fontSize": 14, "lineSpacing": 1.2}
	step := jetbrains.NewSettingsStep(jetbrains.IDEIntelliJ, settings, "", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify preflight.xml was created with settings
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEIntelliJ)
	settingsPath := filepath.Join(configPath, "options", "preflight.xml")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "fontSize")
	assert.Contains(t, content, "lineSpacing")
}

// =============================================================================
// Additional SettingsSyncStep Tests
// =============================================================================

func TestSettingsSyncStep_Check_Satisfied_FileExists(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupJetBrainsConfigPath(t, tmpDir)

	// Create options directory with settings sync file
	configPath := getJetBrainsConfigPathForTest(tmpDir, jetbrains.IDEWebStorm)
	optionsPath := filepath.Join(configPath, "options")
	err := os.MkdirAll(optionsPath, 0o755)
	require.NoError(t, err)

	// Create existing settingsSync.xml
	settingsSyncPath := filepath.Join(optionsPath, "settingsSync.xml")
	err = os.WriteFile(settingsSyncPath, []byte("<application></application>"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	// Always returns NeedsApply to ensure config is correct
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsSyncStep_Plan_Disabled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: false}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "disabled")
}

func TestSettingsSyncStep_Plan_Enabled(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	config := &jetbrains.SettingsSyncConfig{Enabled: true}
	step := jetbrains.NewSettingsSyncStep(jetbrains.IDEWebStorm, config, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "enabled")
}

// =============================================================================
// Discovery Tests
// =============================================================================

func TestNewDiscoveryWithOS(t *testing.T) {
	t.Parallel()

	d := jetbrains.NewDiscoveryWithOS("linux")
	assert.NotNil(t, d)

	// Test that it works for different OS
	dDarwin := jetbrains.NewDiscoveryWithOS("darwin")
	assert.NotNil(t, dDarwin)

	dWindows := jetbrains.NewDiscoveryWithOS("windows")
	assert.NotNil(t, dWindows)
}

func TestSearchOpts(t *testing.T) {
	t.Parallel()

	opts := jetbrains.SearchOpts(jetbrains.IDEGoLand)
	assert.Equal(t, "", opts.EnvVar)
	assert.Contains(t, opts.MacOSPaths[0], "GoLand")
	assert.Contains(t, opts.LinuxPaths[0], "GoLand")
	assert.Contains(t, opts.WindowsPaths[0], "GoLand")

	// Test with different IDE
	opts2 := jetbrains.SearchOpts(jetbrains.IDEPyCharm)
	assert.Contains(t, opts2.MacOSPaths[0], "PyCharm")
}

func TestDiscovery_BestPracticePath(t *testing.T) {
	// Can't use t.Parallel() because we modify HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := jetbrains.NewDiscoveryWithOS("darwin")
	path := d.BestPracticePath(jetbrains.IDEGoLand)
	assert.Contains(t, path, "GoLand")
	assert.Contains(t, path, "JetBrains")

	// Test for linux
	dLinux := jetbrains.NewDiscoveryWithOS("linux")
	pathLinux := dLinux.BestPracticePath(jetbrains.IDEPyCharm)
	assert.Contains(t, pathLinux, "PyCharm")
	assert.Contains(t, pathLinux, ".config/JetBrains")

	// Test for windows
	dWindows := jetbrains.NewDiscoveryWithOS("windows")
	t.Setenv("APPDATA", tmpDir)
	pathWindows := dWindows.BestPracticePath(jetbrains.IDEWebStorm)
	assert.Contains(t, pathWindows, "WebStorm")

	// Test with empty APPDATA
	t.Setenv("APPDATA", "")
	pathWindowsNoAppData := dWindows.BestPracticePath(jetbrains.IDEWebStorm)
	assert.Contains(t, pathWindowsNoAppData, "WebStorm")

	// Test for unknown OS
	dUnknown := jetbrains.NewDiscoveryWithOS("freebsd")
	pathUnknown := dUnknown.BestPracticePath(jetbrains.IDEGoLand)
	assert.Contains(t, pathUnknown, "GoLand")
}

func TestDiscovery_FindCodeStylesDir(t *testing.T) {
	// Can't use t.Parallel() because we modify HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := jetbrains.NewDiscovery()
	path := d.FindCodeStylesDir(jetbrains.IDEGoLand)
	assert.Contains(t, path, "codestyles")
	assert.Contains(t, path, "GoLand")
}

func TestDiscovery_FindKeymapsDir(t *testing.T) {
	// Can't use t.Parallel() because we modify HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := jetbrains.NewDiscovery()
	path := d.FindKeymapsDir(jetbrains.IDEGoLand)
	assert.Contains(t, path, "keymaps")
	assert.Contains(t, path, "GoLand")
}

func TestDiscovery_GetInstalledIDEs(t *testing.T) {
	// Can't use t.Parallel() because we modify HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create JetBrains config directories for some IDEs
	var baseDir string
	switch runtime.GOOS {
	case "darwin":
		baseDir = filepath.Join(tmpDir, "Library", "Application Support", "JetBrains")
	case "windows":
		t.Setenv("APPDATA", tmpDir)
		baseDir = filepath.Join(tmpDir, "JetBrains")
	default:
		baseDir = filepath.Join(tmpDir, ".config", "JetBrains")
	}

	// Create GoLand and PyCharm directories
	err := os.MkdirAll(filepath.Join(baseDir, "GoLand2024.1"), 0o755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(baseDir, "PyCharm2024.1"), 0o755)
	require.NoError(t, err)

	d := jetbrains.NewDiscovery()
	installed := d.GetInstalledIDEs()

	assert.Len(t, installed, 2)
	// Check that both IDEs are found
	ideNames := make(map[jetbrains.IDE]bool)
	for _, ide := range installed {
		ideNames[ide] = true
	}
	assert.True(t, ideNames[jetbrains.IDEGoLand])
	assert.True(t, ideNames[jetbrains.IDEPyCharm])
}

func TestDiscovery_GetInstalledIDEs_NoDir(t *testing.T) {
	// Can't use t.Parallel() because we modify HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Don't create any JetBrains directories
	d := jetbrains.NewDiscovery()
	installed := d.GetInstalledIDEs()

	assert.Nil(t, installed)
}

func TestDiscovery_GetCandidatePaths(t *testing.T) {
	t.Parallel()

	d := jetbrains.NewDiscovery()
	paths := d.GetCandidatePaths(jetbrains.IDEGoLand)

	// Should have paths for all platforms
	assert.NotEmpty(t, paths)
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupJetBrainsConfigPath(t *testing.T, tmpDir string) {
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

func getJetBrainsConfigPathForTest(tmpDir string, ide jetbrains.IDE) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(tmpDir, "Library", "Application Support", "JetBrains", string(ide)+"2024.1")
	case "linux":
		return filepath.Join(tmpDir, ".config", "JetBrains", string(ide)+"2024.1")
	default: // windows
		return filepath.Join(tmpDir, "JetBrains", string(ide)+"2024.1")
	}
}
