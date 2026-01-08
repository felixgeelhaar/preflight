package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type captureFixture struct {
	nvimConfig string
	sshConfig  string
	findings   *CaptureFindings
	dotfiles   *DotfilesCaptureResult
}

func newCaptureFixture(t *testing.T, target string) *captureFixture {
	t.Helper()

	nvimConfig := filepath.Join(target, "nvim-config")
	require.NoError(t, os.MkdirAll(filepath.Join(nvimConfig, ".git"), 0o755))
	lockPath := filepath.Join(nvimConfig, "lazy-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"plugin-one": {}}`), 0o644))

	sshDir := filepath.Join(target, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0o755))
	sshConfig := filepath.Join(sshDir, "config")
	require.NoError(t, os.WriteFile(sshConfig, []byte("Host example\n  HostName example.com\n"), 0o644))
	keyPath := filepath.Join(sshDir, "id_example")
	require.NoError(t, os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nPRIVATE KEY\n"), 0o600))
	require.NoError(t, os.WriteFile(keyPath+".pub", []byte("ssh-ed25519 AAAA comment\n"), 0o644))

	dotfiles := &DotfilesCaptureResult{
		Dotfiles: []CapturedDotfile{
			{Provider: "git", HomeRelPath: ".gitconfig"},
			{Provider: "shell", HomeRelPath: ".zshrc"},
			{Provider: "starship", HomeRelPath: ".config/starship.toml"},
			{Provider: "nvim", HomeRelPath: ".config/nvim", IsDirectory: true},
			{Provider: "vscode", HomeRelPath: ".config/Code/User", IsDirectory: true},
			{Provider: "tmux", HomeRelPath: ".tmux.conf"},
			{Provider: "ssh", HomeRelPath: ".ssh", IsDirectory: true},
		},
	}

	findings := &CaptureFindings{
		Items: []CapturedItem{
			{Provider: "brew", Name: "formula-one"},
			{Provider: "brew-cask", Name: "cask-one"},
			{Provider: "npm", Name: "npm-tool", Value: "npm-tool@1.0"},
			{Provider: "go", Name: "golang.org/x/tools", Value: "golang.org/x/tools"},
			{Provider: "pip", Name: "pip-tool", Value: "pip-tool==2.0"},
			{Provider: "gem", Name: "gem-tool", Value: "gem-tool"},
			{Provider: "cargo", Name: "cargo-tool", Value: "cargo-tool"},
			{Provider: "git", Name: "user.name", Value: "Preflight Test"},
			{Provider: "shell", Name: ".zshrc"},
			{Provider: "vscode", Name: "ms-vscode.go"},
			{Provider: "runtime", Name: "node", Value: "18.0.0"},
			{Provider: "nvim", Name: "config", Value: nvimConfig},
			{Provider: "nvim", Name: "lazy-lock.json", Value: lockPath},
			{Provider: "ssh", Name: "config", Value: sshConfig},
		},
	}

	return &captureFixture{
		nvimConfig: nvimConfig,
		sshConfig:  sshConfig,
		findings:   findings,
		dotfiles:   dotfiles,
	}
}

type fakeAICategorizer struct{}

func (fakeAICategorizer) Categorize(context.Context, AICategorizationRequest) (*AICategorizationResult, error) {
	return nil, nil
}

func TestCaptureConfigGenerator_BuilderHelpers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	g := NewCaptureConfigGenerator(dir)
	require.NotNil(t, g)

	assert.False(t, g.smartSplit)
	g.WithSmartSplit(true)
	assert.True(t, g.smartSplit)

	g.WithSplitStrategy(SplitByProvider)
	assert.True(t, g.smartSplit)
	assert.Equal(t, SplitByProvider, g.splitStrategy)

	ai := fakeAICategorizer{}
	g.WithAICategorizer(ai)
	assert.Equal(t, ai, g.aiCategorizer)

	dotfiles := &DotfilesCaptureResult{}
	g.WithDotfiles(dotfiles)
	assert.Equal(t, dotfiles, g.dotfilesResult)
}

