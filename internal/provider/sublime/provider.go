// Package sublime provides configuration management for Sublime Text editor.
package sublime

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Sublime Text.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Sublime Text provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "sublime"
}

// Compile generates steps from the configuration.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("sublime")
	if rawConfig == nil {
		return nil, nil
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add packages step if specified
	if config.HasPackages() {
		steps = append(steps, NewPackagesStep(config.Packages, p.runner))
	}

	// Add settings step if specified
	if config.HasSettings() {
		steps = append(steps, NewSettingsStep(config.Settings, config.Theme, config.ColorScheme, p.runner))
	}

	// Add keybindings step if specified
	if config.HasKeybindings() {
		steps = append(steps, NewKeybindingsStep(config.Keybindings, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
