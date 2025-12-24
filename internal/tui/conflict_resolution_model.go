package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/merge"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// ConflictResolutionOptions configures the conflict resolution TUI.
type ConflictResolutionOptions struct {
	ShowBase bool // Show base (original) content in diff view
}

// ConflictResolutionResult holds the result of conflict resolution.
type ConflictResolutionResult struct {
	Resolutions []merge.Resolution
	Cancelled   bool
}

// conflictResolutionModel is the Bubble Tea model for conflict resolution.
type conflictResolutionModel struct {
	filePath        string
	conflicts       []merge.Conflict
	resolutions     []merge.Resolution
	currentConflict int
	scrollOffset    int
	options         ConflictResolutionOptions
	styles          ui.Styles
	width           int
	height          int
	done            bool
	cancelled       bool
}

// newConflictResolutionModel creates a new conflict resolution model.
func newConflictResolutionModel(filePath string, conflicts []merge.Conflict, opts ConflictResolutionOptions) conflictResolutionModel {
	styles := ui.DefaultStyles()

	return conflictResolutionModel{
		filePath:        filePath,
		conflicts:       conflicts,
		resolutions:     make([]merge.Resolution, len(conflicts)),
		currentConflict: 0,
		scrollOffset:    0,
		options:         opts,
		styles:          styles,
		width:           80,
		height:          24,
	}
}

// Init initializes the model.
func (m conflictResolutionModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m conflictResolutionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg handles key input.
func (m conflictResolutionModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // We only handle specific key types
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyUp:
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil

	case tea.KeyDown:
		m.scrollOffset++
		return m, nil

	case tea.KeyRunes:
		return m.handleRuneKey(msg)
	}

	return m, nil
}

// handleRuneKey handles character key input.
func (m conflictResolutionModel) handleRuneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.Runes) == 0 {
		return m, nil
	}

	key := msg.Runes[0]

	switch key {
	case 'q':
		m.cancelled = true
		return m, tea.Quit

	case 'n':
		// Next conflict
		if m.currentConflict < len(m.conflicts)-1 {
			m.currentConflict++
			m.scrollOffset = 0
		}
		return m, nil

	case 'p':
		// Previous conflict
		if m.currentConflict > 0 {
			m.currentConflict--
			m.scrollOffset = 0
		}
		return m, nil

	case 'o':
		// Pick ours (config)
		return m.resolveCurrentConflict(merge.ResolveOurs)

	case 't':
		// Pick theirs (file)
		return m.resolveCurrentConflict(merge.ResolveTheirs)

	case 'b':
		// Pick base (original)
		return m.resolveCurrentConflict(merge.ResolveBase)

	case 'O':
		// Resolve all with ours
		return m.resolveAllConflicts(merge.ResolveOurs)

	case 'T':
		// Resolve all with theirs
		return m.resolveAllConflicts(merge.ResolveTheirs)

	case 'B':
		// Resolve all with base
		return m.resolveAllConflicts(merge.ResolveBase)

	case 'j':
		m.scrollOffset++
		return m, nil

	case 'k':
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	}

	return m, nil
}

// resolveCurrentConflict resolves the current conflict with the given resolution.
func (m conflictResolutionModel) resolveCurrentConflict(resolution merge.Resolution) (tea.Model, tea.Cmd) {
	m.resolutions[m.currentConflict] = resolution

	// Check if all conflicts are resolved
	if m.allResolved() {
		m.done = true
		return m, tea.Quit
	}

	// Move to next unresolved conflict
	m.scrollOffset = 0
	for i := m.currentConflict + 1; i < len(m.conflicts); i++ {
		if m.resolutions[i] == "" {
			m.currentConflict = i
			return m, nil
		}
	}

	// If no more after current, check from beginning
	for i := 0; i < m.currentConflict; i++ {
		if m.resolutions[i] == "" {
			m.currentConflict = i
			return m, nil
		}
	}

	return m, nil
}

// resolveAllConflicts resolves all conflicts with the given resolution.
func (m conflictResolutionModel) resolveAllConflicts(resolution merge.Resolution) (tea.Model, tea.Cmd) {
	for i := range m.resolutions {
		m.resolutions[i] = resolution
	}
	m.done = true
	return m, tea.Quit
}

// allResolved checks if all conflicts have been resolved.
func (m conflictResolutionModel) allResolved() bool {
	for _, r := range m.resolutions {
		if r == "" {
			return false
		}
	}
	return true
}

