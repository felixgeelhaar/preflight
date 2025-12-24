// Package vscode provides the VSCode editor provider for Preflight.
// It handles extension installation, settings management, and keybindings configuration.
// Includes Remote-WSL support for Windows/WSL environments.
package vscode

// Keybinding represents a single keybinding configuration.
type Keybinding struct {
	Key     string `yaml:"key"`
	Command string `yaml:"command"`
	When    string `yaml:"when,omitempty"`
	Args    string `yaml:"args,omitempty"`
}

// WSLConfig contains configuration specific to Remote-WSL.
type WSLConfig struct {
	// Extensions to install in the WSL remote context
	Extensions []string
	// Settings specific to WSL remote sessions
	Settings map[string]interface{}
	// AutoInstall enables automatic Remote-WSL extension installation on Windows
	AutoInstall bool
	// Distro specifies which WSL distro to target (empty means default)
	Distro string
}

// Config represents VSCode configuration.
type Config struct {
	Extensions  []string
	Settings    map[string]interface{}
	Keybindings []Keybinding
	WSL         *WSLConfig
}

// ParseConfig parses raw config into VSCode Config.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Settings: make(map[string]interface{}),
	}

	if raw == nil {
		return cfg, nil
	}

	// Parse extensions
	if exts, ok := raw["extensions"].([]interface{}); ok {
		for _, ext := range exts {
			if extStr, ok := ext.(string); ok {
				cfg.Extensions = append(cfg.Extensions, extStr)
			}
		}
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		cfg.Settings = settings
	}

	// Parse keybindings
	if keybindings, ok := raw["keybindings"].([]interface{}); ok {
		for _, kb := range keybindings {
			if kbMap, ok := kb.(map[string]interface{}); ok {
				keybinding := Keybinding{}
				if key, ok := kbMap["key"].(string); ok {
					keybinding.Key = key
				}
				if cmd, ok := kbMap["command"].(string); ok {
					keybinding.Command = cmd
				}
				if when, ok := kbMap["when"].(string); ok {
					keybinding.When = when
				}
				if args, ok := kbMap["args"].(string); ok {
					keybinding.Args = args
				}
				cfg.Keybindings = append(cfg.Keybindings, keybinding)
			}
		}
	}

	// Parse WSL configuration
	if wsl, ok := raw["wsl"].(map[string]interface{}); ok {
		cfg.WSL = &WSLConfig{
			Settings: make(map[string]interface{}),
		}

		// Parse WSL extensions
		if exts, ok := wsl["extensions"].([]interface{}); ok {
			for _, ext := range exts {
				if extStr, ok := ext.(string); ok {
					cfg.WSL.Extensions = append(cfg.WSL.Extensions, extStr)
				}
			}
		}

		// Parse WSL settings
		if settings, ok := wsl["settings"].(map[string]interface{}); ok {
			cfg.WSL.Settings = settings
		}

		// Parse auto_install
		if autoInstall, ok := wsl["auto_install"].(bool); ok {
			cfg.WSL.AutoInstall = autoInstall
		}

		// Parse distro
		if distro, ok := wsl["distro"].(string); ok {
			cfg.WSL.Distro = distro
		}
	}

	return cfg, nil
}
