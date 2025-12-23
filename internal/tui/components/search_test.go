package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewSearch(t *testing.T) {
	t.Parallel()

	search := NewSearch()

	assert.Empty(t, search.Value())
	assert.Equal(t, "Filter...", search.Placeholder())
	assert.False(t, search.Focused())
}

func TestSearch_SetValue(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("test query")

	assert.Equal(t, "test query", search.Value())
}

func TestSearch_SetPlaceholder(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetPlaceholder("Search presets...")

	assert.Equal(t, "Search presets...", search.Placeholder())
}

func TestSearch_Focus(t *testing.T) {
	t.Parallel()

	search := NewSearch().Focus()

	assert.True(t, search.Focused())
}

func TestSearch_Blur(t *testing.T) {
	t.Parallel()

	search := NewSearch().Focus().Blur()

	assert.False(t, search.Focused())
}

func TestSearch_Typing(t *testing.T) {
	t.Parallel()

	search := NewSearch().Focus()

	// Type some characters
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	assert.Equal(t, "hi", search.Value())
}

func TestSearch_Backspace(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("hello").Focus()

	// Backspace
	search, _ = search.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	assert.Equal(t, "hell", search.Value())
}

func TestSearch_Clear(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("some query")
	search = search.Clear()

	assert.Empty(t, search.Value())
}

func TestSearch_Submit(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("query").Focus()

	// Press enter to submit
	_, cmd := search.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		msg := cmd()
		result, ok := msg.(SearchSubmitMsg)
		assert.True(t, ok)
		assert.Equal(t, "query", result.Query)
	}
}

func TestSearch_Cancel(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("query").Focus()

	// Press escape to cancel
	search, cmd := search.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, search.Focused())
	if cmd != nil {
		msg := cmd()
		_, ok := msg.(SearchCancelMsg)
		assert.True(t, ok)
	}
}

func TestSearch_View(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetPlaceholder("Type to filter...")

	view := search.View()

	assert.Contains(t, view, "Type to filter...")
}

func TestSearch_ViewWithValue(t *testing.T) {
	t.Parallel()

	search := NewSearch().SetValue("nvim")

	view := search.View()

	assert.Contains(t, view, "nvim")
}

func TestSearch_Width(t *testing.T) {
	t.Parallel()

	search := NewSearch().WithWidth(50)

	assert.Equal(t, 50, search.Width())
}

func TestSearch_OnChange(t *testing.T) {
	t.Parallel()

	search := NewSearch().Focus()

	// Type a character
	search, cmd := search.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Verify value was updated
	assert.Equal(t, "a", search.Value())

	// Should produce a command (batched with change message)
	assert.NotNil(t, cmd)
}
