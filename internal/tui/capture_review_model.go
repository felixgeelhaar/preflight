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
	Layer    string // Target layer (default: "captured")
}

// captureReviewModel is the Bubble Tea model for capture review.
type captureReviewModel struct {
	items             []CaptureItem
	options           CaptureReviewOptions
	styles            ui.Styles
	width             int
	height            int
	cursor            int
	accepted          []CaptureItem
	rejected          []CaptureItem
	done              bool
	cancelled         bool
	history           *ReviewHistory
	searchActive      bool
	searchQuery       string
	filteredIdx       []int
	layerSelectActive bool
	layerCursor       int
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
		history:  NewReviewHistory(),
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
		// When search is active, handle it specially
		if m.searchActive {
			return m.handleSearchKey(msg)
		}

		// When layer selection is active, handle it specially
		if m.layerSelectActive {
			return m.handleLayerSelectKey(msg)
		}

		// Handle quit keys
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			m.cancelled = true
			return m, tea.Quit
		}

		// Handle escape: clear filter if active, else cancel
		if msg.Type == tea.KeyEsc {
			if m.filteredIdx != nil {
				m.filteredIdx = nil
				m.searchQuery = ""
				return m, nil
			}
			m.cancelled = true
			return m, tea.Quit
		}

		// Handle navigation
		if msg.Type == tea.KeyUp || msg.String() == "k" {
			return m.navigateUp()
		}

		if msg.Type == tea.KeyDown || msg.String() == "j" {
			return m.navigateDown()
		}

		// Handle undo/redo
		if msg.Type == tea.KeyCtrlR {
			return m.redo()
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
		case "u":
			return m.undo()
		case "/":
			m.searchActive = true
			return m, nil
		case "l":
			return m.enterLayerSelect()
		case "g":
			return m.goToTop()
		case "G":
			return m.goToBottom()
		}
	}

	return m, nil
}

// handleSearchKey handles key input when search is active.
func (m captureReviewModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.searchActive = false
		return m, nil
	case tea.KeyEnter:
		m.searchActive = false
		m.filteredIdx = m.filterItems(m.searchQuery)
		// Move cursor to first filtered item if filter is active
		if len(m.filteredIdx) > 0 {
			m.cursor = m.filteredIdx[0]
		}
		return m, nil
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			// Live filtering
			m.filteredIdx = m.filterItems(m.searchQuery)
		}
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.searchQuery += string(msg.Runes)
			// Live filtering
			m.filteredIdx = m.filterItems(m.searchQuery)
		}
		return m, nil
	}
}

// navigateUp moves cursor up, respecting filter.
func (m captureReviewModel) navigateUp() (tea.Model, tea.Cmd) {
	if m.filteredIdx == nil {
		// No filter, simple navigation
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	// Find current position in filtered list
	currentPos := -1
	for i, idx := range m.filteredIdx {
		if idx == m.cursor {
			currentPos = i
			break
		}
	}

	// Move to previous filtered item
	if currentPos > 0 {
		m.cursor = m.filteredIdx[currentPos-1]
	}
	return m, nil
}

// navigateDown moves cursor down, respecting filter.
func (m captureReviewModel) navigateDown() (tea.Model, tea.Cmd) {
	if m.filteredIdx == nil {
		// No filter, simple navigation
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
		return m, nil
	}

	// Find current position in filtered list
	currentPos := -1
	for i, idx := range m.filteredIdx {
		if idx == m.cursor {
			currentPos = i
			break
		}
	}

	// Move to next filtered item
	if currentPos >= 0 && currentPos < len(m.filteredIdx)-1 {
		m.cursor = m.filteredIdx[currentPos+1]
	} else if currentPos == -1 && len(m.filteredIdx) > 0 {
		// Cursor not on filtered item, jump to first filtered item
		m.cursor = m.filteredIdx[0]
	}
	return m, nil
}

// goToTop moves cursor to the first item (or first filtered item).
func (m captureReviewModel) goToTop() (tea.Model, tea.Cmd) {
	if len(m.filteredIdx) > 0 {
		m.cursor = m.filteredIdx[0]
	} else {
		m.cursor = 0
	}
	return m, nil
}

// goToBottom moves cursor to the last item (or last filtered item).
func (m captureReviewModel) goToBottom() (tea.Model, tea.Cmd) {
	if len(m.filteredIdx) > 0 {
		m.cursor = m.filteredIdx[len(m.filteredIdx)-1]
	} else if len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
	return m, nil
}

// filterItems returns indices of items matching the query.
func (m captureReviewModel) filterItems(query string) []int {
	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	result := make([]int, 0)

	for i, item := range m.items {
		name := strings.ToLower(item.Name)
		category := strings.ToLower(item.Category)
		typeStr := strings.ToLower(string(item.Type))

		if strings.Contains(name, query) ||
			strings.Contains(category, query) ||
			strings.Contains(typeStr, query) {
			result = append(result, i)
		}
	}

	return result
}

// isInFilter checks if an index is in the filtered set.
func (m captureReviewModel) isInFilter(index int) bool {
	for _, idx := range m.filteredIdx {
		if idx == index {
			return true
		}
	}
	return false
}

// acceptCurrent accepts the current item and moves to next.
func (m captureReviewModel) acceptCurrent() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.items) {
		item := m.items[m.cursor]
		m.accepted = append(m.accepted, item)
		m.history.Push(ReviewAction{
			Type:      ActionAccept,
			ItemIndex: m.cursor,
			Item:      item,
		})
		return m.advanceCursor()
	}
	return m, nil
}

