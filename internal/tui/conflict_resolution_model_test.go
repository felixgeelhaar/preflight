package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictResolutionModel_New(t *testing.T) {
	conflicts := []merge.Conflict{
		{
			Start:  0,
			End:    10,
			Base:   []string{"base line 1", "base line 2"},
			Ours:   []string{"ours line 1", "ours line 2"},
			Theirs: []string{"theirs line 1", "theirs line 2"},
		},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	assert.Equal(t, "test.yaml", m.filePath)
	assert.Len(t, m.conflicts, 1)
	assert.Equal(t, 0, m.currentConflict)
	assert.False(t, m.done)
	assert.False(t, m.cancelled)
}

func TestConflictResolutionModel_Navigation(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"conflict 1"}},
		{Start: 10, End: 15, Ours: []string{"conflict 2"}},
		{Start: 20, End: 25, Ours: []string{"conflict 3"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Start at first conflict
	assert.Equal(t, 0, m.currentConflict)

	// Navigate to next conflict with 'n'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 1, m.currentConflict)

	// Navigate to next conflict
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 2, m.currentConflict)

	// Navigate at end should not change
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 2, m.currentConflict)

	// Navigate to previous conflict with 'p'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 1, m.currentConflict)
}

func TestConflictResolutionModel_PickOurs(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours line"}, Theirs: []string{"theirs line"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Pick ours with 'o'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveOurs, m.resolutions[0])
	assert.True(t, m.done) // All conflicts resolved
	assert.NotNil(t, cmd)  // Should quit
}

func TestConflictResolutionModel_PickTheirs(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours line"}, Theirs: []string{"theirs line"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Pick theirs with 't'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveTheirs, m.resolutions[0])
	assert.True(t, m.done) // All conflicts resolved
	assert.NotNil(t, cmd)  // Should quit
}

func TestConflictResolutionModel_PickBase(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Base: []string{"base line"}, Ours: []string{"ours line"}, Theirs: []string{"theirs line"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Pick base with 'b'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveBase, m.resolutions[0])
	assert.True(t, m.done) // All conflicts resolved
	assert.NotNil(t, cmd)  // Should quit
}

func TestConflictResolutionModel_MultipleConflicts(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours 1"}, Theirs: []string{"theirs 1"}},
		{Start: 10, End: 15, Ours: []string{"ours 2"}, Theirs: []string{"theirs 2"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Resolve first conflict with ours
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveOurs, m.resolutions[0])
	assert.False(t, m.done)               // Still have unresolved conflicts
	assert.Equal(t, 1, m.currentConflict) // Auto-advanced to next conflict

	// Resolve second conflict with theirs
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveTheirs, m.resolutions[1])
	assert.True(t, m.done)
	assert.NotNil(t, cmd) // Should quit
}

func TestConflictResolutionModel_Cancel(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours"}, Theirs: []string{"theirs"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Press Esc to cancel
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(conflictResolutionModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd) // Should quit
}

func TestConflictResolutionModel_View(t *testing.T) {
	conflicts := []merge.Conflict{
		{
			Start:  0,
			End:    10,
			Base:   []string{"base content"},
			Ours:   []string{"ours content"},
			Theirs: []string{"theirs content"},
		},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain title
	assert.Contains(t, view, "Conflict Resolution")

	// Should contain file path
	assert.Contains(t, view, "test.yaml")

	// Should contain conflict indicator
	assert.Contains(t, view, "1/1")

	// Should contain help text
	assert.Contains(t, view, "o")
	assert.Contains(t, view, "t")
	assert.Contains(t, view, "b")
}

func TestConflictResolutionModel_WindowResize(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours"}, Theirs: []string{"theirs"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestConflictResolutionModel_ResolveAll(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours 1"}, Theirs: []string{"theirs 1"}},
		{Start: 10, End: 15, Ours: []string{"ours 2"}, Theirs: []string{"theirs 2"}},
		{Start: 20, End: 25, Ours: []string{"ours 3"}, Theirs: []string{"theirs 3"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Resolve all with 'O' (uppercase)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveOurs, m.resolutions[0])
	assert.Equal(t, merge.ResolveOurs, m.resolutions[1])
	assert.Equal(t, merge.ResolveOurs, m.resolutions[2])
	assert.True(t, m.done)
	assert.NotNil(t, cmd)
}

func TestConflictResolutionModel_ResolveAllTheirs(t *testing.T) {
	conflicts := []merge.Conflict{
		{Start: 0, End: 5, Ours: []string{"ours 1"}, Theirs: []string{"theirs 1"}},
		{Start: 10, End: 15, Ours: []string{"ours 2"}, Theirs: []string{"theirs 2"}},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})

	// Resolve all with 'T' (uppercase)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	m = updated.(conflictResolutionModel)

	assert.Equal(t, merge.ResolveTheirs, m.resolutions[0])
	assert.Equal(t, merge.ResolveTheirs, m.resolutions[1])
	assert.True(t, m.done)
	assert.NotNil(t, cmd)
}

func TestConflictResolutionModel_ScrollContent(t *testing.T) {
	// Create a conflict with many lines
	longContent := make([]string, 20)
	for i := range longContent {
		longContent[i] = "line content"
	}

	conflicts := []merge.Conflict{
		{Start: 0, End: 30, Ours: longContent, Theirs: longContent},
	}

	m := newConflictResolutionModel("test.yaml", conflicts, ConflictResolutionOptions{})
	m.height = 10 // Small height to trigger scrolling

	// Start at top
	assert.Equal(t, 0, m.scrollOffset)

	// Scroll down with 'j'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 1, m.scrollOffset)

	// Scroll up with 'k'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(conflictResolutionModel)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestConflictResolutionResult(t *testing.T) {
	result := &ConflictResolutionResult{
		Resolutions: []merge.Resolution{merge.ResolveOurs, merge.ResolveTheirs},
		Cancelled:   false,
	}

	require.NotNil(t, result)
	assert.Len(t, result.Resolutions, 2)
	assert.False(t, result.Cancelled)
}

func TestConflictResolutionOptions(t *testing.T) {
	opts := ConflictResolutionOptions{
		ShowBase: true,
	}

	assert.True(t, opts.ShowBase)
}
