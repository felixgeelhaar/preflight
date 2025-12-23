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

func TestCaptureReviewModel_IsItemReviewed(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Before any action, no item should be reviewed
	assert.False(t, model.isItemReviewed(0), "item 0 should not be reviewed")

	// Accept first item
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(captureReviewModel)

	// Now item 0 should be reviewed
	assert.True(t, m.isItemReviewed(0), "item 0 should be reviewed after accepting")
	assert.False(t, m.isItemReviewed(1), "item 1 should not be reviewed yet")

	// Index out of bounds should return true
	assert.True(t, m.isItemReviewed(999), "out of bounds index should return true")
}

func TestCaptureReviewModel_FormatItemStatus(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Initial status should be pending
	status := model.formatItemStatus(0)
	assert.Contains(t, status, "?", "unreviewed item should show ?")

	// Accept first item
	model.accepted = append(model.accepted, items[0])
	status = model.formatItemStatus(0)
	assert.Contains(t, status, "+", "accepted item should show +")

	// Reject second item
	model.rejected = append(model.rejected, items[1])
	status = model.formatItemStatus(1)
	assert.Contains(t, status, "-", "rejected item should show -")
}

func TestCaptureReviewModel_AdvanceCursor_WrapAround(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Accept first two items
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(captureReviewModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(captureReviewModel)

	// Cursor should have found the remaining unreviewed item
	assert.Equal(t, 2, m.cursor, "cursor should be at last item")
}

func TestCaptureReviewModel_AdvanceCursor_FindFromStart(t *testing.T) {
	t.Parallel()

	items := createTestCaptureItemsMultiple(t)
	model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
	model.width = 100
	model.height = 24

	// Manually position cursor at the end
	model.cursor = 2
	// Mark last item as accepted
	model.accepted = append(model.accepted, items[2])

	// Accept item at cursor (which is already accepted, but tests the wrap around)
	// Since cursor is at end and that item is reviewed, it should wrap to find first unreviewed
	newModel, _ := model.advanceCursor()
	m := newModel.(captureReviewModel)

	// Should find the first unreviewed item (index 0)
	assert.Equal(t, 0, m.cursor, "cursor should wrap around to first unreviewed item")
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
