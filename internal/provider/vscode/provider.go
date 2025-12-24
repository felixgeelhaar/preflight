package vscode

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider compiles VSCode configuration into executable steps.
type Provider struct {
	fs       ports.FileSystem
	runner   ports.CommandRunner
	platform *platform.Platform
}

// NewProvider creates a new VSCode Provider.
func NewProvider(fs ports.FileSystem, runner ports.CommandRunner, plat *platform.Platform) *Provider {
	return &Provider{
		fs:       fs,
		runner:   runner,
		platform: plat,
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
	// Add WSL steps if applicable
	if cfg.WSL != nil {
		stepCount++ // Setup step
		stepCount += len(cfg.WSL.Extensions)
		if len(cfg.WSL.Settings) > 0 {
			stepCount++
		}
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

	// Add WSL steps if WSL configuration is present and on appropriate platform
	if cfg.WSL != nil && p.shouldApplyWSL() {
		steps = append(steps, p.compileWSLSteps(cfg.WSL)...)
	}

	return steps, nil
}

// shouldApplyWSL returns true if WSL configuration should be applied.
// This is true on Windows (native) or when running in WSL.
func (p *Provider) shouldApplyWSL() bool {
	if p.platform == nil {
		return false
	}
	return p.platform.IsWindows() || p.platform.IsWSL()
}

// compileWSLSteps generates steps for WSL-specific configuration.
func (p *Provider) compileWSLSteps(cfg *WSLConfig) []compiler.Step {
	steps := make([]compiler.Step, 0)

	// Always add the setup step first (installs Remote-WSL extension)
	if cfg.AutoInstall || len(cfg.Extensions) > 0 || len(cfg.Settings) > 0 {
		steps = append(steps, NewRemoteWSLSetupStep(p.runner, p.platform))
	}

	// Add WSL extension steps
	for _, ext := range cfg.Extensions {
		steps = append(steps, NewRemoteWSLExtensionStep(ext, cfg.Distro, p.runner, p.platform))
	}

	// Add WSL settings step if settings are defined
	if len(cfg.Settings) > 0 {
		steps = append(steps, NewRemoteWSLSettingsStep(cfg.Settings, cfg.Distro, p.fs, p.platform))
	}

	return steps
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
