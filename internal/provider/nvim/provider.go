package nvim

import (
	"path/filepath"
	"strings"

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
	case cfg.Preset != "" && cfg.Preset != "custom":
		// Add preset step if specified (skip "custom" which means user has their own config)
		steps = append(steps, NewPresetStep(cfg.Preset, p.fs, p.runner))
	}

	// Add lazy-lock step if using lazy plugin manager
	if cfg.PluginManager == "lazy" {
		steps = append(steps, NewLazyLockStep(p.fs, p.runner))
	}

	return steps, nil
}

// resolveConfigSource resolves a config_source path with per-target override support.
// Uses home-mirrored structure where the config root mirrors $HOME.
// Per-target uses suffixed first path component (e.g., .config.work/nvim).
// Returns empty string if path traversal is detected.
func (p *Provider) resolveConfigSource(configSource, configRoot, target string) string {
	if configRoot == "" || configSource == "" {
		return ""
	}

	// Security: reject paths that could escape configRoot
	if strings.Contains(configSource, "..") {
		return ""
	}

	// Check per-target path first: suffix the first path component
	// e.g., .config/nvim with target "work" -> .config.work/nvim
	if target != "" && target != "default" {
		targetPath := p.applyTargetSuffix(configSource, configRoot, target)
		if p.isPathWithinRoot(configRoot, targetPath) && p.fs.Exists(targetPath) {
			return targetPath
		}
	}

	// Fall back to shared path (home-mirrored structure)
	sharedPath := filepath.Join(configRoot, configSource)
	if p.isPathWithinRoot(configRoot, sharedPath) && p.fs.Exists(sharedPath) {
		return sharedPath
	}

	// Legacy support: check old dotfiles/ structure during migration
	legacyPath := filepath.Join(configRoot, "dotfiles", configSource)
	if p.isPathWithinRoot(configRoot, legacyPath) && p.fs.Exists(legacyPath) {
		return legacyPath
	}

	return ""
}

// isPathWithinRoot validates that a path stays within the config root.
func (p *Provider) isPathWithinRoot(root, path string) bool {
	return ports.IsPathWithinRoot(root, path)
}

// applyTargetSuffix adds the target suffix to the first path component.
// Examples:
//   - ".config/nvim" with target "work" -> configRoot/.config.work/nvim
//   - ".gitconfig" with target "work" -> configRoot/.gitconfig.work
func (p *Provider) applyTargetSuffix(configSource, configRoot, target string) string {
	return ports.ApplyTargetSuffix(configSource, configRoot, target)
}

// Ensure Provider implements compiler.Provider.
var _ compiler.Provider = (*Provider)(nil)
