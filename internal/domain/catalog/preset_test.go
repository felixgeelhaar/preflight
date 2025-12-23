package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestMetadata(t *testing.T) Metadata {
	t.Helper()
	meta, err := NewMetadata("Test Preset", "A test preset description")
	require.NoError(t, err)
	return meta
}

func createTestPresetID(t *testing.T) PresetID {
	t.Helper()
	id, err := NewPresetID("nvim", "balanced")
	require.NoError(t, err)
	return id
}

func TestNewPreset_Valid(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	config := map[string]interface{}{
		"plugins": []string{"telescope", "treesitter"},
	}

	preset, err := NewPreset(id, meta, DifficultyIntermediate, config)

	require.NoError(t, err)
	assert.Equal(t, id, preset.ID())
	assert.Equal(t, meta, preset.Metadata())
	assert.Equal(t, DifficultyIntermediate, preset.Difficulty())
	assert.Equal(t, config, preset.Config())
}

func TestNewPreset_ZeroID(t *testing.T) {
	t.Parallel()

	meta := createTestMetadata(t)

	_, err := NewPreset(PresetID{}, meta, DifficultyBeginner, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidPresetID)
}

func TestNewPreset_ZeroMetadata(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)

	_, err := NewPreset(id, Metadata{}, DifficultyBeginner, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMetadata)
}

func TestPreset_WithRequires(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	preset, _ := NewPreset(id, meta, DifficultyBeginner, nil)

	reqID, _ := NewPresetID("shell", "zsh")
	updated := preset.WithRequires([]PresetID{reqID})

	assert.Empty(t, preset.Requires())
	assert.Len(t, updated.Requires(), 1)
	assert.True(t, updated.Requires()[0].Equals(reqID))
}

func TestPreset_WithConflicts(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	preset, _ := NewPreset(id, meta, DifficultyBeginner, nil)

	conflictID, _ := NewPresetID("nvim", "minimal")
	updated := preset.WithConflicts([]PresetID{conflictID})

	assert.Empty(t, preset.Conflicts())
	assert.Len(t, updated.Conflicts(), 1)
}

func TestPreset_RequiresPreset(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	preset, _ := NewPreset(id, meta, DifficultyBeginner, nil)

	reqID, _ := NewPresetID("shell", "zsh")
	preset = preset.WithRequires([]PresetID{reqID})

	assert.True(t, preset.RequiresPreset(reqID))

	otherID, _ := NewPresetID("shell", "fish")
	assert.False(t, preset.RequiresPreset(otherID))
}

func TestPreset_ConflictsWith(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	preset, _ := NewPreset(id, meta, DifficultyBeginner, nil)

	conflictID, _ := NewPresetID("nvim", "minimal")
	preset = preset.WithConflicts([]PresetID{conflictID})

	assert.True(t, preset.ConflictsWith(conflictID))

	otherID, _ := NewPresetID("nvim", "pro")
	assert.False(t, preset.ConflictsWith(otherID))
}

func TestPreset_IsZero(t *testing.T) {
	t.Parallel()

	var zero Preset
	assert.True(t, zero.IsZero())

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	nonZero, _ := NewPreset(id, meta, DifficultyBeginner, nil)
	assert.False(t, nonZero.IsZero())
}

func TestPreset_String(t *testing.T) {
	t.Parallel()

	id := createTestPresetID(t)
	meta := createTestMetadata(t)
	preset, _ := NewPreset(id, meta, DifficultyIntermediate, nil)

	assert.Equal(t, "nvim:balanced (intermediate)", preset.String())
}

func TestDifficultyLevel_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "beginner", DifficultyBeginner.String())
	assert.Equal(t, "intermediate", DifficultyIntermediate.String())
	assert.Equal(t, "advanced", DifficultyAdvanced.String())
}

func TestDifficultyLevel_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, DifficultyBeginner.IsValid())
	assert.True(t, DifficultyIntermediate.IsValid())
	assert.True(t, DifficultyAdvanced.IsValid())
	assert.False(t, DifficultyLevel("expert").IsValid())
}

func TestParseDifficultyLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected DifficultyLevel
		wantErr  bool
	}{
		{"beginner", DifficultyBeginner, false},
		{"intermediate", DifficultyIntermediate, false},
		{"advanced", DifficultyAdvanced, false},
		{"BEGINNER", DifficultyBeginner, false}, // case insensitive
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			level, err := ParseDifficultyLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}
