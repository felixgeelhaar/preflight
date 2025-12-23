package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/tui/common"
)

// Progress displays a progress bar with optional message.
type Progress struct {
	percent float64
	current int
	total   int
	message string
	width   int
	styles  common.Styles
}

// NewProgress creates a new progress component.
func NewProgress() Progress {
	return Progress{
		width:  40,
		styles: common.DefaultStyles(),
	}
}

// Percent returns the current percentage (0.0 to 1.0).
func (p Progress) Percent() float64 {
	return p.percent
}

// Current returns the current item number.
func (p Progress) Current() int {
	return p.current
}

// Total returns the total number of items.
func (p Progress) Total() int {
	return p.total
}

// Message returns the current message.
func (p Progress) Message() string {
	return p.message
}

// Width returns the progress bar width.
func (p Progress) Width() int {
	return p.width
}

// SetPercent sets the progress percentage.
func (p Progress) SetPercent(percent float64) Progress {
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	p.percent = percent
	return p
}

// SetCurrent sets the current item number and updates percent.
func (p Progress) SetCurrent(current int) Progress {
	if current < 0 {
		current = 0
	}
	if current > p.total && p.total > 0 {
		current = p.total
	}
	p.current = current
	if p.total > 0 {
		p.percent = float64(current) / float64(p.total)
	}
	return p
}

// SetTotal sets the total number of items.
func (p Progress) SetTotal(total int) Progress {
	if total < 0 {
		total = 0
	}
	p.total = total
	if p.total > 0 && p.current > 0 {
		p.percent = float64(p.current) / float64(p.total)
	}
	return p
}

// IncrementCurrent increments the current count.
func (p Progress) IncrementCurrent() Progress {
	if p.current < p.total {
		p.current++
		if p.total > 0 {
			p.percent = float64(p.current) / float64(p.total)
		}
	}
	return p
}

// SetMessage sets the status message.
func (p Progress) SetMessage(message string) Progress {
	p.message = message
	return p
}

// WithWidth sets the progress bar width.
func (p Progress) WithWidth(width int) Progress {
	p.width = width
	return p
}

// WithStyles sets the styles.
func (p Progress) WithStyles(styles common.Styles) Progress {
	p.styles = styles
	return p
}

// View renders the progress bar.
func (p Progress) View() string {
	var b strings.Builder

	// Calculate filled width
	barWidth := p.width - 2 // Account for brackets
	filled := int(p.percent * float64(barWidth))
	empty := barWidth - filled

	// Build the bar
	bar := fmt.Sprintf("[%s%s]",
		strings.Repeat("█", filled),
		strings.Repeat("░", empty),
	)

	barStyle := p.styles.ProgressBar
	b.WriteString(barStyle.Render(bar))

	// Add percentage
	percentStr := fmt.Sprintf(" %3.0f%%", p.percent*100)
	b.WriteString(percentStr)

	// Add message
	if p.message != "" {
		b.WriteString("\n")
		b.WriteString(p.styles.Help.Render(p.message))
	}

	return b.String()
}

// Spinner displays an animated spinner with optional message.
type Spinner struct {
	spinner spinner.Model
	message string
	styles  common.Styles
}

// NewSpinner creates a new spinner component.
func NewSpinner() Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(common.ColorPrimary)

	return Spinner{
		spinner: s,
		styles:  common.DefaultStyles(),
	}
}

// Message returns the current message.
func (s Spinner) Message() string {
	return s.message
}

// SetMessage sets the spinner message.
func (s Spinner) SetMessage(message string) Spinner {
	s.message = message
	return s
}

// WithStyles sets the styles.
func (s Spinner) WithStyles(styles common.Styles) Spinner {
	s.styles = styles
	return s
}

// Init returns the initial command for the spinner.
func (s Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles spinner animation.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner.
func (s Spinner) View() string {
	if s.message != "" {
		return fmt.Sprintf("%s %s", s.spinner.View(), s.message)
	}
	return s.spinner.View()
}
