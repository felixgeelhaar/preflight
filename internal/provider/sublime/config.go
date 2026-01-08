package sublime

// Config represents the Sublime Text configuration section in preflight.yaml.
type Config struct {
	// Packages is a list of Package Control packages to install
	Packages []string `yaml:"packages,omitempty"`
	// Settings are preferences to merge into Preferences.sublime-settings
	Settings map[string]interface{} `yaml:"settings,omitempty"`
	// Keybindings are custom keybindings to set
	Keybindings []Keybinding `yaml:"keybindings,omitempty"`
	// Theme is the color scheme to use
	Theme string `yaml:"theme,omitempty"`
	// ColorScheme is the color scheme file
	ColorScheme string `yaml:"color_scheme,omitempty"`
}

// Keybinding represents a Sublime Text keybinding.
type Keybinding struct {
	Keys    []string `yaml:"keys" json:"keys"`
	Command string   `yaml:"command" json:"command"`
	Args    map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
	Context []KeyContext `yaml:"context,omitempty" json:"context,omitempty"`
}

// KeyContext represents a keybinding context condition.
type KeyContext struct {
	Key      string      `yaml:"key" json:"key"`
	Operator string      `yaml:"operator,omitempty" json:"operator,omitempty"`
	Operand  interface{} `yaml:"operand,omitempty" json:"operand,omitempty"`
}

// ParseConfig parses the sublime section from the manifest.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	config := &Config{}

	// Parse packages
	if pkgs, ok := raw["packages"]; ok {
		pkgList, ok := pkgs.([]interface{})
		if ok {
			for _, p := range pkgList {
				if s, ok := p.(string); ok {
					config.Packages = append(config.Packages, s)
				}
			}
		}
	}

	// Parse settings
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		config.Settings = settings
	}

	// Parse theme
	if theme, ok := raw["theme"].(string); ok {
		config.Theme = theme
	}

	// Parse color_scheme
	if colorScheme, ok := raw["color_scheme"].(string); ok {
		config.ColorScheme = colorScheme
	}

	// Parse keybindings
	if kb, ok := raw["keybindings"]; ok {
		kbList, ok := kb.([]interface{})
		if ok {
			for _, k := range kbList {
				kbMap, ok := k.(map[string]interface{})
				if !ok {
					continue
				}

				keybinding := Keybinding{}

				// Parse keys array
				if keys, ok := kbMap["keys"].([]interface{}); ok {
					for _, key := range keys {
						if s, ok := key.(string); ok {
							keybinding.Keys = append(keybinding.Keys, s)
						}
					}
				}

				// Parse command
				if cmd, ok := kbMap["command"].(string); ok {
					keybinding.Command = cmd
				}

				// Parse args
				if args, ok := kbMap["args"].(map[string]interface{}); ok {
					keybinding.Args = args
				}

				// Parse context
				if ctx, ok := kbMap["context"].([]interface{}); ok {
					for _, c := range ctx {
						if ctxMap, ok := c.(map[string]interface{}); ok {
							keyCtx := KeyContext{}
							if key, ok := ctxMap["key"].(string); ok {
								keyCtx.Key = key
							}
							if op, ok := ctxMap["operator"].(string); ok {
								keyCtx.Operator = op
							}
							keyCtx.Operand = ctxMap["operand"]
							keybinding.Context = append(keybinding.Context, keyCtx)
						}
					}
				}

				config.Keybindings = append(config.Keybindings, keybinding)
			}
		}
	}

	return config, nil
}

// HasPackages returns true if packages are specified.
func (c *Config) HasPackages() bool {
	return len(c.Packages) > 0
}

// HasSettings returns true if settings are specified.
func (c *Config) HasSettings() bool {
	return len(c.Settings) > 0 || c.Theme != "" || c.ColorScheme != ""
}

// HasKeybindings returns true if keybindings are specified.
func (c *Config) HasKeybindings() bool {
	return len(c.Keybindings) > 0
}
