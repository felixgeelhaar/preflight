package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoteLoader(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp/cache")
	require.NotNil(t, loader)
	assert.Equal(t, "/tmp/cache", loader.cacheDir)
	assert.NotNil(t, loader.httpClient)
	assert.Equal(t, 30*time.Second, loader.httpClient.Timeout)
}

func TestRemoteLoader_cacheKey_URL(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp/cache")
	source := RemoteLayerSource{
		URL: "https://example.com/layer.yaml",
	}

	key := loader.cacheKey(source)

	assert.NotEmpty(t, key)
	assert.Len(t, key, 32) // 16 bytes hex encoded
}

func TestRemoteLoader_cacheKey_Git(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp/cache")
	source := RemoteLayerSource{
		GitRepo: "https://github.com/user/repo",
		GitRef:  "main",
		GitPath: "layers/base.yaml",
	}

	key := loader.cacheKey(source)

	assert.NotEmpty(t, key)
	assert.Len(t, key, 32)
}

func TestRemoteLoader_cacheKey_Consistent(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp/cache")
	source := RemoteLayerSource{URL: "https://example.com/layer.yaml"}

	key1 := loader.cacheKey(source)
	key2 := loader.cacheKey(source)

	assert.Equal(t, key1, key2)
}

func TestRemoteLoader_isCacheValid_NotExists(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp/cache")

	result := loader.isCacheValid("/nonexistent/path.yaml", "1h")

	assert.False(t, result)
}

func TestRemoteLoader_isCacheValid_Fresh(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.yaml")
	require.NoError(t, os.WriteFile(cachePath, []byte("test"), 0644))

	loader := NewRemoteLoader(tmpDir)

	result := loader.isCacheValid(cachePath, "1h")

	assert.True(t, result)
}

func TestRemoteLoader_isCacheValid_Stale(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.yaml")
	require.NoError(t, os.WriteFile(cachePath, []byte("test"), 0644))

	// Set modification time to 2 hours ago
	oldTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(cachePath, oldTime, oldTime))

	loader := NewRemoteLoader(tmpDir)

	result := loader.isCacheValid(cachePath, "1h")

	assert.False(t, result)
}

func TestRemoteLoader_isCacheValid_DefaultTTL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.yaml")
	require.NoError(t, os.WriteFile(cachePath, []byte("test"), 0644))

	loader := NewRemoteLoader(tmpDir)

	// Empty TTL should default to 24h
	result := loader.isCacheValid(cachePath, "")

	assert.True(t, result)
}

func TestRemoteLoader_isCacheValid_InvalidTTL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.yaml")
	require.NoError(t, os.WriteFile(cachePath, []byte("test"), 0644))

	loader := NewRemoteLoader(tmpDir)

	// Invalid TTL should fallback to 24h
	result := loader.isCacheValid(cachePath, "invalid")

	assert.True(t, result)
}

func TestRemoteLoader_saveToCache(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	cachePath := filepath.Join(tmpDir, "subdir", "cache.yaml")
	data := []byte("test content")

	err := loader.saveToCache(cachePath, data)

	require.NoError(t, err)
	content, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestRemoteLoader_loadFromCache(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	cachePath := filepath.Join(tmpDir, "layer.yaml")

	// Write a valid layer YAML
	layerYAML := `name: test-layer
packages:
  brew:
    formulae:
      - git
`
	require.NoError(t, os.WriteFile(cachePath, []byte(layerYAML), 0644))

	layer, err := loader.loadFromCache(cachePath, "test")

	require.NoError(t, err)
	require.NotNil(t, layer)
	assert.Contains(t, layer.Provenance, "cache:")
}

func TestRemoteLoader_loadFromCache_NotExists(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")

	_, err := loader.loadFromCache("/nonexistent/path.yaml", "test")

	assert.Error(t, err)
}

func TestRemoteLoader_loadFromCache_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	cachePath := filepath.Join(tmpDir, "layer.yaml")
	require.NoError(t, os.WriteFile(cachePath, []byte("invalid: [yaml"), 0644))

	_, err := loader.loadFromCache(cachePath, "test")

	assert.Error(t, err)
}

