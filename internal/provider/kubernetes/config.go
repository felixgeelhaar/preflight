package kubernetes

import (
	"fmt"
)

// Config represents the kubernetes section of the configuration.
type Config struct {
	Plugins          []string
	Contexts         []Context
	DefaultNamespace string
}

// Context represents a Kubernetes context configuration.
type Context struct {
	Name      string
	Cluster   string
	User      string
	Namespace string
}

// ParseConfig parses the kubernetes configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Plugins:  make([]string, 0),
		Contexts: make([]Context, 0),
	}

	// Parse plugins (krew plugins)
	if plugins, ok := raw["plugins"]; ok {
		pluginList, ok := plugins.([]interface{})
		if !ok {
			return nil, fmt.Errorf("plugins must be a list")
		}
		for _, p := range pluginList {
			pluginStr, ok := p.(string)
			if !ok {
				return nil, fmt.Errorf("plugin must be a string")
			}
			cfg.Plugins = append(cfg.Plugins, pluginStr)
		}
	}

	// Parse contexts
	if contexts, ok := raw["contexts"].([]interface{}); ok {
		for _, c := range contexts {
			context, err := parseContext(c)
			if err != nil {
				return nil, err
			}
			cfg.Contexts = append(cfg.Contexts, context)
		}
	}

	// Parse default namespace
	if ns, ok := raw["default_namespace"].(string); ok {
		cfg.DefaultNamespace = ns
	}

	return cfg, nil
}

func parseContext(raw interface{}) (Context, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Context{}, fmt.Errorf("context must be an object")
	}

	ctx := Context{}

	if name, ok := m["name"].(string); ok {
		ctx.Name = name
	} else {
		return Context{}, fmt.Errorf("context must have a name")
	}

	if cluster, ok := m["cluster"].(string); ok {
		ctx.Cluster = cluster
	}

	if user, ok := m["user"].(string); ok {
		ctx.User = user
	}

	if namespace, ok := m["namespace"].(string); ok {
		ctx.Namespace = namespace
	}

	return ctx, nil
}
