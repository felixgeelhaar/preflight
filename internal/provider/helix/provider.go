// Package helix provides configuration management for Helix editor.
package helix

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Helix.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Helix provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "helix"
}

// Compile generates steps from the configuration.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("helix")
	if rawConfig == nil {
		return nil, nil
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add config step if source or settings specified
	if config.HasConfigSource() || config.HasSettings() {
		steps = append(steps, NewConfigStep(
			config.Source,
			config.Link,
			config.Settings,
			config.EditorSettings,
			config.KeysSettings,
			p.runner,
		))
	}

	// Add languages step if specified
	if config.HasLanguages() {
		steps = append(steps, NewLanguagesStep(config.Languages, config.Link, p.runner))
	}

	// Add theme step if specified
	if config.HasTheme() {
		steps = append(steps, NewThemeStep(config.Theme, config.ThemeSource, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
