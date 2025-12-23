package vscode

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider compiles VSCode configuration into executable steps.
type Provider struct {
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewProvider creates a new VSCode Provider.
func NewProvider(fs ports.FileSystem, runner ports.CommandRunner) *Provider {
	return &Provider{
		fs:     fs,
		runner: runner,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "vscode"
}

// Compile transforms VSCode configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("vscode")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	// Pre-allocate steps slice
	stepCount := len(cfg.Extensions)
	if len(cfg.Settings) > 0 {
		stepCount++
	}
	if len(cfg.Keybindings) > 0 {
		stepCount++
	}
	steps := make([]compiler.Step, 0, stepCount)

	// Add extension steps
	for _, ext := range cfg.Extensions {
		steps = append(steps, NewExtensionStep(ext, p.runner))
	}

	// Add settings step if settings are defined
	if len(cfg.Settings) > 0 {
		steps = append(steps, NewSettingsStep(cfg.Settings, p.fs))
	}

	// Add keybindings step if keybindings are defined
	if len(cfg.Keybindings) > 0 {
		steps = append(steps, NewKeybindingsStep(cfg.Keybindings, p.fs))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
