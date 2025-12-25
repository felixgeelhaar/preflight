package catalog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Add(t *testing.T) {
	t.Parallel()

	t.Run("add catalog", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		src := NewBuiltinSource()
		manifest, _ := NewManifestBuilder("builtin").Build()
		rc := NewRegisteredCatalog(src, manifest, NewCatalog())

		err := r.Add(rc)
		require.NoError(t, err)
		assert.Equal(t, 1, r.Count())
	})

	t.Run("duplicate fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		src := NewBuiltinSource()
		manifest, _ := NewManifestBuilder("builtin").Build()
		rc := NewRegisteredCatalog(src, manifest, NewCatalog())

		_ = r.Add(rc)
		err := r.Add(rc)
		assert.ErrorIs(t, err, ErrCatalogExists)
	})

	t.Run("nil fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Add(nil)
		assert.Error(t, err)
	})
}

func TestRegistry_Remove(t *testing.T) {
	t.Parallel()

	t.Run("remove external catalog", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		src, _ := NewURLSource("test", "https://example.com")
		manifest, _ := NewManifestBuilder("test").Build()
		rc := NewRegisteredCatalog(src, manifest, NewCatalog())

		_ = r.Add(rc)
		err := r.Remove("test")
		require.NoError(t, err)
		assert.Equal(t, 0, r.Count())
	})

	t.Run("remove builtin fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		src := NewBuiltinSource()
		manifest, _ := NewManifestBuilder("builtin").Build()
		rc := NewRegisteredCatalog(src, manifest, NewCatalog())

		_ = r.Add(rc)
		err := r.Remove("builtin")
		assert.ErrorIs(t, err, ErrBuiltinImmutable)
	})

	t.Run("remove missing fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Remove("missing")
		assert.ErrorIs(t, err, ErrCatalogNotFound)
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	src := NewBuiltinSource()
	manifest, _ := NewManifestBuilder("builtin").Build()
	rc := NewRegisteredCatalog(src, manifest, NewCatalog())
	_ = r.Add(rc)

	t.Run("existing", func(t *testing.T) {
		t.Parallel()
		got, ok := r.Get("builtin")
		assert.True(t, ok)
		assert.Equal(t, "builtin", got.Name())
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		_, ok := r.Get("missing")
		assert.False(t, ok)
	})
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	// Add builtin
	builtinSrc := NewBuiltinSource()
	builtinManifest, _ := NewManifestBuilder("builtin").Build()
	builtinRC := NewRegisteredCatalog(builtinSrc, builtinManifest, NewCatalog())
	_ = r.Add(builtinRC)

	// Add external
	extSrc, _ := NewURLSource("external", "https://example.com")
	extManifest, _ := NewManifestBuilder("external").Build()
	extRC := NewRegisteredCatalog(extSrc, extManifest, NewCatalog())
	_ = r.Add(extRC)

	list := r.List()
	assert.Len(t, list, 2)
	// Builtin should be first
	assert.Equal(t, "builtin", list[0].Name())
	assert.Equal(t, "external", list[1].Name())
}

func TestRegistry_ListEnabled(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	src1, _ := NewURLSource("enabled", "https://example.com")
	manifest1, _ := NewManifestBuilder("enabled").Build()
	rc1 := NewRegisteredCatalog(src1, manifest1, NewCatalog())
	_ = r.Add(rc1)

	src2, _ := NewURLSource("disabled", "https://other.com")
	manifest2, _ := NewManifestBuilder("disabled").Build()
	rc2 := NewRegisteredCatalog(src2, manifest2, NewCatalog())
	rc2.SetEnabled(false)
	_ = r.Add(rc2)

	enabled := r.ListEnabled()
	assert.Len(t, enabled, 1)
	assert.Equal(t, "enabled", enabled[0].Name())
}

