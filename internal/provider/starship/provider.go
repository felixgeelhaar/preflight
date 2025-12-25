// Package starship provides the Starship prompt provider for configuration management.
package starship

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Starship prompt.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Starship provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "starship"
}

// Compile transforms starship configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("starship")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add installation step
	steps = append(steps, NewInstallStep(p.runner))

	// Add config step if settings are defined
	if len(cfg.Settings) > 0 || cfg.Preset != "" {
		installStep := compiler.MustNewStepID("starship:install")
		steps = append(steps, NewConfigStep(cfg.Settings, cfg.Preset, installStep, p.runner))
	}

	// Add shell integration step
	if cfg.Shell != "" {
		installStep := compiler.MustNewStepID("starship:install")
		steps = append(steps, NewShellIntegrationStep(cfg.Shell, installStep, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
