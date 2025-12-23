// Package runtime provides the runtime version manager provider for Preflight.
// It handles tool version management via rtx/asdf.
package runtime

import "fmt"

// Config represents runtime tool version configuration.
type Config struct {
	Backend string
	Scope   string
	Tools   []ToolConfig
	Plugins []PluginConfig
}

// ToolConfig represents a tool with its desired version.
type ToolConfig struct {
	Name    string
	Version string
}

// PluginConfig represents a custom plugin source for asdf.
type PluginConfig struct {
	Name string
	URL  string
}

// ToolVersionsPath returns the path to the .tool-versions file.
func (c *Config) ToolVersionsPath() string {
	if c.Scope == "project" {
		return ".tool-versions"
	}
	return "~/.tool-versions"
}

// ParseConfig parses runtime configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{}

	// Parse backend (rtx or asdf)
	if backend, ok := raw["backend"].(string); ok {
		cfg.Backend = backend
	}

	// Parse scope (global or project)
	if scope, ok := raw["scope"].(string); ok {
		cfg.Scope = scope
	}

	// Parse tools
	if tools, ok := raw["tools"].([]interface{}); ok {
		for i, t := range tools {
			toolMap, ok := t.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid tool entry at index %d", i)
			}

			tool := ToolConfig{}
			if v, ok := toolMap["name"].(string); ok {
				tool.Name = v
			}
			if v, ok := toolMap["version"].(string); ok {
				tool.Version = v
			}

			cfg.Tools = append(cfg.Tools, tool)
		}
	}

	// Parse plugins
	if plugins, ok := raw["plugins"].([]interface{}); ok {
		for i, p := range plugins {
			pluginMap, ok := p.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid plugin entry at index %d", i)
			}

			plugin := PluginConfig{}
			if v, ok := pluginMap["name"].(string); ok {
				plugin.Name = v
			}
			if v, ok := pluginMap["url"].(string); ok {
				plugin.URL = v
			}

			cfg.Plugins = append(cfg.Plugins, plugin)
		}
	}

	return cfg, nil
}
