package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlacrittyConfigStep_ID(t *testing.T) {
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml"},
		nil, // No global config
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:alacritty:config", step.ID().String())
}

func TestAlacrittyConfigStep_DependsOn(t *testing.T) {
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml"},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestAlacrittyConfigStep_Check_SourceMode_NotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: true},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Check_NoSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{}, // No source, no settings
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestAlacrittyConfigStep_Plan_WithSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: true},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	// diff.Summary() contains the target path
	assert.NotEmpty(t, diff.Summary())
}

func TestAlacrittyConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml"},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure Alacritty", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
}

func TestKittyConfigStep_ID(t *testing.T) {
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:kitty:config", step.ID().String())
}

func TestKittyConfigStep_Check_SettingsMode_NotExists(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWezTermConfigStep_ID(t *testing.T) {
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua"},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:wezterm:config", step.ID().String())
}

func TestGhosttyConfigStep_ID(t *testing.T) {
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:ghostty:config", step.ID().String())
}

func TestHyperConfigStep_ID(t *testing.T) {
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js"},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:hyper:config", step.ID().String())
}

func TestWindowsTerminalConfigStep_ID(t *testing.T) {
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:windows-terminal:config", step.ID().String())
}

func TestITerm2SettingsStep_ID(t *testing.T) {
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		mocks.NewCommandRunner(),
	)
	assert.Equal(t, "terminal:iterm2:settings", step.ID().String())
}

func TestITerm2ProfilesStep_ID(t *testing.T) {
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		mocks.NewFileSystem(),
	)
	assert.Equal(t, "terminal:iterm2:profiles", step.ID().String())
}

func TestDiscovery_AlacrittySearchOpts(t *testing.T) {
	opts := AlacrittySearchOpts()
	assert.Equal(t, "ALACRITTY_CONFIG_DIR", opts.EnvVar)
	assert.Equal(t, "alacritty.toml", opts.ConfigFileName)
	assert.Equal(t, "alacritty/alacritty.toml", opts.XDGSubpath)
	assert.Contains(t, opts.LegacyPaths, "~/.alacritty.toml")
}

func TestDiscovery_KittySearchOpts(t *testing.T) {
	opts := KittySearchOpts()
	assert.Equal(t, "KITTY_CONFIG_DIRECTORY", opts.EnvVar)
	assert.Equal(t, "kitty.conf", opts.ConfigFileName)
}

func TestDiscovery_WezTermSearchOpts(t *testing.T) {
	opts := WezTermSearchOpts()
	assert.Equal(t, "WEZTERM_CONFIG_DIR", opts.EnvVar)
	assert.Contains(t, opts.LegacyPaths, "~/.wezterm.lua")
}

func TestDiscovery_GhosttySearchOpts(t *testing.T) {
	opts := GhosttySearchOpts()
	// Ghostty doesn't have an env var, it uses XDG_CONFIG_HOME only
	assert.Empty(t, opts.EnvVar)
	assert.Equal(t, "ghostty/config", opts.XDGSubpath)
}

func TestDiscovery_FindAlacrittyConfig(t *testing.T) {
	// Create a temporary directory with an alacritty config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "alacritty")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "alacritty.toml")
	err = os.WriteFile(configPath, []byte("[window]"), 0o644)
	require.NoError(t, err)

	// Clear XDG_CONFIG_HOME to use default
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpDir)
	t.Setenv("ALACRITTY_CONFIG_DIR", "") // Clear env var

	discovery := NewDiscovery()
	found := discovery.FindAlacrittyConfig()
	assert.Equal(t, configPath, found)
}

// =============================================================================
// GhosttyConfigStep Tests
// =============================================================================

func TestGhosttyConfigStep_DependsOn(t *testing.T) {
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestGhosttyConfigStep_Check_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{}, // No source, no settings
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestGhosttyConfigStep_Check_SettingsMode_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestGhosttyConfigStep_Plan_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestGhosttyConfigStep_Plan_WithSettings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14, "theme": "dark"}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "2 settings")
}

func TestGhosttyConfigStep_Plan_WithSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: true},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "symlink")
}

func TestGhosttyConfigStep_Apply_Settings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/tmp/test/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify the file was written
	content, err := fs.ReadFile("/tmp/test/.config/ghostty/config")
	require.NoError(t, err)
	assert.Contains(t, string(content), "font-size = 14")
}

func TestGhosttyConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure Ghostty", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// HyperConfigStep Tests
// =============================================================================

func TestHyperConfigStep_DependsOn(t *testing.T) {
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js"},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestHyperConfigStep_Check_NoSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestHyperConfigStep_Check_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js"},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestHyperConfigStep_Plan_NoSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestHyperConfigStep_Plan_WithSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: true},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "symlink")
}

func TestHyperConfigStep_Apply_Link(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", []byte("module.exports = {}"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: true},
		"/tmp/test/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestHyperConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js"},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure Hyper", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// WezTermConfigStep Tests
// =============================================================================

