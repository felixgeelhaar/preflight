// Package apt provides the apt provider for package management on Debian/Ubuntu.
package apt

import (
	"fmt"
)

// Config represents the apt section of the configuration.
type Config struct {
	PPAs     []string
	Packages []Package
}

// Package represents an apt package to install.
type Package struct {
	Name    string
	Version string // Optional: specific version
}

// FullName returns the package name with optional version specifier.
func (p Package) FullName() string {
	if p.Version != "" {
		return fmt.Sprintf("%s=%s", p.Name, p.Version)
	}
	return p.Name
}

// ParseConfig parses the apt configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		PPAs:     make([]string, 0),
		Packages: make([]Package, 0),
	}

	// Parse PPAs
	if ppas, ok := raw["ppas"]; ok {
		ppaList, ok := ppas.([]interface{})
		if !ok {
			return nil, fmt.Errorf("ppas must be a list")
		}
		for _, ppa := range ppaList {
			ppaStr, ok := ppa.(string)
			if !ok {
				return nil, fmt.Errorf("ppa must be a string")
			}
			cfg.PPAs = append(cfg.PPAs, ppaStr)
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

// parsePackage parses a single package from either a string or a map.
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
		return pkg, nil
	default:
		return Package{}, fmt.Errorf("package must be a string or object")
	}
}
