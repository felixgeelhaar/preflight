package ui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap(t *testing.T) {
	t.Parallel()

	km := ui.DefaultKeyMap()

	// Check that key bindings are set
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Left.Keys())
	assert.NotEmpty(t, km.Right.Keys())
	assert.NotEmpty(t, km.Select.Keys())
	assert.NotEmpty(t, km.Quit.Keys())
}

func TestKeyMap_IsUp(t *testing.T) {
	t.Parallel()

	km := ui.DefaultKeyMap()

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"arrow up", "up", true},
		{"vim k", "k", true},
		{"arrow down", "down", false},
		{"j key", "j", false},
		{"random key", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "up" {
				msg = tea.KeyMsg{Type: tea.KeyUp}
			} else if tt.key == "down" {
				msg = tea.KeyMsg{Type: tea.KeyDown}
			}
			assert.Equal(t, tt.expected, km.IsUp(msg))
		})
	}
}

func TestKeyMap_IsDown(t *testing.T) {
	t.Parallel()

	km := ui.DefaultKeyMap()

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"arrow down", "down", true},
		{"vim j", "j", true},
		{"arrow up", "up", false},
		{"k key", "k", false},
		{"random key", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "up" {
				msg = tea.KeyMsg{Type: tea.KeyUp}
			} else if tt.key == "down" {
				msg = tea.KeyMsg{Type: tea.KeyDown}
			}
			assert.Equal(t, tt.expected, km.IsDown(msg))
		})
	}
}

func TestKeyMap_IsLeft(t *testing.T) {
	t.Parallel()

	km := ui.DefaultKeyMap()

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"arrow left", "left", true},
		{"vim h", "h", true},
		{"arrow right", "right", false},
		{"l key", "l", false},
		{"random key", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "left" {
				msg = tea.KeyMsg{Type: tea.KeyLeft}
			} else if tt.key == "right" {
				msg = tea.KeyMsg{Type: tea.KeyRight}
			}
			assert.Equal(t, tt.expected, km.IsLeft(msg))
		})
	}
}

func TestKeyMap_IsRight(t *testing.T) {
	t.Parallel()

	km := ui.DefaultKeyMap()

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"arrow right", "right", true},
		{"vim l", "l", true},
		{"arrow left", "left", false},
		{"h key", "h", false},
		{"random key", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "left" {
				msg = tea.KeyMsg{Type: tea.KeyLeft}
			} else if tt.key == "right" {
				msg = tea.KeyMsg{Type: tea.KeyRight}
			}
			assert.Equal(t, tt.expected, km.IsRight(msg))
		})
	}
}
