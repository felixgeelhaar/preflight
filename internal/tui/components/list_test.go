package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewList(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First", Description: "First item"},
		{ID: "2", Title: "Second", Description: "Second item"},
	}

	list := NewList(items)

	assert.Equal(t, 2, len(list.Items()))
	assert.Equal(t, 0, list.SelectedIndex())
	assert.Equal(t, "1", list.SelectedItem().ID)
}

func TestList_EmptyList(t *testing.T) {
	t.Parallel()

	list := NewList([]ListItem{})

	assert.Equal(t, 0, len(list.Items()))
	assert.Equal(t, 0, list.SelectedIndex())
	assert.Nil(t, list.SelectedItem())
}

func TestList_Navigation(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
		{ID: "3", Title: "Third"},
	}

	list := NewList(items)

	// Move down
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, list.SelectedIndex())

	// Move down again
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, list.SelectedIndex())

	// Move down at end (should wrap or stay)
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, list.SelectedIndex()) // Stay at end

	// Move up
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, list.SelectedIndex())
}

func TestList_VimNavigation(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
		{ID: "3", Title: "Third"},
	}

	list := NewList(items)

	// Move down with j
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, list.SelectedIndex())

	// Move up with k
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, list.SelectedIndex())
}

func TestList_Select(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
	}

	list := NewList(items)
	list, _ = list.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press enter to select
	_, cmd := list.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should return a command that produces SelectedMsg
	if cmd != nil {
		msg := cmd()
		selectedMsg, ok := msg.(ListSelectedMsg)
		assert.True(t, ok)
		assert.Equal(t, "2", selectedMsg.Item.ID)
		assert.Equal(t, 1, selectedMsg.Index)
	}
}

func TestList_SetItems(t *testing.T) {
	t.Parallel()

	list := NewList([]ListItem{
		{ID: "1", Title: "First"},
	})

	newItems := []ListItem{
		{ID: "a", Title: "Alpha"},
		{ID: "b", Title: "Beta"},
		{ID: "c", Title: "Gamma"},
	}

	list = list.SetItems(newItems)

	assert.Equal(t, 3, len(list.Items()))
	assert.Equal(t, "a", list.SelectedItem().ID)
}

func TestList_SetSelected(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
		{ID: "3", Title: "Third"},
	}

	list := NewList(items)

	// Set to valid index
	list = list.SetSelected(2)
	assert.Equal(t, 2, list.SelectedIndex())

	// Set to invalid index (should clamp)
	list = list.SetSelected(10)
	assert.Equal(t, 2, list.SelectedIndex())

	list = list.SetSelected(-1)
	assert.Equal(t, 0, list.SelectedIndex())
}

func TestList_WithWidth(t *testing.T) {
	t.Parallel()

	list := NewList([]ListItem{{ID: "1", Title: "Test"}})
	list = list.WithWidth(80)

	assert.Equal(t, 80, list.Width())
}

func TestList_WithHeight(t *testing.T) {
	t.Parallel()

	list := NewList([]ListItem{{ID: "1", Title: "Test"}})
	list = list.WithHeight(20)

	assert.Equal(t, 20, list.Height())
}

func TestList_View(t *testing.T) {
	t.Parallel()

	items := []ListItem{
		{ID: "1", Title: "First", Description: "First item"},
		{ID: "2", Title: "Second", Description: "Second item"},
	}

	list := NewList(items)
	view := list.View()

	// Should contain item titles
	assert.Contains(t, view, "First")
	assert.Contains(t, view, "Second")
}

func TestListItem_FilterValue(t *testing.T) {
	t.Parallel()

	item := ListItem{
		ID:          "test-1",
		Title:       "Test Item",
		Description: "A test description",
	}

	// FilterValue should return title for filtering
	assert.Equal(t, "Test Item", item.FilterValue())
}
