package apt

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
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

	steps := make([]compiler.Step, 0, len(cfg.PPAs)+len(cfg.Packages)+2)
	if len(cfg.PPAs) > 0 || len(cfg.Packages) > 0 {
		steps = append(steps, NewReadyStep(p.runner))
	}

	// Add PPA steps first
	for _, ppa := range cfg.PPAs {
		steps = append(steps, NewPPAStep(ppa, p.runner))
	}

	if len(cfg.PPAs) > 0 || len(cfg.Packages) > 0 {
		deps := make([]compiler.StepID, 0, len(cfg.PPAs)+1)
		deps = append(deps, compiler.MustNewStepID(aptReadyStepID))
		for _, ppa := range cfg.PPAs {
			deps = append(deps, compiler.MustNewStepID(ppaStepID(ppa)))
		}
		steps = append(steps, NewUpdateStep(p.runner, deps))
	}

	// Add package steps
	for _, pkg := range cfg.Packages {
		version, err := versionutil.ResolvePackageVersion(ctx, "apt", pkg.Name, pkg.Version)
		if err != nil {
			return nil, err
		}
		pkg.Version = version
		steps = append(steps, NewPackageStep(pkg, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
