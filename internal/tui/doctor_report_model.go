package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// IssueSeverity represents the severity of a doctor issue.
type IssueSeverity string

// IssueSeverity constants define the severity levels for doctor issues.
const (
	IssueSeverityInfo    IssueSeverity = "info"
	IssueSeverityWarning IssueSeverity = "warning"
	IssueSeverityError   IssueSeverity = "error"
)

// DoctorIssue represents an issue found by the doctor command.
type DoctorIssue struct {
	Severity   IssueSeverity
	Category   string
	Message    string
	Details    string
	CanAutoFix bool
	FixCommand string
}

// DoctorReport holds the results of a doctor check.
type DoctorReport struct {
	Issues []DoctorIssue
}

// HasIssues returns true if there are any issues.
func (r *DoctorReport) HasIssues() bool {
	return len(r.Issues) > 0
}

// FixableCount returns the number of issues that can be auto-fixed.
func (r *DoctorReport) FixableCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.CanAutoFix {
			count++
		}
	}
	return count
}

// doctorReportModel is the Bubble Tea model for doctor report.
type doctorReportModel struct {
	report  *DoctorReport
	options DoctorReportOptions
	styles  ui.Styles
	width   int
	height  int
	cursor  int
	fixing  bool
	done    bool
}

// newDoctorReportModel creates a new doctor report model.
func newDoctorReportModel(report *DoctorReport, opts DoctorReportOptions) doctorReportModel {
	styles := ui.DefaultStyles()

	return doctorReportModel{
		report:  report,
		options: opts,
		styles:  styles,
		width:   80,
		height:  24,
		cursor:  0,
	}
}

// Init initializes the model.
func (m doctorReportModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m doctorReportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC, msg.Type == tea.KeyEsc, msg.String() == "q":
			m.done = true
			return m, tea.Quit

		case msg.Type == tea.KeyUp, msg.String() == "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case msg.Type == tea.KeyDown, msg.String() == "j":
			if m.cursor < len(m.report.Issues)-1 {
				m.cursor++
			}
			return m, nil

		case msg.String() == "f":
			if m.report.FixableCount() > 0 && m.options.AutoFix {
				m.fixing = true
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the model.
func (m doctorReportModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Doctor Report")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle no issues
	if !m.report.HasIssues() {
		noIssues := m.styles.Success.Render("No issues found. Your system is healthy!")
		b.WriteString(noIssues)
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("Press q or Esc to exit"))
		return b.String()
	}

	// Summary
	fixable := m.report.FixableCount()
	summaryLine := fmt.Sprintf("Found %d issue(s)", len(m.report.Issues))
	if fixable > 0 {
		summaryLine += fmt.Sprintf(", %d fixable", fixable)
	}
	b.WriteString(m.styles.Help.Render(summaryLine))
	b.WriteString("\n\n")

	// Issues list
	for i, issue := range m.report.Issues {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		severity := m.formatSeverity(issue.Severity)
		line := fmt.Sprintf("%s%s [%s] %s", prefix, severity, issue.Category, issue.Message)

		// Highlight selected line
		if i == m.cursor {
			line = m.styles.ListItemActive.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Show details for selected issue if verbose
		if i == m.cursor && m.options.Verbose && issue.Details != "" {
			details := fmt.Sprintf("      %s", issue.Details)
			b.WriteString(m.styles.Help.Render(details))
			b.WriteString("\n")
		}

		// Show fix command if available and selected
		if i == m.cursor && issue.CanAutoFix && issue.FixCommand != "" {
			fixLine := fmt.Sprintf("      Fix: %s", issue.FixCommand)
			b.WriteString(m.styles.Info.Render(fixLine))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Footer with keybindings
	var helpItems []string
	helpItems = append(helpItems, "↑/k up", "↓/j down")
	if fixable > 0 && m.options.AutoFix {
		helpItems = append(helpItems, "f fix")
	}
	helpItems = append(helpItems, "q/Esc quit")

	help := m.styles.Help.Render(strings.Join(helpItems, " • "))
	b.WriteString(help)

	return b.String()
}

// formatSeverity returns a formatted severity indicator.
func (m doctorReportModel) formatSeverity(severity IssueSeverity) string {
	switch severity {
	case IssueSeverityError:
		return m.styles.Error.Render("✗")
	case IssueSeverityWarning:
		return m.styles.Warning.Render("!")
	case IssueSeverityInfo:
		return m.styles.Info.Render("i")
	}
	return "?"
}
