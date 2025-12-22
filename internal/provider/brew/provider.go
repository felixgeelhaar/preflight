package brew

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Homebrew.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Homebrew provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "brew"
}

// Compile transforms brew configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("brew")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add tap steps first (they have no dependencies on other brew steps)
	for _, tap := range cfg.Taps {
		steps = append(steps, NewTapStep(tap, p.runner))
	}

	// Add formula steps
	for _, formula := range cfg.Formulae {
		steps = append(steps, NewFormulaStep(formula, p.runner))
	}

	// Add cask steps
	for _, cask := range cfg.Casks {
		steps = append(steps, NewCaskStep(cask, p.runner))
	}

	return steps, nil
}
