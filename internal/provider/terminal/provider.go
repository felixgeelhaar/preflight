package terminal

import (
	"fmt"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider implements the compiler.Provider interface for terminal emulators.
type Provider struct {
	fs        ports.FileSystem
	runner    ports.CommandRunner
	discovery *Discovery
	goos      string
}

// NewProvider creates a new terminal provider.
func NewProvider(fs ports.FileSystem, runner ports.CommandRunner) *Provider {
	return &Provider{
		fs:        fs,
		runner:    runner,
		discovery: NewDiscovery(),
		goos:      runtime.GOOS,
	}
}

// NewProviderWithDiscovery creates a provider with a custom discovery (for testing).
func NewProviderWithDiscovery(fs ports.FileSystem, runner ports.CommandRunner, discovery *Discovery, goos string) *Provider {
	return &Provider{
		fs:        fs,
		runner:    runner,
		discovery: discovery,
		goos:      goos,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "terminal"
}

// Compile transforms terminal configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("terminal")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if !cfg.HasAnyTerminal() {
		return nil, nil
	}

	var steps []compiler.Step

	// Compile Alacritty
	if cfg.Alacritty != nil {
		alacrittySteps, err := p.compileAlacritty(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("alacritty: %w", err)
		}
		steps = append(steps, alacrittySteps...)
	}

	// Compile Kitty
	if cfg.Kitty != nil {
		kittySteps, err := p.compileKitty(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("kitty: %w", err)
		}
		steps = append(steps, kittySteps...)
	}

	// Compile WezTerm
	if cfg.WezTerm != nil {
		weztermSteps, err := p.compileWezTerm(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("wezterm: %w", err)
		}
		steps = append(steps, weztermSteps...)
	}

	// Compile Ghostty
	if cfg.Ghostty != nil {
		ghosttySteps, err := p.compileGhostty(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("ghostty: %w", err)
		}
		steps = append(steps, ghosttySteps...)
	}

	// Compile iTerm2 (macOS only)
	if cfg.ITerm2 != nil && p.goos == "darwin" {
		iterm2Steps, err := p.compileITerm2(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("iterm2: %w", err)
		}
		steps = append(steps, iterm2Steps...)
	}

	// Compile Hyper
	if cfg.Hyper != nil {
		hyperSteps, err := p.compileHyper(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("hyper: %w", err)
		}
		steps = append(steps, hyperSteps...)
	}

	// Compile Windows Terminal (Windows only)
	if cfg.WindowsTerminal != nil && p.goos == "windows" {
		wtSteps, err := p.compileWindowsTerminal(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("windows_terminal: %w", err)
		}
		steps = append(steps, wtSteps...)
	}

	return steps, nil
}

// compileAlacritty generates steps for Alacritty configuration.
func (p *Provider) compileAlacritty(ctx compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.AlacrittyBestPracticePath()
	if cfg.Alacritty.ConfigPath != "" {
		targetPath = cfg.Alacritty.ConfigPath
	}

	step := NewAlacrittyConfigStep(
		cfg.Alacritty,
		cfg,
		targetPath,
		ctx.ConfigRoot(),
		p.fs,
	)

	return []compiler.Step{step}, nil
}

// compileKitty generates steps for Kitty configuration.
func (p *Provider) compileKitty(ctx compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.KittyBestPracticePath()
	if cfg.Kitty.ConfigPath != "" {
		targetPath = cfg.Kitty.ConfigPath
	}

	step := NewKittyConfigStep(
		cfg.Kitty,
		cfg,
		targetPath,
		ctx.ConfigRoot(),
		p.fs,
	)

	return []compiler.Step{step}, nil
}

// compileWezTerm generates steps for WezTerm configuration.
func (p *Provider) compileWezTerm(ctx compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.WezTermBestPracticePath()
	if cfg.WezTerm.ConfigPath != "" {
		targetPath = cfg.WezTerm.ConfigPath
	}

	step := NewWezTermConfigStep(
		cfg.WezTerm,
		targetPath,
		ctx.ConfigRoot(),
		p.fs,
	)

	return []compiler.Step{step}, nil
}

// compileGhostty generates steps for Ghostty configuration.
func (p *Provider) compileGhostty(ctx compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.GhosttyBestPracticePath()
	if cfg.Ghostty.ConfigPath != "" {
		targetPath = cfg.Ghostty.ConfigPath
	}

	step := NewGhosttyConfigStep(
		cfg.Ghostty,
		targetPath,
		ctx.ConfigRoot(),
		p.fs,
	)

	return []compiler.Step{step}, nil
}

// compileITerm2 generates steps for iTerm2 configuration.
func (p *Provider) compileITerm2(_ compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	var steps []compiler.Step

	// Settings step
	if len(cfg.ITerm2.Settings) > 0 {
		steps = append(steps, NewITerm2SettingsStep(cfg.ITerm2, p.runner))
	}

	// Dynamic profiles step
	if len(cfg.ITerm2.DynamicProfiles) > 0 {
		profilesDir := p.discovery.ITerm2DynamicProfilesDir()
		steps = append(steps, NewITerm2ProfilesStep(cfg.ITerm2, profilesDir, p.fs))
	}

	return steps, nil
}

// compileHyper generates steps for Hyper configuration.
func (p *Provider) compileHyper(ctx compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.HyperBestPracticePath()
	if cfg.Hyper.ConfigPath != "" {
		targetPath = cfg.Hyper.ConfigPath
	}

	step := NewHyperConfigStep(
		cfg.Hyper,
		targetPath,
		ctx.ConfigRoot(),
		p.fs,
	)

	return []compiler.Step{step}, nil
}

// compileWindowsTerminal generates steps for Windows Terminal configuration.
func (p *Provider) compileWindowsTerminal(_ compiler.CompileContext, cfg *Config) ([]compiler.Step, error) {
	targetPath := p.discovery.WindowsTerminalBestPracticePath()

	step := NewWindowsTerminalConfigStep(
		cfg.WindowsTerminal,
		targetPath,
		p.fs,
	)

	return []compiler.Step{step}, nil
}
