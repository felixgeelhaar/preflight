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
