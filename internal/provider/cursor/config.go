package cursor

import (
	"fmt"
)

// Config represents the cursor section of the configuration.
type Config struct {
	Extensions  []string
	Settings    map[string]interface{}
	Keybindings []Keybinding
}

// Keybinding represents a keyboard shortcut.
type Keybinding struct {
	Key     string
	Command string
	When    string
}

// ParseConfig parses the cursor configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Extensions:  make([]string, 0),
		Settings:    make(map[string]interface{}),
		Keybindings: make([]Keybinding, 0),
	}

	// Parse extensions
	if extensions, ok := raw["extensions"]; ok {
		extList, ok := extensions.([]interface{})
		if !ok {
			return nil, fmt.Errorf("extensions must be a list")
		}
		for _, ext := range extList {
			extStr, ok := ext.(string)
			if !ok {
				return nil, fmt.Errorf("extension must be a string")
			}
			cfg.Extensions = append(cfg.Extensions, extStr)
		}
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		cfg.Settings = settings
	}

	// Parse keybindings
	if keybindings, ok := raw["keybindings"].([]interface{}); ok {
		for _, kb := range keybindings {
			keybinding, err := parseKeybinding(kb)
			if err != nil {
				return nil, err
			}
			cfg.Keybindings = append(cfg.Keybindings, keybinding)
		}
	}

	return cfg, nil
}

func parseKeybinding(raw interface{}) (Keybinding, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Keybinding{}, fmt.Errorf("keybinding must be an object")
	}

	kb := Keybinding{}

	if key, ok := m["key"].(string); ok {
		kb.Key = key
	} else {
		return Keybinding{}, fmt.Errorf("keybinding must have a key")
	}

	if cmd, ok := m["command"].(string); ok {
		kb.Command = cmd
	} else {
		return Keybinding{}, fmt.Errorf("keybinding must have a command")
	}

	if when, ok := m["when"].(string); ok {
		kb.When = when
	}

	return kb, nil
}
