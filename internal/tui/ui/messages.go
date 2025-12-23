package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// WindowSizeMsg is sent when the terminal window is resized.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// ErrorMsg represents an error that occurred during processing.
type ErrorMsg struct {
	Err error
}

func (e ErrorMsg) Error() string {
	return e.Err.Error()
}

// SuccessMsg indicates a successful operation.
type SuccessMsg struct {
	Message string
}

// ProgressMsg indicates progress on an operation.
type ProgressMsg struct {
	Current int
	Total   int
	Message string
}

// CompletedMsg indicates an operation has completed.
type CompletedMsg struct {
	Result interface{}
}

// SelectedMsg indicates an item was selected.
type SelectedMsg struct {
	ID    string
	Index int
	Value interface{}
}

// ConfirmedMsg indicates a confirmation action.
type ConfirmedMsg struct {
	Confirmed bool
}

// NavigateMsg requests navigation to a different view.
type NavigateMsg struct {
	View   string
	Params map[string]interface{}
}

// RefreshMsg requests a data refresh.
type RefreshMsg struct{}

// QuitMsg requests to quit the application.
type QuitMsg struct{}

// Helper functions to create messages

// NewErrorMsg creates a new error message.
func NewErrorMsg(err error) tea.Msg {
	return ErrorMsg{Err: err}
}

// NewSuccessMsg creates a new success message.
func NewSuccessMsg(message string) tea.Msg {
	return SuccessMsg{Message: message}
}

// NewProgressMsg creates a new progress message.
func NewProgressMsg(current, total int, message string) tea.Msg {
	return ProgressMsg{
		Current: current,
		Total:   total,
		Message: message,
	}
}

// NewCompletedMsg creates a new completed message.
func NewCompletedMsg(result interface{}) tea.Msg {
	return CompletedMsg{Result: result}
}

// NewSelectedMsg creates a new selection message.
func NewSelectedMsg(id string, index int, value interface{}) tea.Msg {
	return SelectedMsg{
		ID:    id,
		Index: index,
		Value: value,
	}
}

// NewNavigateMsg creates a new navigation message.
func NewNavigateMsg(view string, params map[string]interface{}) tea.Msg {
	return NavigateMsg{
		View:   view,
		Params: params,
	}
}

// NewConfirmedMsg creates a new confirmation message.
func NewConfirmedMsg(confirmed bool) tea.Msg {
	return ConfirmedMsg{Confirmed: confirmed}
}
