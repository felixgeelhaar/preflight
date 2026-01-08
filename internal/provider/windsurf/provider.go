// Package windsurf provides configuration management for Windsurf editor.
package windsurf

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Windsurf.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Windsurf provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "windsurf"
}

// Compile generates steps from the configuration.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("windsurf")
	if rawConfig == nil {
		return nil, nil
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add extension steps
	for _, ext := range config.Extensions {
		steps = append(steps, NewExtensionStep(ext, p.runner))
	}

	// Add settings step
	if len(config.Settings) > 0 {
		steps = append(steps, NewSettingsStep(config.Settings, p.runner))
	}

	// Add keybindings step
	if len(config.Keybindings) > 0 {
		steps = append(steps, NewKeybindingsStep(config.Keybindings, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
