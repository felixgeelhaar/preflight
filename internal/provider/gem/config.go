// Package gem provides the RubyGems provider for gem installation.
package gem

import (
	"fmt"
	"strings"
)

// Config represents the gem section of the configuration.
type Config struct {
	Gems []Gem
}

// Gem represents a Ruby gem to install.
type Gem struct {
	Name    string
	Version string // Optional: specific version
}

// FullName returns the gem name with optional version.
func (g Gem) FullName() string {
	if g.Version != "" {
		return fmt.Sprintf("%s@%s", g.Name, g.Version)
	}
	return g.Name
}

// ParseConfig parses the gem configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Gems: make([]Gem, 0),
	}

	// Parse gems
	if gems, ok := raw["gems"]; ok {
		gemList, ok := gems.([]interface{})
		if !ok {
			return nil, fmt.Errorf("gems must be a list")
		}
		for _, g := range gemList {
			gem, err := parseGem(g)
			if err != nil {
				return nil, err
			}
			cfg.Gems = append(cfg.Gems, gem)
		}
	}

	return cfg, nil
}

// parseGem parses a single gem from either a string or a map.
func parseGem(raw interface{}) (Gem, error) {
	switch v := raw.(type) {
	case string:
		return parseGemString(v), nil
	case map[string]interface{}:
		gem := Gem{}
		if name, ok := v["name"].(string); ok {
			gem.Name = name
		} else {
			return Gem{}, fmt.Errorf("gem must have a name")
		}
		if version, ok := v["version"].(string); ok {
			gem.Version = version
		}
		return gem, nil
	default:
		return Gem{}, fmt.Errorf("gem must be a string or object")
	}
}

// parseGemString parses a gem string like "gem" or "gem@version".
func parseGemString(s string) Gem {
	if atIndex := strings.LastIndex(s, "@"); atIndex > 0 {
		return Gem{
			Name:    s[:atIndex],
			Version: s[atIndex+1:],
		}
	}
	return Gem{Name: s}
}
