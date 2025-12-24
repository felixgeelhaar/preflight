package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestNewTourModel(t *testing.T) {
	t.Parallel()

	opts := TourOptions{}
	model := newTourModel(opts)

	assert.Equal(t, tourStepMenu, model.step)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.False(t, model.cancelled)
	assert.False(t, model.completed)
	assert.Empty(t, model.selectedTopic)
}

func TestNewTourModel_WithInitialTopic(t *testing.T) {
	t.Parallel()

	opts := TourOptions{InitialTopic: "basics"}
	model := newTourModel(opts)

	assert.Equal(t, tourStepContent, model.step)
	assert.Equal(t, "Preflight Fundamentals", model.currentTopic.Title)
	assert.Equal(t, 0, model.currentSection)
}

func TestNewTourModel_WithInvalidInitialTopic(t *testing.T) {
	t.Parallel()

	opts := TourOptions{InitialTopic: "invalid-topic"}
	model := newTourModel(opts)

	// Should stay at menu since topic doesn't exist
	assert.Equal(t, tourStepMenu, model.step)
}

func TestTourModel_Init(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestTourModel_Update_WindowSize(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.Nil(t, cmd)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestTourModel_HandleKeyMsg_QuitFromMenu(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}

	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd) // Should be tea.Quit
}

func TestTourModel_HandleKeyMsg_QFromMenu(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd)
}

func TestTourModel_HandleKeyMsg_EscFromMenu(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	msg := tea.KeyMsg{Type: tea.KeyEsc}

	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd)
}

func TestTourModel_HandleKeyMsg_EnterSelectsTopic(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	msg := tea.KeyMsg{Type: tea.KeyEnter}

	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.Nil(t, cmd)
	assert.Equal(t, tourStepContent, m.step)
	assert.NotEmpty(t, m.selectedTopic)
	assert.NotEmpty(t, m.currentTopic.Title)
}

func TestTourModel_HandleKeyMsg_EscFromContent(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	assert.Equal(t, tourStepContent, model.step)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	m := updated.(tourModel)

	assert.Nil(t, cmd)
	assert.Equal(t, tourStepMenu, m.step)
	assert.Equal(t, 0, m.currentSection)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_NextSection(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	assert.Equal(t, 0, model.currentSection)

	// Test 'n' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.currentSection)
}

func TestTourModel_HandleKeyMsg_NextSectionRight(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})

	msg := tea.KeyMsg{Type: tea.KeyRight}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.currentSection)
}

func TestTourModel_HandleKeyMsg_NextSectionL(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.currentSection)
}

func TestTourModel_HandleKeyMsg_PreviousSection(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.currentSection = 2

	// Test 'p' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.currentSection)
}

func TestTourModel_HandleKeyMsg_PreviousSectionLeft(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.currentSection = 1

	msg := tea.KeyMsg{Type: tea.KeyLeft}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.currentSection)
}

func TestTourModel_HandleKeyMsg_PreviousSectionH(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.currentSection = 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.currentSection)
}

func TestTourModel_HandleKeyMsg_CannotGoPastFirstSection(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	assert.Equal(t, 0, model.currentSection)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.currentSection)
}

func TestTourModel_HandleKeyMsg_CannotGoPastLastSection(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	lastSection := len(model.currentTopic.Sections) - 1
	model.currentSection = lastSection

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, lastSection, m.currentSection)
}

func TestTourModel_HandleKeyMsg_ScrollDown(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_ScrollUp(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.scrollOffset = 5

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 4, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_ScrollUpCannotGoNegative(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	assert.Equal(t, 0, model.scrollOffset)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_GoToFirst(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.currentSection = 2
	model.scrollOffset = 5

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.currentSection)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_GoToLast(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.scrollOffset = 5
	lastSection := len(model.currentTopic.Sections) - 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, lastSection, m.currentSection)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestTourModel_HandleKeyMsg_NumberJump(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})

	// Jump to section 2 (0-indexed, so '2' goes to index 1)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 1, m.currentSection)
}

func TestTourModel_HandleKeyMsg_NumberJumpOutOfRange(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	initialSection := model.currentSection

	// Try to jump to section 9 (likely out of range)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	// Should stay at current section if out of range
	assert.Equal(t, initialSection, m.currentSection)
}

func TestTourModel_HandleTopicSelection(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})

	msg := components.ListSelectedMsg{
		Item: components.ListItem{ID: "config", Title: "Configuration Deep-Dive"},
	}

	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, tourStepContent, m.step)
	assert.Equal(t, "config", m.selectedTopic)
	assert.Equal(t, "Configuration Deep-Dive", m.currentTopic.Title)
	assert.Equal(t, 0, m.currentSection)
}

