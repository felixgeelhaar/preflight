package tui

// ReviewActionType represents the type of review action.
type ReviewActionType int

const (
	// ActionAccept represents accepting an item.
	ActionAccept ReviewActionType = iota
	// ActionReject represents rejecting an item.
	ActionReject
	// ActionLayerChange represents changing an item's target layer.
	ActionLayerChange
)

// ReviewAction represents a single action in the review history.
type ReviewAction struct {
	Type      ReviewActionType
	ItemIndex int
	Item      CaptureItem
	PrevLayer string // For layer changes, the previous layer
}

// ReviewHistory tracks review actions for undo/redo support.
type ReviewHistory struct {
	actions []ReviewAction
	cursor  int // Points to the next position for new actions
}

// NewReviewHistory creates a new ReviewHistory.
func NewReviewHistory() *ReviewHistory {
	return &ReviewHistory{
		actions: make([]ReviewAction, 0),
		cursor:  0,
	}
}

// Push adds a new action to the history.
// This clears any redo history (actions after the cursor).
func (h *ReviewHistory) Push(action ReviewAction) {
	// Truncate any redo history
	h.actions = h.actions[:h.cursor]
	// Add new action
	h.actions = append(h.actions, action)
	h.cursor++
}

// Undo returns the last action and moves the cursor back.
// Returns false if there's nothing to undo.
func (h *ReviewHistory) Undo() (ReviewAction, bool) {
	if !h.CanUndo() {
		return ReviewAction{}, false
	}
	h.cursor--
	return h.actions[h.cursor], true
}

// Redo returns the next undone action and moves the cursor forward.
// Returns false if there's nothing to redo.
func (h *ReviewHistory) Redo() (ReviewAction, bool) {
	if !h.CanRedo() {
		return ReviewAction{}, false
	}
	action := h.actions[h.cursor]
	h.cursor++
	return action, true
}

// CanUndo returns true if there are actions to undo.
func (h *ReviewHistory) CanUndo() bool {
	return h.cursor > 0
}

// CanRedo returns true if there are actions to redo.
func (h *ReviewHistory) CanRedo() bool {
	return h.cursor < len(h.actions)
}
