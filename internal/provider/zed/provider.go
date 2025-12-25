// Package zed provides the Zed editor provider for configuration management.
package zed

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for Zed editor.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new Zed provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "zed"
}

// Compile transforms zed configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("zed")
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

	// Add settings step if settings are defined
	if len(cfg.Settings) > 0 {
		steps = append(steps, NewSettingsStep(cfg.Settings, p.runner))
	}

	// Add keymap step if keymap is defined
	if len(cfg.Keymap) > 0 {
		steps = append(steps, NewKeymapStep(cfg.Keymap, p.runner))
	}

	// Add theme step if theme is specified
	if cfg.Theme != "" {
		steps = append(steps, NewThemeStep(cfg.Theme, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
