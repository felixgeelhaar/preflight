package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestCaptureReviewModel_Init(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItems(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
}

func TestCaptureReviewModel_View(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItems(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Capture Review", "should contain header")
}

func TestCaptureReviewModel_EmptyItems(t *testing.T) {
	t.Parallel()

	model := newCaptureReviewModel([]CaptureItem{}, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Nothing captured", "should show empty message")
}

func TestCaptureReviewModel_Navigation(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Initial cursor should be at 0
	assert.Equal(t, 0, model.cursor)

	// Navigate down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(captureReviewModel)
	assert.Equal(t, 1, m.cursor)

	// Navigate up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(captureReviewModel)
	assert.Equal(t, 0, m.cursor)
}

func TestCaptureReviewModel_AcceptItem(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Press 'y' to accept current item
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(captureReviewModel)

	assert.Len(t, m.accepted, 1, "should have one accepted item")
	assert.Equal(t, 1, m.cursor, "cursor should advance to next item")
}

func TestCaptureReviewModel_RejectItem(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Press 'n' to reject current item
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := newModel.(captureReviewModel)

	assert.Len(t, m.rejected, 1, "should have one rejected item")
	assert.Equal(t, 1, m.cursor, "cursor should advance to next item")
}

func TestCaptureReviewModel_AcceptAll(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Press 'a' to accept all remaining items
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m := newModel.(captureReviewModel)

	assert.Len(t, m.accepted, 3, "should have all items accepted")
	assert.True(t, m.done, "should be done")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestCaptureReviewModel_RejectAll(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Press 'd' to reject all remaining items
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m := newModel.(captureReviewModel)

	assert.Len(t, m.rejected, 3, "should have all items rejected")
	assert.True(t, m.done, "should be done")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestCaptureReviewModel_WindowResize(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItems(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(captureReviewModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestCaptureReviewModel_Quit(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItems(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Press 'q' to quit
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(captureReviewModel)

	assert.True(t, m.cancelled, "should be cancelled")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestCaptureReviewModel_AutoComplete(t *testing.T) {
	t.Parallel()

	items := []CaptureItem{
		{Category: "brew", Name: "git", Type: CaptureTypeFormula},
	}
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Accept the only item
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(captureReviewModel)

	assert.True(t, m.done, "should be done when all items reviewed")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestCaptureReviewModel_ShowDetails(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItems(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "brew", "should show item category")
	assert.Contains(t, view, "git", "should show item name")
}

func TestCaptureReviewModel_StatusIndicators(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Accept first, reject second
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(captureReviewModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(captureReviewModel)

	view := m.View()

	// Should show summary of decisions
	assert.Contains(t, view, "1 accepted", "should show accepted count")
	assert.Contains(t, view, "1 rejected", "should show rejected count")
}

// Helper functions to create test capture items

func createTestCaptureItems(t *testing.T) []CaptureItem {
	t.Helper()

	return []CaptureItem{
		{
			Category: "brew",
			Name:     "git",
			Type:     CaptureTypeFormula,
			Details:  "Git version control",
		},
	}
}

func createTestCaptureItemsMultiple(t *testing.T) []CaptureItem {
	t.Helper()

	return []CaptureItem{
		{
			Category: "brew",
			Name:     "git",
			Type:     CaptureTypeFormula,
			Details:  "Git version control",
		},
		{
			Category: "brew",
			Name:     "neovim",
			Type:     CaptureTypeFormula,
			Details:  "Neovim editor",
		},
		{
			Category: "files",
			Name:     "~/.gitconfig",
			Type:     CaptureTypeFile,
			Details:  "Git configuration file",
		},
	}
}
