package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// tourStep represents the current step in the tour.
type tourStep int

const (
	tourStepMenu tourStep = iota
	tourStepContent
)

// tourModel implements the interactive tour TUI.
type tourModel struct {
	step           tourStep
	styles         ui.Styles
	keys           ui.KeyMap
	width          int
	height         int
	cancelled      bool
	completed      bool
	catalogService CatalogServiceInterface

	// Topic menu
	topicList     components.List
	selectedTopic string

	// Content viewing
	currentTopic   TopicContent
	currentSection int
	scrollOffset   int

	// Progress tracking
	progress      TourProgress
	progressStore TourProgressStore
	trackProgress bool
}

func newTourModel(opts TourOptions) tourModel {
	styles := ui.DefaultStyles()
	keys := ui.DefaultKeyMap()

	// Load progress if store provided
	var progress TourProgress
	if opts.ProgressStore != nil && opts.TrackProgress {
		if loaded, err := opts.ProgressStore.Load(); err == nil {
			progress = loaded
		}
	}

	// Build topic list with progress indicators
	topics := GetAllTopics()
	items := make([]components.ListItem, len(topics))
	for i, t := range topics {
		description := t.Description
		if progress != nil {
			if progress.IsTopicCompleted(t.ID) {
				description = "✓ " + description
			} else if progress.IsTopicStarted(t.ID) {
				pct := progress.TopicCompletionPercent(t.ID)
				description = fmt.Sprintf("(%d%%) %s", pct, description)
			}
		}
		items[i] = components.ListItem{
			ID:          t.ID,
			Title:       t.Title,
			Description: description,
		}
	}

	topicList := components.NewList(items).
		WithWidth(60).
		WithHeight(12)

	// If initial topic provided, go directly to content
	startStep := tourStepMenu
	var currentTopic TopicContent
	if opts.InitialTopic != "" {
		if topic, found := GetTopic(opts.InitialTopic); found {
			startStep = tourStepContent
			currentTopic = topic
			// Start tracking this topic
			if progress != nil {
				progress.StartTopic(topic.ID, len(topic.Sections))
			}
		}
	}

	catalogService := opts.CatalogService
	if catalogService == nil {
		catalogService = &fallbackCatalogService{}
	}

	return tourModel{
		step:           startStep,
		styles:         styles,
		keys:           keys,
		width:          80,
		height:         24,
		topicList:      topicList,
		currentTopic:   currentTopic,
		catalogService: catalogService,
		progress:       progress,
		progressStore:  opts.ProgressStore,
		trackProgress:  opts.TrackProgress,
	}
}

func (m tourModel) Init() tea.Cmd {
	return nil
}

func (m tourModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = m.styles.WithWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case components.ListSelectedMsg:
		return m.handleTopicSelection(msg)
	}

	// Update active component
	var cmd tea.Cmd
	if m.step == tourStepMenu {
		m.topicList, cmd = m.topicList.Update(msg)
	}

	return m, cmd
}

func (m tourModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		// Mark current section as complete before quitting
		m.markCurrentSectionComplete()
		m.cancelled = true
		return m, tea.Quit

	case "esc":
		if m.step == tourStepContent {
			// Mark current section as complete before going back
			m.markCurrentSectionComplete()
			// Go back to menu and refresh topic list
			m.step = tourStepMenu
			m.currentSection = 0
			m.scrollOffset = 0
			m.refreshTopicList()
			return m, nil
		}
		m.cancelled = true
		return m, tea.Quit

	case "enter":
		if m.step == tourStepMenu {
			// Select topic
			if item := m.topicList.SelectedItem(); item != nil {
				if topic, found := GetTopic(item.ID); found {
					m.selectedTopic = item.ID
					m.currentTopic = topic
					m.currentSection = 0
					m.scrollOffset = 0
					m.step = tourStepContent
					// Start tracking this topic
					if m.progress != nil {
						m.progress.StartTopic(topic.ID, len(topic.Sections))
					}
					return m, nil
				}
			}
		}

	case "n", "right", "l":
		if m.step == tourStepContent {
			// Mark current section as complete before moving
			m.markCurrentSectionComplete()
			// Next section
			if m.currentSection < len(m.currentTopic.Sections)-1 {
				m.currentSection++
				m.scrollOffset = 0
			}
			return m, nil
		}

	case "p", "left", "h":
		if m.step == tourStepContent {
			// Mark current section as complete before moving
			m.markCurrentSectionComplete()
			// Previous section
			if m.currentSection > 0 {
				m.currentSection--
				m.scrollOffset = 0
			}
			return m, nil
		}

	case "j", "down":
		if m.step == tourStepContent {
			m.scrollOffset++
			return m, nil
		}

	case "k", "up":
		if m.step == tourStepContent {
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil
		}

	case "g":
		if m.step == tourStepContent {
			// Mark current section as complete before jumping
			m.markCurrentSectionComplete()
			// Go to first section
			m.currentSection = 0
			m.scrollOffset = 0
			return m, nil
		}

	case "G":
		if m.step == tourStepContent {
			// Mark current section as complete before jumping
			m.markCurrentSectionComplete()
			// Go to last section
			m.currentSection = len(m.currentTopic.Sections) - 1
			m.scrollOffset = 0
			return m, nil
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.step == tourStepContent {
			// Mark current section as complete before jumping
			m.markCurrentSectionComplete()
			// Quick jump to section
			idx := int(msg.String()[0] - '1')
			if idx < len(m.currentTopic.Sections) {
				m.currentSection = idx
				m.scrollOffset = 0
			}
			return m, nil
		}
	}

	// Let list component handle keys in menu mode
	var cmd tea.Cmd
	if m.step == tourStepMenu {
		m.topicList, cmd = m.topicList.Update(msg)
	}

	return m, cmd
}

