package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPresetID_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		provider     string
		presetName   string
		wantProvider string
		wantName     string
	}{
		{"simple", "nvim", "balanced", "nvim", "balanced"},
		{"with-dash", "oh-my-zsh", "minimal", "oh-my-zsh", "minimal"},
		{"with-underscore", "my_provider", "my_preset", "my_provider", "my_preset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := NewPresetID(tt.provider, tt.presetName)
			require.NoError(t, err)
			assert.Equal(t, tt.wantProvider, id.Provider())
			assert.Equal(t, tt.wantName, id.Name())
		})
	}
}

func TestNewPresetID_EmptyProvider(t *testing.T) {
	t.Parallel()

	_, err := NewPresetID("", "balanced")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPresetProvider)
}

func TestNewPresetID_EmptyName(t *testing.T) {
	t.Parallel()

	_, err := NewPresetID("nvim", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPresetName)
}

func TestParsePresetID_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		provider string
		preset   string
	}{
		{"simple", "nvim:balanced", "nvim", "balanced"},
		{"with-dash", "oh-my-zsh:minimal", "oh-my-zsh", "minimal"},
		{"shell-starship", "shell:starship", "shell", "starship"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ParsePresetID(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.provider, id.Provider())
			assert.Equal(t, tt.preset, id.Name())
		})
	}
}

func TestParsePresetID_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"no colon", "nvimbalanced"},
		{"empty provider", ":balanced"},
		{"empty name", "nvim:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParsePresetID(tt.input)
			require.Error(t, err)
		})
	}
}

func TestPresetID_String(t *testing.T) {
	t.Parallel()

	id, _ := NewPresetID("nvim", "balanced")

	assert.Equal(t, "nvim:balanced", id.String())
}

func TestPresetID_Equals(t *testing.T) {
	t.Parallel()

	id1, _ := NewPresetID("nvim", "balanced")
	id2, _ := NewPresetID("nvim", "balanced")
	id3, _ := NewPresetID("nvim", "minimal")
	id4, _ := NewPresetID("shell", "balanced")

	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
	assert.False(t, id1.Equals(id4))
}

func TestPresetID_IsZero(t *testing.T) {
	t.Parallel()

	var zero PresetID
	assert.True(t, zero.IsZero())

	nonZero, _ := NewPresetID("nvim", "balanced")
	assert.False(t, nonZero.IsZero())
}

func TestPresetID_MatchesProvider(t *testing.T) {
	t.Parallel()

	id, _ := NewPresetID("nvim", "balanced")

	assert.True(t, id.MatchesProvider("nvim"))
	assert.False(t, id.MatchesProvider("shell"))
	assert.False(t, id.MatchesProvider(""))
}
