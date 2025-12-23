package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/tui/common"
)

// ConfirmResultMsg is sent when the user confirms or cancels.
type ConfirmResultMsg struct {
	Confirmed bool
}

// Confirm is a yes/no confirmation dialog.
type Confirm struct {
	message  string
	yesLabel string
	noLabel  string
	focused  bool // true = yes, false = no
	width    int
	keys     common.KeyMap
	styles   common.Styles
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(message string) Confirm {
	return Confirm{
		message:  message,
		yesLabel: "Yes",
		noLabel:  "No",
		focused:  true,
		width:    40,
		keys:     common.DefaultKeyMap(),
		styles:   common.DefaultStyles(),
	}
}

// Message returns the confirmation message.
func (c Confirm) Message() string {
	return c.message
}

// YesLabel returns the yes button label.
func (c Confirm) YesLabel() string {
	return c.yesLabel
}

// NoLabel returns the no button label.
func (c Confirm) NoLabel() string {
	return c.noLabel
}

// Focused returns true if yes is focused, false if no is focused.
func (c Confirm) Focused() bool {
	return c.focused
}

// Width returns the dialog width.
func (c Confirm) Width() int {
	return c.width
}

// WithMessage sets the message.
func (c Confirm) WithMessage(message string) Confirm {
	c.message = message
	return c
}

// WithYesLabel sets the yes button label.
func (c Confirm) WithYesLabel(label string) Confirm {
	c.yesLabel = label
	return c
}

// WithNoLabel sets the no button label.
func (c Confirm) WithNoLabel(label string) Confirm {
	c.noLabel = label
	return c
}

// WithWidth sets the dialog width.
func (c Confirm) WithWidth(width int) Confirm {
	c.width = width
	return c
}

// WithStyles sets the styles.
func (c Confirm) WithStyles(styles common.Styles) Confirm {
	c.styles = styles
	return c
}

// Init implements tea.Model.
func (c Confirm) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (c Confirm) Update(msg tea.Msg) (Confirm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return c.handleKeyMsg(msg)
	}
	return c, nil
}

func (c Confirm) handleKeyMsg(msg tea.KeyMsg) (Confirm, tea.Cmd) {
	switch {
	case key.Matches(msg, c.keys.Left) || key.Matches(msg, c.keys.VimLeft):
		c.focused = true
	case key.Matches(msg, c.keys.Right) || key.Matches(msg, c.keys.VimRight):
		c.focused = false
	case key.Matches(msg, c.keys.Select):
		return c, c.confirmCmd(c.focused)
	case key.Matches(msg, c.keys.Accept): // 'y' key
		return c, c.confirmCmd(true)
	case key.Matches(msg, c.keys.Reject): // 'n' key
		return c, c.confirmCmd(false)
	case key.Matches(msg, c.keys.Cancel):
		return c, c.confirmCmd(false)
	}
	return c, nil
}

func (c Confirm) confirmCmd(confirmed bool) tea.Cmd {
	return func() tea.Msg {
		return ConfirmResultMsg{Confirmed: confirmed}
	}
}

// View renders the confirmation dialog.
func (c Confirm) View() string {
	messageStyle := c.styles.Paragraph.Width(c.width)

	// Button styles
	yesStyle := c.styles.Button
	noStyle := c.styles.Button

	if c.focused {
		yesStyle = c.styles.ButtonActive
	} else {
		noStyle = c.styles.ButtonActive
	}

	// Render message
	message := messageStyle.Render(c.message)

	// Render buttons
	yesBtn := yesStyle.Render(c.yesLabel)
	noBtn := noStyle.Render(c.noLabel)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "  ", noBtn)

	// Center buttons
	buttonRow := lipgloss.NewStyle().Width(c.width).Align(lipgloss.Center).Render(buttons)

	return lipgloss.JoinVertical(lipgloss.Left, message, "", buttonRow)
}
