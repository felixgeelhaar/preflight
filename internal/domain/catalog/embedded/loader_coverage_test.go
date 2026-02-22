package embedded

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePreset_InvalidID(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "", // invalid: empty ID
		Metadata: metadataDTO{
			Title:       "Test",
			Description: "Test desc",
		},
		Difficulty: "beginner",
	}

	_, err := parsePreset(dto)
	assert.Error(t, err)
}

func TestParsePreset_InvalidMetadata(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "nvim:test",
		Metadata: metadataDTO{
			Title:       "", // invalid: empty title
			Description: "Test desc",
		},
		Difficulty: "beginner",
	}

	_, err := parsePreset(dto)
	assert.Error(t, err)
}

func TestParsePreset_InvalidDifficulty(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "nvim:test",
		Metadata: metadataDTO{
			Title:       "Test",
			Description: "Test desc",
		},
		Difficulty: "impossible", // invalid difficulty
	}

	_, err := parsePreset(dto)
	assert.Error(t, err)
}

func TestParsePreset_InvalidRequires(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "nvim:test",
		Metadata: metadataDTO{
			Title:       "Test",
			Description: "Test desc",
		},
		Difficulty: "beginner",
		Requires:   []string{"invalid-no-colon"}, // invalid: no colon separator
	}

	_, err := parsePreset(dto)
	assert.Error(t, err)
}

func TestParsePreset_InvalidConflicts(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "nvim:test",
		Metadata: metadataDTO{
			Title:       "Test",
			Description: "Test desc",
		},
		Difficulty: "beginner",
		Conflicts:  []string{"invalid-no-colon"}, // invalid: no colon separator
	}

	_, err := parsePreset(dto)
	assert.Error(t, err)
}

func TestParsePreset_ValidWithRequiresAndConflicts(t *testing.T) {
	t.Parallel()

	dto := presetDTO{
		ID: "nvim:test",
		Metadata: metadataDTO{
			Title:       "Test Preset",
			Description: "A test preset",
			Tradeoffs:   []string{"pro", "con"},
			DocLinks:    map[string]string{"docs": "https://example.com"},
			Tags:        []string{"editor"},
		},
		Difficulty: "intermediate",
		Config:     map[string]interface{}{"key": "value"},
		Requires:   []string{"shell:zsh"},
		Conflicts:  []string{"nvim:minimal"},
	}

	preset, err := parsePreset(dto)
	require.NoError(t, err)
	assert.Equal(t, "test", preset.ID().Name())
	assert.Equal(t, "nvim", preset.ID().Provider())
	assert.NotEmpty(t, preset.Requires())
	assert.NotEmpty(t, preset.Conflicts())
}

func TestParseMetadata_InvalidTitle(t *testing.T) {
	t.Parallel()

	dto := metadataDTO{
		Title:       "",
		Description: "desc",
	}

	_, err := parseMetadata(dto)
	assert.Error(t, err)
}

func TestParseMetadata_InvalidDescription(t *testing.T) {
	t.Parallel()

	dto := metadataDTO{
		Title:       "title",
		Description: "",
	}

	_, err := parseMetadata(dto)
	assert.Error(t, err)
}

func TestParseMetadata_WithAllOptionalFields(t *testing.T) {
	t.Parallel()

	dto := metadataDTO{
		Title:       "Test Title",
		Description: "Test Description",
		Tradeoffs:   []string{"trade-off 1", "trade-off 2"},
		DocLinks:    map[string]string{"docs": "https://example.com"},
		Tags:        []string{"tag1", "tag2"},
	}

	meta, err := parseMetadata(dto)
	require.NoError(t, err)
	assert.Equal(t, "Test Title", meta.Title())
	assert.Equal(t, "Test Description", meta.Description())
	assert.Len(t, meta.Tradeoffs(), 2)
	assert.Len(t, meta.DocLinks(), 1)
	assert.Len(t, meta.Tags(), 2)
}

func TestParseMetadata_WithNoOptionalFields(t *testing.T) {
	t.Parallel()

	dto := metadataDTO{
		Title:       "Minimal Title",
		Description: "Minimal Description",
	}

	meta, err := parseMetadata(dto)
	require.NoError(t, err)
	assert.Equal(t, "Minimal Title", meta.Title())
	assert.Empty(t, meta.Tradeoffs())
	assert.Empty(t, meta.DocLinks())
	assert.Empty(t, meta.Tags())
}

func TestParseCapabilityPack_InvalidMetadata(t *testing.T) {
	t.Parallel()

	dto := capabilityPackDTO{
		ID: "test-pack",
		Metadata: metadataDTO{
			Title:       "", // invalid
			Description: "desc",
		},
	}

	_, err := parseCapabilityPack(dto)
	assert.Error(t, err)
}

