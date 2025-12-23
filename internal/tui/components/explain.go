package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// ExplainSection represents a titled section in an explanation.
type ExplainSection struct {
	Title   string
	Content string
}

// Explain displays explanatory content with sections and links.
type Explain struct {
	title        string
	content      string
	sections     []ExplainSection
	links        map[string]string
	loading      bool
	width        int
	height       int
	scrollOffset int
	viewport     viewport.Model
	keys         ui.KeyMap
	styles       ui.Styles
}

// NewExplain creates a new explain panel.
func NewExplain() Explain {
	vp := viewport.New(40, 10)
	return Explain{
		width:    40,
		height:   10,
		links:    make(map[string]string),
		sections: make([]ExplainSection, 0),
		viewport: vp,
		keys:     ui.DefaultKeyMap(),
		styles:   ui.DefaultStyles(),
	}
}

// Title returns the explanation title.
func (e Explain) Title() string {
	return e.title
}

// Content returns the main content.
func (e Explain) Content() string {
	return e.content
}

// Sections returns the sections.
func (e Explain) Sections() []ExplainSection {
	result := make([]ExplainSection, len(e.sections))
	copy(result, e.sections)
	return result
}

// Links returns the documentation links.
func (e Explain) Links() map[string]string {
	result := make(map[string]string)
	for k, v := range e.links {
		result[k] = v
	}
	return result
}

// Loading returns whether loading state is active.
func (e Explain) Loading() bool {
	return e.loading
}

// Width returns the panel width.
func (e Explain) Width() int {
	return e.width
}

// Height returns the panel height.
func (e Explain) Height() int {
	return e.height
}

// ScrollOffset returns the current scroll position.
func (e Explain) ScrollOffset() int {
	return e.scrollOffset
}

// SetTitle sets the title.
func (e Explain) SetTitle(title string) Explain {
	e.title = title
	return e
}

// SetContent sets the main content.
func (e Explain) SetContent(content string) Explain {
	e.content = content
	e.updateViewport()
	return e
}

// SetSections sets the sections.
func (e Explain) SetSections(sections []ExplainSection) Explain {
	e.sections = make([]ExplainSection, len(sections))
	copy(e.sections, sections)
	e.updateViewport()
	return e
}

// AddSection adds a section.
func (e Explain) AddSection(title, content string) Explain {
	e.sections = append(e.sections, ExplainSection{
		Title:   title,
		Content: content,
	})
	e.updateViewport()
	return e
}

// SetLinks sets the documentation links.
func (e Explain) SetLinks(links map[string]string) Explain {
	e.links = make(map[string]string)
	for k, v := range links {
		e.links[k] = v
	}
	return e
}

// SetLoading sets the loading state.
func (e Explain) SetLoading(loading bool) Explain {
	e.loading = loading
	return e
}

// WithWidth sets the width.
func (e Explain) WithWidth(width int) Explain {
	e.width = width
	e.viewport.Width = width - 4 // Account for padding
	e.updateViewport()
	return e
}

// WithHeight sets the height.
func (e Explain) WithHeight(height int) Explain {
	e.height = height
	e.viewport.Height = height - 4 // Account for title and borders
	e.updateViewport()
	return e
}

// WithStyles sets the styles.
func (e Explain) WithStyles(styles ui.Styles) Explain {
	e.styles = styles
	return e
}

func (e *Explain) updateViewport() {
	e.viewport.SetContent(e.renderContent())
}

// Init implements tea.Model.
func (e Explain) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (e Explain) Update(msg tea.Msg) (Explain, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return e.handleKeyMsg(msg)
	}
	return e, nil
}

func (e Explain) handleKeyMsg(msg tea.KeyMsg) (Explain, tea.Cmd) {
	switch {
	case key.Matches(msg, e.keys.Up) || key.Matches(msg, e.keys.VimUp):
		if e.scrollOffset > 0 {
			e.scrollOffset--
		}
		e.viewport.LineUp(1)
	case key.Matches(msg, e.keys.Down) || key.Matches(msg, e.keys.VimDown):
		e.scrollOffset++
		e.viewport.LineDown(1)
	case key.Matches(msg, e.keys.PageUp):
		e.viewport.HalfViewUp()
	case key.Matches(msg, e.keys.PageDown):
		e.viewport.HalfViewDown()
	}
	return e, nil
}

func (e Explain) renderContent() string {
	var b strings.Builder

	// Main content
	if e.content != "" {
		b.WriteString(e.content)
		b.WriteString("\n")
	}

	// Sections
	for _, section := range e.sections {
		b.WriteString("\n")
		b.WriteString(e.styles.Subtitle.Render(section.Title))
		b.WriteString("\n")
		b.WriteString(section.Content)
		b.WriteString("\n")
	}

	// Links
	if len(e.links) > 0 {
		b.WriteString("\n")
		b.WriteString(e.styles.Subtitle.Render("Documentation"))
		b.WriteString("\n")
		for name, url := range e.links {
			b.WriteString("• ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(e.styles.Help.Render(url))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// View renders the explain panel.
func (e Explain) View() string {
	var b strings.Builder

	// Title
	if e.title != "" {
		b.WriteString(e.styles.PanelTitle.Render(e.title))
		b.WriteString("\n\n")
	}

	// Loading state
	if e.loading {
		spinner := NewSpinner().SetMessage("Loading explanation...")
		b.WriteString(spinner.View())
		return e.styles.Panel.Width(e.width - 4).Render(b.String())
	}

	// Content via viewport
	b.WriteString(e.viewport.View())

	return e.styles.Panel.Width(e.width - 4).Render(b.String())
}