// View renders the model.
func (m conflictResolutionModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Conflict Resolution")
	b.WriteString(header)
	b.WriteString("\n")

	// File path and conflict counter
	fileInfo := fmt.Sprintf("%s - Conflict %d/%d", m.filePath, m.currentConflict+1, len(m.conflicts))
	b.WriteString(m.styles.Subtitle.Render(fileInfo))
	b.WriteString("\n\n")

	// Resolution progress
	resolved := 0
	for _, r := range m.resolutions {
		if r != "" {
			resolved++
		}
	}
	progress := fmt.Sprintf("Resolved: %d/%d", resolved, len(m.conflicts))
	b.WriteString(m.styles.Help.Render(progress))
	b.WriteString("\n\n")

	// Current conflict view
	if m.currentConflict < len(m.conflicts) {
		conflict := m.conflicts[m.currentConflict]
		b.WriteString(m.renderConflict(conflict))
	}

	b.WriteString("\n")

	// Footer with keybindings
	helpItems := []string{
		"o pick ours",
		"t pick theirs",
		"b pick base",
		"n/p next/prev",
		"O/T/B resolve all",
		"j/k scroll",
		"q/Esc cancel",
	}
	help := m.styles.Help.Render(strings.Join(helpItems, " • "))
	b.WriteString(help)

	return b.String()
}

// renderConflict renders a conflict in a side-by-side diff view.
func (m conflictResolutionModel) renderConflict(conflict merge.Conflict) string {
	var b strings.Builder

	// Calculate panel width
	panelWidth := (m.width - 6) / 2 // Leave space for borders and separator
	if panelWidth < 20 {
		panelWidth = 20
	}

	// Render headers
	leftHeader := m.styles.DiffRemove.Render(padOrTruncate("<<<< OURS (config)", panelWidth))
	rightHeader := m.styles.DiffAdd.Render(padOrTruncate(">>>> THEIRS (file)", panelWidth))
	b.WriteString(leftHeader)
	b.WriteString(" │ ")
	b.WriteString(rightHeader)
	b.WriteString("\n")

	// Separator
	b.WriteString(m.styles.Help.Render(strings.Repeat("─", panelWidth)))
	b.WriteString("─┼─")
	b.WriteString(m.styles.Help.Render(strings.Repeat("─", panelWidth)))
	b.WriteString("\n")

	// Determine visible lines
	visibleLines := m.height - 12
	if visibleLines < 5 {
		visibleLines = 5
	}

	maxLines := max(len(conflict.Ours), len(conflict.Theirs))

	startLine := m.scrollOffset
	if startLine > maxLines-visibleLines {
		startLine = max(0, maxLines-visibleLines)
	}
	endLine := min(startLine+visibleLines, maxLines)

	// Render content lines
	for i := startLine; i < endLine; i++ {
		leftLine := ""
		rightLine := ""

		if i < len(conflict.Ours) {
			leftLine = conflict.Ours[i]
		}
		if i < len(conflict.Theirs) {
			rightLine = conflict.Theirs[i]
		}

		// Pad or truncate to panel width
		leftLine = padOrTruncate(leftLine, panelWidth)
		rightLine = padOrTruncate(rightLine, panelWidth)

		// Style based on differences
		if i < len(conflict.Ours) && i < len(conflict.Theirs) && conflict.Ours[i] != conflict.Theirs[i] {
			leftLine = m.styles.DiffRemove.Render(leftLine)
			rightLine = m.styles.DiffAdd.Render(rightLine)
		} else {
			leftLine = m.styles.Paragraph.Render(leftLine)
			rightLine = m.styles.Paragraph.Render(rightLine)
		}

		b.WriteString(leftLine)
		b.WriteString(" │ ")
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Show scroll indicator if content is scrollable
	if maxLines > visibleLines {
		scrollInfo := fmt.Sprintf("─── %d/%d lines ───", startLine+1, maxLines)
		b.WriteString(m.styles.Help.Render(scrollInfo))
		b.WriteString("\n")
	}

	// Show base content if option enabled and base exists
	if m.options.ShowBase && len(conflict.Base) > 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.DiffHeader.Render("|||| BASE (original):"))
		b.WriteString("\n")
		for i, line := range conflict.Base {
			if i >= 5 {
				remaining := len(conflict.Base) - 5
				b.WriteString(m.styles.Help.Render(fmt.Sprintf("... +%d more lines", remaining)))
				break
			}
			b.WriteString(m.styles.Help.Render("  " + line))
			b.WriteString("\n")
		}
	}

	// Show current resolution if set
	if res := m.resolutions[m.currentConflict]; res != "" {
		b.WriteString("\n")
		resolutionText := fmt.Sprintf("✓ Resolved with: %s", res)
		b.WriteString(m.styles.Success.Render(resolutionText))
	}

	return b.String()
}

// padOrTruncate pads or truncates a string to the given width.
func padOrTruncate(s string, width int) string {
	if len(s) > width {
		if width <= 3 {
			return s[:width]
		}
		return s[:width-3] + "..."
	}
	return s + strings.Repeat(" ", width-len(s))
}
