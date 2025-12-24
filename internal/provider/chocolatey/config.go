// Package chocolatey provides the Chocolatey package manager provider for Windows.
package chocolatey

import (
	"fmt"
)

// Config represents the chocolatey section of the configuration.
type Config struct {
	Sources  []Source
	Packages []Package
}

// Source represents a Chocolatey source (feed/repository).
type Source struct {
	Name     string // Source name (e.g., "chocolatey", "internal")
	URL      string // Source URL (e.g., "https://community.chocolatey.org/api/v2/")
	Priority int    // Optional: priority (lower = higher priority)
	Disabled bool   // Optional: whether source is disabled
}

// Package represents a Chocolatey package to install.
type Package struct {
	Name    string // Package name (e.g., "git", "nodejs")
	Version string // Optional: specific version (e.g., "2.40.0")
	Source  string // Optional: source name to install from
	Args    string // Optional: package-specific arguments
	Pin     bool   // Optional: pin package to prevent upgrades
}

// ParseConfig parses the chocolatey configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Sources:  make([]Source, 0),
		Packages: make([]Package, 0),
	}

	// Parse sources
	if sources, ok := raw["sources"]; ok {
		sourceList, ok := sources.([]interface{})
		if !ok {
			return nil, fmt.Errorf("sources must be a list")
		}
		for _, s := range sourceList {
			src, err := parseSource(s)
			if err != nil {
				return nil, err
			}
			cfg.Sources = append(cfg.Sources, src)
		}
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

// parseSource parses a source from either a string or a map.
func parseSource(raw interface{}) (Source, error) {
	switch v := raw.(type) {
	case map[string]interface{}:
		src := Source{}
		if name, ok := v["name"].(string); ok {
			src.Name = name
		} else {
			return Source{}, fmt.Errorf("source must have a name")
		}
		if url, ok := v["url"].(string); ok {
			src.URL = url
		} else {
			return Source{}, fmt.Errorf("source must have a url")
		}
		if priority, ok := v["priority"].(int); ok {
			src.Priority = priority
		}
		if disabled, ok := v["disabled"].(bool); ok {
			src.Disabled = disabled
		}
		return src, nil
	default:
		return Source{}, fmt.Errorf("source must be an object with name and url")
	}
}

// parsePackage parses a package from either a string or a map.
func parsePackage(raw interface{}) (Package, error) {
	switch v := raw.(type) {
	case string:
		return Package{Name: v}, nil
	case map[string]interface{}:
		pkg := Package{}
		if name, ok := v["name"].(string); ok {
			pkg.Name = name
		} else {
			return Package{}, fmt.Errorf("package must have a name")
		}
		if version, ok := v["version"].(string); ok {
			pkg.Version = version
		}
		if source, ok := v["source"].(string); ok {
			pkg.Source = source
		}
		if args, ok := v["args"].(string); ok {
			pkg.Args = args
		}
		if pin, ok := v["pin"].(bool); ok {
			pkg.Pin = pin
		}
		return pkg, nil
	default:
		return Package{}, fmt.Errorf("package must be a string or object")
	}
}