func TestRegistry_FindPreset(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	// Create catalog with preset
	cat := NewCatalog()
	id, _ := ParsePresetID("nvim:balanced")
	meta, _ := NewMetadata("Balanced Neovim", "A balanced setup")
	preset, _ := NewPreset(id, meta, DifficultyIntermediate, map[string]interface{}{"key": "value"})
	_ = cat.AddPreset(preset)

	src := NewBuiltinSource()
	manifest, _ := NewManifestBuilder("builtin").Build()
	rc := NewRegisteredCatalog(src, manifest, cat)
	_ = r.Add(rc)

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		found, fromCatalog, ok := r.FindPreset(id)
		assert.True(t, ok)
		assert.Equal(t, "Balanced Neovim", found.Metadata().Title())
		assert.Equal(t, "builtin", fromCatalog.Name())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		missingID, _ := ParsePresetID("nvim:missing")
		_, _, ok := r.FindPreset(missingID)
		assert.False(t, ok)
	})
}

func TestRegistry_FindPack(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	// Create catalog with pack
	cat := NewCatalog()
	meta, _ := NewMetadata("Go Developer", "Go development tools")
	pack, _ := NewCapabilityPack("go-developer", meta)
	_ = cat.AddPack(pack)

	src := NewBuiltinSource()
	manifest, _ := NewManifestBuilder("builtin").Build()
	rc := NewRegisteredCatalog(src, manifest, cat)
	_ = r.Add(rc)

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		found, _, ok := r.FindPack("go-developer")
		assert.True(t, ok)
		assert.Equal(t, "Go Developer", found.Metadata().Title())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, _, ok := r.FindPack("missing")
		assert.False(t, ok)
	})
}

func TestRegistry_Stats(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	// Add builtin
	builtinSrc := NewBuiltinSource()
	builtinManifest, _ := NewManifestBuilder("builtin").Build()
	builtinCat := NewCatalog()
	meta, _ := NewMetadata("Test", "Test")
	id, _ := ParsePresetID("test:preset")
	preset, _ := NewPreset(id, meta, DifficultyBeginner, nil)
	_ = builtinCat.AddPreset(preset)
	_ = r.Add(NewRegisteredCatalog(builtinSrc, builtinManifest, builtinCat))

	// Add URL catalog
	urlSrc, _ := NewURLSource("url-cat", "https://example.com")
	urlManifest, _ := NewManifestBuilder("url-cat").Build()
	_ = r.Add(NewRegisteredCatalog(urlSrc, urlManifest, NewCatalog()))

	// Add local catalog
	localSrc, _ := NewLocalSource("local-cat", "/path/to/catalog")
	localManifest, _ := NewManifestBuilder("local-cat").Build()
	_ = r.Add(NewRegisteredCatalog(localSrc, localManifest, NewCatalog()))

	stats := r.Stats()
	assert.Equal(t, 3, stats.TotalCatalogs)
	assert.Equal(t, 3, stats.EnabledCatalogs)
	assert.Equal(t, 1, stats.BuiltinCatalogs)
	assert.Equal(t, 1, stats.URLCatalogs)
	assert.Equal(t, 1, stats.LocalCatalogs)
	assert.Equal(t, 1, stats.TotalPresets)
}

func TestRegisteredCatalog(t *testing.T) {
	t.Parallel()

	src := NewBuiltinSource()
	manifest, _ := NewManifestBuilder("test").Build()
	cat := NewCatalog()
	rc := NewRegisteredCatalog(src, manifest, cat)

	assert.Equal(t, src, rc.Source())
	assert.Equal(t, manifest, rc.Manifest())
	assert.Equal(t, cat, rc.Catalog())
	assert.Equal(t, "builtin", rc.Name())
	assert.True(t, rc.Enabled())
	assert.WithinDuration(t, time.Now(), rc.AddedAt(), time.Second)
	assert.WithinDuration(t, time.Now(), rc.VerifiedAt(), time.Second)

	// Test setters
	rc.SetEnabled(false)
	assert.False(t, rc.Enabled())

	newTime := time.Now().Add(-time.Hour)
	rc.SetVerifiedAt(newTime)
	assert.Equal(t, newTime, rc.VerifiedAt())
}
