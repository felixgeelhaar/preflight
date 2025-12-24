// Package scoop provides the Scoop provider for package management on Windows.
package scoop

import (
	"fmt"
)

// Config represents the scoop section of the configuration.
type Config struct {
	Buckets  []Bucket
	Packages []Package
}

// Bucket represents a Scoop bucket to add.
type Bucket struct {
	Name string
	URL  string // Optional: custom URL for the bucket
}

// Package represents a Scoop package to install.
type Package struct {
	Name    string
	Bucket  string // Optional: specific bucket (e.g., "extras")
	Version string // Optional: specific version
}

// FullName returns the fully qualified package name with bucket prefix.
func (p Package) FullName() string {
	if p.Bucket != "" {
		return fmt.Sprintf("%s/%s", p.Bucket, p.Name)
	}
	return p.Name
}

// ParseConfig parses the scoop configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Buckets:  make([]Bucket, 0),
		Packages: make([]Package, 0),
	}

	// Parse buckets
	if buckets, ok := raw["buckets"]; ok {
		bucketList, ok := buckets.([]interface{})
		if !ok {
			return nil, fmt.Errorf("buckets must be a list")
		}
		for _, b := range bucketList {
			bucket, err := parseBucket(b)
			if err != nil {
				return nil, err
			}
			cfg.Buckets = append(cfg.Buckets, bucket)
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

// parseBucket parses a single bucket from either a string or a map.
func parseBucket(raw interface{}) (Bucket, error) {
	switch v := raw.(type) {
	case string:
		return Bucket{Name: v}, nil
	case map[string]interface{}:
		bucket := Bucket{}
		if name, ok := v["name"].(string); ok {
			bucket.Name = name
		} else {
			return Bucket{}, fmt.Errorf("bucket must have a name")
		}
		if url, ok := v["url"].(string); ok {
			bucket.URL = url
		}
		return bucket, nil
	default:
		return Bucket{}, fmt.Errorf("bucket must be a string or object")
	}
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
		if bucket, ok := v["bucket"].(string); ok {
			pkg.Bucket = bucket
		}
		if version, ok := v["version"].(string); ok {
			pkg.Version = version
		}
		return pkg, nil
	default:
		return Package{}, fmt.Errorf("package must be a string or object")
	}
}
