// Package jetbrains provides configuration management for JetBrains IDEs.
package jetbrains

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for JetBrains IDEs.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new JetBrains provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "jetbrains"
}

// Compile generates steps from the configuration.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("jetbrains")
	if rawConfig == nil {
		return nil, nil
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Process each configured IDE
	for _, ideConfig := range config.IDEs {
		if ideConfig.Disabled {
			continue
		}

		ide := IDE(ideConfig.Name)

		// Get all plugins for this IDE (shared + IDE-specific)
		plugins := config.GetAllPluginsForIDE(ide)
		if len(plugins) > 0 {
			steps = append(steps, NewPluginStep(ide, plugins, p.runner))
		}

		// Add settings step if any settings are specified
		if len(ideConfig.Settings) > 0 || ideConfig.Keymap != "" || ideConfig.CodeStyle != "" {
			steps = append(steps, NewSettingsStep(
				ide,
				ideConfig.Settings,
				ideConfig.Keymap,
				ideConfig.CodeStyle,
				p.runner,
			))
		}

		// Add settings sync step if configured
		if config.HasSettingsSync() {
			steps = append(steps, NewSettingsSyncStep(ide, config.SettingsSync, p.runner))
		}
	}

	// If only shared plugins are specified (no specific IDEs), apply to all installed IDEs
	if !config.HasIDEConfig() && config.HasSharedPlugins() {
		discovery := NewDiscovery()
		installedIDEs := discovery.GetInstalledIDEs()
		for _, ide := range installedIDEs {
			steps = append(steps, NewPluginStep(ide, config.SharedPlugins, p.runner))
		}
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