func TestWezTermConfigStep_DependsOn(t *testing.T) {
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua"},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestWezTermConfigStep_Check_NoSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestWezTermConfigStep_Check_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua"},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWezTermConfigStep_Plan_NoSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestWezTermConfigStep_Plan_WithSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: true},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "symlink")
}

func TestWezTermConfigStep_Apply_Link(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", []byte("return {}"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: true},
		"/tmp/test/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestWezTermConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua"},
		"/home/user/.wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure WezTerm", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// WindowsTerminalConfigStep Tests
// =============================================================================

func TestWindowsTerminalConfigStep_DependsOn(t *testing.T) {
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestWindowsTerminalConfigStep_Check_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestWindowsTerminalConfigStep_Check_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWindowsTerminalConfigStep_Plan_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestWindowsTerminalConfigStep_Plan_WithSettings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
}

func TestWindowsTerminalConfigStep_Apply_Settings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify file was written
	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)
	assert.Contains(t, string(content), "copyOnSelect")
}

func TestWindowsTerminalConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure Windows Terminal", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ITerm2SettingsStep Tests
// =============================================================================

func TestITerm2SettingsStep_DependsOn(t *testing.T) {
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		mocks.NewCommandRunner(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestITerm2SettingsStep_Check_NoSettings(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{},
		runner,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestITerm2SettingsStep_Check_NeedsApply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestITerm2SettingsStep_Plan_NoSettings(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{},
		runner,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestITerm2SettingsStep_Plan_WithSettings(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "1 preferences")
}

func TestITerm2SettingsStep_Apply(t *testing.T) {
	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.googlecode.iterm2", "HideScrollbar", "-bool", "YES"}, ports.CommandResult{ExitCode: 0})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestITerm2SettingsStep_Explain(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure iTerm2 Settings", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// ITerm2ProfilesStep Tests
// =============================================================================

func TestITerm2ProfilesStep_DependsOn(t *testing.T) {
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestITerm2ProfilesStep_Check_NoProfiles(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestITerm2ProfilesStep_Check_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestITerm2ProfilesStep_Plan_NoProfiles(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeNone, diff.Type())
}

func TestITerm2ProfilesStep_Plan_WithProfiles(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}, {Name: "Work"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "2 dynamic profiles")
}

func TestITerm2ProfilesStep_Apply(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test", ColorScheme: "Solarized Dark"}}},
		"/tmp/test/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify file was written (preflight-profiles.json)
	content, err := fs.ReadFile("/tmp/test/DynamicProfiles/preflight-profiles.json")
	require.NoError(t, err)
	assert.Contains(t, string(content), "Test")
}

func TestITerm2ProfilesStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure iTerm2 Dynamic Profiles", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// KittyConfigStep Additional Tests
// =============================================================================

func TestKittyConfigStep_DependsOn(t *testing.T) {
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)
	assert.Empty(t, step.DependsOn())
}

func TestKittyConfigStep_Check_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestKittyConfigStep_Plan_NoConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	// Kitty always reports a modify diff, even with 0 settings
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "0 settings")
}

func TestKittyConfigStep_Plan_WithSettings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "1 settings")
}

func TestKittyConfigStep_Apply_Settings(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/tmp/test/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify file was written
	content, err := fs.ReadFile("/tmp/test/.config/kitty/kitty.conf")
	require.NoError(t, err)
	assert.Contains(t, string(content), "font_size")
}

func TestKittyConfigStep_Explain(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Settings: map[string]interface{}{"font_size": 12}},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewExplainContext()
	explanation := step.Explain(ctx)
	assert.Equal(t, "Configure Kitty", explanation.Summary())
	assert.NotEmpty(t, explanation.DocLinks())
	assert.NotEmpty(t, explanation.Tradeoffs())
}

// =============================================================================
// Discovery Additional Tests
// =============================================================================

func TestDiscovery_HyperSearchOpts(t *testing.T) {
	opts := HyperSearchOpts()
	assert.Empty(t, opts.EnvVar)            // Hyper doesn't use an env var
	assert.Empty(t, opts.LegacyPaths)       // Hyper uses direct home paths
	assert.Contains(t, opts.MacOSPaths, "~/.hyper.js")
	assert.Contains(t, opts.LinuxPaths, "~/.hyper.js")
}

func TestDiscovery_WindowsTerminalSearchOpts(t *testing.T) {
	opts := WindowsTerminalSearchOpts()
	assert.Empty(t, opts.EnvVar)
}

func TestDiscovery_AlacrittyBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.AlacrittyBestPracticePath()
	assert.Contains(t, path, "alacritty")
}

func TestDiscovery_KittyBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.KittyBestPracticePath()
	assert.Contains(t, path, "kitty")
}

func TestDiscovery_WezTermBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.WezTermBestPracticePath()
	assert.NotEmpty(t, path)
}

func TestDiscovery_GhosttyBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.GhosttyBestPracticePath()
	assert.Contains(t, path, "ghostty")
}

func TestDiscovery_HyperBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.HyperBestPracticePath()
	assert.Contains(t, path, ".hyper.js")
}

