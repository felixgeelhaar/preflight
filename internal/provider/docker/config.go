// Package docker provides the Docker provider for container runtime management.
package docker

import (
	"fmt"
)

// Config represents the docker section of the configuration.
type Config struct {
	// Install enables Docker Desktop installation
	Install bool
	// Compose enables Docker Compose
	Compose bool
	// Kubernetes enables Kubernetes in Docker Desktop
	Kubernetes bool
	// BuildKit enables BuildKit for improved builds
	BuildKit bool
	// ResourceLimits configures container resource limits
	ResourceLimits *ResourceLimits
	// Registries configures additional container registries
	Registries []Registry
	// Contexts configures Docker contexts for multi-host management
	Contexts []Context
}

// ResourceLimits defines Docker Desktop resource allocation.
type ResourceLimits struct {
	CPUs   int    `yaml:"cpus"`
	Memory string `yaml:"memory"` // e.g., "4GB", "8GB"
	Swap   string `yaml:"swap"`   // e.g., "1GB", "2GB"
	Disk   string `yaml:"disk"`   // e.g., "60GB", "100GB"
}

// Registry represents a container registry configuration.
type Registry struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty"`
}

// Context represents a Docker context for multi-host management.
type Context struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Host        string `yaml:"host"` // e.g., "ssh://user@host", "tcp://host:2376"
	Default     bool   `yaml:"default,omitempty"`
}

// ParseConfig parses the docker configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Install:    true, // Default to installing Docker
		Compose:    true, // Docker Compose is included by default
		Kubernetes: false,
		BuildKit:   true, // BuildKit is recommended
		Registries: make([]Registry, 0),
		Contexts:   make([]Context, 0),
	}

	// Parse install
	if install, ok := raw["install"]; ok {
		if b, ok := install.(bool); ok {
			cfg.Install = b
		}
	}

	// Parse compose
	if compose, ok := raw["compose"]; ok {
		if b, ok := compose.(bool); ok {
			cfg.Compose = b
		}
	}

	// Parse kubernetes
	if kubernetes, ok := raw["kubernetes"]; ok {
		if b, ok := kubernetes.(bool); ok {
			cfg.Kubernetes = b
		}
	}

	// Parse buildkit
	if buildkit, ok := raw["buildkit"]; ok {
		if b, ok := buildkit.(bool); ok {
			cfg.BuildKit = b
		}
	}

	// Parse resource_limits
	if limits, ok := raw["resource_limits"]; ok {
		limitsMap, ok := limits.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("resource_limits must be an object")
		}
		cfg.ResourceLimits = &ResourceLimits{}
		if cpus, ok := limitsMap["cpus"].(int); ok {
			cfg.ResourceLimits.CPUs = cpus
		}
		if memory, ok := limitsMap["memory"].(string); ok {
			cfg.ResourceLimits.Memory = memory
		}
		if swap, ok := limitsMap["swap"].(string); ok {
			cfg.ResourceLimits.Swap = swap
		}
		if disk, ok := limitsMap["disk"].(string); ok {
			cfg.ResourceLimits.Disk = disk
		}
	}

	// Parse registries
	if registries, ok := raw["registries"]; ok {
		registryList, ok := registries.([]interface{})
		if !ok {
			return nil, fmt.Errorf("registries must be a list")
		}
		for _, r := range registryList {
			registry, err := parseRegistry(r)
			if err != nil {
				return nil, err
			}
			cfg.Registries = append(cfg.Registries, registry)
		}
	}

	// Parse contexts
	if contexts, ok := raw["contexts"]; ok {
		contextList, ok := contexts.([]interface{})
		if !ok {
			return nil, fmt.Errorf("contexts must be a list")
		}
		for _, c := range contextList {
			context, err := parseContext(c)
			if err != nil {
				return nil, err
			}
			cfg.Contexts = append(cfg.Contexts, context)
		}
	}

	return cfg, nil
}

// parseRegistry parses a single registry configuration.
func parseRegistry(raw interface{}) (Registry, error) {
	switch v := raw.(type) {
	case string:
		return Registry{URL: v}, nil
	case map[string]interface{}:
		registry := Registry{}
		if url, ok := v["url"].(string); ok {
			registry.URL = url
		} else {
			return Registry{}, fmt.Errorf("registry must have a url")
		}
		if username, ok := v["username"].(string); ok {
			registry.Username = username
		}
		if insecure, ok := v["insecure"].(bool); ok {
			registry.Insecure = insecure
		}
		return registry, nil
	default:
		return Registry{}, fmt.Errorf("registry must be a string or object")
	}
}

// parseContext parses a single Docker context configuration.
func parseContext(raw interface{}) (Context, error) {
	switch v := raw.(type) {
	case map[string]interface{}:
		context := Context{}
		if name, ok := v["name"].(string); ok {
			context.Name = name
		} else {
			return Context{}, fmt.Errorf("context must have a name")
		}
		if description, ok := v["description"].(string); ok {
			context.Description = description
		}
		if host, ok := v["host"].(string); ok {
			context.Host = host
		} else {
			return Context{}, fmt.Errorf("context must have a host")
		}
		if isDefault, ok := v["default"].(bool); ok {
			context.Default = isDefault
		}
		return context, nil
	default:
		return Context{}, fmt.Errorf("context must be an object")
	}
}
