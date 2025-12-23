package nvim

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Provider compiles nvim configuration into executable steps.
type Provider struct {
	fs     ports.FileSystem
	runner ports.CommandRunner
}

// NewProvider creates a new nvim Provider.
func NewProvider(fs ports.FileSystem, runner ports.CommandRunner) *Provider {
	return &Provider{
		fs:     fs,
		runner: runner,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "nvim"
}

// Compile transforms nvim configuration into executable steps.
func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
	rawConfig := ctx.GetSection("nvim")
	if rawConfig == nil {
		return nil, nil
	}

	cfg, err := ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	var steps []compiler.Step

	// Add preset step if specified
	if cfg.Preset != "" {
		steps = append(steps, NewPresetStep(cfg.Preset, p.fs, p.runner))
	}

	// Add config repo step if specified (alternative to preset)
	if cfg.ConfigRepo != "" && cfg.Preset == "" {
		steps = append(steps, NewConfigRepoStep(cfg.ConfigRepo, p.fs, p.runner))
	}

	// Add lazy-lock step if using lazy plugin manager
	if cfg.PluginManager == "lazy" {
		steps = append(steps, NewLazyLockStep(p.fs))
	}

	return steps, nil
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
