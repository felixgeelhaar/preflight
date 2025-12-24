// Package winget provides the Windows Package Manager (winget) provider.
package winget

import (
	"fmt"
)

// Config represents the winget section of the configuration.
type Config struct {
	Packages []Package
}

// Package represents a winget package to install.
type Package struct {
	ID      string // Package ID (e.g., "Microsoft.VisualStudioCode")
	Version string // Optional: specific version (e.g., "1.85.0")
	Source  string // Optional: source name (e.g., "winget", "msstore")
}

// ParseConfig parses the winget configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Packages: make([]Package, 0),
	}

	// Parse packages
	if packages, ok := raw["packages"]; ok {
		packageList, ok := packages.([]interface{})
		if !ok {
			return nil, fmt.Errorf("packages must be a list")
		}
		for _, p := range packageList {
			pkg, err := parsePackage(p)
			if err != nil {
				return nil, err
			}
			cfg.Packages = append(cfg.Packages, pkg)
		}
	}

	return cfg, nil
}

// parsePackage parses a single package from either a string or a map.
func parsePackage(raw interface{}) (Package, error) {
	switch v := raw.(type) {
	case string:
		return Package{ID: v}, nil
	case map[string]interface{}:
		pkg := Package{}
		if id, ok := v["id"].(string); ok {
			pkg.ID = id
		} else {
			return Package{}, fmt.Errorf("package must have an id")
		}
		if version, ok := v["version"].(string); ok {
			pkg.Version = version
		}
		if source, ok := v["source"].(string); ok {
			pkg.Source = source
		}
		return pkg, nil
	default:
		return Package{}, fmt.Errorf("package must be a string or object")
	}
}
