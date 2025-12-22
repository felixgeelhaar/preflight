package git

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for git configuration.
type Provider struct {
	fs ports.FileSystem
}

// NewProvider creates a new git provider.
func NewProvider(fs ports.FileSystem) *Provider {
	return &Provider{fs: fs}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "git"
}

// Compile transforms git configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("git")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Only create step if there's actual config to write
	if cfg.User.Name == "" && cfg.User.Email == "" && len(cfg.Aliases) == 0 &&
		len(cfg.Includes) == 0 && cfg.Core.Editor == "" && !cfg.Commit.GPGSign {
		return nil, nil
	}

	steps := make([]compiler.Step, 0)
	steps = append(steps, NewConfigStep(cfg, p.fs))

	return steps, nil
}
