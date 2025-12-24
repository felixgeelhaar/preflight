package docker

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Docker.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Docker provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "docker"
}

// Compile transforms docker configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("docker")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add Docker Desktop installation step
	if cfg.Install {
		steps = append(steps, NewInstallStep(p.runner))
	}

	// Add BuildKit configuration step
	if cfg.BuildKit {
		steps = append(steps, NewBuildKitStep(cfg.Install, p.runner))
	}

	// Add Kubernetes step if enabled
	if cfg.Kubernetes {
		steps = append(steps, NewKubernetesStep(cfg.Install, p.runner))
	}

	// Add context steps
	for _, context := range cfg.Contexts {
		steps = append(steps, NewContextStep(context, cfg.Install, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
