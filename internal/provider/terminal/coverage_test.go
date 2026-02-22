package terminal

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ParseConfig Tests
// =============================================================================

func TestParseConfig_ValidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    map[string]interface{}
		verify func(t *testing.T, cfg *Config)
	}{
		{
			name: "alacritty only",
			raw: map[string]interface{}{
				"alacritty": map[string]interface{}{
					"source": "alacritty.toml",
					"link":   true,
				},
			},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Alacritty)
				assert.Equal(t, "alacritty.toml", cfg.Alacritty.Source)
				assert.True(t, cfg.Alacritty.Link)
			},
		},
		{
			name: "font and theme globals",
			raw: map[string]interface{}{
				"font": map[string]interface{}{
					"family": "JetBrains Mono",
					"size":   14.0,
				},
				"theme": map[string]interface{}{
					"name": "catppuccin-mocha",
				},
			},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Font)
				assert.Equal(t, "JetBrains Mono", cfg.Font.Family)
				assert.InDelta(t, 14.0, cfg.Font.Size, 0.001)
				require.NotNil(t, cfg.Theme)
				assert.Equal(t, "catppuccin-mocha", cfg.Theme.Name)
			},
		},
		{
			name: "empty map returns empty config",
			raw:  map[string]interface{}{},
			verify: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.False(t, cfg.HasAnyTerminal())
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseConfig(tc.raw)
			require.NoError(t, err)
			tc.verify(t, cfg)
		})
	}
}

func TestHasAnyTerminal_AllNil(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	assert.False(t, cfg.HasAnyTerminal())
}

func TestHasAnyTerminal_EachTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
	}{
		{"alacritty", Config{Alacritty: &AlacrittyConfig{}}},
		{"kitty", Config{Kitty: &KittyConfig{}}},
		{"wezterm", Config{WezTerm: &WezTermConfig{}}},
		{"ghostty", Config{Ghostty: &GhosttyConfig{}}},
		{"iterm2", Config{ITerm2: &ITerm2Config{}}},
		{"hyper", Config{Hyper: &HyperConfig{}}},
		{"windows_terminal", Config{WindowsTerminal: &WindowsTerminalConfig{}}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tc.cfg.HasAnyTerminal())
		})
	}
}

// =============================================================================
// Alacritty Extended Coverage Tests
// =============================================================================

func TestAlacrittyConfigStep_Check_SettingsMode_WithGlobalFont(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	globalCfg := &Config{
		Font: &FontConfig{Family: "JetBrains Mono", Size: 14},
	}
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{}, // No settings, no source
		globalCfg,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	// Has global font, target file doesn't exist => needs apply
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Check_SettingsMode_WithGlobalTheme(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	globalCfg := &Config{
		Theme: &ThemeConfig{Name: "catppuccin-mocha"},
	}
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{},
		globalCfg,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Check_SettingsMode_FileExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/home/user/.config/alacritty/alacritty.toml", []byte("[general]"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{
			Settings: map[string]interface{}{"font": map[string]interface{}{"size": 14}},
		},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Check_CopyMode_NeedsApply_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", []byte("[general]"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Check_CopyMode_DifferentHash(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", []byte("[general]\nlive_config_reload = true"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/alacritty/alacritty.toml", []byte("[general]\nlive_config_reload = false"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestAlacrittyConfigStep_Plan_WithSettings(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{
			Settings: map[string]interface{}{"font": map[string]interface{}{"size": 14}},
		},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "merge 1 settings")
}

func TestAlacrittyConfigStep_Plan_CopyMode(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "copy")
}

func TestAlacrittyConfigStep_Apply_SettingsMode_WithExistingConfig(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Write existing TOML content
	err := fs.WriteFile("/home/user/.config/alacritty/alacritty.toml", []byte(""), 0o644)
	require.NoError(t, err)

	globalCfg := &Config{
		Font: &FontConfig{Family: "Fira Code", Size: 13},
	}
	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{
			Settings: map[string]interface{}{
				"live_config_reload": true,
			},
		},
		globalCfg,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/home/user/.config/alacritty/alacritty.toml")
	require.NoError(t, err)
	assert.Contains(t, string(content), "live_config_reload")
	assert.Contains(t, string(content), "Fira Code")
}

