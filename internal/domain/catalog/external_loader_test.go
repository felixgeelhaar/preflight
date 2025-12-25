package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalLoader_Load(t *testing.T) {
	t.Parallel()

	// Create test catalog content
	catalogContent := `
presets:
  - id: "nvim:test"
    metadata:
      title: "Test Preset"
      description: "A test preset"
    difficulty: beginner
    config:
      plugins: ["test"]

capability_packs:
  - id: "test-pack"
    metadata:
      title: "Test Pack"
      description: "A test pack"
    presets: ["nvim:test"]
`
	catalogHash := ComputeSHA256([]byte(catalogContent))

	manifestContent := `
version: "1.0"
name: "test-catalog"
description: "Test catalog"
author: "Test Author"
repository: "https://github.com/test/catalog"
license: "MIT"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: ` + catalogHash + `
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog-manifest.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write([]byte(manifestContent))
		case "/catalog.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write([]byte(catalogContent))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create loader with temp cache dir
	tmpDir := t.TempDir()
	config := ExternalLoaderConfig{
		Timeout:  DefaultExternalLoaderConfig().Timeout,
		CacheDir: tmpDir,
	}
	loader := NewExternalLoader(config)

	// Create source
	source, err := NewURLSource("test-catalog", server.URL)
	require.NoError(t, err)

	// Load catalog
	rc, err := loader.Load(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "test-catalog", rc.Name())
	assert.Equal(t, "Test catalog", rc.Manifest().Description())
	assert.Equal(t, 1, rc.Catalog().PresetCount())
	assert.Equal(t, 1, rc.Catalog().PackCount())
}

func TestExternalLoader_LoadFromCache(t *testing.T) {
	t.Parallel()

	// Create test files in temp directory
	tmpDir := t.TempDir()
	catalogDir := filepath.Join(tmpDir, "test-catalog")
	require.NoError(t, os.MkdirAll(catalogDir, 0o755))

	// Create catalog content
	catalogContent := `
presets:
  - id: "nvim:cached"
    metadata:
      title: "Cached Preset"
      description: "A cached preset"
    difficulty: beginner
`
	catalogHash := ComputeSHA256([]byte(catalogContent))

	manifestContent := `
version: "1.0"
name: "test-catalog"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: ` + catalogHash + `
`

	require.NoError(t, os.WriteFile(
		filepath.Join(catalogDir, "catalog-manifest.yaml"),
		[]byte(manifestContent),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(catalogDir, "catalog.yaml"),
		[]byte(catalogContent),
		0o644,
	))

	// Create loader
	config := ExternalLoaderConfig{
		Timeout:  DefaultExternalLoaderConfig().Timeout,
		CacheDir: tmpDir,
	}
	loader := NewExternalLoader(config)

	// Create source
	source, err := NewLocalSource("test-catalog", "/unused/path")
	require.NoError(t, err)

	// Load from cache
	rc, err := loader.LoadFromCache(source)
	require.NoError(t, err)

	assert.Equal(t, "test-catalog", rc.Manifest().Name())
	assert.Equal(t, 1, rc.Catalog().PresetCount())
}

func TestExternalLoader_LoadFromCache_IntegrityMismatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	catalogDir := filepath.Join(tmpDir, "test-catalog")
	require.NoError(t, os.MkdirAll(catalogDir, 0o755))

	// Create catalog with wrong hash
	catalogContent := `
presets:
  - id: "nvim:test"
    metadata:
      title: "Test"
      description: "Test"
    difficulty: beginner
`

	manifestContent := `
version: "1.0"
name: "test-catalog"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: sha256:0000000000000000000000000000000000000000000000000000000000000000
`

	require.NoError(t, os.WriteFile(
		filepath.Join(catalogDir, "catalog-manifest.yaml"),
		[]byte(manifestContent),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(catalogDir, "catalog.yaml"),
		[]byte(catalogContent),
		0o644,
	))

	config := ExternalLoaderConfig{
		Timeout:  DefaultExternalLoaderConfig().Timeout,
		CacheDir: tmpDir,
	}
	loader := NewExternalLoader(config)

	source, _ := NewLocalSource("test-catalog", "/unused")
	_, err := loader.LoadFromCache(source)
	assert.ErrorIs(t, err, ErrIntegrityMismatch)
}

func TestExternalLoader_Verify(t *testing.T) {
	t.Parallel()

	catalogContent := `
presets: []
`
	catalogHash := ComputeSHA256([]byte(catalogContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte(catalogContent))
	}))
	defer server.Close()

	config := DefaultExternalLoaderConfig()
	config.CacheDir = t.TempDir()
	loader := NewExternalLoader(config)

	source, _ := NewURLSource("test", server.URL)
	manifest, _ := NewManifestBuilder("test").
		AddFile("catalog.yaml", catalogHash).
		Build()
	rc := NewRegisteredCatalog(source, manifest, NewCatalog())

	err := loader.Verify(context.Background(), rc)
	assert.NoError(t, err)
}

func TestExternalLoader_ClearCache(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	catalogDir := filepath.Join(tmpDir, "test-catalog")
	require.NoError(t, os.MkdirAll(catalogDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(catalogDir, "test.txt"),
		[]byte("test"),
		0o644,
	))

	config := ExternalLoaderConfig{
		Timeout:  DefaultExternalLoaderConfig().Timeout,
		CacheDir: tmpDir,
	}
	loader := NewExternalLoader(config)

	source, _ := NewLocalSource("test-catalog", "/unused")
	err := loader.ClearCache(source)
	require.NoError(t, err)

	_, err = os.Stat(catalogDir)
	assert.True(t, os.IsNotExist(err))
}

func TestExternalLoader_Load_BuiltinFails(t *testing.T) {
	t.Parallel()

	loader := NewExternalLoader(DefaultExternalLoaderConfig())
	source := NewBuiltinSource()

	_, err := loader.Load(context.Background(), source)
	assert.ErrorIs(t, err, ErrInvalidSource)
}

func TestExternalLoader_Load_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := DefaultExternalLoaderConfig()
	config.CacheDir = t.TempDir()
	loader := NewExternalLoader(config)

	source, _ := NewURLSource("test", server.URL)
	_, err := loader.Load(context.Background(), source)
	assert.ErrorIs(t, err, ErrSourceNotFound)
}

func TestExternalLoader_LocalSource(t *testing.T) {
	t.Parallel()

	// Create test catalog in temp directory
	tmpDir := t.TempDir()

	catalogContent := `
presets:
  - id: "nvim:local"
    metadata:
      title: "Local Preset"
      description: "A local preset"
    difficulty: beginner
`
	catalogHash := ComputeSHA256([]byte(catalogContent))

	manifestContent := `
version: "1.0"
name: "local-catalog"
integrity:
  algorithm: sha256
  files:
    catalog.yaml: ` + catalogHash + `
`

	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "catalog-manifest.yaml"),
		[]byte(manifestContent),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "catalog.yaml"),
		[]byte(catalogContent),
		0o644,
	))

	config := ExternalLoaderConfig{
		Timeout:  DefaultExternalLoaderConfig().Timeout,
		CacheDir: t.TempDir(),
	}
	loader := NewExternalLoader(config)

	source, err := NewLocalSource("local-catalog", tmpDir)
	require.NoError(t, err)

	rc, err := loader.Load(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "local-catalog", rc.Manifest().Name())
	assert.Equal(t, 1, rc.Catalog().PresetCount())
}