// markCurrentSectionComplete marks the current section as completed in progress tracking.
func (m *tourModel) markCurrentSectionComplete() {
	if m.progress != nil && m.currentTopic.ID != "" {
		m.progress.CompleteSection(m.currentTopic.ID, m.currentSection)
	}
}

// refreshTopicList rebuilds the topic list with updated progress indicators.
func (m *tourModel) refreshTopicList() {
	topics := GetAllTopics()
	items := make([]components.ListItem, len(topics))
	for i, t := range topics {
		description := t.Description
		if m.progress != nil {
			if m.progress.IsTopicCompleted(t.ID) {
				description = "✓ " + description
			} else if m.progress.IsTopicStarted(t.ID) {
				pct := m.progress.TopicCompletionPercent(t.ID)
				description = fmt.Sprintf("(%d%%) %s", pct, description)
			}
		}
		items[i] = components.ListItem{
			ID:          t.ID,
			Title:       t.Title,
			Description: description,
		}
	}
	m.topicList = components.NewList(items).
		WithWidth(60).
		WithHeight(12)
}

func (m tourModel) handleTopicSelection(msg components.ListSelectedMsg) (tea.Model, tea.Cmd) {
	if m.step == tourStepMenu {
		if topic, found := GetTopic(msg.Item.ID); found {
			m.selectedTopic = msg.Item.ID
			m.currentTopic = topic
			m.currentSection = 0
			m.scrollOffset = 0
			m.step = tourStepContent
			// Start tracking this topic
			if m.progress != nil {
				m.progress.StartTopic(topic.ID, len(topic.Sections))
			}
		}
	}
	return m, nil
}

func (m tourModel) View() string {
	var content string

	switch m.step {
	case tourStepMenu:
		content = m.viewMenu()
	case tourStepContent:
		content = m.viewContent()
	}

	return content
}

func (m tourModel) viewMenu() string {
	var b strings.Builder

	// Header
	title := m.styles.Title.Render("Preflight Tour")
	subtitle := m.styles.Subtitle.Render("Interactive guided walkthroughs")
	b.WriteString(title + "\n")
	b.WriteString(subtitle + "\n")

	// Overall progress indicator
	if m.progress != nil {
		totalTopics := len(GetAllTopics())
		completed := m.progress.CompletedTopicsCount()
		pct := m.progress.OverallCompletionPercent(totalTopics)
		if completed > 0 || pct > 0 {
			progressText := fmt.Sprintf("Progress: %d/%d topics (%d%%)", completed, totalTopics, pct)
			progress := m.styles.Help.Render(progressText)
			b.WriteString(progress + "\n")
		}
	}
	b.WriteString("\n")

	// Topic list
	b.WriteString(m.topicList.View())
	b.WriteString("\n\n")

	// Help
	help := m.styles.Help.Render("↑/↓ navigate • enter select • q quit")
	b.WriteString(help)

	return b.String()
}

func (m tourModel) viewContent() string {
	var b strings.Builder

	if len(m.currentTopic.Sections) == 0 {
		return "No content available"
	}

	section := m.currentTopic.Sections[m.currentSection]

	// Header with topic title and section indicator
	topicTitle := m.styles.Title.Render(m.currentTopic.Title)

	// Build section indicator with completion status
	var sectionStatus string
	if m.progress != nil {
		completedCount := 0
		for i := range m.currentTopic.Sections {
			if m.progress.IsSectionCompleted(m.currentTopic.ID, i) {
				completedCount++
			}
		}
		sectionStatus = fmt.Sprintf("Section %d/%d (✓%d)", m.currentSection+1, len(m.currentTopic.Sections), completedCount)
	} else {
		sectionStatus = fmt.Sprintf("Section %d/%d", m.currentSection+1, len(m.currentTopic.Sections))
	}
	sectionIndicator := m.styles.Help.Render(sectionStatus)
	b.WriteString(topicTitle + "  " + sectionIndicator + "\n")

	// Section title
	sectionTitle := m.styles.Subtitle.Render(section.Title)
	b.WriteString(sectionTitle + "\n\n")

	// Content
	contentStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		PaddingLeft(2)
	b.WriteString(contentStyle.Render(section.Content))
	b.WriteString("\n")

	// Code block if present
	if section.Code != "" {
		b.WriteString("\n")
		codeStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(1, 2).
			Width(m.width - 8).
			MarginLeft(2)
		b.WriteString(codeStyle.Render(section.Code))
		b.WriteString("\n")
	}

	// Navigation footer
	b.WriteString("\n")

	// Section navigation hints
	var navHints []string
	if m.currentSection > 0 {
		navHints = append(navHints, "← prev")
	}
	if m.currentSection < len(m.currentTopic.Sections)-1 {
		navHints = append(navHints, "next →")
	}

	nav := strings.Join(navHints, " • ")
	if nav != "" {
		nav = "h/l or ←/→: " + nav
	}

	// Next topics suggestion on last section
	if m.currentSection == len(m.currentTopic.Sections)-1 && len(m.currentTopic.NextTopics) > 0 {
		next := m.styles.Help.Render("Next: " + strings.Join(m.currentTopic.NextTopics, ", "))
		b.WriteString(next + "\n")
	}

	// Help footer
	help := m.styles.Help.Render(nav + " • 1-9 jump • esc menu • q quit")
	b.WriteString(help)

	return b.String()
}

// Cancelled returns true if the tour was cancelled.
func (m tourModel) Cancelled() bool {
	return m.cancelled
}

// Completed returns true if the tour was completed.
func (m tourModel) Completed() bool {
	return m.completed
}

// SelectedTopic returns the last selected topic.
func (m tourModel) SelectedTopic() string {
	return m.selectedTopic
}
