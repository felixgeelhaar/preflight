// Package tmux provides the tmux provider for configuration and plugin management.
package tmux

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for tmux.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new tmux provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "tmux"
}

// Compile transforms tmux configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("tmux")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add TPM installation step if plugins are defined
	if len(cfg.Plugins) > 0 {
		steps = append(steps, NewTPMStep(p.runner))
	}

	// Add plugin steps
	for _, plugin := range cfg.Plugins {
		tpmStep := compiler.MustNewStepID("tmux:tpm")
		steps = append(steps, NewPluginStep(plugin, tpmStep, p.runner))
	}

	// Add config step if settings are defined
	if len(cfg.Settings) > 0 || cfg.ConfigFile != "" {
		steps = append(steps, NewConfigStep(cfg.Settings, cfg.ConfigFile, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
