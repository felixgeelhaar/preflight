package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewHistory_Push(t *testing.T) {
	t.Parallel()

	t.Run("push adds action to history", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		action := ReviewAction{
			Type:      ActionAccept,
			ItemIndex: 0,
			Item:      CaptureItem{Name: "git", Category: "brew"},
		}
		h.Push(action)

		assert.True(t, h.CanUndo())
		assert.False(t, h.CanRedo())
	})

	t.Run("push clears redo stack", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		// Push, undo, then push new action
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		h.Undo()
		assert.True(t, h.CanRedo())

		h.Push(ReviewAction{Type: ActionReject, ItemIndex: 1})
		assert.False(t, h.CanRedo()) // Redo stack should be cleared
	})
}

func TestReviewHistory_Undo(t *testing.T) {
	t.Parallel()

	t.Run("undo returns last action", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		action := ReviewAction{
			Type:      ActionAccept,
			ItemIndex: 0,
			Item:      CaptureItem{Name: "git", Category: "brew"},
		}
		h.Push(action)

		undone, ok := h.Undo()
		assert.True(t, ok)
		assert.Equal(t, ActionAccept, undone.Type)
		assert.Equal(t, "git", undone.Item.Name)
	})

	t.Run("undo returns false when empty", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		_, ok := h.Undo()
		assert.False(t, ok)
	})

	t.Run("undo multiple actions in order", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0, Item: CaptureItem{Name: "first"}})
		h.Push(ReviewAction{Type: ActionReject, ItemIndex: 1, Item: CaptureItem{Name: "second"}})
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 2, Item: CaptureItem{Name: "third"}})

		undone, _ := h.Undo()
		assert.Equal(t, "third", undone.Item.Name)

		undone, _ = h.Undo()
		assert.Equal(t, "second", undone.Item.Name)

		undone, _ = h.Undo()
		assert.Equal(t, "first", undone.Item.Name)

		_, ok := h.Undo()
		assert.False(t, ok)
	})
}

func TestReviewHistory_Redo(t *testing.T) {
	t.Parallel()

	t.Run("redo returns undone action", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		action := ReviewAction{
			Type:      ActionAccept,
			ItemIndex: 0,
			Item:      CaptureItem{Name: "git", Category: "brew"},
		}
		h.Push(action)
		h.Undo()

		redone, ok := h.Redo()
		assert.True(t, ok)
		assert.Equal(t, ActionAccept, redone.Type)
		assert.Equal(t, "git", redone.Item.Name)
	})

	t.Run("redo returns false when nothing to redo", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		_, ok := h.Redo()
		assert.False(t, ok)

		// Also test after push without undo
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		_, ok = h.Redo()
		assert.False(t, ok)
	})

	t.Run("redo multiple actions in order", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0, Item: CaptureItem{Name: "first"}})
		h.Push(ReviewAction{Type: ActionReject, ItemIndex: 1, Item: CaptureItem{Name: "second"}})

		h.Undo()
		h.Undo()

		redone, _ := h.Redo()
		assert.Equal(t, "first", redone.Item.Name)

		redone, _ = h.Redo()
		assert.Equal(t, "second", redone.Item.Name)

		_, ok := h.Redo()
		assert.False(t, ok)
	})
}

func TestReviewHistory_CanUndo(t *testing.T) {
	t.Parallel()

	t.Run("false when empty", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		assert.False(t, h.CanUndo())
	})

	t.Run("true after push", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		assert.True(t, h.CanUndo())
	})

	t.Run("false after undoing all", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		h.Undo()
		assert.False(t, h.CanUndo())
	})
}

func TestReviewHistory_CanRedo(t *testing.T) {
	t.Parallel()

	t.Run("false when empty", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		assert.False(t, h.CanRedo())
	})

	t.Run("false after push", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		assert.False(t, h.CanRedo())
	})

	t.Run("true after undo", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()
		h.Push(ReviewAction{Type: ActionAccept, ItemIndex: 0})
		h.Undo()
		assert.True(t, h.CanRedo())
	})
}

func TestReviewHistory_LayerChange(t *testing.T) {
	t.Parallel()

	t.Run("tracks layer changes with previous layer", func(t *testing.T) {
		t.Parallel()
		h := NewReviewHistory()

		action := ReviewAction{
			Type:      ActionLayerChange,
			ItemIndex: 0,
			Item:      CaptureItem{Name: "git", Layer: "base"},
			PrevLayer: "captured",
		}
		h.Push(action)

		undone, ok := h.Undo()
		assert.True(t, ok)
		assert.Equal(t, ActionLayerChange, undone.Type)
		assert.Equal(t, "captured", undone.PrevLayer)
		assert.Equal(t, "base", undone.Item.Layer)
	})
}
