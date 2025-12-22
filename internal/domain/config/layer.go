// Package config provides the configuration domain for Preflight.
// It handles loading, parsing, merging, and validating workstation configurations.
package config

import (
	"gopkg.in/yaml.v3"
)

// FileMode represents how a dotfile is managed.
type FileMode string

const (
	// FileModeGenerated means Preflight owns the file completely.
	FileModeGenerated FileMode = "generated"
	// FileModeTemplate means Preflight manages a base with user extensions.
	FileModeTemplate FileMode = "template"
	// FileModeBYO means user owns the file; Preflight links/validates only.
	FileModeBYO FileMode = "byo"
)

// FileDeclaration represents a managed dotfile.
type FileDeclaration struct {
	Path     string   `yaml:"path"`
	Mode     FileMode `yaml:"mode"`
	Template string   `yaml:"template,omitempty"`
}

// BrewPackages represents Homebrew package configuration.
type BrewPackages struct {
	Taps     []string `yaml:"taps,omitempty"`
	Formulae []string `yaml:"formulae,omitempty"`
	Casks    []string `yaml:"casks,omitempty"`
}

// AptPackages represents apt package configuration.
type AptPackages struct {
	Packages []string `yaml:"packages,omitempty"`
}

// PackageSet represents all package manager configurations.
type PackageSet struct {
	Brew BrewPackages `yaml:"brew,omitempty"`
	Apt  AptPackages  `yaml:"apt,omitempty"`
}

// GitUserConfig represents git user configuration.
type GitUserConfig struct {
	Name       string `yaml:"name,omitempty"`
	Email      string `yaml:"email,omitempty"`
	SigningKey string `yaml:"signingkey,omitempty"`
}

// GitCoreConfig represents git core configuration.
type GitCoreConfig struct {
	Editor       string `yaml:"editor,omitempty"`
	AutoCRLF     string `yaml:"autocrlf,omitempty"`
	ExcludesFile string `yaml:"excludesfile,omitempty"`
}

// GitCommitConfig represents git commit configuration.
type GitCommitConfig struct {
	GPGSign bool `yaml:"gpgsign,omitempty"`
}

// GitGPGConfig represents git gpg configuration.
type GitGPGConfig struct {
	Format  string `yaml:"format,omitempty"`
	Program string `yaml:"program,omitempty"`
}

// GitInclude represents a conditional include directive.
type GitInclude struct {
	Path     string `yaml:"path"`
	IfConfig string `yaml:"ifconfig,omitempty"`
}

// GitConfig represents git configuration.
type GitConfig struct {
	User     GitUserConfig     `yaml:"user,omitempty"`
	Core     GitCoreConfig     `yaml:"core,omitempty"`
	Commit   GitCommitConfig   `yaml:"commit,omitempty"`
	GPG      GitGPGConfig      `yaml:"gpg,omitempty"`
	Aliases  map[string]string `yaml:"alias,omitempty"`
	Includes []GitInclude      `yaml:"includes,omitempty"`
}

// Layer is a composable configuration overlay.
type Layer struct {
	Name       LayerName
	Provenance string
	Packages   PackageSet
	Files      []FileDeclaration
	Git        GitConfig
}

// layerYAML is the YAML representation for unmarshaling.
type layerYAML struct {
	Name     string            `yaml:"name"`
	Packages PackageSet        `yaml:"packages,omitempty"`
	Files    []FileDeclaration `yaml:"files,omitempty"`
	Git      GitConfig         `yaml:"git,omitempty"`
}

// ParseLayer parses a Layer from YAML bytes.
func ParseLayer(data []byte) (*Layer, error) {
	var raw layerYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	name, err := NewLayerName(raw.Name)
	if err != nil {
		return nil, err
	}

	return &Layer{
		Name:     name,
		Packages: raw.Packages,
		Files:    raw.Files,
		Git:      raw.Git,
	}, nil
}

// SetProvenance sets the file path origin for this layer.
func (l *Layer) SetProvenance(path string) {
	l.Provenance = path
}
