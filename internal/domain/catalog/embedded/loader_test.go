package embedded

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()

	require.NoError(t, err)
	assert.NotNil(t, catalog)
}

func TestLoadCatalog_HasPresets(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	// Should have at least the defined presets
	assert.GreaterOrEqual(t, catalog.PresetCount(), 8)
}

func TestLoadCatalog_HasPacks(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	// Should have at least the defined capability packs
	assert.GreaterOrEqual(t, catalog.PackCount(), 3)
}

func TestLoadCatalog_NvimPresetsExist(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	nvimPresets := catalog.FindPresetsByProvider("nvim")

	assert.Len(t, nvimPresets, 3) // minimal, balanced, pro
}

func TestLoadCatalog_GoDevPackExists(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	pack, ok := catalog.GetPack("go-developer")

	assert.True(t, ok)
	assert.Equal(t, "go-developer", pack.ID())
	assert.Contains(t, pack.Tools(), "go")
	assert.Contains(t, pack.Tools(), "gopls")
}

func TestLoadCatalog_PresetHasMetadata(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	presets := catalog.FindPresetsByProvider("nvim")
	require.NotEmpty(t, presets)

	// Find the balanced preset
	var balanced bool
	for _, p := range presets {
		if p.ID().Name() == "balanced" {
			assert.Equal(t, "Balanced Neovim", p.Metadata().Title())
			assert.NotEmpty(t, p.Metadata().Description())
			balanced = true
		}
	}
	assert.True(t, balanced, "should find nvim:balanced preset")
}

func TestLoadCatalog_PresetHasDifficulty(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	// Check beginner presets exist
	beginnerPresets := catalog.FindPresetsByDifficulty("beginner")
	assert.NotEmpty(t, beginnerPresets)

	// Check intermediate presets exist
	intermediatePresets := catalog.FindPresetsByDifficulty("intermediate")
	assert.NotEmpty(t, intermediatePresets)
}

func TestLoadCatalog_PresetRequirements(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	// nvim:balanced requires shell:zsh
	presets := catalog.FindPresetsByProvider("nvim")
	for _, p := range presets {
		if p.ID().Name() == "balanced" {
			requires := p.Requires()
			assert.NotEmpty(t, requires)
		}
	}
}

func TestLoadCatalog_CapabilityPackHasPresets(t *testing.T) {
	t.Parallel()

	catalog, err := LoadCatalog()
	require.NoError(t, err)

	pack, ok := catalog.GetPack("go-developer")
	require.True(t, ok)

	presets := pack.Presets()
	assert.NotEmpty(t, presets)
}