func TestCaptureConfigGenerator_GenerateSmartSplitProviders(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	setup := newCaptureFixture(t, target)

	generator := NewCaptureConfigGenerator(target).
		WithSmartSplit(true).
		WithSplitStrategy(SplitByProvider).
		WithDotfiles(setup.dotfiles)

	err := generator.GenerateFromCapture(setup.findings, "default")
	require.NoError(t, err)

	manifestData, err := os.ReadFile(filepath.Join(target, "preflight.yaml"))
	require.NoError(t, err)

	var manifest captureManifestYAML
	require.NoError(t, yaml.Unmarshal(manifestData, &manifest))
	expectedLayers := []string{
		"brew", "git", "shell", "vscode", "runtime", "nvim", "ssh",
		"npm", "go", "pip", "gem", "cargo",
	}
	assert.ElementsMatch(t, expectedLayers, manifest.Targets["default"])

	brew := mustReadLayer(t, target, "brew")
	require.NotNil(t, brew.Packages)
	assert.Contains(t, brew.Packages.Brew.Formulae, "formula-one")
	assert.Contains(t, brew.Packages.Brew.Casks, "cask-one")

	assert.Contains(t, mustReadLayer(t, target, "npm").Packages.Npm.Packages, "npm-tool@1.0")
	assert.Contains(t, mustReadLayer(t, target, "go").Packages.Go.Tools, "golang.org/x/tools")
	assert.Contains(t, mustReadLayer(t, target, "pip").Packages.Pip.Packages, "pip-tool==2.0")
	assert.Contains(t, mustReadLayer(t, target, "gem").Packages.Gem.Gems, "gem-tool")
	assert.Contains(t, mustReadLayer(t, target, "cargo").Packages.Cargo.Crates, "cargo-tool")

	assert.Equal(t, ".gitconfig", mustReadLayer(t, target, "git").Git.ConfigSource)

	shell := mustReadLayer(t, target, "shell")
	require.NotNil(t, shell.Shell)
	assert.Equal(t, ".zshrc", shell.Shell.ConfigSource.Dir)
	require.NotNil(t, shell.Shell.Starship)
	assert.Equal(t, ".config", shell.Shell.Starship.ConfigSource)
	require.NotNil(t, shell.Tmux)
	assert.Equal(t, ".tmux.conf", shell.Tmux.ConfigSource)

	vscodeLayer := mustReadLayer(t, target, "vscode")
	require.NotNil(t, vscodeLayer.VSCode)
	assert.Equal(t, ".config/Code/User", vscodeLayer.VSCode.ConfigSource)
	assert.Contains(t, vscodeLayer.VSCode.Extensions, "ms-vscode.go")

	nvm := mustReadLayer(t, target, "nvim")
	require.NotNil(t, nvm.Nvim)
	assert.Equal(t, setup.nvimConfig, nvm.Nvim.ConfigPath)
	assert.Equal(t, ".config/nvim", nvm.Nvim.ConfigSource)
	assert.Equal(t, "lazy.nvim", nvm.Nvim.PluginManager)
	assert.Equal(t, 1, nvm.Nvim.PluginCount)
	assert.True(t, nvm.Nvim.ConfigManaged)

	runtimeLayer := mustReadLayer(t, target, "runtime")
	require.NotNil(t, runtimeLayer.Runtime)
	require.Len(t, runtimeLayer.Runtime.Tools, 1)
	assert.Equal(t, "node", runtimeLayer.Runtime.Tools[0].Name)
	assert.Equal(t, "18.0.0", runtimeLayer.Runtime.Tools[0].Version)

	ssh := mustReadLayer(t, target, "ssh")
	require.NotNil(t, ssh.SSH)
	assert.Equal(t, ".ssh", ssh.SSH.ConfigSource)
	require.NotEmpty(t, ssh.SSH.Hosts)
	assert.Equal(t, "example.com", ssh.SSH.Hosts[0].HostName)
	require.NotEmpty(t, ssh.SSH.Keys)
	assert.Equal(t, "ed25519", ssh.SSH.Keys[0].Type)

}

