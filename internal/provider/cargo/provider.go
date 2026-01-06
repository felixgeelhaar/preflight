package cargo

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for Cargo.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new cargo provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "cargo"
}

// Compile transforms cargo configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("cargo")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0, len(cfg.Crates))
	deps := tooldeps.ResolveToolDeps(ctx, nil, tooldeps.ToolRust)

	for _, crate := range cfg.Crates {
		version, err := versionutil.ResolvePackageVersion(ctx, "cargo", crate.Name, crate.Version)
		if err != nil {
			return nil, err
		}
		crate.Version = version
		steps = append(steps, NewCrateStep(crate, p.runner, deps))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