func TestParseCapabilityPack_InvalidID(t *testing.T) {
	t.Parallel()

	dto := capabilityPackDTO{
		ID: "", // invalid
		Metadata: metadataDTO{
			Title:       "Pack",
			Description: "A pack",
		},
	}

	_, err := parseCapabilityPack(dto)
	assert.Error(t, err)
}

func TestParseCapabilityPack_InvalidPresetID(t *testing.T) {
	t.Parallel()

	dto := capabilityPackDTO{
		ID: "test-pack",
		Metadata: metadataDTO{
			Title:       "Test Pack",
			Description: "A test pack",
		},
		Presets: []string{"invalid-no-colon"}, // invalid preset ID
	}

	_, err := parseCapabilityPack(dto)
	assert.Error(t, err)
}

func TestParseCapabilityPack_ValidWithPresetsAndTools(t *testing.T) {
	t.Parallel()

	dto := capabilityPackDTO{
		ID: "valid-pack",
		Metadata: metadataDTO{
			Title:       "Valid Pack",
			Description: "A valid pack",
			Tags:        []string{"go", "backend"},
		},
		Presets: []string{"nvim:balanced", "shell:starship"},
		Tools:   []string{"go", "gopls", "delve"},
	}

	pack, err := parseCapabilityPack(dto)
	require.NoError(t, err)
	assert.Equal(t, "valid-pack", pack.ID())
	assert.Len(t, pack.Presets(), 2)
	assert.Len(t, pack.Tools(), 3)
}

func TestParseCapabilityPack_NoPresetsNoTools(t *testing.T) {
	t.Parallel()

	dto := capabilityPackDTO{
		ID: "empty-pack",
		Metadata: metadataDTO{
			Title:       "Empty Pack",
			Description: "A pack with no presets or tools",
		},
	}

	pack, err := parseCapabilityPack(dto)
	require.NoError(t, err)
	assert.Equal(t, "empty-pack", pack.ID())
	assert.Empty(t, pack.Presets())
	assert.Empty(t, pack.Tools())
}

func TestLoadCatalog_PresetsHaveConfigs(t *testing.T) {
	t.Parallel()

	cat, err := LoadCatalog()
	require.NoError(t, err)

	// All presets should have valid non-empty configs
	for _, preset := range cat.FindPresetsByProvider("nvim") {
		assert.NotNil(t, preset.Config(), "preset %s should have config", preset.ID())
	}
}

func TestLoadCatalog_PresetsConflicts(t *testing.T) {
	t.Parallel()

	cat, err := LoadCatalog()
	require.NoError(t, err)

	// nvim:pro conflicts with nvim:minimal
	proPresets := cat.FindPresetsByProvider("nvim")
	for _, p := range proPresets {
		if p.ID().Name() == "pro" {
			conflicts := p.Conflicts()
			assert.NotEmpty(t, conflicts, "nvim:pro should have conflicts")
		}
	}
}

func TestLoadCatalog_AllPacksHaveTools(t *testing.T) {
	t.Parallel()

	cat, err := LoadCatalog()
	require.NoError(t, err)

	// All capability packs should have tools
	packIDs := []string{"go-developer", "frontend-developer", "devops-engineer"}
	for _, id := range packIDs {
		pack, ok := cat.GetPack(id)
		require.True(t, ok, "pack %s should exist", id)
		assert.NotEmpty(t, pack.Tools(), "pack %s should have tools", id)
		assert.NotEmpty(t, pack.Presets(), "pack %s should have presets", id)
	}
}

func TestLoadCatalog_DifficultyLevels(t *testing.T) {
	t.Parallel()

	cat, err := LoadCatalog()
	require.NoError(t, err)

	// Check that all difficulty levels have presets
	levels := []catalog.DifficultyLevel{"beginner", "intermediate", "advanced"}
	for _, level := range levels {
		presets := cat.FindPresetsByDifficulty(level)
		assert.NotEmpty(t, presets, "should have %s presets", level)
	}
}

func TestLoadCatalog_AllPresetsHaveMetadata(t *testing.T) {
	t.Parallel()

	cat, err := LoadCatalog()
	require.NoError(t, err)

	// Check metadata on some known presets
	shellPresets := cat.FindPresetsByProvider("shell")
	for _, p := range shellPresets {
		meta := p.Metadata()
		assert.NotEmpty(t, meta.Title(), "preset %s should have a title", p.ID())
		assert.NotEmpty(t, meta.Description(), "preset %s should have a description", p.ID())
	}
}
