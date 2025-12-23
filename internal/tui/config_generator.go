package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigGenerator generates preflight configuration files from presets.
type ConfigGenerator struct {
	targetDir string
}

// NewConfigGenerator creates a new ConfigGenerator.
func NewConfigGenerator(targetDir string) *ConfigGenerator {
	return &ConfigGenerator{
		targetDir: targetDir,
	}
}

// GenerateFromPreset generates configuration files from a preset.
func (g *ConfigGenerator) GenerateFromPreset(preset PresetItem) error {
	// Parse preset ID to get provider and preset name
	parts := strings.SplitN(preset.ID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid preset ID format: %s", preset.ID)
	}
	provider := parts[0]
	presetName := parts[1]

	// Ensure target directory exists
	if err := os.MkdirAll(g.targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Ensure layers directory exists
	layersDir := filepath.Join(g.targetDir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		return fmt.Errorf("failed to create layers directory: %w", err)
	}

	// Generate manifest
	if err := g.generateManifest(); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Generate base layer
	if err := g.generateBaseLayer(provider, presetName); err != nil {
		return fmt.Errorf("failed to generate base layer: %w", err)
	}

	return nil
}

// generateManifest generates the preflight.yaml manifest file.
func (g *ConfigGenerator) generateManifest() error {
	manifest := manifestYAML{
		Defaults: defaultsYAML{
			Mode: "intent",
		},
		Targets: map[string][]string{
			"default": {"base"},
		},
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(g.targetDir, "preflight.yaml")
	return os.WriteFile(manifestPath, data, 0o644)
}

// generateBaseLayer generates the layers/base.yaml file.
func (g *ConfigGenerator) generateBaseLayer(provider, presetName string) error {
	layer := layerYAML{
		Name: "base",
	}

	// Configure the appropriate provider section based on the preset
	switch provider {
	case "nvim":
		layer.Nvim = &nvimYAML{
			Preset: presetName,
		}
	case "shell":
		layer.Shell = g.generateShellConfig(presetName)
	case "git":
		layer.Git = g.generateGitConfig(presetName)
	case "brew":
		layer.Packages = g.generateBrewConfig(presetName)
	}

	data, err := yaml.Marshal(layer)
	if err != nil {
		return err
	}

	layerPath := filepath.Join(g.targetDir, "layers", "base.yaml")
	return os.WriteFile(layerPath, data, 0o644)
}

// generateShellConfig generates shell configuration for a preset.
func (g *ConfigGenerator) generateShellConfig(presetName string) *shellYAML {
	switch presetName {
	case "zsh":
		return &shellYAML{
			Default: "zsh",
			Shells: []shellEntryYAML{
				{Name: "zsh"},
			},
		}
	case "oh-my-zsh":
		return &shellYAML{
			Default: "zsh",
			Shells: []shellEntryYAML{
				{
					Name:      "zsh",
					Framework: "oh-my-zsh",
					Plugins:   []string{"git", "docker", "kubectl"},
				},
			},
		}
	case "starship":
		return &shellYAML{
			Default: "zsh",
			Shells: []shellEntryYAML{
				{Name: "zsh"},
			},
			Starship: &starshipYAML{
				Enabled: true,
				Preset:  "gruvbox-rainbow",
			},
		}
	default:
		return &shellYAML{
			Default: "zsh",
		}
	}
}

// generateGitConfig generates git configuration for a preset.
func (g *ConfigGenerator) generateGitConfig(presetName string) *gitYAML {
	switch presetName {
	case "standard":
		return &gitYAML{
			Core: &gitCoreYAML{
				Editor:   "nvim",
				AutoCRLF: "input",
			},
			Aliases: map[string]string{
				"co": "checkout",
				"br": "branch",
				"ci": "commit",
				"st": "status",
			},
		}
	case "secure":
		return &gitYAML{
			Core: &gitCoreYAML{
				Editor:   "nvim",
				AutoCRLF: "input",
			},
			Commit: &gitCommitYAML{
				GPGSign: true,
			},
			GPG: &gitGPGYAML{
				Format: "ssh",
			},
			Aliases: map[string]string{
				"co": "checkout",
				"br": "branch",
				"ci": "commit",
				"st": "status",
			},
		}
	default:
		return &gitYAML{}
	}
}

// generateBrewConfig generates Homebrew configuration for a preset.
func (g *ConfigGenerator) generateBrewConfig(presetName string) *packagesYAML {
	switch presetName {
	case "minimal":
		return &packagesYAML{
			Brew: &brewYAML{
				Formulae: []string{"git", "neovim", "ripgrep", "fd"},
			},
		}
	case "developer":
		return &packagesYAML{
			Brew: &brewYAML{
				Taps:     []string{"homebrew/cask-fonts"},
				Formulae: []string{"git", "neovim", "ripgrep", "fd", "fzf", "jq", "gh", "lazygit"},
				Casks:    []string{"font-jetbrains-mono-nerd-font", "wezterm"},
			},
		}
	default:
		return &packagesYAML{}
	}
}

// YAML structure types for marshaling

type manifestYAML struct {
	Defaults defaultsYAML        `yaml:"defaults,omitempty"`
	Targets  map[string][]string `yaml:"targets"`
}

type defaultsYAML struct {
	Mode string `yaml:"mode,omitempty"`
}

type layerYAML struct {
	Name     string        `yaml:"name"`
	Packages *packagesYAML `yaml:"packages,omitempty"`
	Git      *gitYAML      `yaml:"git,omitempty"`
	Shell    *shellYAML    `yaml:"shell,omitempty"`
	Nvim     *nvimYAML     `yaml:"nvim,omitempty"`
}

type packagesYAML struct {
	Brew *brewYAML `yaml:"brew,omitempty"`
}

type brewYAML struct {
	Taps     []string `yaml:"taps,omitempty"`
	Formulae []string `yaml:"formulae,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

type gitYAML struct {
	User    *gitUserYAML      `yaml:"user,omitempty"`
	Core    *gitCoreYAML      `yaml:"core,omitempty"`
	Commit  *gitCommitYAML    `yaml:"commit,omitempty"`
	GPG     *gitGPGYAML       `yaml:"gpg,omitempty"`
	Aliases map[string]string `yaml:"alias,omitempty"`
}

type gitUserYAML struct {
	Name  string `yaml:"name,omitempty"`
	Email string `yaml:"email,omitempty"`
}

type gitCoreYAML struct {
	Editor   string `yaml:"editor,omitempty"`
	AutoCRLF string `yaml:"autocrlf,omitempty"`
}

type gitCommitYAML struct {
	GPGSign bool `yaml:"gpgsign,omitempty"`
}

type gitGPGYAML struct {
	Format string `yaml:"format,omitempty"`
}

type shellYAML struct {
	Default  string           `yaml:"default,omitempty"`
	Shells   []shellEntryYAML `yaml:"shells,omitempty"`
	Starship *starshipYAML    `yaml:"starship,omitempty"`
}

type shellEntryYAML struct {
	Name      string   `yaml:"name"`
	Framework string   `yaml:"framework,omitempty"`
	Theme     string   `yaml:"theme,omitempty"`
	Plugins   []string `yaml:"plugins,omitempty"`
}

type starshipYAML struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Preset  string `yaml:"preset,omitempty"`
}

type nvimYAML struct {
	Preset        string `yaml:"preset,omitempty"`
	PluginManager string `yaml:"plugin_manager,omitempty"`
	EnsureInstall bool   `yaml:"ensure_install,omitempty"`
}
