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

func TestCaptureReviewModel_Undo(t *testing.T) {
	t.Parallel()

	t.Run("undo restores accepted item", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Accept first item
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := newModel.(captureReviewModel)
		assert.Len(t, m.accepted, 1, "should have one accepted item")

		// Undo
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)

		assert.Len(t, m.accepted, 0, "accepted should be empty after undo")
		assert.Equal(t, 0, m.cursor, "cursor should return to undone item")
	})

	t.Run("undo restores rejected item", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Reject first item
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		m := newModel.(captureReviewModel)
		assert.Len(t, m.rejected, 1, "should have one rejected item")

		// Undo
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)

		assert.Len(t, m.rejected, 0, "rejected should be empty after undo")
		assert.Equal(t, 0, m.cursor, "cursor should return to undone item")
	})

	t.Run("undo multiple actions in order", func(t *testing.T) {
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

		assert.Len(t, m.accepted, 1)
		assert.Len(t, m.rejected, 1)

		// Undo reject
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)
		assert.Len(t, m.rejected, 0, "should undo reject first")
		assert.Len(t, m.accepted, 1, "accept should remain")
		assert.Equal(t, 1, m.cursor, "cursor should be at second item")

		// Undo accept
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)
		assert.Len(t, m.accepted, 0, "should undo accept")
		assert.Equal(t, 0, m.cursor, "cursor should be at first item")
	})

	t.Run("undo does nothing when history empty", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.cursor = 1 // Move cursor

		// Undo with nothing to undo
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 1, m.cursor, "cursor should not change")
		assert.Len(t, m.accepted, 0, "accepted should remain empty")
	})
}

func TestCaptureReviewModel_Redo(t *testing.T) {
	t.Parallel()

	t.Run("redo restores undone action", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Accept, undo, then redo
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := newModel.(captureReviewModel)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)
		assert.Len(t, m.accepted, 0)

		// Redo
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		m = newModel.(captureReviewModel)

		assert.Len(t, m.accepted, 1, "should restore accepted item")
		assert.Equal(t, 1, m.cursor, "cursor should advance after redo")
	})

	t.Run("redo does nothing when nothing to redo", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Redo with nothing to redo
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 0, m.cursor, "cursor should not change")
		assert.Len(t, m.accepted, 0, "accepted should remain empty")
	})

	t.Run("new action clears redo stack", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Accept, undo, then perform new action
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := newModel.(captureReviewModel)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = newModel.(captureReviewModel)

		// New action should clear redo stack
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		m = newModel.(captureReviewModel)

		// Try to redo (should do nothing)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
		m = newModel.(captureReviewModel)

		assert.Len(t, m.rejected, 1, "only the new rejection should exist")
		assert.Len(t, m.accepted, 0, "original accept should not be restored via redo")
	})
}

func TestCaptureReviewModel_UndoRedoWithHistory(t *testing.T) {
	t.Parallel()

	t.Run("model has history after initialization", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		assert.NotNil(t, model.history, "model should have history initialized")
		assert.False(t, model.history.CanUndo(), "should not be able to undo initially")
		assert.False(t, model.history.CanRedo(), "should not be able to redo initially")
	})

	t.Run("history tracks actions", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		// Accept first item
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		m := newModel.(captureReviewModel)

		assert.True(t, m.history.CanUndo(), "should be able to undo after action")
	})
}

func TestCaptureReviewModel_SearchActivation(t *testing.T) {
	t.Parallel()

	t.Run("slash key activates search", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24

		assert.False(t, model.searchActive, "search should be inactive initially")

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m := newModel.(captureReviewModel)

		assert.True(t, m.searchActive, "search should be active after /")
	})

	t.Run("escape key deactivates search", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.searchActive = true

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := newModel.(captureReviewModel)

		assert.False(t, m.searchActive, "search should be deactivated on Esc")
	})
}

func TestCaptureReviewModel_FilterItems(t *testing.T) {
	t.Parallel()

	t.Run("filters by name", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		filtered := model.filterItems("git")

		assert.Len(t, filtered, 2, "should match 'git' and '~/.gitconfig'")
		assert.Contains(t, filtered, 0, "should include git (index 0)")
		assert.Contains(t, filtered, 2, "should include ~/.gitconfig (index 2)")
	})

	t.Run("filters by category", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		filtered := model.filterItems("brew")

		assert.Len(t, filtered, 2, "should match brew category items")
		assert.Contains(t, filtered, 0, "should include git (brew)")
		assert.Contains(t, filtered, 1, "should include neovim (brew)")
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		filtered := model.filterItems("NEOVIM")

		assert.Len(t, filtered, 1, "should match neovim case-insensitively")
		assert.Contains(t, filtered, 1, "should include neovim")
	})

	t.Run("empty query returns all items", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		filtered := model.filterItems("")

		assert.Nil(t, filtered, "empty query should return nil (no filter)")
	})

	t.Run("no matches returns empty slice", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})

		filtered := model.filterItems("nonexistent")

		assert.NotNil(t, filtered, "should return non-nil slice")
		assert.Len(t, filtered, 0, "should return empty slice for no matches")
	})
}

func TestCaptureReviewModel_FilteredNavigation(t *testing.T) {
	t.Parallel()

	t.Run("navigation respects filter", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.filteredIdx = []int{0, 2} // Only git and ~/.gitconfig visible

		// Start at first filtered item
		model.cursor = 0

		// Navigate down should skip neovim (index 1)
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 2, m.cursor, "cursor should skip to next filtered item")
	})
}

func TestCaptureReviewModel_ViewWithFilter(t *testing.T) {
	t.Parallel()

	t.Run("view shows search input when active", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.searchActive = true

		view := model.View()

		assert.Contains(t, view, "Filter", "should show filter prompt when search active")
	})
}

func TestCaptureReviewModel_GoToTop(t *testing.T) {
	t.Parallel()

	t.Run("g moves cursor to first item", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.cursor = 2 // Start at last item

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 0, m.cursor, "cursor should move to first item")
	})

	t.Run("g moves to first filtered item when filter active", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.filteredIdx = []int{1, 2} // Only neovim and .gitconfig
		model.cursor = 2

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 1, m.cursor, "cursor should move to first filtered item")
	})
}

func TestCaptureReviewModel_GoToBottom(t *testing.T) {
	t.Parallel()

	t.Run("G moves cursor to last item", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.cursor = 0 // Start at first item

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 2, m.cursor, "cursor should move to last item")
	})

	t.Run("G moves to last filtered item when filter active", func(t *testing.T) {
		t.Parallel()
		items := createTestCaptureItemsMultiple(t)
		model := newCaptureReviewModel(items, CaptureReviewOptions{Interactive: true})
		model.width = 100
		model.height = 24
		model.filteredIdx = []int{0, 1} // Only git and neovim
		model.cursor = 0

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		m := newModel.(captureReviewModel)

		assert.Equal(t, 1, m.cursor, "cursor should move to last filtered item")
	})
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
