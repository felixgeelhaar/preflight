package helix

import "fmt"

// Config represents the Helix configuration section in preflight.yaml.
type Config struct {
	// Source is the path to a config.toml file to copy/link
	Source string `yaml:"source,omitempty"`
	// Link indicates whether to symlink (true) or copy (false)
	Link bool `yaml:"link,omitempty"`
	// Languages is the path to a languages.toml file to copy/link
	Languages string `yaml:"languages,omitempty"`
	// Theme is the name of the theme to use
	Theme string `yaml:"theme,omitempty"`
	// ThemeSource is the path to a custom theme file
	ThemeSource string `yaml:"theme_source,omitempty"`
	// Settings are individual config.toml settings to merge
	Settings map[string]interface{} `yaml:"settings,omitempty"`
	// EditorSettings are settings under [editor] section
	EditorSettings map[string]interface{} `yaml:"editor,omitempty"`
	// KeysSettings are keybinding settings under [keys] section
	KeysSettings map[string]interface{} `yaml:"keys,omitempty"`
}

// ParseConfig parses the helix section from the manifest.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	config := &Config{}

	// Parse source
	if src, ok := raw["source"].(string); ok {
		config.Source = src
	}

	// Parse link flag
	if link, ok := raw["link"].(bool); ok {
		config.Link = link
	}

	// Parse languages
	if lang, ok := raw["languages"].(string); ok {
		config.Languages = lang
	}

	// Parse theme
	if theme, ok := raw["theme"].(string); ok {
		config.Theme = theme
	}

	// Parse theme_source
	if themeSrc, ok := raw["theme_source"].(string); ok {
		config.ThemeSource = themeSrc
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		config.Settings = settings
	}

	// Parse editor settings
	if editor, ok := raw["editor"].(map[string]interface{}); ok {
		config.EditorSettings = editor
	}

	// Parse keys settings
	if keys, ok := raw["keys"].(map[string]interface{}); ok {
		config.KeysSettings = keys
	}

	return config, nil
}

// HasConfigSource returns true if a source config file is specified.
func (c *Config) HasConfigSource() bool {
	return c.Source != ""
}

// HasLanguages returns true if a languages.toml is specified.
func (c *Config) HasLanguages() bool {
	return c.Languages != ""
}

// HasTheme returns true if a theme is specified.
func (c *Config) HasTheme() bool {
	return c.Theme != "" || c.ThemeSource != ""
}

// HasSettings returns true if any settings are specified.
func (c *Config) HasSettings() bool {
	return len(c.Settings) > 0 || len(c.EditorSettings) > 0 || len(c.KeysSettings) > 0
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Cannot have both source file and settings
	if c.HasConfigSource() && c.HasSettings() {
		return fmt.Errorf("cannot specify both 'source' and individual settings (settings, editor, keys)")
	}

	return nil
}
