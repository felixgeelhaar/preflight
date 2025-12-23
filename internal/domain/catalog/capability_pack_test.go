package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCapabilityPack_Valid(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")

	pack, err := NewCapabilityPack("go-developer", meta)

	require.NoError(t, err)
	assert.Equal(t, "go-developer", pack.ID())
	assert.Equal(t, meta, pack.Metadata())
	assert.Empty(t, pack.Presets())
	assert.Empty(t, pack.Tools())
}

func TestNewCapabilityPack_EmptyID(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")

	_, err := NewCapabilityPack("", meta)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPackID)
}

func TestNewCapabilityPack_ZeroMetadata(t *testing.T) {
	t.Parallel()

	_, err := NewCapabilityPack("go-developer", Metadata{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMetadata)
}

func TestCapabilityPack_WithPresets(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	pack, _ := NewCapabilityPack("go-developer", meta)

	nvimID, _ := NewPresetID("nvim", "balanced")
	shellID, _ := NewPresetID("shell", "starship")
	presets := []PresetID{nvimID, shellID}

	updated := pack.WithPresets(presets)

	assert.Empty(t, pack.Presets())
	assert.Len(t, updated.Presets(), 2)
}

func TestCapabilityPack_WithTools(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	pack, _ := NewCapabilityPack("go-developer", meta)

	tools := []string{"go", "gopls", "golangci-lint", "delve"}

	updated := pack.WithTools(tools)

	assert.Empty(t, pack.Tools())
	assert.Equal(t, tools, updated.Tools())
}

func TestCapabilityPack_HasPreset(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	pack, _ := NewCapabilityPack("go-developer", meta)

	nvimID, _ := NewPresetID("nvim", "balanced")
	pack = pack.WithPresets([]PresetID{nvimID})

	assert.True(t, pack.HasPreset(nvimID))

	otherID, _ := NewPresetID("nvim", "minimal")
	assert.False(t, pack.HasPreset(otherID))
}

func TestCapabilityPack_HasTool(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	pack, _ := NewCapabilityPack("go-developer", meta)

	pack = pack.WithTools([]string{"go", "gopls"})

	assert.True(t, pack.HasTool("go"))
	assert.True(t, pack.HasTool("gopls"))
	assert.False(t, pack.HasTool("node"))
}

func TestCapabilityPack_IsZero(t *testing.T) {
	t.Parallel()

	var zero CapabilityPack
	assert.True(t, zero.IsZero())

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	nonZero, _ := NewCapabilityPack("go-developer", meta)
	assert.False(t, nonZero.IsZero())
}

func TestCapabilityPack_String(t *testing.T) {
	t.Parallel()

	meta, _ := NewMetadata("Go Developer", "Tools for Go development")
	pack, _ := NewCapabilityPack("go-developer", meta)
	pack = pack.WithPresets([]PresetID{createTestPresetID(t)})
	pack = pack.WithTools([]string{"go", "gopls", "golangci-lint"})

	expected := "go-developer (1 presets, 3 tools)"
	assert.Equal(t, expected, pack.String())
}