func TestDiscovery_WindowsTerminalBestPracticePath(t *testing.T) {
	discovery := NewDiscovery()
	path := discovery.WindowsTerminalBestPracticePath()
	// Windows Terminal path is only available on Windows
	if os.Getenv("OS") == "Windows_NT" {
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "settings.json")
	}
	// On non-Windows, path is empty which is expected
}

// =============================================================================
// Alacritty Apply Tests
// =============================================================================

func TestAlacrittyConfigStep_Apply_SourceMode_Link(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file in dotfiles
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", []byte("[general]\nlive_config_reload = true"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: true},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestAlacrittyConfigStep_Apply_SourceMode_Copy(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file in dotfiles
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", []byte("[general]\nlive_config_reload = true"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)

	// Verify file was copied
	content, err := fs.ReadFile("/home/user/.config/alacritty/alacritty.toml")
	require.NoError(t, err)
	assert.Contains(t, string(content), "live_config_reload")
}

func TestAlacrittyConfigStep_Apply_SettingsMode(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Settings: map[string]interface{}{
			"font": map[string]interface{}{
				"size": 14,
			},
		}},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Verify file was written
	content, err := fs.ReadFile("/home/user/.config/alacritty/alacritty.toml")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestAlacrittyConfigStep_Check_CopyMode_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	content := []byte("[general]\nlive_config_reload = true")
	// Create source and target with same content
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", content, 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/alacritty/alacritty.toml", content, 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// =============================================================================
// Kitty Apply Tests
// =============================================================================

func TestKittyConfigStep_Apply_SourceMode_Link(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file in dotfiles
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: true},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestKittyConfigStep_Check_SourceMode_NeedsApply(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file but no target
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: true},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKittyConfigStep_Plan_WithSource(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: true},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "symlink")
}

// =============================================================================
// Ghostty Apply Extended Tests
// =============================================================================

func TestGhosttyConfigStep_Apply_SourceMode(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create source file in dotfiles
	err := fs.WriteFile("/tmp/dotfiles/ghostty.conf", []byte("font-size = 14"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty.conf", Link: true},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestGhosttyConfigStep_Check_CopyMode_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	content := []byte("font-size = 14")
	// Create source and target with same content
	err := fs.WriteFile("/tmp/dotfiles/ghostty.conf", content, 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/ghostty/config", content, 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty.conf", Link: false},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// =============================================================================
// Hyper Apply Extended Tests
// =============================================================================

func TestHyperConfigStep_Check_CopyMode_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	content := []byte("module.exports = {}")
	// Create source and target with same content
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", content, 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.hyper.js", content, 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// =============================================================================
// WezTerm Apply Extended Tests
// =============================================================================

func TestWezTermConfigStep_Check_CopyMode_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	content := []byte("return {}")
	// Create source and target with same content
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", content, 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/wezterm/wezterm.lua", content, 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// =============================================================================
// Windows Terminal Extended Tests
// =============================================================================

func TestWindowsTerminalConfigStep_Check_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create existing config with matching settings
	existing := `{"profiles":{"defaults":{}}}`
	err := fs.WriteFile("/home/user/AppData/Local/Microsoft/Windows Terminal/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/AppData/Local/Microsoft/Windows Terminal/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestWindowsTerminalConfigStep_Plan_WithProfiles(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{{Name: "PowerShell", CommandLine: "pwsh.exe"}},
		},
		"/home/user/AppData/Local/Microsoft/Windows Terminal/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "1 profiles")
}

// =============================================================================
// iTerm2 Extended Tests
// =============================================================================

func TestITerm2ProfilesStep_Check_Satisfied(t *testing.T) {
	fs := mocks.NewFileSystem()
	// Create existing profiles file
	err := fs.WriteFile("/home/user/Library/Application Support/iTerm2/DynamicProfiles/preflight-profiles.json", []byte(`{"Profiles":[]}`), 0o644)
	require.NoError(t, err)

	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/Library/Application Support/iTerm2/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	status, err := step.Check(ctx)
	require.NoError(t, err)
	// Profiles always needs apply since we always regenerate
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

// =============================================================================
// LockInfo and InstalledVersion Tests
// =============================================================================

func TestAlacrittyConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml"},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	info := step.LockInfo()
	// Returns empty struct (no lock info for config steps)
	assert.Empty(t, info.Provider)
	assert.Empty(t, info.Name)
	assert.Empty(t, info.Version)
}

func TestAlacrittyConfigStep_InstalledVersion(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml"},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(nil)
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

func TestGhosttyConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{"font-size": 14}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestHyperConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js"},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestWezTermConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua"},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestWindowsTerminalConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/settings.json",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestKittyConfigStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestITerm2SettingsStep_LockInfo(t *testing.T) {
	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"key": "value"}},
		runner,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}

func TestITerm2ProfilesStep_LockInfo(t *testing.T) {
	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{{Name: "Test"}}},
		"/home/user/DynamicProfiles",
		fs,
	)

	info := step.LockInfo()
	assert.Empty(t, info.Provider)
}
