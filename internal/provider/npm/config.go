// Package npm provides the npm provider for global package management.
package npm

import (
	"fmt"
	"strings"
)

// Config represents the npm section of the configuration.
type Config struct {
	Packages []Package
}

// Package represents an npm package to install globally.
type Package struct {
	Name    string
	Version string // Optional: specific version
}

// FullName returns the package name with optional version.
func (p Package) FullName() string {
	if p.Version != "" {
		return fmt.Sprintf("%s@%s", p.Name, p.Version)
	}
	return p.Name
}

// ParseConfig parses the npm configuration from a raw map.
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
		return parsePackageString(v), nil
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

// parsePackageString parses a package string like "pkg" or "pkg@version" or "@scope/pkg@version".
func parsePackageString(s string) Package {
	// Handle scoped packages: @scope/pkg@version
	if strings.HasPrefix(s, "@") {
		// Find the second @ which separates name from version
		atIndex := strings.LastIndex(s, "@")
		if atIndex > 0 && atIndex != strings.Index(s, "@") {
			return Package{
				Name:    s[:atIndex],
				Version: s[atIndex+1:],
			}
		}
		return Package{Name: s}
	}

	// Regular package: pkg@version
	if atIndex := strings.Index(s, "@"); atIndex > 0 {
		return Package{
			Name:    s[:atIndex],
			Version: s[atIndex+1:],
		}
	}

	return Package{Name: s}
}
