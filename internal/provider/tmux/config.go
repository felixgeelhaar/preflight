package tmux

import (
	"fmt"
)

// Config represents the tmux section of the configuration.
type Config struct {
	Plugins    []string
	Settings   map[string]string
	ConfigFile string
}

// ParseConfig parses the tmux configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Plugins:  make([]string, 0),
		Settings: make(map[string]string),
	}

	// Parse plugins
	if plugins, ok := raw["plugins"]; ok {
		pluginList, ok := plugins.([]interface{})
		if !ok {
			return nil, fmt.Errorf("plugins must be a list")
		}
		for _, p := range pluginList {
			pluginStr, ok := p.(string)
			if !ok {
				return nil, fmt.Errorf("plugin must be a string")
			}
			cfg.Plugins = append(cfg.Plugins, pluginStr)
		}
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		for key, value := range settings {
			if valueStr, ok := value.(string); ok {
				cfg.Settings[key] = valueStr
			}
		}
	}

	// Parse config file path
	if configFile, ok := raw["config_file"].(string); ok {
		cfg.ConfigFile = configFile
	}

	return cfg, nil
}
