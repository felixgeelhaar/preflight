package gotools

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	tooldeps "github.com/felixgeelhaar/preflight/internal/domain/deps"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/provider/versionutil"
)

// Provider implements the compiler.Provider interface for Go tools.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Go tools provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "go"
}

// Compile transforms go configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("go")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0, len(cfg.Tools))
	deps := tooldeps.ResolveToolDeps(ctx, nil, tooldeps.ToolGo)

	for _, tool := range cfg.Tools {
		version, err := versionutil.ResolvePackageVersion(ctx, "go", tool.Module, tool.Version)
		if err != nil {
			return nil, err
		}
		tool.Version = version
		steps = append(steps, NewToolStep(tool, p.runner, deps))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