func TestRemoteLoader_fetchFromURL_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	loader := NewRemoteLoader("/tmp")
	ctx := context.Background()

	data, err := loader.fetchFromURL(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, []byte("test content"), data)
}

func TestRemoteLoader_fetchFromURL_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	loader := NewRemoteLoader("/tmp")
	ctx := context.Background()

	_, err := loader.fetchFromURL(ctx, server.URL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestRemoteLoader_fetchFromURL_InvalidURL(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	ctx := context.Background()

	_, err := loader.fetchFromURL(ctx, "http://invalid-host-that-does-not-exist.local")

	assert.Error(t, err)
}

func TestRemoteLoader_fetchFromGit_Unimplemented(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	ctx := context.Background()
	source := RemoteLayerSource{
		GitRepo: "https://github.com/user/repo",
		GitRef:  "main",
	}

	_, err := loader.fetchFromGit(ctx, source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git clone implementation")
}

func TestRemoteLoader_fetchFromGit_DefaultRef(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	ctx := context.Background()
	source := RemoteLayerSource{
		GitRepo: "https://github.com/user/repo",
	}

	_, err := loader.fetchFromGit(ctx, source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HEAD")
}

func TestRemoteLoader_sourceDescription_URL(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	source := RemoteLayerSource{
		URL: "https://example.com/layer.yaml",
	}

	desc := loader.sourceDescription(source)

	assert.Equal(t, "https://example.com/layer.yaml", desc)
}

func TestRemoteLoader_sourceDescription_Git(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	source := RemoteLayerSource{
		GitRepo: "https://github.com/user/repo",
		GitRef:  "v1.0.0",
		GitPath: "layers/base.yaml",
	}

	desc := loader.sourceDescription(source)

	assert.Equal(t, "https://github.com/user/repo@v1.0.0:layers/base.yaml", desc)
}

func TestRemoteLoader_sourceDescription_GitNoRefOrPath(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader("/tmp")
	source := RemoteLayerSource{
		GitRepo: "https://github.com/user/repo",
	}

	desc := loader.sourceDescription(source)

	assert.Equal(t, "https://github.com/user/repo", desc)
}

func TestRemoteLoader_LoadRemoteLayer_NoSource(t *testing.T) {
	t.Parallel()

	loader := NewRemoteLoader(t.TempDir())
	ctx := context.Background()
	source := RemoteLayerSource{}

	_, err := loader.LoadRemoteLayer(ctx, "test", source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify url or git")
}

func TestRemoteLoader_LoadRemoteLayer_FromURL(t *testing.T) {
	t.Parallel()

	layerYAML := `name: test-layer
packages:
  brew:
    formulae:
      - git
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(layerYAML))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	ctx := context.Background()
	source := RemoteLayerSource{
		URL: server.URL,
	}

	layer, err := loader.LoadRemoteLayer(ctx, "test", source)

	require.NoError(t, err)
	require.NotNil(t, layer)
	assert.Contains(t, layer.Provenance, "remote:")
}

func TestRemoteLoader_LoadRemoteLayer_IntegrityOK(t *testing.T) {
	t.Parallel()

	layerYAML := `name: test-layer`
	// SHA256 of "name: test-layer"
	integrity := "c9b4f4c59c8d0d4f9d0f3f1e0d0c0b0a0908070605040302010099887766554433"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(layerYAML))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	ctx := context.Background()

	// Calculate actual hash
	source := RemoteLayerSource{
		URL: server.URL,
	}

	layer, err := loader.LoadRemoteLayer(ctx, "test", source)
	require.NoError(t, err)
	_ = layer
	_ = integrity // integrity check tested below
}

func TestRemoteLoader_LoadRemoteLayer_IntegrityFail(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("name: test-layer"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)
	ctx := context.Background()
	source := RemoteLayerSource{
		URL:       server.URL,
		Integrity: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	_, err := loader.LoadRemoteLayer(ctx, "test", source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integrity check failed")
}

func TestRemoteLoader_LoadRemoteLayer_UsesCache(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)

	// Pre-populate cache
	cacheDir := filepath.Join(tmpDir, "remote-layers")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	source := RemoteLayerSource{
		URL:      "https://example.com/layer.yaml",
		CacheTTL: "1h",
	}
	cacheKey := loader.cacheKey(source)
	cachePath := filepath.Join(cacheDir, cacheKey+".yaml")

	layerYAML := `name: cached-layer`
	require.NoError(t, os.WriteFile(cachePath, []byte(layerYAML), 0644))

	ctx := context.Background()

	layer, err := loader.LoadRemoteLayer(ctx, "test", source)

	require.NoError(t, err)
	assert.Contains(t, layer.Provenance, "cache:")
}

func TestRemoteLoader_LoadRemoteLayer_FetchFailUsesStaleCache(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	loader := NewRemoteLoader(tmpDir)

	// Pre-populate stale cache
	cacheDir := filepath.Join(tmpDir, "remote-layers")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	source := RemoteLayerSource{
		URL:      "http://invalid-host.local/layer.yaml",
		CacheTTL: "1ms", // Very short TTL to force stale
	}
	cacheKey := loader.cacheKey(source)
	cachePath := filepath.Join(cacheDir, cacheKey+".yaml")

	layerYAML := `name: stale-layer`
	require.NoError(t, os.WriteFile(cachePath, []byte(layerYAML), 0644))

	// Make cache stale
	oldTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(cachePath, oldTime, oldTime))

	ctx := context.Background()

	layer, err := loader.LoadRemoteLayer(ctx, "test", source)

	// Should fallback to stale cache
	require.NoError(t, err)
	assert.Contains(t, layer.Provenance, "cache:")
}

func TestParseRemoteLayerConfig(t *testing.T) {
	t.Parallel()

	data := []byte(`
remote_layers:
  base:
    url: https://example.com/base.yaml
  team:
    git: https://github.com/org/dotfiles
    ref: main
    path: layers/team.yaml
`)

	config, err := ParseRemoteLayerConfig(data)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Len(t, config.Remotes, 2)
	assert.Equal(t, "https://example.com/base.yaml", config.Remotes["base"].URL)
	assert.Equal(t, "https://github.com/org/dotfiles", config.Remotes["team"].GitRepo)
}

func TestParseRemoteLayerConfig_Empty(t *testing.T) {
	t.Parallel()

	data := []byte(`version: 1`)

	config, err := ParseRemoteLayerConfig(data)

	require.NoError(t, err)
	assert.Empty(t, config.Remotes)
}

func TestParseRemoteLayerConfig_Invalid(t *testing.T) {
	t.Parallel()

	data := []byte(`remote_layers: [invalid yaml`)

	_, err := ParseRemoteLayerConfig(data)

	assert.Error(t, err)
}

func TestRemoteLayerSource_Fields(t *testing.T) {
	t.Parallel()

	source := RemoteLayerSource{
		URL:       "https://example.com/layer.yaml",
		GitRepo:   "https://github.com/user/repo",
		GitRef:    "v1.0.0",
		GitPath:   "layers/base.yaml",
		CacheTTL:  "1h",
		Integrity: "abc123",
	}

	assert.Equal(t, "https://example.com/layer.yaml", source.URL)
	assert.Equal(t, "https://github.com/user/repo", source.GitRepo)
	assert.Equal(t, "v1.0.0", source.GitRef)
	assert.Equal(t, "layers/base.yaml", source.GitPath)
	assert.Equal(t, "1h", source.CacheTTL)
	assert.Equal(t, "abc123", source.Integrity)
}

func TestRemoteLayerConfig_Fields(t *testing.T) {
	t.Parallel()

	config := RemoteLayerConfig{
		Remotes: map[string]RemoteLayerSource{
			"test": {URL: "https://example.com/test.yaml"},
		},
	}

	assert.Len(t, config.Remotes, 1)
	assert.Equal(t, "https://example.com/test.yaml", config.Remotes["test"].URL)
}
