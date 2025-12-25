// Package cursor provides the Cursor editor provider for extensions and settings.
package cursor

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Cursor editor.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Cursor provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "cursor"
}

// Compile transforms cursor configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("cursor")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add extension steps
	for _, ext := range cfg.Extensions {
		steps = append(steps, NewExtensionStep(ext, p.runner))
	}

	// Add settings step if settings are defined
	if len(cfg.Settings) > 0 {
		steps = append(steps, NewSettingsStep(cfg.Settings, p.runner))
	}

	// Add keybindings step if keybindings are defined
	if len(cfg.Keybindings) > 0 {
		steps = append(steps, NewKeybindingsStep(cfg.Keybindings, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
