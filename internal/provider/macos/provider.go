// Package macos provides the macOS defaults provider for system preferences.
package macos

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for macOS defaults.
type Provider struct {
	runner ports.CommandRunner
}

// NewProvider creates a new macOS defaults provider.
func NewProvider(runner ports.CommandRunner) *Provider {
	return &Provider{runner: runner}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "macos"
}

// Compile transforms macos configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("macos")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	steps := make([]compiler.Step, 0)

	// Add defaults steps
	for _, def := range cfg.Defaults {
		steps = append(steps, NewDefaultsStep(def, p.runner))
	}

	// Add dock steps
	for _, item := range cfg.Dock.Add {
		steps = append(steps, NewDockStep(item, true, p.runner))
	}
	for _, item := range cfg.Dock.Remove {
		steps = append(steps, NewDockStep(item, false, p.runner))
	}

	// Add Finder settings
	if cfg.Finder.ShowHidden != nil {
		steps = append(steps, NewFinderStep("AppleShowAllFiles", *cfg.Finder.ShowHidden, p.runner))
	}
	if cfg.Finder.ShowExtensions != nil {
		steps = append(steps, NewFinderStep("AppleShowAllExtensions", *cfg.Finder.ShowExtensions, p.runner))
	}
	if cfg.Finder.ShowPathBar != nil {
		steps = append(steps, NewFinderStep("ShowPathbar", *cfg.Finder.ShowPathBar, p.runner))
	}

	// Add keyboard settings
	if cfg.Keyboard.KeyRepeat != nil {
		steps = append(steps, NewKeyboardStep("KeyRepeat", *cfg.Keyboard.KeyRepeat, p.runner))
	}
	if cfg.Keyboard.InitialKeyRepeat != nil {
		steps = append(steps, NewKeyboardStep("InitialKeyRepeat", *cfg.Keyboard.InitialKeyRepeat, p.runner))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
