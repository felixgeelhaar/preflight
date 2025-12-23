// Package nvim provides the Neovim configuration provider for Preflight.
package nvim

import (
	"gopkg.in/yaml.v3"
)

// Config represents the nvim configuration section.
type Config struct {
	Preset        string `yaml:"preset,omitempty"`
	PluginManager string `yaml:"plugin_manager,omitempty"`
	ConfigRepo    string `yaml:"config_repo,omitempty"`
	EnsureInstall bool   `yaml:"ensure_install,omitempty"`
}

// ConfigPath returns the path to the Neovim configuration directory.
func (c *Config) ConfigPath() string {
	return "~/.config/nvim"
}

// LazyLockPath returns the path to lazy-lock.json.
func (c *Config) LazyLockPath() string {
	return "~/.config/nvim/lazy-lock.json"
}

// ParseConfig parses the nvim configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	// Marshal the map back to YAML, then unmarshal to our struct
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
