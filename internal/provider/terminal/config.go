// Package terminal provides configuration management for terminal emulators.
package terminal

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config represents the terminal section of the configuration.
type Config struct {
	// Global settings applied to all terminals where applicable
	Font  *FontConfig  `yaml:"font,omitempty"`
	Theme *ThemeConfig `yaml:"theme,omitempty"`

	// Per-terminal configurations
	Alacritty       *AlacrittyConfig       `yaml:"alacritty,omitempty"`
	Kitty           *KittyConfig           `yaml:"kitty,omitempty"`
	WezTerm         *WezTermConfig         `yaml:"wezterm,omitempty"`
	Ghostty         *GhosttyConfig         `yaml:"ghostty,omitempty"`
	ITerm2          *ITerm2Config          `yaml:"iterm2,omitempty"`
	Hyper           *HyperConfig           `yaml:"hyper,omitempty"`
	WindowsTerminal *WindowsTerminalConfig `yaml:"windows_terminal,omitempty"`
}

// FontConfig represents font settings applicable to terminals.
type FontConfig struct {
	Family string  `yaml:"family"`
	Size   float64 `yaml:"size"`
}

// ThemeConfig represents theme/colorscheme settings.
type ThemeConfig struct {
	Name   string            `yaml:"name"`   // Preset theme name (e.g., "catppuccin-mocha")
	Custom map[string]string `yaml:"custom"` // Custom color overrides
}

// AlacrittyConfig represents Alacritty terminal configuration.
type AlacrittyConfig struct {
	// ConfigPath overrides the default config location
	ConfigPath string `yaml:"config_path,omitempty"`

	// Source is the path to the config file in dotfiles/
	Source string `yaml:"source,omitempty"`

	// Link creates a symlink instead of copying
	Link bool `yaml:"link,omitempty"`

	// Settings are merged into the config (TOML format internally)
	Settings map[string]interface{} `yaml:"settings,omitempty"`
}

// KittyConfig represents Kitty terminal configuration.
type KittyConfig struct {
	ConfigPath string                 `yaml:"config_path,omitempty"`
	Source     string                 `yaml:"source,omitempty"`
	Link       bool                   `yaml:"link,omitempty"`
	Settings   map[string]interface{} `yaml:"settings,omitempty"`
	Theme      string                 `yaml:"theme,omitempty"` // Kitty theme name
}

// WezTermConfig represents WezTerm terminal configuration.
type WezTermConfig struct {
	ConfigPath string `yaml:"config_path,omitempty"`
	Source     string `yaml:"source,omitempty"`
	Link       bool   `yaml:"link,omitempty"`
	// Note: WezTerm uses Lua, so we only support source/link, not settings merge
}

// GhosttyConfig represents Ghostty terminal configuration.
type GhosttyConfig struct {
	ConfigPath string                 `yaml:"config_path,omitempty"`
	Source     string                 `yaml:"source,omitempty"`
	Link       bool                   `yaml:"link,omitempty"`
	Settings   map[string]interface{} `yaml:"settings,omitempty"`
}

// ITerm2Config represents iTerm2 terminal configuration (macOS only).
type ITerm2Config struct {
	// Settings are plist key-value pairs to set via defaults command
	Settings map[string]interface{} `yaml:"settings,omitempty"`

	// DynamicProfiles are iTerm2 dynamic profiles to create
	DynamicProfiles []ITerm2Profile `yaml:"dynamic_profiles,omitempty"`

	// Source is the path to a plist file to import
	Source string `yaml:"source,omitempty"`
}

// ITerm2Profile represents an iTerm2 dynamic profile.
type ITerm2Profile struct {
	Name        string            `yaml:"name"`
	GUID        string            `yaml:"guid,omitempty"`
	Font        string            `yaml:"font,omitempty"`
	FontSize    float64           `yaml:"font_size,omitempty"`
	ColorScheme string            `yaml:"color_scheme,omitempty"`
	Custom      map[string]string `yaml:"custom,omitempty"`
}

// HyperConfig represents Hyper terminal configuration.
type HyperConfig struct {
	ConfigPath string   `yaml:"config_path,omitempty"`
	Source     string   `yaml:"source,omitempty"`
	Link       bool     `yaml:"link,omitempty"`
	Plugins    []string `yaml:"plugins,omitempty"`
	// Note: Hyper uses JavaScript, so we only support source/link for full config
}

// WindowsTerminalConfig represents Windows Terminal configuration.
type WindowsTerminalConfig struct {
	// Settings are merged into settings.json
	Settings map[string]interface{} `yaml:"settings,omitempty"`

	// Profiles to create or update
	Profiles []WindowsTerminalProfile `yaml:"profiles,omitempty"`

	// Color schemes to add
	Schemes []WindowsTerminalColorScheme `yaml:"schemes,omitempty"`

	// Source is the path to a settings.json to import
	Source string `yaml:"source,omitempty"`
}

// WindowsTerminalProfile represents a Windows Terminal profile.
type WindowsTerminalProfile struct {
	Name           string   `yaml:"name"`
	GUID           string   `yaml:"guid,omitempty"`
	CommandLine    string   `yaml:"command_line,omitempty"`
	ColorScheme    string   `yaml:"color_scheme,omitempty"`
	FontFace       string   `yaml:"font_face,omitempty"`
	FontSize       int      `yaml:"font_size,omitempty"`
	UseAcrylic     *bool    `yaml:"use_acrylic,omitempty"`
	AcrylicOpacity *float64 `yaml:"acrylic_opacity,omitempty"`
}

// WindowsTerminalColorScheme represents a Windows Terminal color scheme.
type WindowsTerminalColorScheme struct {
	Name            string `yaml:"name"`
	Background      string `yaml:"background"`
	Foreground      string `yaml:"foreground"`
	Black           string `yaml:"black,omitempty"`
	Red             string `yaml:"red,omitempty"`
	Green           string `yaml:"green,omitempty"`
	Yellow          string `yaml:"yellow,omitempty"`
	Blue            string `yaml:"blue,omitempty"`
	Purple          string `yaml:"purple,omitempty"`
	Cyan            string `yaml:"cyan,omitempty"`
	White           string `yaml:"white,omitempty"`
	BrightBlack     string `yaml:"bright_black,omitempty"`
	BrightRed       string `yaml:"bright_red,omitempty"`
	BrightGreen     string `yaml:"bright_green,omitempty"`
	BrightYellow    string `yaml:"bright_yellow,omitempty"`
	BrightBlue      string `yaml:"bright_blue,omitempty"`
	BrightPurple    string `yaml:"bright_purple,omitempty"`
	BrightCyan      string `yaml:"bright_cyan,omitempty"`
	BrightWhite     string `yaml:"bright_white,omitempty"`
	CursorColor     string `yaml:"cursor_color,omitempty"`
	SelectionColor  string `yaml:"selection_color,omitempty"`
}

// ParseConfig parses the terminal configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	// Marshal back to YAML and unmarshal into typed struct
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal terminal config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse terminal config: %w", err)
	}

	return &cfg, nil
}

// HasAnyTerminal returns true if any terminal is configured.
func (c *Config) HasAnyTerminal() bool {
	return c.Alacritty != nil ||
		c.Kitty != nil ||
		c.WezTerm != nil ||
		c.Ghostty != nil ||
		c.ITerm2 != nil ||
		c.Hyper != nil ||
		c.WindowsTerminal != nil
}
