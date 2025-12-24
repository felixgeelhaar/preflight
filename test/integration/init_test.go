package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog/embedded"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

func TestInit_ManifestParsing(t *testing.T) {
	t.Parallel()

	manifestYAML := `
targets:
  default:
    - base
  work:
    - base
    - identity.work
`
	manifest, err := config.ParseManifest([]byte(manifestYAML))
	require.NoError(t, err)
	require.NotNil(t, manifest)

	// Should have two targets
	assert.Len(t, manifest.Targets, 2)
	assert.Contains(t, manifest.Targets, "default")
	assert.Contains(t, manifest.Targets, "work")

	// Default target has one layer
	defaultLayers := manifest.Targets["default"]
	assert.Len(t, defaultLayers, 1)
	assert.Equal(t, "base", defaultLayers[0].String())

	// Work target has two layers
	workLayers := manifest.Targets["work"]
	assert.Len(t, workLayers, 2)
	assert.Equal(t, "base", workLayers[0].String())
	assert.Equal(t, "identity.work", workLayers[1].String())
}

func TestInit_LayerParsing_Git(t *testing.T) {
	t.Parallel()

	layerYAML := `
name: base
git:
  user:
    name: "Test User"
    email: "test@example.com"
  core:
    editor: "nvim"
  alias:
    co: "checkout"
    br: "branch"
`
	layer, err := config.ParseLayer([]byte(layerYAML))
	require.NoError(t, err)
	require.NotNil(t, layer)

	assert.Equal(t, "base", layer.Name.String())
	assert.Equal(t, "Test User", layer.Git.User.Name)
	assert.Equal(t, "test@example.com", layer.Git.User.Email)
	assert.Equal(t, "nvim", layer.Git.Core.Editor)
	assert.Equal(t, "checkout", layer.Git.Aliases["co"])
}

func TestInit_LayerParsing_Shell(t *testing.T) {
	t.Parallel()

	layerYAML := `
name: base
shell:
  default: zsh
  shells:
    - name: zsh
      framework: oh-my-zsh
      theme: robbyrussell
      plugins:
        - git
        - docker
        - kubectl
  starship:
    enabled: true
    preset: plain-text
  env:
    EDITOR: nvim
  aliases:
    ll: "ls -la"
`
	layer, err := config.ParseLayer([]byte(layerYAML))
	require.NoError(t, err)
	require.NotNil(t, layer)

	assert.Equal(t, "zsh", layer.Shell.Default)
	require.Len(t, layer.Shell.Shells, 1)
	assert.Equal(t, "zsh", layer.Shell.Shells[0].Name)
	assert.Equal(t, "oh-my-zsh", layer.Shell.Shells[0].Framework)
	assert.Contains(t, layer.Shell.Shells[0].Plugins, "git")
	assert.True(t, layer.Shell.Starship.Enabled)
	assert.Equal(t, "nvim", layer.Shell.Env["EDITOR"])
	assert.Equal(t, "ls -la", layer.Shell.Aliases["ll"])
}

func TestInit_LayerParsing_Packages(t *testing.T) {
	t.Parallel()

	layerYAML := `
name: base
packages:
  brew:
    taps:
      - homebrew/cask-fonts
    formulae:
      - ripgrep
      - fzf
      - bat
    casks:
      - visual-studio-code
      - docker
`
	layer, err := config.ParseLayer([]byte(layerYAML))
	require.NoError(t, err)
	require.NotNil(t, layer)

	assert.Contains(t, layer.Packages.Brew.Taps, "homebrew/cask-fonts")
	assert.Contains(t, layer.Packages.Brew.Formulae, "ripgrep")
	assert.Contains(t, layer.Packages.Brew.Casks, "docker")
}

func TestInit_CatalogHasPresets(t *testing.T) {
	t.Parallel()

	cat, err := embedded.LoadCatalog()
	require.NoError(t, err)

	presets := cat.ListPresets()
	assert.NotEmpty(t, presets, "catalog should have presets")

	// Each preset should have required fields
	for _, p := range presets {
		assert.NotEmpty(t, p.Metadata().Title(), "preset should have title")
		assert.NotEmpty(t, p.Metadata().Description(), "preset should have description")
	}
}

func TestInit_CatalogHasCapabilityPacks(t *testing.T) {
	t.Parallel()

	cat, err := embedded.LoadCatalog()
	require.NoError(t, err)

	packs := cat.ListPacks()
	assert.NotEmpty(t, packs, "catalog should have capability packs")

	// Each pack should have required fields
	for _, p := range packs {
		assert.NotEmpty(t, p.ID(), "pack should have ID")
		assert.NotEmpty(t, p.Metadata().Title(), "pack should have title")
		assert.NotEmpty(t, p.Metadata().Description(), "pack should have description")
	}
}

func TestInit_WritesToDisk(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create directory structure
	layersDir := filepath.Join(tempDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	// Write manifest
	manifest := `
targets:
  default:
    - base
`
	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0o644))

	// Write layer
	layer := `
name: base
git:
  user:
    name: "Test User"
`
	layerPath := filepath.Join(layersDir, "base.yaml")
	require.NoError(t, os.WriteFile(layerPath, []byte(layer), 0o644))

	// Verify files exist
	_, err := os.Stat(manifestPath)
	require.NoError(t, err)

	_, err = os.Stat(layerPath)
	require.NoError(t, err)

	// Verify manifest can be parsed
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	parsed, err := config.ParseManifest(data)
	require.NoError(t, err)
	assert.Len(t, parsed.Targets, 1)
}

func TestInit_ManifestWithDefaults(t *testing.T) {
	t.Parallel()

	manifestYAML := `
defaults:
  mode: locked
  editor: nvim
targets:
  default:
    - base
`
	manifest, err := config.ParseManifest([]byte(manifestYAML))
	require.NoError(t, err)

	assert.Equal(t, config.ModeLocked, manifest.Defaults.Mode)
	assert.Equal(t, "nvim", manifest.Defaults.Editor)
}

func TestInit_ManifestValidation_RequiresTargets(t *testing.T) {
	t.Parallel()

	// Empty manifest should fail
	manifestYAML := `
defaults:
  mode: locked
`
	_, err := config.ParseManifest([]byte(manifestYAML))
	assert.Error(t, err)
	assert.ErrorIs(t, err, config.ErrNoTargets)
}

func TestInit_LayerWithSSH(t *testing.T) {
	t.Parallel()

	layerYAML := `
name: base
ssh:
  defaults:
    addkeystoagent: true
    identitiesonly: true
  hosts:
    - host: github.com
      hostname: github.com
      user: git
      identityfile: ~/.ssh/id_github
`
	layer, err := config.ParseLayer([]byte(layerYAML))
	require.NoError(t, err)
	require.NotNil(t, layer)

	assert.True(t, layer.SSH.Defaults.AddKeysToAgent)
	assert.True(t, layer.SSH.Defaults.IdentitiesOnly)
	require.Len(t, layer.SSH.Hosts, 1)
	assert.Equal(t, "github.com", layer.SSH.Hosts[0].Host)
}

func TestInit_LayerWithNvim(t *testing.T) {
	t.Parallel()

	layerYAML := `
name: role.go
nvim:
  preset: kickstart
  plugin_manager: lazy
  ensure_install: true
`
	layer, err := config.ParseLayer([]byte(layerYAML))
	require.NoError(t, err)
	require.NotNil(t, layer)

	assert.Equal(t, "kickstart", layer.Nvim.Preset)
	assert.Equal(t, "lazy", layer.Nvim.PluginManager)
	assert.True(t, layer.Nvim.EnsureInstall)
}
