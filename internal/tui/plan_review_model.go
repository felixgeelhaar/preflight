package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// planReviewModel is the Bubble Tea model for plan review.
type planReviewModel struct {
	plan      *execution.Plan
	options   PlanReviewOptions
	list      components.List
	explain   components.Explain
	styles    ui.Styles
	keys      ui.KeyMap
	width     int
	height    int
	approved  bool
	cancelled bool
}

// newPlanReviewModel creates a new plan review model.
func newPlanReviewModel(plan *execution.Plan, opts PlanReviewOptions) planReviewModel {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	// Convert plan entries to list items
	items := planEntriesToListItems(plan.Entries())

	// Create list component
	list := components.NewList(items).WithStyles(styles)

	// Create explain panel
	explain := components.NewExplain()
	if opts.ShowExplanations && len(plan.Entries()) > 0 {
		explain = updateExplanationForEntry(explain, plan.Entries()[0])
	}

	return planReviewModel{
		plan:    plan,
		options: opts,
		list:    list,
		explain: explain,
		styles:  styles,
		keys:    keys,
		width:   80,
		height:  24,
	}
}

// Cursor returns the current cursor position (for testing).
func (m planReviewModel) Cursor() int {
	return m.list.SelectedIndex()
}

// Init initializes the model.
func (m planReviewModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m planReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		m.updateComponentSizes()
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC, msg.Type == tea.KeyEsc, msg.String() == "q":
			m.cancelled = true
			return m, tea.Quit

		case msg.Type == tea.KeyEnter, msg.String() == "a":
			m.approved = true
			return m, tea.Quit

		case msg.Type == tea.KeyUp, msg.String() == "k":
			currentIdx := m.list.SelectedIndex()
			if currentIdx > 0 {
				m.list = m.list.SetSelected(currentIdx - 1)
				m.updateExplanation()
			}
			return m, nil

		case msg.Type == tea.KeyDown, msg.String() == "j":
			currentIdx := m.list.SelectedIndex()
			if currentIdx < len(m.plan.Entries())-1 {
				m.list = m.list.SetSelected(currentIdx + 1)
				m.updateExplanation()
			}
			return m, nil
		}

	case components.ListSelectedMsg:
		// Item selected, update explanation
		m.updateExplanation()
		return m, nil
	}

	return m, nil
}

// View renders the model.
func (m planReviewModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Plan Review")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle empty plan
	if m.plan.IsEmpty() || !m.plan.HasChanges() {
		noChanges := m.styles.Help.Render("No changes needed. Your system is up to date.")
		b.WriteString(noChanges)
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("Press q or Esc to exit"))
		return b.String()
	}

	// Summary
	summary := m.plan.Summary()
	summaryLine := fmt.Sprintf("Steps: %d total, %d to apply, %d satisfied",
		summary.Total, summary.NeedsApply, summary.Satisfied)
	b.WriteString(m.styles.Help.Render(summaryLine))
	b.WriteString("\n\n")

	// Calculate available height for content
	headerHeight := 4 // header + summary + spacing
	footerHeight := 3 // help line + spacing
	contentHeight := m.height - headerHeight - footerHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Render split view (list on left, explain on right)
	if m.options.ShowExplanations && m.width > 60 {
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth - 3 // -3 for separator

		leftContent := m.renderList(leftWidth, contentHeight)
		rightContent := m.renderExplain(rightWidth, contentHeight)

		// Join horizontally with separator
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(strings.Repeat("│\n", contentHeight))

		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftContent,
			" "+separator+" ",
			rightContent,
		)
		b.WriteString(row)
	} else {
		// Narrow view: just show list
		b.WriteString(m.renderList(m.width, contentHeight))
	}

	b.WriteString("\n\n")

	// Footer with keybindings
	help := m.styles.Help.Render("↑/k up • ↓/j down • a/Enter approve • q/Esc cancel")
	b.WriteString(help)

	return b.String()
}

