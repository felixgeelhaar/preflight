package starship

// Config represents the starship section of the configuration.
type Config struct {
	Settings map[string]interface{}
	Preset   string
	Shell    string
}

// ParseConfig parses the starship configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Settings: make(map[string]interface{}),
	}

	// Parse settings (the actual starship.toml content)
	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		cfg.Settings = settings
	}

	// Parse preset
	if preset, ok := raw["preset"].(string); ok {
		cfg.Preset = preset
	}

	// Parse shell for integration
	if shell, ok := raw["shell"].(string); ok {
		cfg.Shell = shell
	}

	return cfg, nil
}
