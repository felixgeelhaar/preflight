package helix_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/helix"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ConfigStep Tests
// =============================================================================

func TestConfigStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("", false, nil, nil, nil, runner)

	assert.Equal(t, "helix:config", step.ID().String())
}

func TestConfigStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("", false, nil, nil, nil, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestConfigStep_Check_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"theme": "catppuccin_mocha"}
	step := helix.NewConfigStep("", false, settings, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Plan_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("dotfiles/config.toml", true, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "config", diff.Resource())
	assert.Contains(t, diff.NewValue(), "link")
}

func TestConfigStep_Plan_WithSettings(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"theme": "catppuccin_mocha"}
	step := helix.NewConfigStep("", false, settings, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "merge settings")
}

func TestConfigStep_Apply_WithSettings(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"theme": "catppuccin_mocha"}
	editor := map[string]interface{}{"line-number": "relative"}
	step := helix.NewConfigStep("", false, settings, editor, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify config was written
	configPath := getHelixConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "config.toml"))
	require.NoError(t, err)

	var written map[string]interface{}
	err = toml.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.Equal(t, "catppuccin_mocha", written["theme"])

	editorSection, ok := written["editor"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "relative", editorSection["line-number"])
}

func TestConfigStep_Apply_WithSource_Link(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, true, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify symlink was created
	configPath := getHelixConfigPathForTest(tmpDir)
	configFile := filepath.Join(configPath, "config.toml")

	info, err := os.Lstat(configFile)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")
}

func TestConfigStep_Apply_WithSource_Copy(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	sourceContent := "theme = \"gruvbox\"\n"
	err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, false, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify file was copied
	configPath := getHelixConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, sourceContent, string(data))
}

func TestConfigStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("", false, nil, nil, nil, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Helix", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// LanguagesStep Tests
// =============================================================================

func TestLanguagesStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("languages.toml", true, runner)

	assert.Equal(t, "helix:languages", step.ID().String())
}

func TestLanguagesStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("languages.toml", true, runner)

	deps := step.DependsOn()
	assert.Empty(t, deps)
}

func TestLanguagesStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("languages.toml", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestLanguagesStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("dotfiles/languages.toml", true, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "config", diff.Resource())
	assert.Contains(t, diff.NewValue(), "link")
}

func TestLanguagesStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	sourceContent := "[[language]]\nname = \"go\"\n"
	err := os.WriteFile(sourceFile, []byte(sourceContent), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, false, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify file was written
	configPath := getHelixConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "languages.toml"))
	require.NoError(t, err)
	assert.Equal(t, sourceContent, string(data))
}

func TestLanguagesStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("languages.toml", true, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Configure Helix Languages", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ThemeStep Tests
// =============================================================================

func TestThemeStep_ID(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	assert.Equal(t, "helix:theme", step.ID().String())
}

func TestThemeStep_DependsOn(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	deps := step.DependsOn()
	require.Len(t, deps, 1)
	assert.Equal(t, "helix:config", deps[0].String())
}

func TestThemeStep_Check_NeedsApply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestThemeStep_Plan(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "theme", diff.Resource())
	assert.Contains(t, diff.NewValue(), "catppuccin_mocha")
}

func TestThemeStep_Apply(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify theme was set in config
	configPath := getHelixConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "config.toml"))
	require.NoError(t, err)

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	require.NoError(t, err)
	assert.Equal(t, "catppuccin_mocha", config["theme"])
}

