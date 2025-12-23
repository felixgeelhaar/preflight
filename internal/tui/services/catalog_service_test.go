package services

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat := catalog.NewCatalog()

	// Add test presets
	nvimMinimalID, err := catalog.ParsePresetID("nvim:minimal")
	require.NoError(t, err)
	nvimMeta, err := catalog.NewMetadata("Minimal", "Essential plugins only")
	require.NoError(t, err)
	nvimPreset, err := catalog.NewPreset(nvimMinimalID, nvimMeta, catalog.DifficultyBeginner, nil)
	require.NoError(t, err)
	require.NoError(t, cat.AddPreset(nvimPreset))

	nvimBalancedID, err := catalog.ParsePresetID("nvim:balanced")
	require.NoError(t, err)
	nvimBalancedMeta, err := catalog.NewMetadata("Balanced", "Recommended for most users")
	require.NoError(t, err)
	nvimBalancedPreset, err := catalog.NewPreset(nvimBalancedID, nvimBalancedMeta, catalog.DifficultyIntermediate, nil)
	require.NoError(t, err)
	require.NoError(t, cat.AddPreset(nvimBalancedPreset))

	shellZshID, err := catalog.ParsePresetID("shell:zsh")
	require.NoError(t, err)
	shellMeta, err := catalog.NewMetadata("Zsh", "Basic Zsh configuration")
	require.NoError(t, err)
	shellPreset, err := catalog.NewPreset(shellZshID, shellMeta, catalog.DifficultyBeginner, nil)
	require.NoError(t, err)
	require.NoError(t, cat.AddPreset(shellPreset))

	return cat
}

func TestNewCatalogService(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	assert.NotNil(t, service)
}

func TestCatalogService_GetProviders(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	providers := service.GetProviders()

	assert.Contains(t, providers, "nvim")
	assert.Contains(t, providers, "shell")
}

func TestCatalogService_GetPresetsForProvider(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	presets := service.GetPresetsForProvider("nvim")

	assert.Len(t, presets, 2)
	// Presets are sorted alphabetically by ID
	assert.Equal(t, "nvim:balanced", presets[0].ID)
	assert.Equal(t, "Balanced", presets[0].Title)
	assert.Equal(t, "Recommended for most users", presets[0].Description)
	assert.Equal(t, "nvim:minimal", presets[1].ID)
	assert.Equal(t, "Minimal", presets[1].Title)
}

func TestCatalogService_GetPresetsForProvider_Empty(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	presets := service.GetPresetsForProvider("unknown")

	assert.Empty(t, presets)
}

func TestCatalogService_GetCapabilityPacks(t *testing.T) {
	t.Parallel()
	cat := catalog.NewCatalog()

	// Add a capability pack
	packMeta, err := catalog.NewMetadata("Developer", "Full developer setup")
	require.NoError(t, err)
	pack, err := catalog.NewCapabilityPack("developer", packMeta)
	require.NoError(t, err)
	require.NoError(t, cat.AddPack(pack))

	service := NewCatalogService(cat)
	packs := service.GetCapabilityPacks()

	assert.Len(t, packs, 1)
	assert.Equal(t, "developer", packs[0].ID)
	assert.Equal(t, "Developer", packs[0].Title)
}

func TestCatalogService_GetPreset(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	preset, found := service.GetPreset("nvim:minimal")

	assert.True(t, found)
	assert.Equal(t, "Minimal", preset.Title)
}

func TestCatalogService_GetPreset_NotFound(t *testing.T) {
	t.Parallel()
	cat := createTestCatalog(t)
	service := NewCatalogService(cat)

	_, found := service.GetPreset("unknown:preset")

	assert.False(t, found)
}
