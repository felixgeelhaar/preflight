package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/tui/common"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
)

// StepStartMsg is sent when a step starts executing.
type StepStartMsg struct {
	StepID compiler.StepID
}

// StepCompleteMsg is sent when a step completes execution.
type StepCompleteMsg struct {
	Result execution.StepResult
}

// AllCompleteMsg is sent when all steps have completed.
type AllCompleteMsg struct {
	Results []execution.StepResult
}

// applyProgressModel is the Bubble Tea model for apply progress.
type applyProgressModel struct {
	plan           *execution.Plan
	options        ApplyProgressOptions
	progressBar    components.Progress
	styles         common.Styles
	width          int
	height         int
	stepsTotal     int
	stepsCompleted int
	stepsFailed    int
	currentStep    compiler.StepID
	completed      []execution.StepResult
	done           bool
	cancelled      bool
}

// newApplyProgressModel creates a new apply progress model.
func newApplyProgressModel(plan *execution.Plan, opts ApplyProgressOptions) applyProgressModel {
	styles := common.DefaultStyles()
	progressBar := components.NewProgress().WithWidth(40)

	return applyProgressModel{
		plan:        plan,
		options:     opts,
		progressBar: progressBar,
		styles:      styles,
		width:       80,
		height:      24,
		stepsTotal:  len(plan.NeedsApply()),
		completed:   make([]execution.StepResult, 0),
	}
}

// Init initializes the model.
func (m applyProgressModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m applyProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cancelled = true
			return m, tea.Quit
		}

	case StepStartMsg:
		m.currentStep = msg.StepID
		return m, nil

	case StepCompleteMsg:
		m.completed = append(m.completed, msg.Result)
		m.stepsCompleted++

		if msg.Result.Status() == compiler.StatusFailed {
			m.stepsFailed++
		}

		// Check if all steps are complete
		if m.stepsCompleted >= m.stepsTotal {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case AllCompleteMsg:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the model.
func (m applyProgressModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Applying Changes")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Handle empty plan
	if m.stepsTotal == 0 {
		noChanges := m.styles.Help.Render("Nothing to apply. Your system is up to date.")
		b.WriteString(noChanges)
		return b.String()
	}

	// Progress bar
	if !m.options.Quiet {
		progressPct := m.progressPercent()
		progressBar := m.progressBar.SetPercent(progressPct)
		b.WriteString(progressBar.View())
		b.WriteString("\n\n")
	}

	// Status line
	statusLine := fmt.Sprintf("Progress: %d/%d steps", m.stepsCompleted, m.stepsTotal)
	if m.stepsFailed > 0 {
		statusLine += fmt.Sprintf(" (%d failed)", m.stepsFailed)
	}
	b.WriteString(m.styles.Help.Render(statusLine))
	b.WriteString("\n\n")

	// Current step (if any)
	if m.currentStep.String() != "" && !m.done {
		currentLine := fmt.Sprintf("Running: %s", m.currentStep.String())
		b.WriteString(m.styles.Info.Render(currentLine))
		b.WriteString("\n\n")
	}

	// Recent completions (if showing details)
	if m.options.ShowDetails && len(m.completed) > 0 {
		b.WriteString(m.styles.Subtitle.Render("Completed Steps"))
		b.WriteString("\n")

		// Show last 5 completions
		start := 0
		if len(m.completed) > 5 {
			start = len(m.completed) - 5
		}

		for _, result := range m.completed[start:] {
			status := m.formatResultStatus(result)
			line := fmt.Sprintf("  %s %s", status, result.StepID().String())
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Done message
	if m.done {
		b.WriteString("\n")
		if m.stepsFailed == 0 {
			b.WriteString(m.styles.Success.Render("All steps completed successfully!"))
		} else {
			msg := fmt.Sprintf("Completed with %d failures", m.stepsFailed)
			b.WriteString(m.styles.Error.Render(msg))
		}
		b.WriteString("\n")
	}

	// Footer
	if !m.done {
		b.WriteString("\n")
		b.WriteString(m.styles.Help.Render("Ctrl+C to cancel"))
	}

	return b.String()
}

// progressPercent returns the current progress as a percentage (0.0 to 1.0).
func (m applyProgressModel) progressPercent() float64 {
	if m.stepsTotal == 0 {
		return 0
	}
	return float64(m.stepsCompleted) / float64(m.stepsTotal)
}

// formatResultStatus returns a formatted status indicator for a result.
func (m applyProgressModel) formatResultStatus(result execution.StepResult) string {
	switch result.Status() {
	case compiler.StatusSatisfied:
		return m.styles.Success.Render("✓")
	case compiler.StatusFailed:
		return m.styles.Error.Render("✗")
	case compiler.StatusSkipped:
		return m.styles.Help.Render("-")
	case compiler.StatusNeedsApply, compiler.StatusUnknown:
		return m.styles.Help.Render("?")
	}
	return "?"
}

// progress is an alias for progressPercent for test compatibility.
func (m applyProgressModel) progress() float64 {
	return m.progressPercent()
}
