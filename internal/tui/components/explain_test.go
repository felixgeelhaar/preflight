package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewExplain(t *testing.T) {
	t.Parallel()

	explain := NewExplain()

	assert.Empty(t, explain.Title())
	assert.Empty(t, explain.Content())
	assert.False(t, explain.Loading())
}

func TestExplain_SetContent(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		SetTitle("Why this preset?").
		SetContent("This preset is recommended because...")

	assert.Equal(t, "Why this preset?", explain.Title())
	assert.Equal(t, "This preset is recommended because...", explain.Content())
}

func TestExplain_SetLoading(t *testing.T) {
	t.Parallel()

	explain := NewExplain().SetLoading(true)

	assert.True(t, explain.Loading())
}

func TestExplain_SetSections(t *testing.T) {
	t.Parallel()

	sections := []ExplainSection{
		{Title: "Overview", Content: "This is the overview"},
		{Title: "Tradeoffs", Content: "Here are the tradeoffs"},
	}

	explain := NewExplain().SetSections(sections)

	assert.Len(t, explain.Sections(), 2)
	assert.Equal(t, "Overview", explain.Sections()[0].Title)
}

func TestExplain_AddSection(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		AddSection("Overview", "This is the overview").
		AddSection("Details", "More details here")

	assert.Len(t, explain.Sections(), 2)
}

func TestExplain_SetLinks(t *testing.T) {
	t.Parallel()

	links := map[string]string{
		"Documentation": "https://docs.example.com",
		"GitHub":        "https://github.com/example",
	}

	explain := NewExplain().SetLinks(links)

	assert.Len(t, explain.Links(), 2)
}

func TestExplain_View(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		SetTitle("Explanation").
		SetContent("This is the explanation content.").
		WithWidth(60)

	view := explain.View()

	assert.Contains(t, view, "Explanation")
	assert.Contains(t, view, "This is the explanation content.")
}

func TestExplain_ViewWithSections(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		SetTitle("Preset Details").
		AddSection("Overview", "Basic overview").
		AddSection("Tradeoffs", "Some tradeoffs").
		WithWidth(60)

	view := explain.View()

	assert.Contains(t, view, "Preset Details")
	assert.Contains(t, view, "Overview")
	assert.Contains(t, view, "Tradeoffs")
}

func TestExplain_ViewLoading(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		SetTitle("Loading...").
		SetLoading(true)

	view := explain.View()

	assert.NotEmpty(t, view)
}

func TestExplain_Scroll(t *testing.T) {
	t.Parallel()

	explain := NewExplain().
		SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5").
		WithHeight(3)

	// Scroll down
	explain, _ = explain.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.GreaterOrEqual(t, explain.ScrollOffset(), 0)
}

func TestExplain_Width(t *testing.T) {
	t.Parallel()

	explain := NewExplain().WithWidth(80)

	assert.Equal(t, 80, explain.Width())
}

func TestExplain_Height(t *testing.T) {
	t.Parallel()

	explain := NewExplain().WithHeight(20)

	assert.Equal(t, 20, explain.Height())
}
