// Package pip provides the pip provider for Python package management.
package pip

import (
	"fmt"
	"regexp"
)

// Config represents the pip section of the configuration.
type Config struct {
	Packages []Package
}

// Package represents a pip package to install.
type Package struct {
	Name    string
	Version string // Optional: version specifier (e.g., "==23.1.0", ">=3.0")
}

// versionSpecifierRegex matches pip version specifiers.
var versionSpecifierRegex = regexp.MustCompile(`^([=<>!~]+)(.+)$`)

// FullName returns the package name with optional version specifier.
func (p Package) FullName() string {
	if p.Version != "" {
		// Check if version already has specifier
		if versionSpecifierRegex.MatchString(p.Version) {
			return p.Name + p.Version
		}
		// Default to == for exact version
		return fmt.Sprintf("%s==%s", p.Name, p.Version)
	}
	return p.Name
}

// ParseConfig parses the pip configuration from a raw map.
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

// parsePackageString parses a package string like "pkg" or "pkg==version" or "pkg>=version".
func parsePackageString(s string) Package {
	// Find version specifier: ==, >=, <=, !=, ~=, <, >
	specifiers := []string{"==", ">=", "<=", "!=", "~=", "<", ">"}
	for _, spec := range specifiers {
		if idx := findSpecifier(s, spec); idx > 0 {
			return Package{
				Name:    s[:idx],
				Version: s[idx:], // Include the specifier
			}
		}
	}
	return Package{Name: s}
}

// findSpecifier finds the index of a version specifier in the string.
func findSpecifier(s, spec string) int {
	for i := 0; i <= len(s)-len(spec); i++ {
		if s[i:i+len(spec)] == spec {
			return i
		}
	}
	return -1
}
