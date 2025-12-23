package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPreset(t *testing.T, provider, name string, difficulty DifficultyLevel) Preset {
	t.Helper()
	id, err := NewPresetID(provider, name)
	require.NoError(t, err)
	meta, err := NewMetadata(name+" preset", "A test "+name+" preset")
	require.NoError(t, err)
	preset, err := NewPreset(id, meta, difficulty, nil)
	require.NoError(t, err)
	return preset
}

func createTestPack(t *testing.T, id string) CapabilityPack {
	t.Helper()
	meta, err := NewMetadata(id+" pack", "A test "+id+" capability pack")
	require.NoError(t, err)
	pack, err := NewCapabilityPack(id, meta)
	require.NoError(t, err)
	return pack
}

func TestNewCatalog(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()

	assert.NotNil(t, catalog)
	assert.Empty(t, catalog.ListPresets())
	assert.Empty(t, catalog.ListPacks())
}

func TestCatalog_AddPreset(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	preset := createTestPreset(t, "nvim", "balanced", DifficultyIntermediate)

	err := catalog.AddPreset(preset)

	require.NoError(t, err)
	assert.Len(t, catalog.ListPresets(), 1)
}

func TestCatalog_AddPreset_ZeroValue(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()

	err := catalog.AddPreset(Preset{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPreset)
}

func TestCatalog_AddPreset_Duplicate(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	preset := createTestPreset(t, "nvim", "balanced", DifficultyIntermediate)

	err := catalog.AddPreset(preset)
	require.NoError(t, err)

	err = catalog.AddPreset(preset)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicatePreset)
}

func TestCatalog_GetPreset(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	preset := createTestPreset(t, "nvim", "balanced", DifficultyIntermediate)
	_ = catalog.AddPreset(preset)

	id, _ := NewPresetID("nvim", "balanced")
	found, ok := catalog.GetPreset(id)

	assert.True(t, ok)
	assert.Equal(t, preset.ID(), found.ID())
}

func TestCatalog_GetPreset_NotFound(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()

	id, _ := NewPresetID("nvim", "balanced")
	_, ok := catalog.GetPreset(id)

	assert.False(t, ok)
}

func TestCatalog_FindPresetsByProvider(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "minimal", DifficultyBeginner))
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "balanced", DifficultyIntermediate))
	_ = catalog.AddPreset(createTestPreset(t, "shell", "starship", DifficultyBeginner))

	nvimPresets := catalog.FindPresetsByProvider("nvim")

	assert.Len(t, nvimPresets, 2)
}

func TestCatalog_FindPresetsByDifficulty(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "minimal", DifficultyBeginner))
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "balanced", DifficultyIntermediate))
	_ = catalog.AddPreset(createTestPreset(t, "shell", "starship", DifficultyBeginner))

	beginnerPresets := catalog.FindPresetsByDifficulty(DifficultyBeginner)

	assert.Len(t, beginnerPresets, 2)
}

func TestCatalog_AddPack(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	pack := createTestPack(t, "go-developer")

	err := catalog.AddPack(pack)

	require.NoError(t, err)
	assert.Len(t, catalog.ListPacks(), 1)
}

func TestCatalog_AddPack_ZeroValue(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()

	err := catalog.AddPack(CapabilityPack{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPack)
}

func TestCatalog_AddPack_Duplicate(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	pack := createTestPack(t, "go-developer")

	err := catalog.AddPack(pack)
	require.NoError(t, err)

	err = catalog.AddPack(pack)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDuplicatePack)
}

func TestCatalog_GetPack(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	pack := createTestPack(t, "go-developer")
	_ = catalog.AddPack(pack)

	found, ok := catalog.GetPack("go-developer")

	assert.True(t, ok)
	assert.Equal(t, pack.ID(), found.ID())
}

func TestCatalog_GetPack_NotFound(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()

	_, ok := catalog.GetPack("nonexistent")

	assert.False(t, ok)
}

func TestCatalog_ListPresets(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "minimal", DifficultyBeginner))
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "balanced", DifficultyIntermediate))

	presets := catalog.ListPresets()

	assert.Len(t, presets, 2)
}

func TestCatalog_ListPacks(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPack(createTestPack(t, "go-developer"))
	_ = catalog.AddPack(createTestPack(t, "frontend"))

	packs := catalog.ListPacks()

	assert.Len(t, packs, 2)
}

func TestCatalog_PresetCount(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "minimal", DifficultyBeginner))
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "balanced", DifficultyIntermediate))

	assert.Equal(t, 2, catalog.PresetCount())
}

func TestCatalog_PackCount(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPack(createTestPack(t, "go-developer"))
	_ = catalog.AddPack(createTestPack(t, "frontend"))

	assert.Equal(t, 2, catalog.PackCount())
}

func TestCatalog_String(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog()
	_ = catalog.AddPreset(createTestPreset(t, "nvim", "minimal", DifficultyBeginner))
	_ = catalog.AddPack(createTestPack(t, "go-developer"))

	expected := "Catalog (1 presets, 1 packs)"
	assert.Equal(t, expected, catalog.String())
}
