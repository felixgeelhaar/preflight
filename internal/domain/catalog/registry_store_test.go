package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryStore_LoadEmpty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	sources, err := store.Load()
	require.NoError(t, err)
	assert.Empty(t, sources)
}

func TestRegistryStore_AddAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	sources, err := store.Load()
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "my-catalog", sources[0].Name)
	assert.Equal(t, SourceTypeURL, sources[0].Type)
	assert.Equal(t, "https://example.com/catalog", sources[0].Location)
	assert.True(t, sources[0].Enabled)
}

func TestRegistryStore_AddDuplicate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	err = store.Add(src)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRegistryStore_Remove(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	err = store.Remove("my-catalog")
	require.NoError(t, err)

	sources, err := store.Load()
	require.NoError(t, err)
	assert.Empty(t, sources)
}

func TestRegistryStore_RemoveNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	err := store.Remove("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistryStore_Get(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	stored, found, err := store.Get("my-catalog")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "my-catalog", stored.Name)
}

func TestRegistryStore_GetNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	_, found, err := store.Get("nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRegistryStore_SetEnabled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	err = store.SetEnabled("my-catalog", false)
	require.NoError(t, err)

	stored, found, err := store.Get("my-catalog")
	require.NoError(t, err)
	assert.True(t, found)
	assert.False(t, stored.Enabled)
}

func TestRegistryStore_SetTrusted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	err = store.SetTrusted("my-catalog", true)
	require.NoError(t, err)

	stored, found, err := store.Get("my-catalog")
	require.NoError(t, err)
	assert.True(t, found)
	assert.True(t, stored.Trusted)
}

func TestRegistryStore_UpdateVerifyTime(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src, err := NewURLSource("my-catalog", "https://example.com/catalog")
	require.NoError(t, err)

	err = store.Add(src)
	require.NoError(t, err)

	err = store.UpdateVerifyTime("my-catalog")
	require.NoError(t, err)

	stored, found, err := store.Get("my-catalog")
	require.NoError(t, err)
	assert.True(t, found)
	assert.False(t, stored.LastVerify.IsZero())
}

func TestRegistryStore_CorruptedFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	// Write corrupted JSON
	err := os.MkdirAll(tmpDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "registry.json"), []byte("not valid json"), 0o644)
	require.NoError(t, err)

	_, err = store.Load()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrStoreCorrupted)
}

func TestStoredSource_ToSource_URL(t *testing.T) {
	t.Parallel()

	stored := StoredSource{
		Name:     "test",
		Type:     SourceTypeURL,
		Location: "https://example.com/catalog",
		Enabled:  true,
	}

	src, err := stored.ToSource()
	require.NoError(t, err)
	assert.Equal(t, "test", src.Name())
	assert.Equal(t, SourceTypeURL, src.Type())
	assert.Equal(t, "https://example.com/catalog", src.Location())
}

func TestStoredSource_ToSource_Local(t *testing.T) {
	t.Parallel()

	stored := StoredSource{
		Name:     "test",
		Type:     SourceTypeLocal,
		Location: "/path/to/catalog",
		Enabled:  true,
	}

	src, err := stored.ToSource()
	require.NoError(t, err)
	assert.Equal(t, "test", src.Name())
	assert.Equal(t, SourceTypeLocal, src.Type())
	assert.Equal(t, "/path/to/catalog", src.Location())
}

func TestStoredSource_ToSource_Unknown(t *testing.T) {
	t.Parallel()

	stored := StoredSource{
		Name:     "test",
		Type:     "unknown",
		Location: "somewhere",
		Enabled:  true,
	}

	_, err := stored.ToSource()
	assert.Error(t, err)
}

func TestDefaultRegistryStoreConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRegistryStoreConfig()
	assert.Contains(t, config.BasePath, ".preflight")
	assert.Contains(t, config.BasePath, "catalogs")
}

func TestRegistryStore_MultipleSources(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store := NewRegistryStore(RegistryStoreConfig{BasePath: tmpDir})

	src1, err := NewURLSource("catalog-1", "https://example.com/catalog1")
	require.NoError(t, err)

	src2, err := NewLocalSource("catalog-2", "/path/to/catalog2")
	require.NoError(t, err)

	err = store.Add(src1)
	require.NoError(t, err)

	err = store.Add(src2)
	require.NoError(t, err)

	sources, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, sources, 2)

	// Remove one
	err = store.Remove("catalog-1")
	require.NoError(t, err)

	sources, err = store.Load()
	require.NoError(t, err)
	assert.Len(t, sources, 1)
	assert.Equal(t, "catalog-2", sources[0].Name)
}
