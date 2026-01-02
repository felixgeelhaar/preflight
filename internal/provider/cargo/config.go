// Package cargo provides the Cargo provider for Rust crate installation.
package cargo

import (
	"fmt"
	"strings"
)

// Config represents the cargo section of the configuration.
type Config struct {
	Crates []Crate
}

// Crate represents a Rust crate to install.
type Crate struct {
	Name    string
	Version string // Optional: specific version
}

// FullName returns the crate name with optional version.
func (c Crate) FullName() string {
	if c.Version != "" {
		return fmt.Sprintf("%s@%s", c.Name, c.Version)
	}
	return c.Name
}

// ParseConfig parses the cargo configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Crates: make([]Crate, 0),
	}

	// Parse crates
	if crates, ok := raw["crates"]; ok {
		crateList, ok := crates.([]interface{})
		if !ok {
			return nil, fmt.Errorf("crates must be a list")
		}
		for _, c := range crateList {
			crate, err := parseCrate(c)
			if err != nil {
				return nil, err
			}
			cfg.Crates = append(cfg.Crates, crate)
		}
	}

	return cfg, nil
}

// parseCrate parses a single crate from either a string or a map.
func parseCrate(raw interface{}) (Crate, error) {
	switch v := raw.(type) {
	case string:
		return parseCrateString(v), nil
	case map[string]interface{}:
		crate := Crate{}
		if name, ok := v["name"].(string); ok {
			crate.Name = name
		} else {
			return Crate{}, fmt.Errorf("crate must have a name")
		}
		if version, ok := v["version"].(string); ok {
			crate.Version = version
		}
		return crate, nil
	default:
		return Crate{}, fmt.Errorf("crate must be a string or object")
	}
}

// parseCrateString parses a crate string like "crate" or "crate@version".
func parseCrateString(s string) Crate {
	if atIndex := strings.LastIndex(s, "@"); atIndex > 0 {
		return Crate{
			Name:    s[:atIndex],
			Version: s[atIndex+1:],
		}
	}
	return Crate{Name: s}
}