func TestCaptureConfigGenerator_GenerateCapturedLayer(t *testing.T) {
	t.Parallel()

	target := t.TempDir()
	setup := newCaptureFixture(t, target)

	generator := NewCaptureConfigGenerator(target).WithDotfiles(setup.dotfiles)
	require.NoError(t, generator.GenerateFromCapture(setup.findings, "default"))

	captured := mustReadLayer(t, target, "captured")
	require.NotNil(t, captured.Packages)
	assert.Contains(t, captured.Packages.Brew.Formulae, "formula-one")
	assert.Contains(t, captured.Packages.Brew.Casks, "cask-one")
	require.NotNil(t, captured.Git)
	assert.Equal(t, ".gitconfig", captured.Git.ConfigSource)
	require.NotNil(t, captured.Shell)
	assert.Equal(t, ".zshrc", captured.Shell.ConfigSource.Dir)
	require.NotNil(t, captured.Shell.Starship)
	assert.Equal(t, ".config", captured.Shell.Starship.ConfigSource)
	require.NotNil(t, captured.VSCode)
	assert.Equal(t, ".config/Code/User", captured.VSCode.ConfigSource)
	assert.Contains(t, captured.VSCode.Extensions, "ms-vscode.go")
	require.NotNil(t, captured.Runtime)
	require.Len(t, captured.Runtime.Tools, 1)
	assert.Equal(t, "node", captured.Runtime.Tools[0].Name)
	assert.Equal(t, "18.0.0", captured.Runtime.Tools[0].Version)
	require.NotNil(t, captured.Nvim)
	assert.Equal(t, setup.nvimConfig, captured.Nvim.ConfigPath)
	assert.Equal(t, ".config/nvim", captured.Nvim.ConfigSource)
	require.NotNil(t, captured.SSH)
	assert.Equal(t, ".ssh", captured.SSH.ConfigSource)
	require.NotEmpty(t, captured.SSH.Hosts)
	assert.Equal(t, "example.com", captured.SSH.Hosts[0].HostName)
	require.NotEmpty(t, captured.SSH.Keys)
	assert.Equal(t, "ed25519", captured.SSH.Keys[0].Type)
}

func TestCaptureConfigGenerator_buildLayerFromBrewItems(t *testing.T) {
	t.Parallel()

	generator := NewCaptureConfigGenerator(t.TempDir())
	items := []CapturedItem{
		{Provider: "brew", Name: "formula-one"},
		{Provider: "brew-cask", Name: "cask-one"},
	}

	layer := generator.buildLayerFromBrewItems("brew", items)
	require.NotNil(t, layer)
	require.NotNil(t, layer.Packages)
	require.Contains(t, layer.Packages.Brew.Formulae, "formula-one")
	require.Contains(t, layer.Packages.Brew.Casks, "cask-one")
}

