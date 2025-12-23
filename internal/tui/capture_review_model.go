package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// CaptureType represents the type of captured item.
type CaptureType string

// CaptureType constants define the types of items that can be captured.
const (
	CaptureTypeFormula   CaptureType = "formula"
	CaptureTypeCask      CaptureType = "cask"
	CaptureTypeTap       CaptureType = "tap"
	CaptureTypeFile      CaptureType = "file"
	CaptureTypeRuntime   CaptureType = "runtime"
	CaptureTypeSSH       CaptureType = "ssh"
	CaptureTypeGit       CaptureType = "git"
	CaptureTypeNvim      CaptureType = "nvim"
	CaptureTypeExtension CaptureType = "extension"
	CaptureTypeShell     CaptureType = "shell"
)

// CaptureItem represents an item discovered during capture.
type CaptureItem struct {
	Category string
	Name     string
	Type     CaptureType
	Details  string
	Value    string
}

// captureReviewModel is the Bubble Tea model for capture review.
type captureReviewModel struct {
	items     []CaptureItem
	options   CaptureReviewOptions
	styles    ui.Styles
	width     int
	height    int
	cursor    int
	accepted  []CaptureItem
	rejected  []CaptureItem
	done      bool
	cancelled bool
}

// newCaptureReviewModel creates a new capture review model.
func newCaptureReviewModel(items []CaptureItem, opts CaptureReviewOptions) captureReviewModel {
	styles := ui.DefaultStyles()

	return captureReviewModel{
		items:    items,
		options:  opts,
		styles:   styles,
		width:    80,
		height:   24,
		cursor:   0,
		accepted: make([]CaptureItem, 0),
		rejected: make([]CaptureItem, 0),
	}
}

// Init initializes the model.
func (m captureReviewModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m captureReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		// Handle quit keys
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "q" {
			m.cancelled = true
			return m, tea.Quit
		}

		// Handle navigation
		if msg.Type == tea.KeyUp || msg.String() == "k" {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}

		if msg.Type == tea.KeyDown || msg.String() == "j" {
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil
		}

		// Handle accept/reject actions
		switch msg.String() {
		case "y":
			return m.acceptCurrent()
		case "n":
			return m.rejectCurrent()
		case "a":
			return m.acceptAll()
		case "d":
			return m.rejectAll()
		}
	}

	return m, nil
}

// acceptCurrent accepts the current item and moves to next.
func (m captureReviewModel) acceptCurrent() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.items) {
		item := m.items[m.cursor]
		m.accepted = append(m.accepted, item)
		return m.advanceCursor()
	}
	return m, nil
}

// rejectCurrent rejects the current item and moves to next.
func (m captureReviewModel) rejectCurrent() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.items) {
		item := m.items[m.cursor]
		m.rejected = append(m.rejected, item)
		return m.advanceCursor()
	}
	return m, nil
}

// advanceCursor moves to the next unreviewed item.
func (m captureReviewModel) advanceCursor() (tea.Model, tea.Cmd) {
	// Check if all items have been reviewed
	totalReviewed := len(m.accepted) + len(m.rejected)
	if totalReviewed >= len(m.items) {
		m.done = true
		return m, tea.Quit
	}

	// Move cursor to next unreviewed item
	for m.cursor < len(m.items)-1 {
		m.cursor++
		if !m.isItemReviewed(m.cursor) {
			break
		}
	}

	// If cursor is at an already reviewed item, try to find any unreviewed
	if m.isItemReviewed(m.cursor) {
		for i := 0; i < len(m.items); i++ {
			if !m.isItemReviewed(i) {
				m.cursor = i
				break
			}
		}
	}

	return m, nil
}

// isItemReviewed checks if an item at the given index has been reviewed.
func (m captureReviewModel) isItemReviewed(index int) bool {
	if index >= len(m.items) {
		return true
	}
	item := m.items[index]
	for _, a := range m.accepted {
		if a.Name == item.Name && a.Category == item.Category {
			return true
		}
	}
	for _, r := range m.rejected {
		if r.Name == item.Name && r.Category == item.Category {
			return true
		}
	}
	return false
}

// acceptAll accepts all remaining unreviewed items.
func (m captureReviewModel) acceptAll() (tea.Model, tea.Cmd) {
	for i, item := range m.items {
		if !m.isItemReviewed(i) {
			m.accepted = append(m.accepted, item)
		}
	}
	m.done = true
	return m, tea.Quit
}

// rejectAll rejects all remaining unreviewed items.
func (m captureReviewModel) rejectAll() (tea.Model, tea.Cmd) {
	for i, item := range m.items {
		if !m.isItemReviewed(i) {
			m.rejected = append(m.rejected, item)
		}
	}
	m.done = true
	return m, tea.Quit
}

// View renders the model.
func (m captureReviewModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Capture Review")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle empty items
	if len(m.items) == 0 {
		noItems := m.styles.Help.Render("Nothing captured. Your system scan found no items to review.")
		b.WriteString(noItems)
		return b.String()
	}

	// Progress summary
	acceptedCount := len(m.accepted)
	rejectedCount := len(m.rejected)
	remaining := len(m.items) - acceptedCount - rejectedCount

	summaryParts := make([]string, 0, 3)
	if acceptedCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d accepted", acceptedCount))
	}
	if rejectedCount > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d rejected", rejectedCount))
	}
	summaryParts = append(summaryParts, fmt.Sprintf("%d remaining", remaining))

	summaryLine := fmt.Sprintf("Review progress: %s", strings.Join(summaryParts, ", "))
	b.WriteString(m.styles.Help.Render(summaryLine))
	b.WriteString("\n\n")

	// Items list
	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		status := m.formatItemStatus(i)
		line := fmt.Sprintf("%s%s [%s] %s", prefix, status, item.Category, item.Name)

		// Highlight selected line
		if i == m.cursor {
			line = m.styles.ListItemActive.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Show details for selected item
		if i == m.cursor && item.Details != "" {
			details := fmt.Sprintf("      %s", item.Details)
			b.WriteString(m.styles.Help.Render(details))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Footer with keybindings
	helpItems := []string{"y accept", "n reject", "a accept all", "d reject all", "j/k navigate", "q quit"}
	help := m.styles.Help.Render(strings.Join(helpItems, " • "))
	b.WriteString(help)

	return b.String()
}

// formatItemStatus returns a formatted status indicator for an item.
func (m captureReviewModel) formatItemStatus(index int) string {
	if index >= len(m.items) {
		return "?"
	}
	item := m.items[index]

	// Check if accepted
	for _, a := range m.accepted {
		if a.Name == item.Name && a.Category == item.Category {
			return m.styles.Success.Render("+")
		}
	}

	// Check if rejected
	for _, r := range m.rejected {
		if r.Name == item.Name && r.Category == item.Category {
			return m.styles.Error.Render("-")
		}
	}

	// Pending
	return m.styles.Help.Render("?")
}