func TestThemeStep_Apply_WithCustomTheme(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create custom theme file
	themeSource := filepath.Join(tmpDir, "my_theme.toml")
	themeContent := "[\"ui.background\"]\nbg = \"#1e1e2e\"\n"
	err := os.WriteFile(themeSource, []byte(themeContent), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("my_theme", themeSource, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify theme was installed
	configPath := getHelixConfigPathForTest(tmpDir)
	themePath := filepath.Join(configPath, "themes", "my_theme.toml")
	data, err := os.ReadFile(themePath)
	require.NoError(t, err)
	assert.Equal(t, themeContent, string(data))
}

func TestThemeStep_Explain(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set Helix Theme", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "catppuccin_mocha")
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Additional ConfigStep Check Tests
// =============================================================================

func TestConfigStep_Check_Source_Link_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and symlink
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	absSource, _ := filepath.Abs(sourceFile)
	err = os.Symlink(absSource, configFile)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, true, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigStep_Check_Source_Link_WrongTarget(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	// Create a different file
	wrongTarget := filepath.Join(tmpDir, "wrong-config.toml")
	err = os.WriteFile(wrongTarget, []byte("theme = \"onedark\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and symlink pointing to wrong file
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	err = os.Symlink(wrongTarget, configFile)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, true, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Check_Source_Link_NotSymlink(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and regular file (not symlink)
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	err = os.WriteFile(configFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, true, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestConfigStep_Check_Source_Copy_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and regular file
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	err = os.WriteFile(configFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, false, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestConfigStep_Check_Source_NeedsApply_NoFile(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source-config.toml")
	err := os.WriteFile(sourceFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	// No config file exists

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep(sourceFile, false, nil, nil, nil, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// =============================================================================
// Additional LanguagesStep Check Tests
// =============================================================================

func TestLanguagesStep_Check_Link_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	err := os.WriteFile(sourceFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and symlink
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	langFile := filepath.Join(configPath, "languages.toml")
	absSource, _ := filepath.Abs(sourceFile)
	err = os.Symlink(absSource, langFile)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestLanguagesStep_Check_Link_WrongTarget(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	err := os.WriteFile(sourceFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	// Create wrong target
	wrongTarget := filepath.Join(tmpDir, "wrong-languages.toml")
	err = os.WriteFile(wrongTarget, []byte("[[language]]\nname = \"rust\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and symlink pointing to wrong file
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	langFile := filepath.Join(configPath, "languages.toml")
	err = os.Symlink(wrongTarget, langFile)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestLanguagesStep_Check_Link_NotSymlink(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	err := os.WriteFile(sourceFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and regular file (not symlink)
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	langFile := filepath.Join(configPath, "languages.toml")
	err = os.WriteFile(langFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, true, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestLanguagesStep_Check_Copy_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	err := os.WriteFile(sourceFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	// Create helix config dir and regular file
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	langFile := filepath.Join(configPath, "languages.toml")
	err = os.WriteFile(langFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, false, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestLanguagesStep_Apply_Link(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create source file
	sourceFile := filepath.Join(tmpDir, "languages.toml")
	err := os.WriteFile(sourceFile, []byte("[[language]]\nname = \"go\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep(sourceFile, true, runner)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)

	require.NoError(t, err)

	// Verify symlink was created
	configPath := getHelixConfigPathForTest(tmpDir)
	langFile := filepath.Join(configPath, "languages.toml")

	info, err := os.Lstat(langFile)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")
}

// =============================================================================
// Additional ThemeStep Check Tests
// =============================================================================

func TestThemeStep_Check_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create config dir and config.toml with theme already set
	configPath := getHelixConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	err = os.WriteFile(configFile, []byte("theme = \"catppuccin_mocha\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestThemeStep_Check_WrongTheme(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create config dir and config.toml with different theme
	configPath := getHelixConfigPathForTest(tmpDir)
	err := os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configPath, "config.toml")
	err = os.WriteFile(configFile, []byte("theme = \"gruvbox\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("catppuccin_mocha", "", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestThemeStep_Check_CustomTheme_Satisfied(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create custom theme source
	themeSource := filepath.Join(tmpDir, "my_theme.toml")
	err := os.WriteFile(themeSource, []byte("[\"ui.background\"]\nbg = \"#1e1e2e\"\n"), 0o644)
	require.NoError(t, err)

	// Create config dir, themes dir, and install the theme
	configPath := getHelixConfigPathForTest(tmpDir)
	themesDir := filepath.Join(configPath, "themes")
	err = os.MkdirAll(themesDir, 0o755)
	require.NoError(t, err)

	installedTheme := filepath.Join(themesDir, "my_theme.toml")
	err = os.WriteFile(installedTheme, []byte("[\"ui.background\"]\nbg = \"#1e1e2e\"\n"), 0o644)
	require.NoError(t, err)

	// Create config.toml with theme set
	configFile := filepath.Join(configPath, "config.toml")
	err = os.WriteFile(configFile, []byte("theme = \"my_theme\"\n"), 0o644)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("my_theme", themeSource, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestThemeStep_Check_CustomTheme_NotInstalled(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	// Create custom theme source
	themeSource := filepath.Join(tmpDir, "my_theme.toml")
	err := os.WriteFile(themeSource, []byte("[\"ui.background\"]\nbg = \"#1e1e2e\"\n"), 0o644)
	require.NoError(t, err)

	// Create config dir but no themes
	configPath := getHelixConfigPathForTest(tmpDir)
	err = os.MkdirAll(configPath, 0o755)
	require.NoError(t, err)

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("my_theme", themeSource, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestThemeStep_Plan_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("", "dotfiles/my_theme.toml", runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "theme", diff.Resource())
	assert.Contains(t, diff.NewValue(), "my_theme")
}

func TestThemeStep_Explain_WithSource(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewThemeStep("", "my_theme.toml", runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Equal(t, "Set Helix Theme", explanation.Summary())
	assert.Contains(t, explanation.Detail(), "my_theme")
}

func TestConfigStep_Explain_WithSource_Link(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("dotfiles/config.toml", true, nil, nil, nil, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Detail(), "Symlinks")
	assert.Contains(t, explanation.Detail(), "dotfiles/config.toml")
}

func TestConfigStep_Explain_WithSource_Copy(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewConfigStep("dotfiles/config.toml", false, nil, nil, nil, runner)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)

	assert.Contains(t, explanation.Detail(), "Copies")
	assert.Contains(t, explanation.Detail(), "dotfiles/config.toml")
}

func TestConfigStep_Apply_WithKeys(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	setupHelixConfigPath(t, tmpDir)

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"theme": "catppuccin_mocha"}
	keys := map[string]interface{}{
		"normal": map[string]interface{}{
			"space": map[string]interface{}{
				"f": "file_picker",
			},
		},
	}
	step := helix.NewConfigStep("", false, settings, nil, keys, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)

	require.NoError(t, err)

	// Verify config was written with keys
	configPath := getHelixConfigPathForTest(tmpDir)
	data, err := os.ReadFile(filepath.Join(configPath, "config.toml"))
	require.NoError(t, err)

	var written map[string]interface{}
	err = toml.Unmarshal(data, &written)
	require.NoError(t, err)
	assert.Equal(t, "catppuccin_mocha", written["theme"])

	keysSection, ok := written["keys"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, keysSection["normal"])
}

func TestLanguagesStep_Plan_Copy(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := helix.NewLanguagesStep("dotfiles/languages.toml", false, runner)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Equal(t, "config", diff.Resource())
	assert.Contains(t, diff.NewValue(), "copy")
}

// =============================================================================
// Helper Functions
// =============================================================================

func setupHelixConfigPath(t *testing.T, tmpDir string) {
	t.Helper()
	switch runtime.GOOS {
	case "darwin", "linux":
		t.Setenv("HOME", tmpDir)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	default: // windows
		t.Setenv("APPDATA", tmpDir)
	}
	t.Setenv("HELIX_CONFIG_DIR", "")
}

func getHelixConfigPathForTest(tmpDir string) string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return filepath.Join(tmpDir, ".config", "helix")
	default: // windows
		return filepath.Join(tmpDir, "helix")
	}
}
