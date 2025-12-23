// Package components provides reusable TUI components built on Bubble Tea.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// ListItem represents a single item in the list.
type ListItem struct {
	ID          string
	Title       string
	Description string
	Value       interface{}
}

// FilterValue returns the value used for filtering.
func (i ListItem) FilterValue() string {
	return i.Title
}

// ListSelectedMsg is sent when an item is selected.
type ListSelectedMsg struct {
	Item  ListItem
	Index int
}

// List is a navigable list component.
type List struct {
	items    []ListItem
	selected int
	width    int
	height   int
	keys     ui.KeyMap
	styles   ui.Styles
}

// NewList creates a new list with the given items.
func NewList(items []ListItem) List {
	return List{
		items:    items,
		selected: 0,
		width:    40,
		height:   10,
		keys:     ui.DefaultKeyMap(),
		styles:   ui.DefaultStyles(),
	}
}

// Items returns all items in the list.
func (l List) Items() []ListItem {
	result := make([]ListItem, len(l.items))
	copy(result, l.items)
	return result
}

// SelectedIndex returns the currently selected index.
func (l List) SelectedIndex() int {
	return l.selected
}

// SelectedItem returns the currently selected item, or nil if empty.
func (l List) SelectedItem() *ListItem {
	if len(l.items) == 0 {
		return nil
	}
	item := l.items[l.selected]
	return &item
}

// SetItems replaces the list items.
func (l List) SetItems(items []ListItem) List {
	l.items = items
	l.selected = 0
	return l
}

// SetSelected sets the selected index, clamping to valid range.
func (l List) SetSelected(index int) List {
	if index < 0 {
		index = 0
	}
	if index >= len(l.items) && len(l.items) > 0 {
		index = len(l.items) - 1
	}
	l.selected = index
	return l
}

// Width returns the list width.
func (l List) Width() int {
	return l.width
}

// Height returns the list height.
func (l List) Height() int {
	return l.height
}

// WithWidth returns the list with a new width.
func (l List) WithWidth(width int) List {
	l.width = width
	return l
}

// WithHeight returns the list with a new height.
func (l List) WithHeight(height int) List {
	l.height = height
	return l
}

// WithStyles returns the list with custom styles.
func (l List) WithStyles(styles ui.Styles) List {
	l.styles = styles
	return l
}

// Init implements tea.Model.
func (l List) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (l List) Update(msg tea.Msg) (List, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return l.handleKeyMsg(msg)
	}
	return l, nil
}

func (l List) handleKeyMsg(msg tea.KeyMsg) (List, tea.Cmd) {
	if len(l.items) == 0 {
		return l, nil
	}

	switch {
	case key.Matches(msg, l.keys.Up) || key.Matches(msg, l.keys.VimUp):
		if l.selected > 0 {
			l.selected--
		}
	case key.Matches(msg, l.keys.Down) || key.Matches(msg, l.keys.VimDown):
		if l.selected < len(l.items)-1 {
			l.selected++
		}
	case key.Matches(msg, l.keys.Home):
		l.selected = 0
	case key.Matches(msg, l.keys.End):
		l.selected = len(l.items) - 1
	case key.Matches(msg, l.keys.Select):
		return l, l.selectCmd()
	}

	return l, nil
}

func (l List) selectCmd() tea.Cmd {
	if len(l.items) == 0 {
		return nil
	}
	item := l.items[l.selected]
	index := l.selected
	return func() tea.Msg {
		return ListSelectedMsg{
			Item:  item,
			Index: index,
		}
	}
}

// View implements tea.Model.
func (l List) View() string {
	if len(l.items) == 0 {
		return l.styles.Help.Render("No items")
	}

	var b strings.Builder

	// Calculate visible range
	visibleCount := l.height
	if visibleCount > len(l.items) {
		visibleCount = len(l.items)
	}

	start := 0
	if l.selected >= visibleCount {
		start = l.selected - visibleCount + 1
	}

	end := start + visibleCount
	if end > len(l.items) {
		end = len(l.items)
	}

	for i := start; i < end; i++ {
		item := l.items[i]

		var style lipgloss.Style
		if i == l.selected {
			style = l.styles.ListItemActive
			b.WriteString(style.Render("▸ " + item.Title))
		} else {
			style = l.styles.ListItem
			b.WriteString(style.Render("  " + item.Title))
		}

		if item.Description != "" && i == l.selected {
			b.WriteString("\n")
			b.WriteString(l.styles.Help.Render("    " + item.Description))
		}

		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
