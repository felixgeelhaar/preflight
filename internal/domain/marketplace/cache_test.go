package marketplace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	t.Parallel()

	config := DefaultCacheConfig()
	cache := NewCache(config)
	assert.NotNil(t, cache)
}

func TestDefaultCacheConfig(t *testing.T) {
	t.Parallel()

	config := DefaultCacheConfig()
	assert.Equal(t, 1*time.Hour, config.IndexTTL)
	assert.Equal(t, 24*time.Hour, config.PackageTTL)
	assert.NotEmpty(t, config.BasePath)
}

func TestCache_IndexOperations(t *testing.T) {
	t.Parallel()

	// Create temp dir for cache
	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath: tmpDir,
		IndexTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// Initially no index
	_, err := cache.GetIndex()
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Put an index
	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("test-pkg"),
		Type:     PackageTypePreset,
		Title:    "Test Package",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	err = cache.PutIndex(idx, "https://registry.test.dev")
	require.NoError(t, err)

	// Now we can get it
	retrieved, err := cache.GetIndex()
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.Count())

	pkg, ok := retrieved.Get(MustNewPackageID("test-pkg"))
	assert.True(t, ok)
	assert.Equal(t, "Test Package", pkg.Title)
}

func TestCache_IndexExpired(t *testing.T) {
	t.Parallel()

	// Create temp dir for cache
	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath: tmpDir,
		IndexTTL: 1 * time.Millisecond, // Very short TTL
	}
	cache := NewCache(config)

	// Put an index
	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("test-pkg"),
		Type:     PackageTypePreset,
		Title:    "Test Package",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	err := cache.PutIndex(idx, "https://registry.test.dev")
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Now it should be expired
	_, err = cache.GetIndex()
	assert.ErrorIs(t, err, ErrCacheExpired)
}

func TestCache_PackageOperations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath:   tmpDir,
		IndexTTL:   1 * time.Hour,
		PackageTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	id := MustNewPackageID("my-package")
	version := "1.0.0"
	data := []byte("package content here")

	// Initially not found
	_, err := cache.GetPackage(id, version)
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Put package
	err = cache.PutPackage(id, version, data, "https://registry.test.dev")
	require.NoError(t, err)

	// Now we can get it
	retrieved, err := cache.GetPackage(id, version)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestCache_InstalledPackages(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath: tmpDir,
		IndexTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// Initially empty
	installed, err := cache.GetInstalled()
	require.NoError(t, err)
	assert.Empty(t, installed)

	// Add installed package
	pkg := InstalledPackage{
		Package: Package{
			ID:       MustNewPackageID("installed-pkg"),
			Type:     PackageTypePreset,
			Title:    "Installed Package",
			Versions: []PackageVersion{{Version: "1.0.0"}},
		},
		Version:     "1.0.0",
		InstalledAt: time.Now(),
		Path:        "/path/to/package",
	}

	err = cache.AddInstalled(pkg)
	require.NoError(t, err)

	// Now we can get it
	installed, err = cache.GetInstalled()
	require.NoError(t, err)
	require.Len(t, installed, 1)
	assert.Equal(t, "installed-pkg", installed[0].Package.ID.String())

	// Get specific package
	retrieved, found, err := cache.GetInstalledPackage(MustNewPackageID("installed-pkg"))
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1.0.0", retrieved.Version)

	// Not found
	_, found, err = cache.GetInstalledPackage(MustNewPackageID("not-installed"))
	require.NoError(t, err)
	assert.False(t, found)

	// Remove installed
	err = cache.RemoveInstalled(MustNewPackageID("installed-pkg"))
	require.NoError(t, err)

	installed, err = cache.GetInstalled()
	require.NoError(t, err)
	assert.Empty(t, installed)
}

func TestCache_UpdateInstalled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath: tmpDir,
		IndexTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// Add installed package
	pkg := InstalledPackage{
		Package: Package{
			ID:       MustNewPackageID("update-pkg"),
			Type:     PackageTypePreset,
			Title:    "Update Package",
			Versions: []PackageVersion{{Version: "1.0.0"}},
		},
		Version:     "1.0.0",
		InstalledAt: time.Now(),
		Path:        "/path/to/v1",
	}

	err := cache.AddInstalled(pkg)
	require.NoError(t, err)

	// Update to new version
	pkg.Version = "2.0.0"
	pkg.Path = "/path/to/v2"

	err = cache.AddInstalled(pkg)
	require.NoError(t, err)

	// Should still have only one entry
	installed, err := cache.GetInstalled()
	require.NoError(t, err)
	require.Len(t, installed, 1)
	assert.Equal(t, "2.0.0", installed[0].Version)
	assert.Equal(t, "/path/to/v2", installed[0].Path)
}

func TestCache_Clear(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath:   tmpDir,
		IndexTTL:   1 * time.Hour,
		PackageTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// Add some data
	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("test"),
		Type:     PackageTypePreset,
		Title:    "Test",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})
	_ = cache.PutIndex(idx, "https://registry.test.dev")
	_ = cache.PutPackage(MustNewPackageID("pkg"), "1.0.0", []byte("data"), "https://registry.test.dev")

	// Clear
	err := cache.Clear()
	require.NoError(t, err)

	// Now everything should be gone
	_, err = cache.GetIndex()
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestCache_ClearIndex(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := CacheConfig{
		BasePath:   tmpDir,
		IndexTTL:   1 * time.Hour,
		PackageTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// Add index and package
	idx := NewIndex()
	_ = cache.PutIndex(idx, "https://registry.test.dev")
	_ = cache.PutPackage(MustNewPackageID("pkg"), "1.0.0", []byte("data"), "https://registry.test.dev")

	// Clear only index
	err := cache.ClearIndex()
	require.NoError(t, err)

	// Index should be gone
	_, err = cache.GetIndex()
	assert.ErrorIs(t, err, ErrCacheMiss)

	// Package should still be there
	data, err := cache.GetPackage(MustNewPackageID("pkg"), "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, []byte("data"), data)
}

func TestCache_EnsureDir(t *testing.T) {
	t.Parallel()

	// Use a nested non-existent path
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "deeply", "nested", "cache")
	config := CacheConfig{
		BasePath: nestedDir,
		IndexTTL: 1 * time.Hour,
	}
	cache := NewCache(config)

	// EnsureDir should create directories
	err := cache.EnsureDir()
	require.NoError(t, err)

	// Directory should exist
	_, err = os.Stat(nestedDir)
	assert.NoError(t, err)
}

func TestCacheMeta_IsExpired(t *testing.T) {
	t.Parallel()

	// Not expired
	meta := CacheMeta{
		CachedAt: time.Now(),
	}
	assert.False(t, meta.IsExpired(1*time.Hour))

	// Expired
	meta.CachedAt = time.Now().Add(-2 * time.Hour)
	assert.True(t, meta.IsExpired(1*time.Hour))
}
