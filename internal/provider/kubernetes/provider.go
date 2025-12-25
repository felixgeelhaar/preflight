// Package kubernetes provides the Kubernetes provider for kubectl plugins and contexts.
package kubernetes

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Kubernetes.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Kubernetes provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "kubernetes"
}

// Compile transforms kubernetes configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("kubernetes")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add plugin steps (krew plugins)
	for _, plugin := range cfg.Plugins {
		steps = append(steps, NewPluginStep(plugin, p.runner))
	}

	// Add context steps
	for _, context := range cfg.Contexts {
		steps = append(steps, NewContextStep(context, p.runner))
	}

	// Add namespace step if default namespace is specified
	if cfg.DefaultNamespace != "" {
		steps = append(steps, NewNamespaceStep(cfg.DefaultNamespace, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
