// Package gotools provides the Go tools provider for installing Go CLI tools.
package gotools

import (
	"fmt"
	"path"
	"strings"
)

// Config represents the go section of the configuration.
type Config struct {
	Tools []Tool
}

// Tool represents a Go tool to install.
type Tool struct {
	Module  string // Full module path (e.g., "golang.org/x/tools/gopls")
	Version string // Optional: specific version (e.g., "latest", "v0.14.0")
}

// FullName returns the module path with version for go install.
func (t Tool) FullName() string {
	version := t.Version
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("%s@%s", t.Module, version)
}

// BinaryName returns the expected binary name from the module path.
func (t Tool) BinaryName() string {
	return path.Base(t.Module)
}

// ParseConfig parses the go configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Tools: make([]Tool, 0),
	}

	// Parse tools
	if tools, ok := raw["tools"]; ok {
		toolList, ok := tools.([]interface{})
		if !ok {
			return nil, fmt.Errorf("tools must be a list")
		}
		for _, t := range toolList {
			tool, err := parseTool(t)
			if err != nil {
				return nil, err
			}
			cfg.Tools = append(cfg.Tools, tool)
		}
	}

	return cfg, nil
}

// parseTool parses a single tool from either a string or a map.
func parseTool(raw interface{}) (Tool, error) {
	switch v := raw.(type) {
	case string:
		return parseToolString(v), nil
	case map[string]interface{}:
		tool := Tool{}
		if module, ok := v["module"].(string); ok {
			tool.Module = module
		} else if name, ok := v["name"].(string); ok {
			// Allow "name" as alias for "module"
			tool.Module = name
		} else {
			return Tool{}, fmt.Errorf("tool must have a module or name")
		}
		if version, ok := v["version"].(string); ok {
			tool.Version = version
		}
		return tool, nil
	default:
		return Tool{}, fmt.Errorf("tool must be a string or object")
	}
}

// parseToolString parses a tool string like "module/path@version".
func parseToolString(s string) Tool {
	if atIndex := strings.LastIndex(s, "@"); atIndex > 0 {
		return Tool{
			Module:  s[:atIndex],
			Version: s[atIndex+1:],
		}
	}
	return Tool{Module: s}
}
