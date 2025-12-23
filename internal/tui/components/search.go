package components

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// SearchSubmitMsg is sent when the user submits a search.
type SearchSubmitMsg struct {
	Query string
}

// SearchCancelMsg is sent when the user cancels search.
type SearchCancelMsg struct{}

// SearchChangeMsg is sent when the search query changes.
type SearchChangeMsg struct {
	Query string
}

// Search is a text input for filtering/searching.
type Search struct {
	input  textinput.Model
	width  int
	keys   ui.KeyMap
	styles ui.Styles
}

// NewSearch creates a new search component.
func NewSearch() Search {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.CharLimit = ui.DefaultSearchCharLimit

	return Search{
		input:  ti,
		width:  ui.DefaultWidthSmall,
		keys:   ui.DefaultKeyMap(),
		styles: ui.DefaultStyles(),
	}
}

// Value returns the current search value.
func (s Search) Value() string {
	return s.input.Value()
}

// Placeholder returns the placeholder text.
func (s Search) Placeholder() string {
	return s.input.Placeholder
}

// Focused returns true if the input is focused.
func (s Search) Focused() bool {
	return s.input.Focused()
}

// Width returns the input width.
func (s Search) Width() int {
	return s.width
}

// SetValue sets the search value.
func (s Search) SetValue(value string) Search {
	s.input.SetValue(value)
	return s
}

// SetPlaceholder sets the placeholder text.
func (s Search) SetPlaceholder(placeholder string) Search {
	s.input.Placeholder = placeholder
	return s
}

// Focus focuses the input.
func (s Search) Focus() Search {
	s.input.Focus()
	return s
}

// Blur removes focus from the input.
func (s Search) Blur() Search {
	s.input.Blur()
	return s
}

// Clear clears the search value.
func (s Search) Clear() Search {
	s.input.SetValue("")
	return s
}

// WithWidth sets the input width.
func (s Search) WithWidth(width int) Search {
	s.width = width
	s.input.Width = width - 4 // Account for prompt and padding
	return s
}

// WithStyles sets the styles.
func (s Search) WithStyles(styles ui.Styles) Search {
	s.styles = styles
	return s
}

// Init implements tea.Model.
func (s Search) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (s Search) Update(msg tea.Msg) (Search, tea.Cmd) {
	if !s.input.Focused() {
		return s, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		return s.handleKeyMsg(msg)
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s Search) handleKeyMsg(msg tea.KeyMsg) (Search, tea.Cmd) {
	switch {
	case key.Matches(msg, s.keys.Select): // Enter
		return s, s.submitCmd()
	case key.Matches(msg, s.keys.Cancel): // Escape
		s.input.Blur()
		return s, s.cancelCmd()
	default:
		// Let textinput handle the key
		prevValue := s.input.Value()
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)

		// If value changed, emit change message
		if s.input.Value() != prevValue {
			return s, tea.Batch(cmd, s.changeCmd())
		}
		return s, cmd
	}
}

func (s Search) submitCmd() tea.Cmd {
	query := s.input.Value()
	return func() tea.Msg {
		return SearchSubmitMsg{Query: query}
	}
}

func (s Search) cancelCmd() tea.Cmd {
	return func() tea.Msg {
		return SearchCancelMsg{}
	}
}

func (s Search) changeCmd() tea.Cmd {
	query := s.input.Value()
	return func() tea.Msg {
		return SearchChangeMsg{Query: query}
	}
}

// View renders the search input.
func (s Search) View() string {
	return s.styles.Panel.Width(s.width - 4).Render(s.input.View())
}
