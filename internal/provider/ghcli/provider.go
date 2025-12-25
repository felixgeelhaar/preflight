// Package ghcli provides the GitHub CLI provider for extensions and aliases.
package ghcli

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for GitHub CLI.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new GitHub CLI provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "github-cli"
}

// Compile transforms github-cli configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("github-cli")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add extension steps
	for _, ext := range cfg.Extensions {
		steps = append(steps, NewExtensionStep(ext, p.runner))
	}

	// Add alias steps
	for name, command := range cfg.Aliases {
		steps = append(steps, NewAliasStep(name, command, p.runner))
	}

	// Add config steps
	for key, value := range cfg.Config {
		steps = append(steps, NewConfigStep(key, value, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
