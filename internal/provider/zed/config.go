package zed

import (
	"fmt"
)

// Config represents the zed section of the configuration.
type Config struct {
	Extensions []string
	Settings   map[string]interface{}
	Keymap     []KeyBinding
	Theme      string
}

// KeyBinding represents a Zed key binding.
type KeyBinding struct {
	Context  string
	Bindings map[string]string
}

// ParseConfig parses the zed configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Extensions: make([]string, 0),
		Settings:   make(map[string]interface{}),
		Keymap:     make([]KeyBinding, 0),
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

	// Parse keymap
	if keymap, ok := raw["keymap"].([]interface{}); ok {
		for _, km := range keymap {
			keybinding, err := parseKeyBinding(km)
			if err != nil {
				return nil, err
			}
			cfg.Keymap = append(cfg.Keymap, keybinding)
		}
	}

	// Parse theme
	if theme, ok := raw["theme"].(string); ok {
		cfg.Theme = theme
	}

	return cfg, nil
}

func parseKeyBinding(raw interface{}) (KeyBinding, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return KeyBinding{}, fmt.Errorf("keybinding must be an object")
	}

	kb := KeyBinding{
		Bindings: make(map[string]string),
	}

	if context, ok := m["context"].(string); ok {
		kb.Context = context
	}

	if bindings, ok := m["bindings"].(map[string]interface{}); ok {
		for key, cmd := range bindings {
			if cmdStr, ok := cmd.(string); ok {
				kb.Bindings[key] = cmdStr
			}
		}
	}

	return kb, nil
}
