package macos

import (
	"fmt"
)

// Config represents the macos section of the configuration.
type Config struct {
	Defaults []Default
	Dock     DockConfig
	Finder   FinderConfig
	Keyboard KeyboardConfig
}

// Default represents a single macOS defaults setting.
type Default struct {
	Domain string
	Key    string
	Type   string // string, int, float, bool, array, dict
	Value  interface{}
}

// DockConfig represents Dock preferences.
type DockConfig struct {
	Add    []string // Apps to add to dock
	Remove []string // Apps to remove from dock
}

// FinderConfig represents Finder preferences.
type FinderConfig struct {
	ShowHidden     *bool
	ShowExtensions *bool
	ShowPathBar    *bool
}

// KeyboardConfig represents keyboard preferences.
type KeyboardConfig struct {
	KeyRepeat        *int
	InitialKeyRepeat *int
}

// ParseConfig parses the macos configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Defaults: make([]Default, 0),
	}

	// Parse defaults
	if defaults, ok := raw["defaults"]; ok {
		defaultsList, ok := defaults.([]interface{})
		if !ok {
			return nil, fmt.Errorf("defaults must be a list")
		}
		for _, d := range defaultsList {
			def, err := parseDefault(d)
			if err != nil {
				return nil, err
			}
			cfg.Defaults = append(cfg.Defaults, def)
		}
	}

	// Parse dock
	if dock, ok := raw["dock"].(map[string]interface{}); ok {
		if add, ok := dock["add"].([]interface{}); ok {
			for _, item := range add {
				if s, ok := item.(string); ok {
					cfg.Dock.Add = append(cfg.Dock.Add, s)
				}
			}
		}
		if remove, ok := dock["remove"].([]interface{}); ok {
			for _, item := range remove {
				if s, ok := item.(string); ok {
					cfg.Dock.Remove = append(cfg.Dock.Remove, s)
				}
			}
		}
	}

	// Parse finder
	if finder, ok := raw["finder"].(map[string]interface{}); ok {
		if v, ok := finder["show_hidden"].(bool); ok {
			cfg.Finder.ShowHidden = &v
		}
		if v, ok := finder["show_extensions"].(bool); ok {
			cfg.Finder.ShowExtensions = &v
		}
		if v, ok := finder["show_path_bar"].(bool); ok {
			cfg.Finder.ShowPathBar = &v
		}
	}

	// Parse keyboard
	if keyboard, ok := raw["keyboard"].(map[string]interface{}); ok {
		if v, ok := keyboard["key_repeat"].(int); ok {
			cfg.Keyboard.KeyRepeat = &v
		}
		if v, ok := keyboard["initial_key_repeat"].(int); ok {
			cfg.Keyboard.InitialKeyRepeat = &v
		}
	}

	return cfg, nil
}

func parseDefault(raw interface{}) (Default, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Default{}, fmt.Errorf("default must be an object")
	}

	def := Default{}

	if domain, ok := m["domain"].(string); ok {
		def.Domain = domain
	} else {
		return Default{}, fmt.Errorf("default must have a domain")
	}

	if key, ok := m["key"].(string); ok {
		def.Key = key
	} else {
		return Default{}, fmt.Errorf("default must have a key")
	}

	if t, ok := m["type"].(string); ok {
		def.Type = t
	} else {
		def.Type = "string"
	}

	def.Value = m["value"]

	return def, nil
}
