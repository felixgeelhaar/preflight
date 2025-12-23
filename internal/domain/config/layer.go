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
	PPAs     []string `yaml:"ppas,omitempty"`
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

// SSHDefaultsConfig represents SSH global defaults (Host *).
type SSHDefaultsConfig struct {
	AddKeysToAgent      bool `yaml:"addkeystoagent,omitempty"`
	IdentitiesOnly      bool `yaml:"identitiesonly,omitempty"`
	ForwardAgent        bool `yaml:"forwardagent,omitempty"`
	ServerAliveInterval int  `yaml:"serveraliveinterval,omitempty"`
	ServerAliveCountMax int  `yaml:"serveralivecountmax,omitempty"`
}

// SSHHostConfig represents an SSH Host block.
type SSHHostConfig struct {
	Host           string `yaml:"host"`
	HostName       string `yaml:"hostname,omitempty"`
	User           string `yaml:"user,omitempty"`
	Port           int    `yaml:"port,omitempty"`
	IdentityFile   string `yaml:"identityfile,omitempty"`
	IdentitiesOnly bool   `yaml:"identitiesonly,omitempty"`
	ForwardAgent   bool   `yaml:"forwardagent,omitempty"`
	ProxyCommand   string `yaml:"proxycommand,omitempty"`
	ProxyJump      string `yaml:"proxyjump,omitempty"`
	LocalForward   string `yaml:"localforward,omitempty"`
	RemoteForward  string `yaml:"remoteforward,omitempty"`
	AddKeysToAgent bool   `yaml:"addkeystoagent,omitempty"`
	UseKeychain    bool   `yaml:"usekeychain,omitempty"`
}

// SSHMatchConfig represents an SSH Match block.
type SSHMatchConfig struct {
	Match        string `yaml:"match"`
	HostName     string `yaml:"hostname,omitempty"`
	User         string `yaml:"user,omitempty"`
	IdentityFile string `yaml:"identityfile,omitempty"`
	ProxyCommand string `yaml:"proxycommand,omitempty"`
	ProxyJump    string `yaml:"proxyjump,omitempty"`
}

// SSHConfig represents SSH configuration.
type SSHConfig struct {
	Include  string            `yaml:"include,omitempty"`
	Defaults SSHDefaultsConfig `yaml:"defaults,omitempty"`
	Hosts    []SSHHostConfig   `yaml:"hosts,omitempty"`
	Matches  []SSHMatchConfig  `yaml:"matches,omitempty"`
}

// RuntimeToolConfig represents a tool with its version.
type RuntimeToolConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// RuntimePluginConfig represents a custom plugin source.
type RuntimePluginConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url,omitempty"`
}

// RuntimeConfig represents runtime version manager configuration.
type RuntimeConfig struct {
	Backend string                `yaml:"backend,omitempty"`
	Scope   string                `yaml:"scope,omitempty"`
	Tools   []RuntimeToolConfig   `yaml:"tools,omitempty"`
	Plugins []RuntimePluginConfig `yaml:"plugins,omitempty"`
}

// ShellCustomPlugin represents a custom shell plugin from a git repository.
type ShellCustomPlugin struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
}

// ShellConfigEntry represents a single shell configuration.
type ShellConfigEntry struct {
	Name          string              `yaml:"name"`
	Framework     string              `yaml:"framework,omitempty"`
	Theme         string              `yaml:"theme,omitempty"`
	Plugins       []string            `yaml:"plugins,omitempty"`
	CustomPlugins []ShellCustomPlugin `yaml:"custom_plugins,omitempty"`
}

// ShellStarshipConfig represents starship prompt configuration.
type ShellStarshipConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Preset  string `yaml:"preset,omitempty"`
}

// ShellConfig represents shell configuration.
type ShellConfig struct {
	Default  string              `yaml:"default,omitempty"`
	Shells   []ShellConfigEntry  `yaml:"shells,omitempty"`
	Starship ShellStarshipConfig `yaml:"starship,omitempty"`
	Env      map[string]string   `yaml:"env,omitempty"`
	Aliases  map[string]string   `yaml:"aliases,omitempty"`
}

// NvimConfig represents Neovim editor configuration.
type NvimConfig struct {
	Preset        string `yaml:"preset,omitempty"`
	PluginManager string `yaml:"plugin_manager,omitempty"`
	ConfigRepo    string `yaml:"config_repo,omitempty"`
	EnsureInstall bool   `yaml:"ensure_install,omitempty"`
}

// Layer is a composable configuration overlay.
type Layer struct {
	Name       LayerName
	Provenance string
	Packages   PackageSet
	Files      []FileDeclaration
	Git        GitConfig
	SSH        SSHConfig
	Runtime    RuntimeConfig
	Shell      ShellConfig
	Nvim       NvimConfig
}

// layerYAML is the YAML representation for unmarshaling.
type layerYAML struct {
	Name     string            `yaml:"name"`
	Packages PackageSet        `yaml:"packages,omitempty"`
	Files    []FileDeclaration `yaml:"files,omitempty"`
	Git      GitConfig         `yaml:"git,omitempty"`
	SSH      SSHConfig         `yaml:"ssh,omitempty"`
	Runtime  RuntimeConfig     `yaml:"runtime,omitempty"`
	Shell    ShellConfig       `yaml:"shell,omitempty"`
	Nvim     NvimConfig        `yaml:"nvim,omitempty"`
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
		SSH:      raw.SSH,
		Runtime:  raw.Runtime,
		Shell:    raw.Shell,
		Nvim:     raw.Nvim,
	}, nil
}

// SetProvenance sets the file path origin for this layer.
func (l *Layer) SetProvenance(path string) {
	l.Provenance = path
}