func TestAlacrittyConfigStep_Apply_SourceMode_RemoveExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Pre-create source and existing target
	err := fs.WriteFile("/tmp/dotfiles/alacritty.toml", []byte("[general]"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/alacritty/alacritty.toml", []byte("old content"), 0o644)
	require.NoError(t, err)

	step := NewAlacrittyConfigStep(
		&AlacrittyConfig{Source: "alacritty.toml", Link: false},
		nil,
		"/home/user/.config/alacritty/alacritty.toml",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/home/user/.config/alacritty/alacritty.toml")
	require.NoError(t, err)
	assert.Equal(t, "[general]", string(content))
}

// =============================================================================
// Kitty Extended Coverage Tests
// =============================================================================

func TestKittyConfigStep_Check_SourceMode_CopyMode_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: false},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKittyConfigStep_Check_SourceMode_CopyMode_Satisfied(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	content := []byte("font_size 14")
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", content, 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/kitty/kitty.conf", content, 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: false},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestKittyConfigStep_Check_SourceMode_CopyMode_DifferentHash(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/kitty/kitty.conf", []byte("font_size 12"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: false},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKittyConfigStep_Check_SourceMode_SourceHashError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// No source file exists => FileHash returns error
	err := fs.WriteFile("/home/user/.config/kitty/kitty.conf", []byte("old"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: false},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestKittyConfigStep_Check_SettingsMode_WithGlobalFont(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	globalCfg := &Config{
		Font: &FontConfig{Family: "JetBrains Mono", Size: 14},
	}

	step := NewKittyConfigStep(
		&KittyConfig{},
		globalCfg,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKittyConfigStep_Check_SettingsMode_WithTheme(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewKittyConfigStep(
		&KittyConfig{Theme: "catppuccin-mocha"},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestKittyConfigStep_Apply_SettingsMode_MergeExistingLines(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Create existing config with a setting that will be overwritten
	err := fs.WriteFile("/tmp/test/.config/kitty/kitty.conf", []byte("font_size 12\nbold_font auto\n# comment line\n"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{
			Settings: map[string]interface{}{
				"font_size": 14,
				"cursor":    "beam",
			},
		},
		nil,
		"/tmp/test/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/kitty/kitty.conf")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "font_size 14")
	assert.Contains(t, contentStr, "cursor beam")
	assert.Contains(t, contentStr, "bold_font auto")
}

func TestKittyConfigStep_Apply_SettingsMode_WithGlobalFontAndTheme(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	globalCfg := &Config{
		Font: &FontConfig{Family: "Fira Code", Size: 13},
	}

	step := NewKittyConfigStep(
		&KittyConfig{
			Settings: map[string]interface{}{
				"background": "#1e1e2e",
			},
			Theme: "catppuccin-mocha",
		},
		globalCfg,
		"/tmp/test/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/kitty/kitty.conf")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "font_family Fira Code")
	assert.Contains(t, contentStr, "font_size 13.0")
	assert.Contains(t, contentStr, "include themes/catppuccin-mocha.conf")
	assert.Contains(t, contentStr, "background #1e1e2e")
}

func TestKittyConfigStep_Apply_SourceMode_Copy(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: false},
		nil,
		"/tmp/test/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/kitty/kitty.conf")
	require.NoError(t, err)
	assert.Equal(t, "font_size 14", string(content))
}

func TestKittyConfigStep_Apply_SourceMode_RemoveExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/kitty.conf", []byte("font_size 14"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/tmp/test/.config/kitty/kitty.conf", []byte("old content"), 0o644)
	require.NoError(t, err)

	step := NewKittyConfigStep(
		&KittyConfig{Source: "kitty.conf", Link: true},
		nil,
		"/tmp/test/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)
}

func TestKittyConfigStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	step := NewKittyConfigStep(
		&KittyConfig{},
		nil,
		"/home/user/.config/kitty/kitty.conf",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

// =============================================================================
// Ghostty Extended Coverage Tests
// =============================================================================

func TestGhosttyConfigStep_Check_SourceMode_CopyMode_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/ghostty/config", []byte("font-size = 14"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: false},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestGhosttyConfigStep_Check_SourceMode_CopyMode_DifferentHash(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/ghostty/config", []byte("font-size = 14"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/ghostty/config", []byte("font-size = 12"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: false},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestGhosttyConfigStep_Check_SourceMode_CopyMode_SourceHashError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Target exists but source doesn't
	err := fs.WriteFile("/home/user/.config/ghostty/config", []byte("font-size = 12"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: false},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestGhosttyConfigStep_Check_SettingsMode_ExistingConfigMatches(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/home/user/.config/ghostty/config", []byte("font-size = 14\ntheme = dark\n"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{
			"font-size": "14",
			"theme":     "dark",
		}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestGhosttyConfigStep_Check_SettingsMode_ExistingConfigDiffers(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/home/user/.config/ghostty/config", []byte("font-size = 12\n"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{
			"font-size": "14",
		}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestGhosttyConfigStep_Apply_SettingsMode_MergeExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/test/.config/ghostty/config", []byte("font-size = 12\ntheme = light\n"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{
			"font-size": 14,
			"cursor":    "beam",
		}},
		"/tmp/test/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/ghostty/config")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "font-size = 14")
	assert.Contains(t, contentStr, "cursor = beam")
	assert.Contains(t, contentStr, "theme = light")
}

func TestGhosttyConfigStep_Apply_SettingsMode_NoSettings(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{}},
		"/tmp/test/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestGhosttyConfigStep_Apply_SourceMode_Copy(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/ghostty/config", []byte("font-size = 14"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: false},
		"/tmp/test/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/ghostty/config")
	require.NoError(t, err)
	assert.Equal(t, "font-size = 14", string(content))
}