func TestCaptureConfigGenerator_addProviderConfigToLayer(t *testing.T) {
	baseDir := t.TempDir()
	nvimDir := filepath.Join(baseDir, "nvim-config")
	require.NoError(t, os.MkdirAll(nvimDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte(""), 0o644))

	sshDir := filepath.Join(baseDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0o755))
	sshConfig := filepath.Join(sshDir, "config")
	require.NoError(t, os.WriteFile(sshConfig, []byte("Host example\n  HostName example.com\n"), 0o644))

	generator := NewCaptureConfigGenerator(baseDir)

	cases := []struct {
		name     string
		provider string
		items    []CapturedItem
		verify   func(t *testing.T, layer *captureLayerYAML)
	}{
		{
			name:     "git",
			provider: "git",
			items: []CapturedItem{
				{Name: "user.name", Value: "Name"},
				{Name: "user.email", Value: "email"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Git)
				assert.Equal(t, "Name", layer.Git.User.Name)
			},
		},
		{
			name:     "shell",
			provider: "shell",
			items:    []CapturedItem{{Name: ".zshrc"}},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Shell)
				assert.Equal(t, "zsh", layer.Shell.Default)
			},
		},
		{
			name:     "vscode",
			provider: "vscode",
			items: []CapturedItem{
				{Name: "ms-vscode.go"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.VSCode)
				assert.Contains(t, layer.VSCode.Extensions, "ms-vscode.go")
			},
		},
		{
			name:     "runtime",
			provider: "runtime",
			items: []CapturedItem{
				{Name: "node", Value: "18.0.0"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Runtime)
				assert.Equal(t, "node", layer.Runtime.Tools[0].Name)
			},
		},
		{
			name:     "nvim",
			provider: "nvim",
			items: []CapturedItem{
				{Name: "config", Value: nvimDir},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Nvim)
				assert.Equal(t, nvimDir, layer.Nvim.ConfigPath)
			},
		},
		{
			name:     "ssh",
			provider: "ssh",
			items: []CapturedItem{
				{Name: "config", Value: sshConfig},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.SSH)
				assert.Equal(t, sshConfig, layer.SSH.ConfigPath)
			},
		},
		{
			name:     "npm",
			provider: "npm",
			items: []CapturedItem{
				{Name: "npm-tool", Value: "npm-tool@1.0"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Packages)
				require.NotNil(t, layer.Packages.Npm)
				assert.Contains(t, layer.Packages.Npm.Packages, "npm-tool@1.0")
			},
		},
		{
			name:     "go",
			provider: "go",
			items: []CapturedItem{
				{Name: "golang", Value: "golang.org/x/tools"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Packages)
				require.NotNil(t, layer.Packages.Go)
				assert.Contains(t, layer.Packages.Go.Tools, "golang.org/x/tools")
			},
		},
		{
			name:     "pip",
			provider: "pip",
			items: []CapturedItem{
				{Name: "pip-tool", Value: "pip-tool==2.0"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Packages)
				require.NotNil(t, layer.Packages.Pip)
				assert.Contains(t, layer.Packages.Pip.Packages, "pip-tool==2.0")
			},
		},
		{
			name:     "gem",
			provider: "gem",
			items: []CapturedItem{
				{Name: "gem-tool", Value: "gem-tool"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Packages)
				require.NotNil(t, layer.Packages.Gem)
				assert.Contains(t, layer.Packages.Gem.Gems, "gem-tool")
			},
		},
		{
			name:     "cargo",
			provider: "cargo",
			items: []CapturedItem{
				{Name: "cargo-tool", Value: "cargo-tool"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Packages)
				require.NotNil(t, layer.Packages.Cargo)
				assert.Contains(t, layer.Packages.Cargo.Crates, "cargo-tool")
			},
		},
		{
			name:     "terminal",
			provider: "terminal",
			items: []CapturedItem{
				{Name: "alacritty", Value: ".config/alacritty/alacritty.toml"},
				{Name: "wezterm", Value: ".config/wezterm/wezterm.lua"},
				{Name: "kitty", Value: ".config/kitty/kitty.conf"},
			},
			verify: func(t *testing.T, layer *captureLayerYAML) {
				require.NotNil(t, layer.Terminal)
				require.NotNil(t, layer.Terminal.Alacritty)
				assert.Equal(t, ".config/alacritty/alacritty.toml", layer.Terminal.Alacritty.Source)
				assert.True(t, layer.Terminal.Alacritty.Link)
				require.NotNil(t, layer.Terminal.WezTerm)
				assert.Equal(t, ".config/wezterm/wezterm.lua", layer.Terminal.WezTerm.Source)
				require.NotNil(t, layer.Terminal.Kitty)
				assert.Equal(t, ".config/kitty/kitty.conf", layer.Terminal.Kitty.Source)
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			layer := &captureLayerYAML{}
			generator.addProviderConfigToLayer(layer, tt.provider, tt.items)
			tt.verify(t, layer)
		})
	}
}

func TestCaptureConfigGenerator_writeLayerFileAndOrderedList(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(target, "layers"), 0o755))

	generator := NewCaptureConfigGenerator(target)
	layer := &captureLayerYAML{Name: "features"}

	require.NoError(t, generator.writeLayerFile("features", layer, "Important layer"))

	data, err := os.ReadFile(filepath.Join(target, "layers", "features.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Important layer")

	created := map[string]bool{
		"base":    true,
		"shell":   true,
		"runtime": true,
	}
	order := generator.buildOrderedLayerList([]string{"runtime", "base"}, created)
	assert.Equal(t, []string{"runtime", "base", "shell"}, order)
}

func mustReadLayer(t *testing.T, target, name string) captureLayerYAML {
	t.Helper()
	path := filepath.Join(target, "layers", name+".yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var layer captureLayerYAML
	require.NoError(t, yaml.Unmarshal(data, &layer))
	return layer
}
