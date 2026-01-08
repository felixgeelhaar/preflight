package windsurf

import "fmt"

// Config represents the Windsurf configuration section in preflight.yaml.
type Config struct {
	Extensions  []string               `yaml:"extensions,omitempty"`
	Settings    map[string]interface{} `yaml:"settings,omitempty"`
	Keybindings []Keybinding           `yaml:"keybindings,omitempty"`
}

// Keybinding represents a Windsurf keybinding.
type Keybinding struct {
	Key     string `yaml:"key" json:"key"`
	Command string `yaml:"command" json:"command"`
	When    string `yaml:"when,omitempty" json:"when,omitempty"`
}

// ParseConfig parses the windsurf section from the manifest.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	config := &Config{}

	// Parse extensions
	if ext, ok := raw["extensions"]; ok {
		extList, ok := ext.([]interface{})
		if !ok {
			return nil, fmt.Errorf("windsurf.extensions must be a list")
		}
		for _, e := range extList {
			extStr, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("windsurf.extensions items must be strings")
			}
			config.Extensions = append(config.Extensions, extStr)
		}
	}

	// Parse settings
	if settings, ok := raw["settings"]; ok {
		settingsMap, ok := settings.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("windsurf.settings must be a map")
		}
		config.Settings = settingsMap
	}

	// Parse keybindings
	if kb, ok := raw["keybindings"]; ok {
		kbList, ok := kb.([]interface{})
		if !ok {
			return nil, fmt.Errorf("windsurf.keybindings must be a list")
		}
		for _, k := range kbList {
			kbMap, ok := k.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("windsurf.keybindings items must be objects")
			}
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
			config.Keybindings = append(config.Keybindings, keybinding)
		}
	}

	return config, nil
}
