// Package shell provides the shell configuration provider for Preflight.
package shell

import (
	"gopkg.in/yaml.v3"
)

// Config represents the shell configuration section.
type Config struct {
	Default  string            `yaml:"default,omitempty"`
	Shells   []Entry           `yaml:"shells,omitempty"`
	Starship StarshipConfig    `yaml:"starship,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Aliases  map[string]string `yaml:"aliases,omitempty"`
}

// Entry represents configuration for a single shell.
type Entry struct {
	Name          string         `yaml:"name"`
	Framework     string         `yaml:"framework,omitempty"`
	Theme         string         `yaml:"theme,omitempty"`
	Plugins       []string       `yaml:"plugins,omitempty"`
	CustomPlugins []CustomPlugin `yaml:"custom_plugins,omitempty"`
}

// CustomPlugin represents a custom plugin to install from a git repository.
type CustomPlugin struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
}

// StarshipConfig represents starship prompt configuration.
type StarshipConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Preset  string `yaml:"preset,omitempty"`
}

// ConfigPath returns the path to the shell's configuration file.
func (s *Entry) ConfigPath() string {
	switch s.Name {
	case "zsh":
		return "~/.zshrc"
	case "bash":
		return "~/.bashrc"
	case "fish":
		return "~/.config/fish/config.fish"
	default:
		return ""
	}
}

// ParseConfig parses a raw map into a shell Config.
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

	// Initialize empty maps if nil
	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	if cfg.Aliases == nil {
		cfg.Aliases = make(map[string]string)
	}

	return &cfg, nil
}
