package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/tui/common"
)

// Panel is a bordered container with a title.
type Panel struct {
	title     string
	content   string
	width     int
	height    int
	hasBorder bool
	styles    common.Styles
}

// NewPanel creates a new panel with the given title.
func NewPanel(title string) Panel {
	return Panel{
		title:     title,
		width:     40,
		height:    10,
		hasBorder: true,
		styles:    common.DefaultStyles(),
	}
}

// Title returns the panel title.
func (p Panel) Title() string {
	return p.title
}

// Content returns the panel content.
func (p Panel) Content() string {
	return p.content
}

// Width returns the panel width.
func (p Panel) Width() int {
	return p.width
}

// Height returns the panel height.
func (p Panel) Height() int {
	return p.height
}

// HasBorder returns whether the panel has a border.
func (p Panel) HasBorder() bool {
	return p.hasBorder
}

// WithTitle returns the panel with a new title.
func (p Panel) WithTitle(title string) Panel {
	p.title = title
	return p
}

// WithContent returns the panel with new content.
func (p Panel) WithContent(content string) Panel {
	p.content = content
	return p
}

// WithWidth returns the panel with a new width.
func (p Panel) WithWidth(width int) Panel {
	p.width = width
	return p
}

// WithHeight returns the panel with a new height.
func (p Panel) WithHeight(height int) Panel {
	p.height = height
	return p
}

// WithBorder returns the panel with border enabled/disabled.
func (p Panel) WithBorder(hasBorder bool) Panel {
	p.hasBorder = hasBorder
	return p
}

// WithStyles returns the panel with custom styles.
func (p Panel) WithStyles(styles common.Styles) Panel {
	p.styles = styles
	return p
}

// View renders the panel.
func (p Panel) View() string {
	titleStyle := p.styles.PanelTitle
	contentStyle := lipgloss.NewStyle()

	var panelStyle lipgloss.Style
	if p.hasBorder {
		panelStyle = p.styles.Panel.Width(p.width - 4)
	} else {
		panelStyle = lipgloss.NewStyle().Width(p.width).Padding(1, 2)
	}

	var b strings.Builder

	// Render title
	b.WriteString(titleStyle.Render(p.title))

	// Render content
	if p.content != "" {
		b.WriteString("\n")
		b.WriteString(contentStyle.Render(p.content))
	}

	return panelStyle.Render(b.String())
}

// SplitPanel displays two panels side by side or stacked.
type SplitPanel struct {
	left     Panel
	right    Panel
	ratio    float64
	vertical bool
	width    int
	height   int
	gap      int
	styles   common.Styles
}

// NewSplitPanel creates a new split panel with left and right panels.
func NewSplitPanel(left, right Panel) SplitPanel {
	return SplitPanel{
		left:   left,
		right:  right,
		ratio:  0.5,
		width:  80,
		height: 20,
		gap:    1,
		styles: common.DefaultStyles(),
	}
}

// Left returns the left panel.
func (s SplitPanel) Left() Panel {
	return s.left
}

// Right returns the right panel.
func (s SplitPanel) Right() Panel {
	return s.right
}

// Ratio returns the split ratio.
func (s SplitPanel) Ratio() float64 {
	return s.ratio
}

// IsVertical returns true if the split is vertical (top/bottom).
func (s SplitPanel) IsVertical() bool {
	return s.vertical
}

// SetLeft replaces the left panel.
func (s SplitPanel) SetLeft(panel Panel) SplitPanel {
	s.left = panel
	return s
}

// SetRight replaces the right panel.
func (s SplitPanel) SetRight(panel Panel) SplitPanel {
	s.right = panel
	return s
}

// WithRatio sets the split ratio (0.0 to 1.0).
func (s SplitPanel) WithRatio(ratio float64) SplitPanel {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	s.ratio = ratio
	return s
}

// WithWidth sets the total width.
func (s SplitPanel) WithWidth(width int) SplitPanel {
	s.width = width
	return s
}

// WithHeight sets the total height.
func (s SplitPanel) WithHeight(height int) SplitPanel {
	s.height = height
	return s
}

// Vertical makes the split vertical (top/bottom).
func (s SplitPanel) Vertical() SplitPanel {
	s.vertical = true
	return s
}

// Horizontal makes the split horizontal (left/right).
func (s SplitPanel) Horizontal() SplitPanel {
	s.vertical = false
	return s
}

// View renders the split panel.
func (s SplitPanel) View() string {
	if s.vertical {
		return s.viewVertical()
	}
	return s.viewHorizontal()
}

func (s SplitPanel) viewHorizontal() string {
	leftWidth := int(float64(s.width) * s.ratio)
	rightWidth := s.width - leftWidth - s.gap

	leftPanel := s.left.WithWidth(leftWidth).WithHeight(s.height)
	rightPanel := s.right.WithWidth(rightWidth).WithHeight(s.height)

	leftView := leftPanel.View()
	rightView := rightPanel.View()

	gap := strings.Repeat(" ", s.gap)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, gap, rightView)
}

func (s SplitPanel) viewVertical() string {
	topHeight := int(float64(s.height) * s.ratio)
	bottomHeight := s.height - topHeight - s.gap

	topPanel := s.left.WithWidth(s.width).WithHeight(topHeight)
	bottomPanel := s.right.WithWidth(s.width).WithHeight(bottomHeight)

	topView := topPanel.View()
	bottomView := bottomPanel.View()

	return lipgloss.JoinVertical(lipgloss.Left, topView, bottomView)
}