// rejectCurrent rejects the current item and moves to next.
func (m captureReviewModel) rejectCurrent() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.items) {
		item := m.items[m.cursor]
		m.rejected = append(m.rejected, item)
		m.history.Push(ReviewAction{
			Type:      ActionReject,
			ItemIndex: m.cursor,
			Item:      item,
		})
		return m.advanceCursor()
	}
	return m, nil
}

// undo reverses the last review action.
func (m captureReviewModel) undo() (tea.Model, tea.Cmd) {
	action, ok := m.history.Undo()
	if !ok {
		return m, nil
	}

	switch action.Type {
	case ActionAccept:
		// Remove from accepted
		m.accepted = removeItem(m.accepted, action.Item)
	case ActionReject:
		// Remove from rejected
		m.rejected = removeItem(m.rejected, action.Item)
	case ActionLayerChange:
		// Restore the previous layer
		if action.ItemIndex < len(m.items) {
			m.items[action.ItemIndex].Layer = action.PrevLayer
		}
	}

	// Restore cursor position
	m.cursor = action.ItemIndex

	return m, nil
}

// redo re-applies the last undone action.
func (m captureReviewModel) redo() (tea.Model, tea.Cmd) {
	action, ok := m.history.Redo()
	if !ok {
		return m, nil
	}

	switch action.Type {
	case ActionAccept:
		m.accepted = append(m.accepted, action.Item)
	case ActionReject:
		m.rejected = append(m.rejected, action.Item)
	case ActionLayerChange:
		// Apply the new layer from the action item
		if action.ItemIndex < len(m.items) {
			m.items[action.ItemIndex].Layer = action.Item.Layer
		}
	}

	// Move cursor to next unreviewed item
	return m.advanceCursorFrom(action.ItemIndex)
}

// enterLayerSelect enters layer selection mode for the current item.
func (m captureReviewModel) enterLayerSelect() (tea.Model, tea.Cmd) {
	if len(m.options.AvailableLayers) == 0 || m.cursor >= len(m.items) {
		return m, nil
	}

	m.layerSelectActive = true
	m.layerCursor = 0

	// Position cursor at current layer if exists
	currentLayer := m.items[m.cursor].Layer
	if currentLayer == "" {
		currentLayer = "captured"
	}
	for i, layer := range m.options.AvailableLayers {
		if layer == currentLayer {
			m.layerCursor = i
			break
		}
	}

	return m, nil
}