func TestTourModel_View_Menu(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	view := model.View()

	assert.Contains(t, view, "Preflight Tour")
	assert.Contains(t, view, "Interactive guided walkthroughs")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "quit")
}

func TestTourModel_View_Content(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	view := model.View()

	assert.Contains(t, view, "Preflight Fundamentals")
	assert.Contains(t, view, "Section 1/")
	assert.Contains(t, view, "What is Preflight?")
}

func TestTourModel_View_ContentEmptySections(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	model.step = tourStepContent
	model.currentTopic = TopicContent{
		Title:    "Empty Topic",
		Sections: []Section{},
	}

	view := model.View()
	assert.Contains(t, view, "No content available")
}

func TestTourModel_Cancelled(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	assert.False(t, model.Cancelled())

	model.cancelled = true
	assert.True(t, model.Cancelled())
}

func TestTourModel_Completed(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	assert.False(t, model.Completed())

	model.completed = true
	assert.True(t, model.Completed())
}

func TestTourModel_SelectedTopic(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{})
	assert.Empty(t, model.SelectedTopic())

	model.selectedTopic = "basics"
	assert.Equal(t, "basics", model.SelectedTopic())
}

func TestTourModel_SectionNavigationResetsScroll(t *testing.T) {
	t.Parallel()

	model := newTourModel(TourOptions{InitialTopic: "basics"})
	model.scrollOffset = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updated, _ := model.Update(msg)
	m := updated.(tourModel)

	assert.Equal(t, 0, m.scrollOffset)
}

func TestGetAllTopics(t *testing.T) {
	t.Parallel()

	topics := GetAllTopics()

	assert.NotEmpty(t, topics)
	assert.GreaterOrEqual(t, len(topics), 6)

	// Verify expected topics exist
	ids := make([]string, len(topics))
	for i, topic := range topics {
		ids[i] = topic.ID
	}

	assert.Contains(t, ids, "basics")
	assert.Contains(t, ids, "config")
	assert.Contains(t, ids, "layers")
	assert.Contains(t, ids, "providers")
	assert.Contains(t, ids, "presets")
	assert.Contains(t, ids, "workflow")
}

func TestGetTopic(t *testing.T) {
	t.Parallel()

	topic, found := GetTopic("basics")
	assert.True(t, found)
	assert.Equal(t, "basics", topic.ID)
	assert.Equal(t, "Preflight Fundamentals", topic.Title)
	assert.NotEmpty(t, topic.Sections)
}

func TestGetTopic_NotFound(t *testing.T) {
	t.Parallel()

	topic, found := GetTopic("nonexistent")
	assert.False(t, found)
	assert.Empty(t, topic.ID)
}

func TestGetTopicIDs(t *testing.T) {
	t.Parallel()

	ids := GetTopicIDs()

	assert.NotEmpty(t, ids)
	assert.Contains(t, ids, "basics")
	assert.Contains(t, ids, "config")
	assert.Contains(t, ids, "layers")
	assert.Contains(t, ids, "providers")
	assert.Contains(t, ids, "presets")
	assert.Contains(t, ids, "workflow")
}

func TestTopicContent_HasSections(t *testing.T) {
	t.Parallel()

	topics := GetAllTopics()
	for _, topic := range topics {
		assert.NotEmpty(t, topic.Sections, "topic %s should have sections", topic.ID)
		assert.NotEmpty(t, topic.Title, "topic %s should have title", topic.ID)
		assert.NotEmpty(t, topic.Description, "topic %s should have description", topic.ID)

		for i, section := range topic.Sections {
			assert.NotEmpty(t, section.Title, "topic %s section %d should have title", topic.ID, i)
			assert.NotEmpty(t, section.Content, "topic %s section %d should have content", topic.ID, i)
		}
	}
}

func TestTopicContent_NextTopics(t *testing.T) {
	t.Parallel()

	topic, found := GetTopic("basics")
	assert.True(t, found)
	assert.NotEmpty(t, topic.NextTopics)

	// Verify next topics are valid
	for _, nextID := range topic.NextTopics {
		_, nextFound := GetTopic(nextID)
		assert.True(t, nextFound, "next topic %s should exist", nextID)
	}
}

func TestTourOptions_NewTourOptions(t *testing.T) {
	t.Parallel()

	opts := NewTourOptions()
	assert.Empty(t, opts.InitialTopic)
	assert.Nil(t, opts.CatalogService)
}

func TestTourOptions_WithInitialTopic(t *testing.T) {
	t.Parallel()

	opts := NewTourOptions().WithInitialTopic("basics")
	assert.Equal(t, "basics", opts.InitialTopic)
}