// renderList renders the step list.
func (m planReviewModel) renderList(width, height int) string {
	entries := m.plan.Entries()
	cursor := m.list.SelectedIndex()

	lines := make([]string, 0, len(entries))
	for i, entry := range entries {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		status := statusIndicator(entry.Status())
		stepID := entry.Step().ID().String()

		line := fmt.Sprintf("%s%s %s", prefix, status, stepID)

		// Add diff summary if available
		diff := entry.Diff()
		if !diff.IsEmpty() {
			line += fmt.Sprintf(" (%s)", diff.Summary())
		}

		// Truncate if too long
		if len(line) > width-2 {
			line = line[:width-5] + "..."
		}

		// Highlight selected line
		if i == cursor {
			line = m.styles.ListItemActive.Render(line)
		}

		lines = append(lines, line)
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

// renderExplain renders the explanation panel.
func (m planReviewModel) renderExplain(width, height int) string {
	var lines []string

	// Title
	title := m.styles.Subtitle.Render("Details")
	lines = append(lines, title)
	lines = append(lines, strings.Repeat("─", min(width, 40)))

	// Explanation content
	for _, section := range m.explain.Sections() {
		if section.Title != "" {
			lines = append(lines, "")
			lines = append(lines, m.styles.Subtitle.Render(section.Title))
		}
		// Wrap content to width
		wrapped := wrapText(section.Content, width-2)
		lines = append(lines, wrapped...)
	}

	// Links
	links := m.explain.Links()
	if len(links) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.styles.Subtitle.Render("Documentation"))
		for name, url := range links {
			lines = append(lines, "  "+m.styles.Help.Render(name+": "+url))
		}
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	// Truncate if too many lines
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// updateComponentSizes updates component sizes after window resize.
func (m *planReviewModel) updateComponentSizes() {
	// Update list width
	if m.options.ShowExplanations && m.width > 60 {
		m.list = m.list.WithWidth(m.width / 2)
	} else {
		m.list = m.list.WithWidth(m.width)
	}
}

// updateExplanation updates the explanation panel for the current selection.
func (m *planReviewModel) updateExplanation() {
	if !m.options.ShowExplanations {
		return
	}

	entries := m.plan.Entries()
	cursor := m.list.SelectedIndex()
	if cursor >= 0 && cursor < len(entries) {
		m.explain = updateExplanationForEntry(m.explain, entries[cursor])
	}
}

// Helper functions

// planEntriesToListItems converts plan entries to list items.
func planEntriesToListItems(entries []execution.PlanEntry) []components.ListItem {
	items := make([]components.ListItem, len(entries))
	for i, entry := range entries {
		status := statusIndicator(entry.Status())
		items[i] = components.ListItem{
			ID:          entry.Step().ID().String(),
			Title:       status + " " + entry.Step().ID().String(),
			Description: entry.Diff().Summary(),
			Value:       entry.Step().ID().String(),
		}
	}
	return items
}

// statusIndicator returns a visual indicator for step status.
func statusIndicator(status compiler.StepStatus) string {
	switch status {
	case compiler.StatusNeedsApply:
		return "+"
	case compiler.StatusSatisfied:
		return "✓"
	case compiler.StatusFailed:
		return "✗"
	case compiler.StatusSkipped:
		return "-"
	case compiler.StatusUnknown:
		return "?"
	}
	return "?"
}

// updateExplanationForEntry updates the explain panel with entry details.
func updateExplanationForEntry(explain components.Explain, entry execution.PlanEntry) components.Explain {
	step := entry.Step()

	// Get explanation from step if available
	sections := []components.ExplainSection{
		{
			Title:   "Step",
			Content: step.ID().String(),
		},
		{
			Title:   "Status",
			Content: string(entry.Status()),
		},
	}

	diff := entry.Diff()
	if !diff.IsEmpty() {
		sections = append(sections, components.ExplainSection{
			Title:   "Change",
			Content: diff.Summary(),
		})
	}

	return explain.SetSections(sections)
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		lines = []string{""}
	}

	return lines
}