// handleLayerSelectKey handles key input when layer selection is active.
func (m captureReviewModel) handleLayerSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.layerSelectActive = false
		return m, nil
	case tea.KeyEnter:
		return m.applyLayerChange()
	case tea.KeyUp:
		if m.layerCursor > 0 {
			m.layerCursor--
		}
		return m, nil
	case tea.KeyDown:
		if m.layerCursor < len(m.options.AvailableLayers)-1 {
			m.layerCursor++
		}
		return m, nil
	default:
		switch msg.String() {
		case "k":
			if m.layerCursor > 0 {
				m.layerCursor--
			}
			return m, nil
		case "j":
			if m.layerCursor < len(m.options.AvailableLayers)-1 {
				m.layerCursor++
			}
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Quick select by number
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(m.options.AvailableLayers) {
				m.layerCursor = idx
				return m.applyLayerChange()
			}
			return m, nil
		}
	}
	return m, nil
}

// applyLayerChange applies the selected layer to the current item.
func (m captureReviewModel) applyLayerChange() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.items) || m.layerCursor >= len(m.options.AvailableLayers) {
		m.layerSelectActive = false
		return m, nil
	}

	item := m.items[m.cursor]
	prevLayer := item.Layer
	if prevLayer == "" {
		prevLayer = "captured"
	}
	newLayer := m.options.AvailableLayers[m.layerCursor]

	// Only record if layer actually changed
	if prevLayer != newLayer {
		m.items[m.cursor].Layer = newLayer

		// Record in history for undo
		m.history.Push(ReviewAction{
			Type:      ActionLayerChange,
			ItemIndex: m.cursor,
			Item:      m.items[m.cursor],
			PrevLayer: prevLayer,
		})
	}

	m.layerSelectActive = false
	return m, nil
}

// advanceCursorFrom advances cursor from a specific position.
func (m captureReviewModel) advanceCursorFrom(from int) (tea.Model, tea.Cmd) {
	m.cursor = from
	return m.advanceCursor()
}

// removeItem removes an item from a slice by matching name and category.
func removeItem(items []CaptureItem, target CaptureItem) []CaptureItem {
	result := make([]CaptureItem, 0, len(items))
	for _, item := range items {
		if item.Name != target.Name || item.Category != target.Category {
			result = append(result, item)
		}
	}
	return result
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

	// Show layer selection when active
	if m.layerSelectActive {
		return m.renderLayerSelect()
	}

	// Show search input when active
	if m.searchActive {
		searchPrompt := fmt.Sprintf("Filter: %s_", m.searchQuery)
		b.WriteString(m.styles.Help.Render(searchPrompt))
		b.WriteString("\n\n")
	} else if m.searchQuery != "" {
		// Show active filter
		filterInfo := fmt.Sprintf("Filter: %s (Esc to clear)", m.searchQuery)
		b.WriteString(m.styles.Help.Render(filterInfo))
		b.WriteString("\n\n")
	}

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
		// Skip items not in filter when filter is active
		if m.filteredIdx != nil && !m.isInFilter(i) {
			continue
		}

		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		status := m.formatItemStatus(i)
		layer := item.Layer
		if layer == "" {
			layer = "captured"
		}
		line := fmt.Sprintf("%s%s [%s] %s → %s", prefix, status, item.Category, item.Name, layer)

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
	helpItems := []string{"y accept", "n reject", "l layer", "a all", "d reject all", "u undo", "ctrl+r redo", "/ filter", "g/G top/bottom", "q quit"}
	help := m.styles.Help.Render(strings.Join(helpItems, " • "))
	b.WriteString(help)

	return b.String()
}

// renderLayerSelect renders the layer selection UI.
func (m captureReviewModel) renderLayerSelect() string {
	var b strings.Builder

	// Header
	header := m.styles.Title.Render("Capture Review")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Show current item being modified
	if m.cursor < len(m.items) {
		item := m.items[m.cursor]
		itemInfo := fmt.Sprintf("Select layer for: [%s] %s", item.Category, item.Name)
		b.WriteString(m.styles.Help.Render(itemInfo))
		b.WriteString("\n\n")
	}

	// Layer selection list
	for i, layer := range m.options.AvailableLayers {
		prefix := "  "
		if i == m.layerCursor {
			prefix = "> "
		}

		// Show number shortcut
		line := fmt.Sprintf("%s%d. %s", prefix, i+1, layer)

		// Highlight selected line
		if i == m.layerCursor {
			line = m.styles.ListItemActive.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Footer with keybindings
	helpItems := []string{"Enter select", "1-9 quick select", "j/k up/down", "Esc cancel"}
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
