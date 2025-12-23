// Package vscode provides the VSCode editor provider for Preflight.
// It handles extension installation, settings management, and keybindings configuration.
package vscode

// Keybinding represents a single keybinding configuration.
type Keybinding struct {
	Key     string `yaml:"key"`
	Command string `yaml:"command"`
	When    string `yaml:"when,omitempty"`
	Args    string `yaml:"args,omitempty"`
}

// Config represents VSCode configuration.
type Config struct {
	Extensions  []string
	Settings    map[string]interface{}
	Keybindings []Keybinding
}

// ParseConfig parses raw config into VSCode Config.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Settings: make(map[string]interface{}),
	}

	if raw == nil {
		return cfg, nil
	}

	// Parse extensions
	if exts, ok := raw["extensions"].([]interface{}); ok {
		for _, ext := range exts {
			if extStr, ok := ext.(string); ok {
				cfg.Extensions = append(cfg.Extensions, extStr)
			}
		}
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		cfg.Settings = settings
	}

	// Parse keybindings
	if keybindings, ok := raw["keybindings"].([]interface{}); ok {
		for _, kb := range keybindings {
			if kbMap, ok := kb.(map[string]interface{}); ok {
				keybinding := Keybinding{}
				if key, ok := kbMap["key"].(string); ok {
					keybinding.Key = key
				}
				if cmd, ok := kbMap["command"].(string); ok {
					keybinding.Command = cmd
				}
				if when, ok := kbMap["when"].(string); ok {
					keybinding.When = when
				}
				if args, ok := kbMap["args"].(string); ok {
					keybinding.Args = args
				}
				cfg.Keybindings = append(cfg.Keybindings, keybinding)
			}
		}
	}

	return cfg, nil
}
