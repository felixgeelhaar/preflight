package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_LoadManifest_LoadsFromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	err := os.WriteFile(manifestPath, []byte(`
targets:
  work:
    - base
    - identity.work
`), 0o644)
	require.NoError(t, err)

	loader := config.NewLoader()
	manifest, err := loader.LoadManifest(manifestPath)

	require.NoError(t, err)
	require.Len(t, manifest.Targets, 1)
	assert.Len(t, manifest.Targets["work"], 2)
}

func TestLoader_LoadManifest_FileNotFound_ReturnsError(t *testing.T) {
	t.Parallel()

	loader := config.NewLoader()
	_, err := loader.LoadManifest("/nonexistent/path/preflight.yaml")

	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLoader_LoadLayer_LoadsFromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	layerPath := filepath.Join(tempDir, "base.yaml")
	err := os.WriteFile(layerPath, []byte(`
name: base
packages:
  brew:
    formulae:
      - git
`), 0o644)
	require.NoError(t, err)

	loader := config.NewLoader()
	layer, err := loader.LoadLayer(layerPath)

	require.NoError(t, err)
	assert.Equal(t, "base", layer.Name.String())
	assert.Equal(t, layerPath, layer.Provenance)
	assert.Contains(t, layer.Packages.Brew.Formulae, "git")
}

func TestLoader_LoadLayer_FileNotFound_ReturnsError(t *testing.T) {
	t.Parallel()

	loader := config.NewLoader()
	_, err := loader.LoadLayer("/nonexistent/path/base.yaml")

	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestLoader_LoadTarget_ResolvesLayers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	layersDir := filepath.Join(tempDir, "layers")
	err := os.MkdirAll(layersDir, 0o755)
	require.NoError(t, err)

	// Create manifest
	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	err = os.WriteFile(manifestPath, []byte(`
targets:
  work:
    - base
    - identity.work
`), 0o644)
	require.NoError(t, err)

	// Create layers
	err = os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(`
name: base
packages:
  brew:
    formulae:
      - git
`), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(layersDir, "identity.work.yaml"), []byte(`
name: identity.work
files:
  - path: ~/.gitconfig
    mode: generated
`), 0o644)
	require.NoError(t, err)

	loader := config.NewLoader()
	manifest, err := loader.LoadManifest(manifestPath)
	require.NoError(t, err)

	targetName, _ := config.NewTargetName("work")
	target, err := loader.LoadTarget(manifest, targetName, layersDir)

	require.NoError(t, err)
	assert.Equal(t, "work", target.Name.String())
	require.Len(t, target.Layers, 2)
	assert.Equal(t, "base", target.Layers[0].Name.String())
	assert.Equal(t, "identity.work", target.Layers[1].Name.String())
}

func TestLoader_LoadTarget_MissingLayer_ReturnsError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	layersDir := filepath.Join(tempDir, "layers")
	err := os.MkdirAll(layersDir, 0o755)
	require.NoError(t, err)

	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	err = os.WriteFile(manifestPath, []byte(`
targets:
  work:
    - base
    - nonexistent
`), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(`
name: base
`), 0o644)
	require.NoError(t, err)

	loader := config.NewLoader()
	manifest, err := loader.LoadManifest(manifestPath)
	require.NoError(t, err)

	targetName, _ := config.NewTargetName("work")
	_, err = loader.LoadTarget(manifest, targetName, layersDir)

	require.Error(t, err)
}
