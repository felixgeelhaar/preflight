package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// PreviewFile represents a file to be previewed.
type PreviewFile struct {
	Path    string
	Content string
}

// LayerPreviewOptions configures the layer preview TUI.
type LayerPreviewOptions struct {
	Title        string
	ShowLineNums bool
}

// LayerPreviewResult holds the result of layer preview.
type LayerPreviewResult struct {
	Confirmed bool
	Cancelled bool
}

// layerPreviewModel is the Bubble Tea model for layer preview.
type layerPreviewModel struct {
	files        []PreviewFile
	options      LayerPreviewOptions
	styles       ui.Styles
	width        int
	height       int
	currentFile  int
	scrollOffset int
	confirmed    bool
	cancelled    bool
}

// newLayerPreviewModel creates a new layer preview model.
func newLayerPreviewModel(files []PreviewFile, opts LayerPreviewOptions) layerPreviewModel {
	styles := ui.DefaultStyles()

	title := opts.Title
	if title == "" {
		title = "Layer Preview"
	}
	opts.Title = title

	return layerPreviewModel{
		files:        files,
		options:      opts,
		styles:       styles,
		width:        80,
		height:       24,
		currentFile:  0,
		scrollOffset: 0,
	}
}

// Init initializes the model.
func (m layerPreviewModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update handles messages.
func (m layerPreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m layerPreviewModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // We only handle specific key types
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyEnter:
		m.confirmed = true
		return m, tea.Quit

	case tea.KeyLeft:
		if m.currentFile > 0 {
			m.currentFile--
			m.scrollOffset = 0
		}
		return m, nil

	case tea.KeyRight:
		if m.currentFile < len(m.files)-1 {
			m.currentFile++
			m.scrollOffset = 0
		}
		return m, nil

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
func (m layerPreviewModel) handleRuneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(msg.Runes) == 0 {
		return m, nil
	}

	key := msg.Runes[0]

	switch key {
	case 'q':
		m.cancelled = true
		return m, tea.Quit

	case 'h':
		if m.currentFile > 0 {
			m.currentFile--
			m.scrollOffset = 0
		}
		return m, nil

	case 'l':
		if m.currentFile < len(m.files)-1 {
			m.currentFile++
			m.scrollOffset = 0
		}
		return m, nil

	case 'j':
		m.scrollOffset++
		return m, nil

	case 'k':
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil

	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		idx := int(key - '1')
		if idx >= 0 && idx < len(m.files) {
			m.currentFile = idx
			m.scrollOffset = 0
		}
		return m, nil
	}

	return m, nil
}

// View renders the model.
func (m layerPreviewModel) View() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render(m.options.Title)
	b.WriteString(header)
	b.WriteString("\n\n")

	// File tabs
	b.WriteString(m.renderFileTabs())
	b.WriteString("\n\n")

	// Current file content
	if len(m.files) > 0 && m.currentFile < len(m.files) {
		file := m.files[m.currentFile]
		b.WriteString(m.renderFileContent(file))
	}

	b.WriteString("\n")

	// Footer with keybindings
	helpItems := []string{
		"Enter confirm",
		"Esc cancel",
		"←/→ or h/l switch file",
		"↑/↓ or j/k scroll",
		"1-9 quick select",
	}
	help := m.styles.Help.Render(strings.Join(helpItems, " • "))
	b.WriteString(help)

	return b.String()
}

// renderFileTabs renders the file tab bar.
func (m layerPreviewModel) renderFileTabs() string {
	tabs := make([]string, 0, len(m.files))

	for i, file := range m.files {
		// Extract just the filename for display
		name := file.Path
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}

		// Show number prefix
		tab := fmt.Sprintf("%d:%s", i+1, name)

		if i == m.currentFile {
			tab = m.styles.ListItemActive.Render("[" + tab + "]")
		} else {
			tab = m.styles.Help.Render(" " + tab + " ")
		}

		tabs = append(tabs, tab)
	}

	return strings.Join(tabs, " ")
}

// renderFileContent renders the file content with optional line numbers.
func (m layerPreviewModel) renderFileContent(file PreviewFile) string {
	var b strings.Builder

	// Show file path
	pathLine := m.styles.DiffHeader.Render("# " + file.Path)
	b.WriteString(pathLine)
	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render(strings.Repeat("─", min(60, m.width-4))))
	b.WriteString("\n")

	// Split content into lines
	lines := strings.Split(file.Content, "\n")

	// Calculate visible area (reserve space for header, tabs, and footer)
	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	// Apply scroll offset
	startLine := m.scrollOffset
	if startLine > len(lines)-visibleLines {
		startLine = max(0, len(lines)-visibleLines)
	}
	endLine := min(startLine+visibleLines, len(lines))

	// Render visible lines with syntax highlighting
	for i := startLine; i < endLine; i++ {
		line := lines[i]

		if m.options.ShowLineNums {
			lineNum := m.styles.Help.Render(fmt.Sprintf("%3d │ ", i+1))
			b.WriteString(lineNum)
		}

		// Apply YAML syntax highlighting
		highlighted := highlightYAMLLine(line)
		b.WriteString(highlighted)
		b.WriteString("\n")
	}

	// Show scroll indicator if content is scrollable
	if len(lines) > visibleLines {
		scrollInfo := fmt.Sprintf("─── %d/%d lines ───", startLine+1, len(lines))
		b.WriteString(m.styles.Help.Render(scrollInfo))
	}

	return b.String()
}

// highlightYAML applies basic syntax highlighting to YAML content.
func highlightYAML(content string) string {
	lines := strings.Split(content, "\n")
	highlighted := make([]string, len(lines))

	for i, line := range lines {
		highlighted[i] = highlightYAMLLine(line)
	}

	return strings.Join(highlighted, "\n")
}

// highlightYAMLLine applies syntax highlighting to a single YAML line.
func highlightYAMLLine(line string) string {
	styles := ui.DefaultStyles()

	// Handle empty lines
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Handle comments
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return styles.Help.Render(line)
	}

	// Handle list items
	if strings.HasPrefix(trimmed, "- ") {
		indent := len(line) - len(trimmed)
		rest := trimmed[2:]
		return strings.Repeat(" ", indent) +
			styles.Info.Render("- ") +
			highlightYAMLValue(rest)
	}

	// Handle key-value pairs
	if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		key := strings.TrimSpace(line[:colonIdx])
		rest := ""
		if colonIdx+1 < len(line) {
			rest = line[colonIdx+1:]
		}

		result := strings.Repeat(" ", indent) +
			styles.Info.Render(key) +
			styles.Subtitle.Render(":")

		if strings.TrimSpace(rest) != "" {
			result += highlightYAMLValue(rest)
		}

		return result
	}

	return line
}

// highlightYAMLValue applies highlighting to a YAML value.
func highlightYAMLValue(value string) string {
	styles := ui.DefaultStyles()
	trimmed := strings.TrimSpace(value)

	// Preserve leading space
	leadingSpace := ""
	if len(value) > 0 && value[0] == ' ' {
		leadingSpace = " "
	}

	// Boolean values
	if trimmed == "true" || trimmed == "false" {
		return leadingSpace + styles.Warning.Render(trimmed)
	}

	// Numeric values
	if isNumeric(trimmed) {
		return leadingSpace + styles.Success.Render(trimmed)
	}

	// String values
	return leadingSpace + styles.Paragraph.Render(trimmed)
}

// isNumeric checks if a string represents a number.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	// Handle negative numbers
	if s[0] == '-' {
		s = s[1:]
	}

	dotSeen := false
	for _, c := range s {
		if c == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}

	return len(s) > 0 || dotSeen
}
