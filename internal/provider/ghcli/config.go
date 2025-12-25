package ghcli

import (
	"fmt"
)

// Config represents the github-cli section of the configuration.
type Config struct {
	Extensions []string
	Aliases    map[string]string
	Config     map[string]string
}

// ParseConfig parses the github-cli configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Extensions: make([]string, 0),
		Aliases:    make(map[string]string),
		Config:     make(map[string]string),
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

	// Parse aliases
	if aliases, ok := raw["aliases"].(map[string]interface{}); ok {
		for name, cmd := range aliases {
			if cmdStr, ok := cmd.(string); ok {
				cfg.Aliases[name] = cmdStr
			}
		}
	}

	// Parse config
	if config, ok := raw["config"].(map[string]interface{}); ok {
		for key, val := range config {
			if valStr, ok := val.(string); ok {
				cfg.Config[key] = valStr
			}
		}
	}

	return cfg, nil
}
