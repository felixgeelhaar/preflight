package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// DiffLineType indicates the type of a diff line.
type DiffLineType string

const (
	// DiffLineContext is an unchanged context line.
	DiffLineContext DiffLineType = "context"
	// DiffLineAdd is an added line.
	DiffLineAdd DiffLineType = "add"
	// DiffLineRemove is a removed line.
	DiffLineRemove DiffLineType = "remove"
)

// DiffLine represents a single line in a diff.
type DiffLine struct {
	Type    DiffLineType
	Content string
}

// DiffHunk represents a contiguous section of changes.
type DiffHunk struct {
	StartLine int
	EndLine   int
	Lines     []DiffLine
}

// DiffSummary provides statistics about the diff.
type DiffSummary struct {
	Additions int
	Deletions int
}

// DiffView displays a unified diff with syntax highlighting.
type DiffView struct {
	title        string
	hunks        []DiffHunk
	width        int
	height       int
	scrollOffset int
	viewport     viewport.Model
	keys         ui.KeyMap
	styles       ui.Styles
}

// NewDiffView creates a new diff view.
func NewDiffView() DiffView {
	vp := viewport.New(ui.DefaultWidthMedium, ui.DefaultHeightMedium)
	return DiffView{
		hunks:    make([]DiffHunk, 0),
		width:    ui.DefaultWidthMedium,
		height:   ui.DefaultHeightMedium,
		viewport: vp,
		keys:     ui.DefaultKeyMap(),
		styles:   ui.DefaultStyles(),
	}
}

// Title returns the diff title.
func (d DiffView) Title() string {
	return d.title
}

// Hunks returns the diff hunks.
func (d DiffView) Hunks() []DiffHunk {
	result := make([]DiffHunk, len(d.hunks))
	copy(result, d.hunks)
	return result
}

// Width returns the view width.
func (d DiffView) Width() int {
	return d.width
}

// Height returns the view height.
func (d DiffView) Height() int {
	return d.height
}

// ScrollOffset returns the current scroll position.
func (d DiffView) ScrollOffset() int {
	return d.scrollOffset
}

// SetTitle sets the diff title.
func (d DiffView) SetTitle(title string) DiffView {
	d.title = title
	return d
}

// SetHunks sets the diff hunks.
func (d DiffView) SetHunks(hunks []DiffHunk) DiffView {
	d.hunks = make([]DiffHunk, len(hunks))
	copy(d.hunks, hunks)
	d.updateViewport()
	return d
}

// AddHunk adds a hunk to the diff.
func (d DiffView) AddHunk(hunk DiffHunk) DiffView {
	d.hunks = append(d.hunks, hunk)
	d.updateViewport()
	return d
}

// FromUnified parses a unified diff string.
func (d DiffView) FromUnified(diff string) DiffView {
	lines := strings.Split(diff, "\n")
	var currentHunk *DiffHunk

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Skip file headers
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}

		// Hunk header
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				d.hunks = append(d.hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{
				Lines: make([]DiffLine, 0),
			}
			continue
		}

		if currentHunk == nil {
			continue
		}

		// Parse line type
		var lineType DiffLineType
		var content string

		switch {
		case strings.HasPrefix(line, "+"):
			lineType = DiffLineAdd
			content = strings.TrimPrefix(line, "+")
		case strings.HasPrefix(line, "-"):
			lineType = DiffLineRemove
			content = strings.TrimPrefix(line, "-")
		default:
			lineType = DiffLineContext
			content = strings.TrimPrefix(line, " ")
		}

		currentHunk.Lines = append(currentHunk.Lines, DiffLine{
			Type:    lineType,
			Content: content,
		})
	}

	// Add final hunk
	if currentHunk != nil && len(currentHunk.Lines) > 0 {
		d.hunks = append(d.hunks, *currentHunk)
	}

	d.updateViewport()
	return d
}

// WithWidth sets the view width.
func (d DiffView) WithWidth(width int) DiffView {
	d.width = width
	d.viewport.Width = width - 4
	d.updateViewport()
	return d
}

// WithHeight sets the view height.
func (d DiffView) WithHeight(height int) DiffView {
	d.height = height
	d.viewport.Height = height - 4
	d.updateViewport()
	return d
}

// WithStyles sets the styles.
func (d DiffView) WithStyles(styles ui.Styles) DiffView {
	d.styles = styles
	return d
}

// Summary returns statistics about the diff.
func (d DiffView) Summary() DiffSummary {
	var summary DiffSummary

	for _, hunk := range d.hunks {
		for _, line := range hunk.Lines {
			switch line.Type {
			case DiffLineAdd:
				summary.Additions++
			case DiffLineRemove:
				summary.Deletions++
			case DiffLineContext:
				// Context lines don't contribute to summary
			}
		}
	}

	return summary
}

func (d *DiffView) updateViewport() {
	d.viewport.SetContent(d.renderContent())
}

// Init implements tea.Model.
func (d DiffView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (d DiffView) Update(msg tea.Msg) (DiffView, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return d.handleKeyMsg(msg)
	}
	return d, nil
}

func (d DiffView) handleKeyMsg(msg tea.KeyMsg) (DiffView, tea.Cmd) {
	switch {
	case key.Matches(msg, d.keys.Up) || key.Matches(msg, d.keys.VimUp):
		if d.scrollOffset > 0 {
			d.scrollOffset--
		}
		d.viewport.LineUp(1)
	case key.Matches(msg, d.keys.Down) || key.Matches(msg, d.keys.VimDown):
		d.scrollOffset++
		d.viewport.LineDown(1)
	case key.Matches(msg, d.keys.PageUp):
		d.viewport.HalfViewUp()
	case key.Matches(msg, d.keys.PageDown):
		d.viewport.HalfViewDown()
	}
	return d, nil
}

func (d DiffView) renderContent() string {
	var b strings.Builder

	for hunkIdx, hunk := range d.hunks {
		if hunkIdx > 0 {
			b.WriteString("\n")
		}

		for _, line := range hunk.Lines {
			var prefix string
			var style = d.styles.Paragraph

			switch line.Type {
			case DiffLineAdd:
				prefix = "+ "
				style = d.styles.DiffAdd
			case DiffLineRemove:
				prefix = "- "
				style = d.styles.DiffRemove
			case DiffLineContext:
				prefix = "  "
			}

			b.WriteString(style.Render(prefix + line.Content))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// View renders the diff view.
func (d DiffView) View() string {
	var b strings.Builder

	// Title / header
	if d.title != "" {
		b.WriteString(d.styles.DiffHeader.Render(d.title))
		b.WriteString("\n")

		summary := d.Summary()
		summaryText := d.styles.DiffAdd.Render("+", string(rune('0'+summary.Additions))) +
			" " +
			d.styles.DiffRemove.Render("-", string(rune('0'+summary.Deletions)))
		b.WriteString(summaryText)
		b.WriteString("\n\n")
	}

	// Content via viewport
	b.WriteString(d.viewport.View())

	return d.styles.Panel.Width(d.width - 4).Render(b.String())
}
