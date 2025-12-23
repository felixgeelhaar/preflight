package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPanel(t *testing.T) {
	t.Parallel()

	panel := NewPanel("Test Panel")

	assert.Equal(t, "Test Panel", panel.Title())
	assert.Empty(t, panel.Content())
}

func TestPanel_WithContent(t *testing.T) {
	t.Parallel()

	panel := NewPanel("Title").WithContent("Panel content here")

	assert.Equal(t, "Panel content here", panel.Content())
}

func TestPanel_WithDimensions(t *testing.T) {
	t.Parallel()

	panel := NewPanel("Title").
		WithWidth(80).
		WithHeight(20)

	assert.Equal(t, 80, panel.Width())
	assert.Equal(t, 20, panel.Height())
}

func TestPanel_WithBorder(t *testing.T) {
	t.Parallel()

	panel := NewPanel("Title").WithBorder(true)
	assert.True(t, panel.HasBorder())

	panel = panel.WithBorder(false)
	assert.False(t, panel.HasBorder())
}

func TestPanel_View(t *testing.T) {
	t.Parallel()

	panel := NewPanel("My Panel").
		WithContent("Hello, World!").
		WithWidth(40)

	view := panel.View()

	assert.Contains(t, view, "My Panel")
	assert.Contains(t, view, "Hello, World!")
}

func TestPanel_ViewEmpty(t *testing.T) {
	t.Parallel()

	panel := NewPanel("Empty Panel")
	view := panel.View()

	assert.Contains(t, view, "Empty Panel")
}

func TestSplitPanel(t *testing.T) {
	t.Parallel()

	left := NewPanel("Left").WithContent("Left content")
	right := NewPanel("Right").WithContent("Right content")

	split := NewSplitPanel(left, right)

	assert.Equal(t, 0.5, split.Ratio())
}

func TestSplitPanel_WithRatio(t *testing.T) {
	t.Parallel()

	left := NewPanel("Left")
	right := NewPanel("Right")

	split := NewSplitPanel(left, right).WithRatio(0.3)

	assert.Equal(t, 0.3, split.Ratio())
}

func TestSplitPanel_View(t *testing.T) {
	t.Parallel()

	left := NewPanel("Left").WithContent("Left content")
	right := NewPanel("Right").WithContent("Right content")

	split := NewSplitPanel(left, right).
		WithWidth(80).
		WithHeight(20)

	view := split.View()

	assert.Contains(t, view, "Left")
	assert.Contains(t, view, "Right")
}

func TestSplitPanel_Vertical(t *testing.T) {
	t.Parallel()

	top := NewPanel("Top")
	bottom := NewPanel("Bottom")

	split := NewSplitPanel(top, bottom).Vertical()

	assert.True(t, split.IsVertical())
}

func TestSplitPanel_SetPanels(t *testing.T) {
	t.Parallel()

	left := NewPanel("Left")
	right := NewPanel("Right")
	split := NewSplitPanel(left, right)

	newLeft := NewPanel("New Left")
	newRight := NewPanel("New Right")

	split = split.SetLeft(newLeft).SetRight(newRight)

	assert.Equal(t, "New Left", split.Left().Title())
	assert.Equal(t, "New Right", split.Right().Title())
}
