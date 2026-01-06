package scoop

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for Scoop.
type Provider struct {
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewProvider creates a new scoop provider.
func NewProvider(runner ports.CommandRunner, plat *platform.Platform) *Provider {
	return &Provider{
		runner:   runner,
		platform: plat,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "scoop"
}

// Compile transforms scoop configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	// Skip if not on Windows or WSL
	if p.platform != nil && !p.platform.IsWindows() && !p.platform.IsWSL() {
		return nil, nil
	}

	rawConfig := ctx.GetSection("scoop")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if len(cfg.Buckets) == 0 && len(cfg.Packages) == 0 {
		return nil, nil
	}

	steps := make([]compiler.Step, 0, len(cfg.Buckets)+len(cfg.Packages)+1)
	steps = append(steps, NewInstallStep(p.runner, p.platform))

	// Add bucket steps first (they have no dependencies on other scoop steps)
	for _, bucket := range cfg.Buckets {
		steps = append(steps, NewBucketStep(bucket, p.runner, p.platform))
	}

	// Add package steps
	for _, pkg := range cfg.Packages {
		version, err := versionutil.ResolvePackageVersion(ctx, "scoop", pkg.FullName(), pkg.Version)
		if err != nil {
			return nil, err
		}
		pkg.Version = version
		steps = append(steps, NewPackageStep(pkg, p.runner, p.platform))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
