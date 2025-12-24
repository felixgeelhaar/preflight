package marketplace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndex(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	assert.Equal(t, "1", idx.Version)
	assert.Empty(t, idx.Packages)
	assert.Equal(t, 0, idx.Count())
}

func TestIndex_Add(t *testing.T) {
	t.Parallel()

	idx := NewIndex()

	pkg := Package{
		ID:       MustNewPackageID("test-package"),
		Type:     PackageTypePreset,
		Title:    "Test Package",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	}

	err := idx.Add(pkg)
	require.NoError(t, err)
	assert.Equal(t, 1, idx.Count())

	// Duplicate
	err = idx.Add(pkg)
	assert.Error(t, err)

	// Invalid package
	err = idx.Add(Package{})
	assert.ErrorIs(t, err, ErrInvalidPackage)
}

func TestIndex_Get(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	pkg := Package{
		ID:       MustNewPackageID("test-package"),
		Type:     PackageTypePreset,
		Title:    "Test Package",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	}
	_ = idx.Add(pkg)

	// Found
	found, ok := idx.Get(MustNewPackageID("test-package"))
	assert.True(t, ok)
	assert.Equal(t, "Test Package", found.Title)

	// Not found
	_, ok = idx.Get(MustNewPackageID("not-found"))
	assert.False(t, ok)
}

func TestIndex_Search(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("nvim-basic"),
		Type:     PackageTypePreset,
		Title:    "Basic Neovim",
		Keywords: []string{"editor", "vim"},
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:       MustNewPackageID("vscode-config"),
		Type:     PackageTypePreset,
		Title:    "VS Code Config",
		Keywords: []string{"editor", "microsoft"},
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	// Empty query returns all
	results := idx.Search("")
	assert.Len(t, results, 2)

	// Search by ID
	results = idx.Search("nvim")
	assert.Len(t, results, 1)
	assert.Equal(t, "nvim-basic", results[0].ID.String())

	// Search by keyword
	results = idx.Search("editor")
	assert.Len(t, results, 2)

	// No results
	results = idx.Search("emacs")
	assert.Len(t, results, 0)
}

func TestIndex_SearchByType(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("preset-one"),
		Type:     PackageTypePreset,
		Title:    "Preset One",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:       MustNewPackageID("pack-one"),
		Type:     PackageTypeCapabilityPack,
		Title:    "Pack One",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	presets := idx.SearchByType(PackageTypePreset)
	assert.Len(t, presets, 1)
	assert.Equal(t, "preset-one", presets[0].ID.String())

	packs := idx.SearchByType(PackageTypeCapabilityPack)
	assert.Len(t, packs, 1)
	assert.Equal(t, "pack-one", packs[0].ID.String())
}

func TestIndex_SearchByKeyword(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("pkg-one"),
		Type:     PackageTypePreset,
		Title:    "Package One",
		Keywords: []string{"go", "backend"},
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:       MustNewPackageID("pkg-two"),
		Type:     PackageTypePreset,
		Title:    "Package Two",
		Keywords: []string{"python", "backend"},
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	results := idx.SearchByKeyword("backend")
	assert.Len(t, results, 2)

	results = idx.SearchByKeyword("go")
	assert.Len(t, results, 1)
	assert.Equal(t, "pkg-one", results[0].ID.String())

	results = idx.SearchByKeyword("frontend")
	assert.Len(t, results, 0)
}

func TestIndex_ListByPopularity(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:        MustNewPackageID("popular"),
		Type:      PackageTypePreset,
		Title:     "Popular",
		Downloads: 1000,
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:        MustNewPackageID("unpopular"),
		Type:      PackageTypePreset,
		Title:     "Unpopular",
		Downloads: 10,
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})

	results := idx.ListByPopularity()
	assert.Len(t, results, 2)
	assert.Equal(t, "popular", results[0].ID.String())
	assert.Equal(t, "unpopular", results[1].ID.String())
}

func TestIndex_ListByRecent(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	now := time.Now()

	_ = idx.Add(Package{
		ID:        MustNewPackageID("old"),
		Type:      PackageTypePreset,
		Title:     "Old",
		UpdatedAt: now.Add(-24 * time.Hour),
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:        MustNewPackageID("new"),
		Type:      PackageTypePreset,
		Title:     "New",
		UpdatedAt: now,
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})

	results := idx.ListByRecent()
	assert.Len(t, results, 2)
	assert.Equal(t, "new", results[0].ID.String())
	assert.Equal(t, "old", results[1].ID.String())
}

func TestIndex_Statistics(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:        MustNewPackageID("preset-one"),
		Type:      PackageTypePreset,
		Title:     "Preset",
		Downloads: 100,
		Provenance: Provenance{
			Verified: true,
		},
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:        MustNewPackageID("pack-one"),
		Type:      PackageTypeCapabilityPack,
		Title:     "Pack",
		Downloads: 50,
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})
	_ = idx.Add(Package{
		ID:        MustNewPackageID("template-one"),
		Type:      PackageTypeLayerTemplate,
		Title:     "Template",
		Downloads: 25,
		Versions:  []PackageVersion{{Version: "1.0.0"}},
	})

	stats := idx.Statistics()
	assert.Equal(t, 3, stats.TotalPackages)
	assert.Equal(t, 1, stats.Presets)
	assert.Equal(t, 1, stats.CapabilityPacks)
	assert.Equal(t, 1, stats.LayerTemplates)
	assert.Equal(t, 175, stats.TotalDownloads)
	assert.Equal(t, 1, stats.VerifiedPackages)
}

func TestParseIndex(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"version": "1",
		"updated_at": "2024-01-01T00:00:00Z",
		"packages": [
			{
				"id": "test",
				"type": "preset",
				"title": "Test",
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`)

	idx, err := ParseIndex(data)
	require.NoError(t, err)
	assert.Equal(t, "1", idx.Version)
	assert.Len(t, idx.Packages, 1)
	assert.Equal(t, "test", idx.Packages[0].ID.String())

	// Invalid JSON
	_, err = ParseIndex([]byte("invalid"))
	assert.Error(t, err)
}

func TestIndex_Marshal(t *testing.T) {
	t.Parallel()

	idx := NewIndex()
	_ = idx.Add(Package{
		ID:       MustNewPackageID("test"),
		Type:     PackageTypePreset,
		Title:    "Test",
		Versions: []PackageVersion{{Version: "1.0.0"}},
	})

	data, err := idx.Marshal()
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")
	assert.Contains(t, string(data), "preset")
}
