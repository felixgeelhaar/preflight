package runtime

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for runtime version management.
type Provider struct {
	fs ports.FileSystem
}

// NewProvider creates a new runtime provider.
func NewProvider(fs ports.FileSystem) *Provider {
	return &Provider{fs: fs}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "runtime"
}

// Compile transforms runtime configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("runtime")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Only create steps if there's actual config
	if len(cfg.Tools) == 0 && len(cfg.Plugins) == 0 {
		return nil, nil
	}

	steps := make([]compiler.Step, 0, len(cfg.Plugins)+1)

	// Add plugin steps first (plugins must be installed before tools)
	for _, plugin := range cfg.Plugins {
		steps = append(steps, NewPluginStep(plugin))
	}

	// Add tool-versions step if there are tools
	if len(cfg.Tools) > 0 {
		steps = append(steps, NewToolVersionStep(cfg, p.fs))
	}

	return steps, nil
}
