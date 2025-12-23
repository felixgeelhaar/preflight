package apt

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider compiles apt configuration into executable steps.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new apt Provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "apt"
}

// Compile transforms apt configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("apt")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0, len(cfg.PPAs)+len(cfg.Packages))

	// Add PPA steps first
	for _, ppa := range cfg.PPAs {
		steps = append(steps, NewPPAStep(ppa, p.runner))
	}

	// Add package steps
	for _, pkg := range cfg.Packages {
		steps = append(steps, NewPackageStep(pkg, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
