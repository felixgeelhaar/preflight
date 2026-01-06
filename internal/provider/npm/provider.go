package npm

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for npm.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new npm provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "npm"
}

// Compile transforms npm configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("npm")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0, len(cfg.Packages))
	deps := tooldeps.ResolveToolDeps(ctx, nil, tooldeps.ToolNode)

	for _, pkg := range cfg.Packages {
		version, err := versionutil.ResolvePackageVersion(ctx, "npm", pkg.Name, pkg.Version)
		if err != nil {
			return nil, err
		}
		pkg.Version = version
		steps = append(steps, NewPackageStep(pkg, p.runner, deps))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
