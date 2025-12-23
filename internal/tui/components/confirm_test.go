package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewConfirm(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Are you sure?")

	assert.Equal(t, "Are you sure?", confirm.Message())
	assert.Equal(t, "Yes", confirm.YesLabel())
	assert.Equal(t, "No", confirm.NoLabel())
	assert.True(t, confirm.Focused())
}

func TestConfirm_WithLabels(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Delete file?").
		WithYesLabel("Delete").
		WithNoLabel("Cancel")

	assert.Equal(t, "Delete", confirm.YesLabel())
	assert.Equal(t, "Cancel", confirm.NoLabel())
}

func TestConfirm_Navigation(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Confirm?")

	// Initially focused on Yes (index 0)
	assert.True(t, confirm.Focused())

	// Move right
	confirm, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.False(t, confirm.Focused())

	// Move left
	confirm, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.True(t, confirm.Focused())
}

func TestConfirm_VimNavigation(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Confirm?")

	// Move right with l
	confirm, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.False(t, confirm.Focused())

	// Move left with h
	confirm, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.True(t, confirm.Focused())
}

func TestConfirm_SelectYes(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Proceed?")

	// Should be on Yes, press enter
	_, cmd := confirm.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		msg := cmd()
		result, ok := msg.(ConfirmResultMsg)
		assert.True(t, ok)
		assert.True(t, result.Confirmed)
	} else {
		t.Fatal("Expected command to be returned")
	}
}

func TestConfirm_SelectNo(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Proceed?")

	// Move to No
	confirm, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyRight})

	// Press enter
	_, cmd := confirm.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		msg := cmd()
		result, ok := msg.(ConfirmResultMsg)
		assert.True(t, ok)
		assert.False(t, result.Confirmed)
	} else {
		t.Fatal("Expected command to be returned")
	}
}

func TestConfirm_QuickYes(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Proceed?")

	// Press 'y' for quick yes
	_, cmd := confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if cmd != nil {
		msg := cmd()
		result, ok := msg.(ConfirmResultMsg)
		assert.True(t, ok)
		assert.True(t, result.Confirmed)
	} else {
		t.Fatal("Expected command to be returned")
	}
}

func TestConfirm_QuickNo(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Proceed?")

	// Press 'n' for quick no
	_, cmd := confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if cmd != nil {
		msg := cmd()
		result, ok := msg.(ConfirmResultMsg)
		assert.True(t, ok)
		assert.False(t, result.Confirmed)
	} else {
		t.Fatal("Expected command to be returned")
	}
}

func TestConfirm_View(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Delete all files?").
		WithYesLabel("Delete").
		WithNoLabel("Cancel")

	view := confirm.View()

	assert.Contains(t, view, "Delete all files?")
	assert.Contains(t, view, "Delete")
	assert.Contains(t, view, "Cancel")
}

func TestConfirm_Width(t *testing.T) {
	t.Parallel()

	confirm := NewConfirm("Confirm?").WithWidth(60)

	assert.Equal(t, 60, confirm.Width())
}
