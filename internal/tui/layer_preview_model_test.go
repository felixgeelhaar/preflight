package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayerPreviewModel_New(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:\n  mode: intent"},
		{Path: "layers/base.yaml", Content: "name: base"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	assert.Len(t, m.files, 2)
	assert.Equal(t, 0, m.currentFile)
	assert.False(t, m.confirmed)
	assert.False(t, m.cancelled)
}

func TestLayerPreviewModel_Navigation(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
		{Path: "layers/base.yaml", Content: "name: base"},
		{Path: "layers/work.yaml", Content: "name: work"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Start at first file
	assert.Equal(t, 0, m.currentFile)

	// Navigate right
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 1, m.currentFile)

	// Navigate right again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 2, m.currentFile)

	// Navigate right at end should not change
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 2, m.currentFile)

	// Navigate left
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 1, m.currentFile)
}

func TestLayerPreviewModel_KeyNavigation(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
		{Path: "layers/base.yaml", Content: "name: base"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Navigate with 'l'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 1, m.currentFile)

	// Navigate with 'h'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 0, m.currentFile)
}

func TestLayerPreviewModel_Confirm(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Press Enter to confirm
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(layerPreviewModel)

	assert.True(t, m.confirmed)
	assert.False(t, m.cancelled)
	assert.NotNil(t, cmd) // Should quit
}

func TestLayerPreviewModel_Cancel(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Press Esc to cancel
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(layerPreviewModel)

	assert.False(t, m.confirmed)
	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd) // Should quit
}

func TestLayerPreviewModel_CancelWithQ(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Press q to cancel
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(layerPreviewModel)

	assert.False(t, m.confirmed)
	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd) // Should quit
}

func TestLayerPreviewModel_ScrollContent(t *testing.T) {
	// Create content with multiple lines
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: content},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})
	m.height = 10 // Small height to trigger scrolling

	// Start at top
	assert.Equal(t, 0, m.scrollOffset)

	// Scroll down with j
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 1, m.scrollOffset)

	// Scroll up with k
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 0, m.scrollOffset)

	// Cannot scroll above 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 0, m.scrollOffset)
}

func TestLayerPreviewModel_NumberKeyNavigation(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
		{Path: "layers/base.yaml", Content: "name: base"},
		{Path: "layers/work.yaml", Content: "name: work"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	// Press '2' to go to second file
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 1, m.currentFile)

	// Press '3' to go to third file
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 2, m.currentFile)

	// Press '1' to go back to first file
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(layerPreviewModel)
	assert.Equal(t, 0, m.currentFile)
}

func TestLayerPreviewModel_View(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:\n  mode: intent"},
		{Path: "layers/base.yaml", Content: "name: base"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain title
	assert.Contains(t, view, "Layer Preview")

	// Should contain file path
	assert.Contains(t, view, "preflight.yaml")

	// Should contain help text
	assert.Contains(t, view, "Enter")
	assert.Contains(t, view, "confirm")
}

func TestLayerPreviewModel_WindowResize(t *testing.T) {
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:"},
	}

	m := newLayerPreviewModel(files, LayerPreviewOptions{})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(layerPreviewModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestHighlightYAML(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains []string
	}{
		{
			name:    "highlight_keys",
			content: "defaults:\n  mode: intent",
			contains: []string{
				"defaults",
				"mode",
				"intent",
			},
		},
		{
			name:    "highlight_list",
			content: "targets:\n  - base\n  - work",
			contains: []string{
				"targets",
				"base",
				"work",
			},
		},
		{
			name:    "preserve_structure",
			content: "name: test\nversion: 1.0",
			contains: []string{
				"name",
				"test",
				"version",
				"1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightYAML(tt.content)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestPreviewFile(t *testing.T) {
	pf := PreviewFile{
		Path:    "layers/base.yaml",
		Content: "name: base",
	}

	assert.Equal(t, "layers/base.yaml", pf.Path)
	assert.Equal(t, "name: base", pf.Content)
}

func TestLayerPreviewOptions(t *testing.T) {
	opts := LayerPreviewOptions{
		Title:        "Custom Title",
		ShowLineNums: true,
	}

	assert.Equal(t, "Custom Title", opts.Title)
	assert.True(t, opts.ShowLineNums)
}

func TestRunLayerPreview_Integration(t *testing.T) {
	// This tests the public API types
	files := []PreviewFile{
		{Path: "preflight.yaml", Content: "defaults:\n  mode: intent"},
	}

	// Verify file structure
	require.Len(t, files, 1)
	assert.Equal(t, "preflight.yaml", files[0].Path)

	result := &LayerPreviewResult{
		Confirmed: true,
		Cancelled: false,
	}

	// Verify result structure
	require.NotNil(t, result)
	assert.True(t, result.Confirmed)
	assert.False(t, result.Cancelled)
}
