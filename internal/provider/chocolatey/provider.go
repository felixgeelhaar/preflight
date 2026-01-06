package chocolatey

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for Chocolatey.
type Provider struct {
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewProvider creates a new chocolatey provider.
func NewProvider(runner ports.CommandRunner, plat *platform.Platform) *Provider {
	return &Provider{
		runner:   runner,
		platform: plat,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "chocolatey"
}

// Compile transforms chocolatey configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	// Skip if not on Windows or WSL
	if p.platform != nil && !p.platform.IsWindows() && !p.platform.IsWSL() {
		return nil, nil
	}

	rawConfig := ctx.GetSection("chocolatey")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if len(cfg.Sources) == 0 && len(cfg.Packages) == 0 {
		return nil, nil
	}

	steps := make([]compiler.Step, 0, len(cfg.Sources)+len(cfg.Packages)+1)
	steps = append(steps, NewInstallStep(p.runner, p.platform))

	// Add source steps first (packages may depend on custom sources)
	sources := make(map[string]struct{}, len(cfg.Sources))
	for _, source := range cfg.Sources {
		sources[source.Name] = struct{}{}
		steps = append(steps, NewSourceStep(source, p.runner, p.platform))
	}

	// Add package steps
	for _, pkg := range cfg.Packages {
		version, err := versionutil.ResolvePackageVersion(ctx, "chocolatey", pkg.Name, pkg.Version)
		if err != nil {
			return nil, err
		}
		pkg.Version = version
		deps := []compiler.StepID{compiler.MustNewStepID(chocoInstallStepID)}
		if pkg.Source != "" {
			if _, ok := sources[pkg.Source]; ok {
				deps = append(deps, compiler.MustNewStepID("chocolatey:source:"+pkg.Source))
			}
		}
		steps = append(steps, NewPackageStep(pkg, p.runner, p.platform, deps))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
