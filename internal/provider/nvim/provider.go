package nvim

import (
	"os"
	"path/filepath"

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

	// Priority: config_source > config_repo > preset
	switch {
	case cfg.ConfigSource != "":
		// Resolve config_source path with per-target override
		sourcePath := p.resolveConfigSource(cfg.ConfigSource, ctx.ConfigRoot(), ctx.Target())
		if sourcePath != "" {
			steps = append(steps, NewConfigSourceStep(sourcePath, "~/.config/nvim", p.fs))
		}
	case cfg.ConfigRepo != "":
		// Add config repo step if specified (alternative to preset)
		steps = append(steps, NewConfigRepoStep(cfg.ConfigRepo, p.fs, p.runner))
	case cfg.Preset != "":
		// Add preset step if specified
		steps = append(steps, NewPresetStep(cfg.Preset, p.fs, p.runner))
	}

	// Add lazy-lock step if using lazy plugin manager
	if cfg.PluginManager == "lazy" {
		steps = append(steps, NewLazyLockStep(p.fs, p.runner))
	}

	return steps, nil
}

// resolveConfigSource resolves a config_source path with per-target override support.
func (p *Provider) resolveConfigSource(configSource, configRoot, target string) string {
	if configRoot == "" {
		return ""
	}

	// Check per-target directory first: dotfiles.{target}/{configSource}
	if target != "" {
		targetPath := filepath.Join(configRoot, "dotfiles."+target, configSource)
		if _, err := os.Stat(targetPath); err == nil {
			return targetPath
		}
	}

	// Fall back to shared directory: dotfiles/{configSource}
	sharedPath := filepath.Join(configRoot, "dotfiles", configSource)
	if _, err := os.Stat(sharedPath); err == nil {
		return sharedPath
	}

	// If configSource is already a path under dotfiles/, use it directly
	fullPath := filepath.Join(configRoot, configSource)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}

	return ""
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
