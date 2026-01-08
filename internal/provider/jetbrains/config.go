package jetbrains

import "fmt"

// Config represents the JetBrains configuration section in preflight.yaml.
type Config struct {
	// IDEs is the list of JetBrains IDEs to configure
	IDEs []IDEConfig `yaml:"ides,omitempty"`
	// SharedPlugins are plugins to install across all configured IDEs
	SharedPlugins []string `yaml:"shared_plugins,omitempty"`
	// SettingsSync enables JetBrains Settings Sync configuration
	SettingsSync *SettingsSyncConfig `yaml:"settings_sync,omitempty"`
}

// IDEConfig represents configuration for a specific JetBrains IDE.
type IDEConfig struct {
	// Name is the IDE identifier (e.g., "GoLand", "IntelliJIdea", "PyCharm")
	Name string `yaml:"name"`
	// Plugins to install for this specific IDE
	Plugins []string `yaml:"plugins,omitempty"`
	// Settings are IDE-specific settings to apply
	Settings map[string]interface{} `yaml:"settings,omitempty"`
	// Keymap is the keymap scheme to use
	Keymap string `yaml:"keymap,omitempty"`
	// CodeStyle is the code style scheme to use
	CodeStyle string `yaml:"code_style,omitempty"`
	// Disabled marks this IDE config as disabled
	Disabled bool `yaml:"disabled,omitempty"`
}

// SettingsSyncConfig represents JetBrains Settings Sync configuration.
type SettingsSyncConfig struct {
	// Enabled enables settings sync
	Enabled bool `yaml:"enabled,omitempty"`
	// SyncPlugins syncs plugins
	SyncPlugins bool `yaml:"sync_plugins,omitempty"`
	// SyncUI syncs UI settings
	SyncUI bool `yaml:"sync_ui,omitempty"`
	// SyncCodeStyles syncs code styles
	SyncCodeStyles bool `yaml:"sync_code_styles,omitempty"`
	// SyncKeymaps syncs keymaps
	SyncKeymaps bool `yaml:"sync_keymaps,omitempty"`
}

// ParseConfig parses the jetbrains section from the manifest.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	config := &Config{}

	// Parse shared_plugins
	if plugins, ok := raw["shared_plugins"]; ok {
		pluginList, ok := plugins.([]interface{})
		if ok {
			for _, p := range pluginList {
				if s, ok := p.(string); ok {
					config.SharedPlugins = append(config.SharedPlugins, s)
				}
			}
		}
	}

	// Parse ides
	if ides, ok := raw["ides"]; ok {
		ideList, ok := ides.([]interface{})
		if ok {
			for _, ide := range ideList {
				ideMap, ok := ide.(map[string]interface{})
				if !ok {
					continue
				}

				ideConfig := IDEConfig{}

				if name, ok := ideMap["name"].(string); ok {
					ideConfig.Name = name
				}

				if plugins, ok := ideMap["plugins"].([]interface{}); ok {
					for _, p := range plugins {
						if s, ok := p.(string); ok {
							ideConfig.Plugins = append(ideConfig.Plugins, s)
						}
					}
				}

				if settings, ok := ideMap["settings"].(map[string]interface{}); ok {
					ideConfig.Settings = settings
				}

				if keymap, ok := ideMap["keymap"].(string); ok {
					ideConfig.Keymap = keymap
				}

				if codeStyle, ok := ideMap["code_style"].(string); ok {
					ideConfig.CodeStyle = codeStyle
				}

				if disabled, ok := ideMap["disabled"].(bool); ok {
					ideConfig.Disabled = disabled
				}

				config.IDEs = append(config.IDEs, ideConfig)
			}
		}
	}

	// Parse settings_sync
	if sync, ok := raw["settings_sync"].(map[string]interface{}); ok {
		config.SettingsSync = &SettingsSyncConfig{}
		if enabled, ok := sync["enabled"].(bool); ok {
			config.SettingsSync.Enabled = enabled
		}
		if syncPlugins, ok := sync["sync_plugins"].(bool); ok {
			config.SettingsSync.SyncPlugins = syncPlugins
		}
		if syncUI, ok := sync["sync_ui"].(bool); ok {
			config.SettingsSync.SyncUI = syncUI
		}
		if syncCodeStyles, ok := sync["sync_code_styles"].(bool); ok {
			config.SettingsSync.SyncCodeStyles = syncCodeStyles
		}
		if syncKeymaps, ok := sync["sync_keymaps"].(bool); ok {
			config.SettingsSync.SyncKeymaps = syncKeymaps
		}
	}

	return config, nil
}

// GetIDEConfig returns the config for a specific IDE.
func (c *Config) GetIDEConfig(ide IDE) *IDEConfig {
	for i := range c.IDEs {
		if c.IDEs[i].Name == string(ide) {
			return &c.IDEs[i]
		}
	}
	return nil
}

// GetAllPluginsForIDE returns all plugins to install for an IDE (shared + IDE-specific).
func (c *Config) GetAllPluginsForIDE(ide IDE) []string {
	plugins := make([]string, len(c.SharedPlugins))
	copy(plugins, c.SharedPlugins)

	ideConfig := c.GetIDEConfig(ide)
	if ideConfig != nil {
		plugins = append(plugins, ideConfig.Plugins...)
	}

	return plugins
}

// HasIDEConfig returns true if any IDE configurations are specified.
func (c *Config) HasIDEConfig() bool {
	return len(c.IDEs) > 0
}

// HasSharedPlugins returns true if shared plugins are specified.
func (c *Config) HasSharedPlugins() bool {
	return len(c.SharedPlugins) > 0
}

// HasSettingsSync returns true if settings sync is configured.
func (c *Config) HasSettingsSync() bool {
	return c.SettingsSync != nil && c.SettingsSync.Enabled
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	// Validate IDE names
	validIDEs := make(map[string]bool)
	for _, ide := range AllIDEs() {
		validIDEs[string(ide)] = true
	}

	for _, ideConfig := range c.IDEs {
		if !validIDEs[ideConfig.Name] {
			return fmt.Errorf("unknown JetBrains IDE: %s", ideConfig.Name)
		}
	}

	return nil
}
