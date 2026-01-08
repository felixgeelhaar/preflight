package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
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
