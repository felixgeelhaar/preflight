package winget

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for Windows Package Manager.
type Provider struct {
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewProvider creates a new winget provider.
func NewProvider(runner ports.CommandRunner, plat *platform.Platform) *Provider {
	return &Provider{
		runner:   runner,
		platform: plat,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "winget"
}

// Compile transforms winget configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	// Skip if not on Windows or WSL
	if p.platform != nil && !p.platform.IsWindows() && !p.platform.IsWSL() {
		return nil, nil
	}

	rawConfig := ctx.GetSection("winget")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if len(cfg.Packages) == 0 {
		return nil, nil
	}

	steps := make([]compiler.Step, 0, len(cfg.Packages)+1)
	steps = append(steps, NewReadyStep(p.platform))

	// Add package steps
	for _, pkg := range cfg.Packages {
		version, err := versionutil.ResolvePackageVersion(ctx, "winget", pkg.ID, pkg.Version)
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
