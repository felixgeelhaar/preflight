package testutil

import (
	"fmt"
	"strings"
)

// TestManifest is a simplified manifest structure for testing.
type TestManifest struct {
	Version  int
	Targets  []TestTarget
	Defaults map[string]interface{}
}

// TestTarget is a simplified target structure for testing.
type TestTarget struct {
	Name   string
	Layers []string
}

// ManifestBuilder builds test manifests.
type ManifestBuilder struct {
	manifest TestManifest
}

// NewManifestBuilder creates a new manifest builder.
func NewManifestBuilder() *ManifestBuilder {
	return &ManifestBuilder{
		manifest: TestManifest{
			Version: 1,
			Targets: make([]TestTarget, 0),
		},
	}
}

// WithVersion sets the manifest version.
func (b *ManifestBuilder) WithVersion(version int) *ManifestBuilder {
	b.manifest.Version = version
	return b
}

// WithTarget adds a target with the specified layers.
func (b *ManifestBuilder) WithTarget(name string, layers ...string) *ManifestBuilder {
	b.manifest.Targets = append(b.manifest.Targets, TestTarget{
		Name:   name,
		Layers: layers,
	})
	return b
}

// WithDefaults sets the default configuration.
func (b *ManifestBuilder) WithDefaults(defaults map[string]interface{}) *ManifestBuilder {
	b.manifest.Defaults = defaults
	return b
}

// Build returns the constructed manifest.
func (b *ManifestBuilder) Build() TestManifest {
	return b.manifest
}

// ToYAML converts the manifest to YAML string.
func (m TestManifest) ToYAML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("version: %d\n", m.Version))

	if len(m.Targets) > 0 {
		sb.WriteString("targets:\n")
		for _, t := range m.Targets {
			sb.WriteString(fmt.Sprintf("  - name: %s\n", t.Name))
			sb.WriteString("    layers:\n")
			for _, l := range t.Layers {
				sb.WriteString(fmt.Sprintf("      - %s\n", l))
			}
		}
	}

	return sb.String()
}

// TestLayer is a simplified layer structure for testing.
type TestLayer struct {
	Name string
	Brew TestBrewConfig
	Git  TestGitConfig
	Nvim TestNvimConfig
}

// TestBrewConfig represents Homebrew configuration for testing.
type TestBrewConfig struct {
	Formulae []string
	Casks    []string
	Taps     []string
}

// TestGitConfig represents Git configuration for testing.
type TestGitConfig struct {
	Config map[string]string
}

// TestNvimConfig represents Neovim configuration for testing.
type TestNvimConfig struct {
	Preset string
}

// LayerBuilder builds test layers.
type LayerBuilder struct {
	layer TestLayer
}

// NewLayerBuilder creates a new layer builder.
func NewLayerBuilder(name string) *LayerBuilder {
	return &LayerBuilder{
		layer: TestLayer{
			Name: name,
			Git: TestGitConfig{
				Config: make(map[string]string),
			},
		},
	}
}

// WithBrew adds Homebrew formulae.
func (b *LayerBuilder) WithBrew(formulae ...string) *LayerBuilder {
	b.layer.Brew.Formulae = append(b.layer.Brew.Formulae, formulae...)
	return b
}

// WithCask adds Homebrew casks.
func (b *LayerBuilder) WithCask(casks ...string) *LayerBuilder {
	b.layer.Brew.Casks = append(b.layer.Brew.Casks, casks...)
	return b
}

// WithTap adds Homebrew taps.
func (b *LayerBuilder) WithTap(taps ...string) *LayerBuilder {
	b.layer.Brew.Taps = append(b.layer.Brew.Taps, taps...)
	return b
}

// WithGit adds a git config setting.
func (b *LayerBuilder) WithGit(key, value string) *LayerBuilder {
	b.layer.Git.Config[key] = value
	return b
}

// WithNvimPreset sets the Neovim preset.
func (b *LayerBuilder) WithNvimPreset(preset string) *LayerBuilder {
	b.layer.Nvim.Preset = preset
	return b
}

// Build returns the constructed layer.
func (b *LayerBuilder) Build() TestLayer {
	return b.layer
}

// ToYAML converts the layer to YAML string.
func (l TestLayer) ToYAML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("name: %s\n", l.Name))

	if len(l.Brew.Formulae) > 0 || len(l.Brew.Casks) > 0 || len(l.Brew.Taps) > 0 {
		sb.WriteString("brew:\n")
		if len(l.Brew.Taps) > 0 {
			sb.WriteString("  taps:\n")
			for _, t := range l.Brew.Taps {
				sb.WriteString(fmt.Sprintf("    - %s\n", t))
			}
		}
		if len(l.Brew.Formulae) > 0 {
			sb.WriteString("  formulae:\n")
			for _, f := range l.Brew.Formulae {
				sb.WriteString(fmt.Sprintf("    - %s\n", f))
			}
		}
		if len(l.Brew.Casks) > 0 {
			sb.WriteString("  casks:\n")
			for _, c := range l.Brew.Casks {
				sb.WriteString(fmt.Sprintf("    - %s\n", c))
			}
		}
	}

	if len(l.Git.Config) > 0 {
		sb.WriteString("git:\n")
		sb.WriteString("  config:\n")
		for k, v := range l.Git.Config {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", k, v))
		}
	}

	if l.Nvim.Preset != "" {
		sb.WriteString("nvim:\n")
		sb.WriteString(fmt.Sprintf("  preset: %s\n", l.Nvim.Preset))
	}

	return sb.String()
}
