package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMap contains all key bindings for the TUI.
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Home     key.Binding
	End      key.Binding
	PageUp   key.Binding
	PageDown key.Binding

	// Vim-style navigation
	VimUp    key.Binding
	VimDown  key.Binding
	VimLeft  key.Binding
	VimRight key.Binding

	// Selection
	Select  key.Binding
	Toggle  key.Binding
	Confirm key.Binding
	Cancel  key.Binding

	// Actions
	Accept key.Binding
	Reject key.Binding
	Skip   key.Binding
	Undo   key.Binding

	// Help and info
	Help    key.Binding
	Explain key.Binding

	// General
	Quit   key.Binding
	Filter key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Arrow key navigation
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "right"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to end"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),

		// Vim-style navigation
		VimUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "up"),
		),
		VimDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "down"),
		),
		VimLeft: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "left"),
		),
		VimRight: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "right"),
		),

		// Selection
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter", "y"),
			key.WithHelp("enter/y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "n"),
			key.WithHelp("esc/n", "cancel"),
		),

		// Actions
		Accept: key.NewBinding(
			key.WithKeys("a", "y"),
			key.WithHelp("a/y", "accept"),
		),
		Reject: key.NewBinding(
			key.WithKeys("d", "n"),
			key.WithHelp("d/n", "reject"),
		),
		Skip: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "skip"),
		),
		Undo: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "undo"),
		),

		// Help and info
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Explain: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "explain"),
		),

		// General
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
	}
}

// IsUp returns true if the key message matches an up navigation key.
func (k KeyMap) IsUp(msg tea.KeyMsg) bool {
	return key.Matches(msg, k.Up) || key.Matches(msg, k.VimUp)
}

// IsDown returns true if the key message matches a down navigation key.
func (k KeyMap) IsDown(msg tea.KeyMsg) bool {
	return key.Matches(msg, k.Down) || key.Matches(msg, k.VimDown)
}

// IsLeft returns true if the key message matches a left navigation key.
func (k KeyMap) IsLeft(msg tea.KeyMsg) bool {
	return key.Matches(msg, k.Left) || key.Matches(msg, k.VimLeft)
}

// IsRight returns true if the key message matches a right navigation key.
func (k KeyMap) IsRight(msg tea.KeyMsg) bool {
	return key.Matches(msg, k.Right) || key.Matches(msg, k.VimRight)
}