func TestGhosttyConfigStep_Apply_SourceMode_RemoveExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/ghostty/config", []byte("font-size = 14"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/tmp/test/.config/ghostty/config", []byte("old content"), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Source: "ghostty/config", Link: false},
		"/tmp/test/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/ghostty/config")
	require.NoError(t, err)
	assert.Equal(t, "font-size = 14", string(content))
}

func TestGhosttyConfigStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	step := NewGhosttyConfigStep(
		&GhosttyConfig{},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

// =============================================================================
// Hyper Extended Coverage Tests
// =============================================================================

func TestHyperConfigStep_Check_CopyMode_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", []byte("module.exports = {}"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestHyperConfigStep_Check_CopyMode_DifferentHash(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", []byte("module.exports = {v2}"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.hyper.js", []byte("module.exports = {v1}"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestHyperConfigStep_Check_CopyMode_SourceHashError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Target exists but source doesn't
	err := fs.WriteFile("/home/user/.hyper.js", []byte("old"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestHyperConfigStep_Apply_NoSource(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestHyperConfigStep_Apply_Copy(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", []byte("module.exports = {}"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/tmp/test/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.hyper.js")
	require.NoError(t, err)
	assert.Equal(t, "module.exports = {}", string(content))
}

func TestHyperConfigStep_Apply_RemoveExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/.hyper.js", []byte("new content"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/tmp/test/.hyper.js", []byte("old content"), 0o644)
	require.NoError(t, err)

	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/tmp/test/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.hyper.js")
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestHyperConfigStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	step := NewHyperConfigStep(
		&HyperConfig{},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

func TestHyperConfigStep_Plan_CopyMode(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewHyperConfigStep(
		&HyperConfig{Source: ".hyper.js", Link: false},
		"/home/user/.hyper.js",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "copy")
}

// =============================================================================
// WezTerm Extended Coverage Tests
// =============================================================================

func TestWezTermConfigStep_Check_CopyMode_NotExists(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", []byte("return {}"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWezTermConfigStep_Check_CopyMode_DifferentHash(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", []byte("return {v2}"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/home/user/.config/wezterm/wezterm.lua", []byte("return {v1}"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWezTermConfigStep_Check_CopyMode_SourceHashError(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Target exists but source doesn't
	err := fs.WriteFile("/home/user/.config/wezterm/wezterm.lua", []byte("old"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestWezTermConfigStep_Apply_Copy(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", []byte("return {}"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/tmp/test/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/wezterm/wezterm.lua")
	require.NoError(t, err)
	assert.Equal(t, "return {}", string(content))
}

func TestWezTermConfigStep_Apply_RemoveExisting(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	err := fs.WriteFile("/tmp/dotfiles/wezterm.lua", []byte("return {new}"), 0o644)
	require.NoError(t, err)
	err = fs.WriteFile("/tmp/test/.config/wezterm/wezterm.lua", []byte("return {old}"), 0o644)
	require.NoError(t, err)

	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/tmp/test/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/.config/wezterm/wezterm.lua")
	require.NoError(t, err)
	assert.Equal(t, "return {new}", string(content))
}

func TestWezTermConfigStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	step := NewWezTermConfigStep(
		&WezTermConfig{},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		mocks.NewFileSystem(),
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

func TestWezTermConfigStep_Plan_CopyMode(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWezTermConfigStep(
		&WezTermConfig{Source: "wezterm.lua", Link: false},
		"/home/user/.config/wezterm/wezterm.lua",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "copy")
}

// =============================================================================
// iTerm2 Extended Coverage Tests
// =============================================================================

func TestITerm2SettingsStep_Check_SettingSatisfied(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.googlecode.iterm2", "HideScrollbar"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "true\n",
	})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestITerm2SettingsStep_Check_SettingDiffers(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"read", "com.googlecode.iterm2", "HideScrollbar"}, ports.CommandResult{
		ExitCode: 0,
		Stdout:   "false\n",
	})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": true}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestITerm2SettingsStep_Apply_BoolFalse(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.googlecode.iterm2", "HideScrollbar", "-bool", "NO"}, ports.CommandResult{ExitCode: 0})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"HideScrollbar": false}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestITerm2SettingsStep_Apply_IntValue(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.googlecode.iterm2", "FontSize", "-float", "14"}, ports.CommandResult{ExitCode: 0})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"FontSize": 14}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestITerm2SettingsStep_Apply_StringValue(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("defaults", []string{"write", "com.googlecode.iterm2", "FontName", "-string", "JetBrains Mono"}, ports.CommandResult{ExitCode: 0})

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"FontName": "JetBrains Mono"}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestITerm2SettingsStep_Apply_Error(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	// No result registered => mock returns error

	step := NewITerm2SettingsStep(
		&ITerm2Config{Settings: map[string]interface{}{"BadKey": "value"}},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set BadKey")
}

func TestITerm2SettingsStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	step := NewITerm2SettingsStep(
		&ITerm2Config{},
		runner,
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

func TestITerm2ProfilesStep_Apply_NoProfiles(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{},
		"/tmp/test/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestITerm2ProfilesStep_Apply_FullProfile(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{
			{
				Name:        "Dev",
				GUID:        "abc-123",
				Font:        "JetBrains Mono",
				FontSize:    14,
				ColorScheme: "Solarized Dark",
				Custom: map[string]string{
					"Cursor Type": "1",
				},
			},
		}},
		"/tmp/test/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/DynamicProfiles/preflight-profiles.json")
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "Dev")
	assert.Contains(t, contentStr, "abc-123")
	assert.Contains(t, contentStr, "JetBrains Mono")
	assert.Contains(t, contentStr, "Solarized Dark")
	assert.Contains(t, contentStr, "Cursor Type")
}

func TestITerm2ProfilesStep_Check_Satisfied_AfterApply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()

	// Generate the expected content for "Dev" profile
	step := NewITerm2ProfilesStep(
		&ITerm2Config{DynamicProfiles: []ITerm2Profile{
			{Name: "Dev", GUID: "abc-123"},
		}},
		"/tmp/test/DynamicProfiles",
		fs,
	)

	// First apply to get the expected content
	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	// Now check should be satisfied
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestITerm2ProfilesStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewITerm2ProfilesStep(
		&ITerm2Config{},
		"/tmp/test/DynamicProfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

// =============================================================================
// WindowsTerminal Extended Coverage Tests
// =============================================================================

func TestWindowsTerminalConfigStep_Check_SettingsMatch(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{"copyOnSelect": true}`
	err := fs.WriteFile("/home/user/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestWindowsTerminalConfigStep_Check_SettingsDiffer(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{"copyOnSelect": false}`
	err := fs.WriteFile("/home/user/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWindowsTerminalConfigStep_Check_WithProfiles_AlwaysNeedsApply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{"profiles": {"defaults": {}, "list": []}}`
	err := fs.WriteFile("/home/user/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{{Name: "PowerShell"}},
		},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWindowsTerminalConfigStep_Check_WithSchemes_AlwaysNeedsApply(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{"schemes": []}`
	err := fs.WriteFile("/home/user/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Schemes: []WindowsTerminalColorScheme{
				{Name: "Custom", Background: "#000", Foreground: "#fff"},
			},
		},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWindowsTerminalConfigStep_Check_UnreadableConfig(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Write invalid JSON
	err := fs.WriteFile("/home/user/settings.json", []byte("not json"), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{Settings: map[string]interface{}{"copyOnSelect": true}},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestWindowsTerminalConfigStep_Plan_WithSettingsAndProfiles(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Settings: map[string]interface{}{"copyOnSelect": true},
			Profiles: []WindowsTerminalProfile{{Name: "PS"}},
		},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "update 1 settings")
	assert.Contains(t, diff.NewValue(), "1 profiles")
}

func TestWindowsTerminalConfigStep_Plan_WithSchemes(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Schemes: []WindowsTerminalColorScheme{
				{Name: "Dark", Background: "#000", Foreground: "#fff"},
				{Name: "Light", Background: "#fff", Foreground: "#000"},
			},
		},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	assert.Contains(t, diff.NewValue(), "2 color schemes")
}

func TestWindowsTerminalConfigStep_Plan_AllTypes(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Settings: map[string]interface{}{"copyOnSelect": true},
			Profiles: []WindowsTerminalProfile{{Name: "PS"}},
			Schemes:  []WindowsTerminalColorScheme{{Name: "Dark", Background: "#000", Foreground: "#fff"}},
		},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	diff, err := step.Plan(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.DiffTypeModify, diff.Type())
	desc := diff.NewValue()
	assert.Contains(t, desc, "update 1 settings")
	assert.Contains(t, desc, "1 profiles")
	assert.Contains(t, desc, "1 color schemes")
}

func TestWindowsTerminalConfigStep_Apply_NoChanges(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)
}

func TestWindowsTerminalConfigStep_Apply_WithProfiles(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	useAcrylic := true
	acrylicOpacity := 0.8

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{
				{
					Name:           "PowerShell",
					GUID:           "{574e775e-4f2a-5b96-ac1e-a2962a402336}",
					CommandLine:    "pwsh.exe",
					ColorScheme:    "Campbell",
					FontFace:       "Cascadia Code",
					FontSize:       12,
					UseAcrylic:     &useAcrylic,
					AcrylicOpacity: &acrylicOpacity,
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	profiles, ok := result["profiles"].(map[string]interface{})
	require.True(t, ok)
	list, ok := profiles["list"].([]interface{})
	require.True(t, ok)
	require.Len(t, list, 1)

	profile := list[0].(map[string]interface{})
	assert.Equal(t, "PowerShell", profile["name"])
	assert.Equal(t, "pwsh.exe", profile["commandline"])
	assert.Equal(t, "Campbell", profile["colorScheme"])
	assert.Equal(t, true, profile["useAcrylic"])
}

func TestWindowsTerminalConfigStep_Apply_MergeExistingProfiles(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{
		"profiles": {
			"defaults": {},
			"list": [
				{"name": "PowerShell", "guid": "{574e775e}", "commandline": "pwsh.exe"}
			]
		}
	}`
	err := fs.WriteFile("/tmp/test/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{
				{
					Name:        "PowerShell",
					GUID:        "{574e775e}",
					ColorScheme: "Campbell",
				},
				{
					Name:        "Ubuntu",
					CommandLine: "wsl.exe -d Ubuntu",
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	profiles := result["profiles"].(map[string]interface{})
	list := profiles["list"].([]interface{})
	assert.Len(t, list, 2) // Updated PowerShell + new Ubuntu

	// PowerShell should be updated in place
	ps := list[0].(map[string]interface{})
	assert.Equal(t, "PowerShell", ps["name"])
	assert.Equal(t, "Campbell", ps["colorScheme"])
	assert.Equal(t, "pwsh.exe", ps["commandline"]) // Should keep original commandline

	// Ubuntu should be appended
	ubuntu := list[1].(map[string]interface{})
	assert.Equal(t, "Ubuntu", ubuntu["name"])
}

func TestWindowsTerminalConfigStep_Apply_WithSchemes(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Schemes: []WindowsTerminalColorScheme{
				{
					Name:           "Catppuccin Mocha",
					Background:     "#1e1e2e",
					Foreground:     "#cdd6f4",
					Black:          "#45475a",
					Red:            "#f38ba8",
					Green:          "#a6e3a1",
					Yellow:         "#f9e2af",
					Blue:           "#89b4fa",
					Purple:         "#f5c2e7",
					Cyan:           "#94e2d5",
					White:          "#bac2de",
					BrightBlack:    "#585b70",
					BrightRed:      "#f38ba8",
					BrightGreen:    "#a6e3a1",
					BrightYellow:   "#f9e2af",
					BrightBlue:     "#89b4fa",
					BrightPurple:   "#f5c2e7",
					BrightCyan:     "#94e2d5",
					BrightWhite:    "#a6adc8",
					CursorColor:    "#f5e0dc",
					SelectionColor: "#585b70",
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	schemes, ok := result["schemes"].([]interface{})
	require.True(t, ok)
	require.Len(t, schemes, 1)

	scheme := schemes[0].(map[string]interface{})
	assert.Equal(t, "Catppuccin Mocha", scheme["name"])
	assert.Equal(t, "#1e1e2e", scheme["background"])
	assert.Equal(t, "#cdd6f4", scheme["foreground"])
	assert.Equal(t, "#45475a", scheme["black"])
	assert.Equal(t, "#f38ba8", scheme["red"])
	assert.Equal(t, "#a6e3a1", scheme["green"])
	assert.Equal(t, "#f9e2af", scheme["yellow"])
	assert.Equal(t, "#89b4fa", scheme["blue"])
	assert.Equal(t, "#f5c2e7", scheme["purple"])
	assert.Equal(t, "#94e2d5", scheme["cyan"])
	assert.Equal(t, "#bac2de", scheme["white"])
	assert.Equal(t, "#585b70", scheme["brightBlack"])
	assert.Equal(t, "#f5e0dc", scheme["cursorColor"])
	assert.Equal(t, "#585b70", scheme["selectionBackground"])
}

func TestWindowsTerminalConfigStep_Apply_MergeExistingSchemes(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	existing := `{
		"schemes": [
			{"name": "Campbell", "background": "#0C0C0C", "foreground": "#CCCCCC"}
		]
	}`
	err := fs.WriteFile("/tmp/test/settings.json", []byte(existing), 0o644)
	require.NoError(t, err)

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Schemes: []WindowsTerminalColorScheme{
				{Name: "Campbell", Background: "#000000", Foreground: "#FFFFFF"},
				{Name: "Solarized", Background: "#002b36", Foreground: "#839496"},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err = step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	schemes := result["schemes"].([]interface{})
	assert.Len(t, schemes, 2) // Updated Campbell + new Solarized

	// Campbell should be updated
	campbell := schemes[0].(map[string]interface{})
	assert.Equal(t, "Campbell", campbell["name"])
	assert.Equal(t, "#000000", campbell["background"])

	// Solarized should be appended
	solarized := schemes[1].(map[string]interface{})
	assert.Equal(t, "Solarized", solarized["name"])
}

func TestWindowsTerminalConfigStep_Apply_ProfilesAndSchemes(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Settings: map[string]interface{}{"copyOnSelect": true},
			Profiles: []WindowsTerminalProfile{
				{Name: "PowerShell", CommandLine: "pwsh.exe"},
			},
			Schemes: []WindowsTerminalColorScheme{
				{Name: "Dark", Background: "#000", Foreground: "#fff"},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	assert.Equal(t, true, result["copyOnSelect"])
	assert.NotNil(t, result["profiles"])
	assert.NotNil(t, result["schemes"])
}

func TestWindowsTerminalConfigStep_InstalledVersion(t *testing.T) {
	t.Parallel()

	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{},
		"/home/user/settings.json",
		mocks.NewFileSystem(),
	)

	ctx := compiler.NewRunContext(context.Background())
	version, err := step.InstalledVersion(ctx)
	require.NoError(t, err)
	assert.Empty(t, version)
}

// =============================================================================
// Provider Compile Extended Tests
// =============================================================================

func TestProvider_Compile_ITerm2_WithDynamicProfiles(t *testing.T) {
	t.Parallel()

	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"darwin",
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"iterm2": map[string]interface{}{
				"settings": map[string]interface{}{
					"HideScrollbar": true,
				},
				"dynamic_profiles": []interface{}{
					map[string]interface{}{
						"name": "Dev",
						"guid": "abc-123",
					},
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 2)

	stepIDs := make(map[string]bool)
	for _, step := range steps {
		stepIDs[step.ID().String()] = true
	}
	assert.True(t, stepIDs["terminal:iterm2:settings"])
	assert.True(t, stepIDs["terminal:iterm2:profiles"])
}

func TestProvider_Compile_ITerm2_ProfilesOnly(t *testing.T) {
	t.Parallel()

	p := NewProviderWithDiscovery(
		mocks.NewFileSystem(),
		mocks.NewCommandRunner(),
		NewDiscovery(),
		"darwin",
	)

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"iterm2": map[string]interface{}{
				"dynamic_profiles": []interface{}{
					map[string]interface{}{
						"name": "Dev",
					},
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:iterm2:profiles", steps[0].ID().String())
}

func TestProvider_Compile_AlacrittyWithCustomConfigPath(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"alacritty": map[string]interface{}{
				"config_path": "/custom/path/alacritty.toml",
				"source":      "alacritty.toml",
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "terminal:alacritty:config", steps[0].ID().String())
}

func TestProvider_Compile_KittyWithCustomConfigPath(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"kitty": map[string]interface{}{
				"config_path": "/custom/kitty.conf",
				"settings": map[string]interface{}{
					"font_size": 12,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
}

func TestProvider_Compile_WezTermWithCustomConfigPath(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"wezterm": map[string]interface{}{
				"config_path": "/custom/wezterm.lua",
				"source":      "wezterm.lua",
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
}

func TestProvider_Compile_GhosttyWithCustomConfigPath(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"ghostty": map[string]interface{}{
				"config_path": "/custom/ghostty/config",
				"settings": map[string]interface{}{
					"font-size": 14,
				},
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg)

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
}

func TestProvider_Compile_HyperWithCustomConfigPath(t *testing.T) {
	t.Parallel()

	p := NewProvider(mocks.NewFileSystem(), mocks.NewCommandRunner())

	cfg := map[string]interface{}{
		"terminal": map[string]interface{}{
			"hyper": map[string]interface{}{
				"config_path": "/custom/.hyper.js",
				"source":      ".hyper.js",
			},
		},
	}
	ctx := compiler.NewCompileContext(cfg).WithConfigRoot("/tmp/dotfiles")

	steps, err := p.Compile(ctx)
	require.NoError(t, err)
	require.Len(t, steps, 1)
}

// =============================================================================
// WindowsTerminal Apply with Profile Font (no FontFace but with FontSize)
// =============================================================================

func TestWindowsTerminalConfigStep_Apply_ProfileFontSizeOnly(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{
				{
					Name:     "Test",
					FontSize: 14,
					// No FontFace => should not set font at all
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	profiles := result["profiles"].(map[string]interface{})
	list := profiles["list"].([]interface{})
	require.Len(t, list, 1)
	profile := list[0].(map[string]interface{})
	assert.Equal(t, "Test", profile["name"])
	// Font should not be set without FontFace
	_, hasFontKey := profile["font"]
	assert.False(t, hasFontKey)
}

func TestWindowsTerminalConfigStep_Apply_ProfileWithFontFaceAndSize(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Profiles: []WindowsTerminalProfile{
				{
					Name:     "Test",
					FontFace: "Cascadia Code",
					FontSize: 14,
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	profiles := result["profiles"].(map[string]interface{})
	list := profiles["list"].([]interface{})
	require.Len(t, list, 1)
	profile := list[0].(map[string]interface{})
	font := profile["font"].(map[string]interface{})
	assert.Equal(t, "Cascadia Code", font["face"])
	// FontSize should be set within font map
	assert.InDelta(t, float64(14), font["size"], 0.001)
}

// =============================================================================
// Ghostty readGhosttyConfig Edge Cases
// =============================================================================

func TestGhosttyConfigStep_ReadConfig_WithComments(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	content := "# Comment line\n\nfont-size = 14\n# Another comment\ntheme = dark\n"
	err := fs.WriteFile("/home/user/.config/ghostty/config", []byte(content), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{
			"font-size": "14",
			"theme":     "dark",
		}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

func TestGhosttyConfigStep_ReadConfig_NoEqualsSign(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	// Lines without = sign should be skipped
	content := "font-size = 14\ninvalid-line-no-equals\ntheme = dark\n"
	err := fs.WriteFile("/home/user/.config/ghostty/config", []byte(content), 0o644)
	require.NoError(t, err)

	step := NewGhosttyConfigStep(
		&GhosttyConfig{Settings: map[string]interface{}{
			"font-size": "14",
			"theme":     "dark",
		}},
		"/home/user/.config/ghostty/config",
		"/tmp/dotfiles",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, compiler.StatusSatisfied, status)
}

// =============================================================================
// WindowsTerminal Scheme with minimal colors
// =============================================================================

func TestWindowsTerminalConfigStep_Apply_SchemeMinimalColors(t *testing.T) {
	t.Parallel()

	fs := mocks.NewFileSystem()
	step := NewWindowsTerminalConfigStep(
		&WindowsTerminalConfig{
			Schemes: []WindowsTerminalColorScheme{
				{
					Name:       "Minimal",
					Background: "#000",
					Foreground: "#fff",
					// All other fields empty
				},
			},
		},
		"/tmp/test/settings.json",
		fs,
	)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	content, err := fs.ReadFile("/tmp/test/settings.json")
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	schemes := result["schemes"].([]interface{})
	require.Len(t, schemes, 1)
	scheme := schemes[0].(map[string]interface{})

	// Should only have name, background, foreground
	assert.Equal(t, "Minimal", scheme["name"])
	assert.Equal(t, "#000", scheme["background"])
	assert.Equal(t, "#fff", scheme["foreground"])
	// Optional colors should NOT be present
	_, hasBlack := scheme["black"]
	assert.False(t, hasBlack)
}
