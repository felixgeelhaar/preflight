package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewDiffView(t *testing.T) {
	t.Parallel()

	diff := NewDiffView()

	assert.Empty(t, diff.Title())
	assert.Empty(t, diff.Hunks())
}

func TestDiffView_SetTitle(t *testing.T) {
	t.Parallel()

	diff := NewDiffView().SetTitle("~/.zshrc")

	assert.Equal(t, "~/.zshrc", diff.Title())
}

func TestDiffView_SetHunks(t *testing.T) {
	t.Parallel()

	hunks := []DiffHunk{
		{
			StartLine: 1,
			Lines: []DiffLine{
				{Type: DiffLineContext, Content: "line 1"},
				{Type: DiffLineAdd, Content: "new line"},
				{Type: DiffLineRemove, Content: "old line"},
			},
		},
	}

	diff := NewDiffView().SetHunks(hunks)

	assert.Len(t, diff.Hunks(), 1)
	assert.Len(t, diff.Hunks()[0].Lines, 3)
}

func TestDiffView_AddHunk(t *testing.T) {
	t.Parallel()

	diff := NewDiffView().
		AddHunk(DiffHunk{StartLine: 1}).
		AddHunk(DiffHunk{StartLine: 10})

	assert.Len(t, diff.Hunks(), 2)
}

func TestDiffView_FromUnified(t *testing.T) {
	t.Parallel()

	unifiedDiff := `--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 context line
-removed line
+added line
+another added line
 more context`

	diff := NewDiffView().FromUnified(unifiedDiff)

	assert.Len(t, diff.Hunks(), 1)
	assert.NotEmpty(t, diff.Hunks()[0].Lines)
}

func TestDiffView_View(t *testing.T) {
	t.Parallel()

	hunks := []DiffHunk{
		{
			StartLine: 1,
			Lines: []DiffLine{
				{Type: DiffLineContext, Content: "context"},
				{Type: DiffLineAdd, Content: "added"},
				{Type: DiffLineRemove, Content: "removed"},
			},
		},
	}

	diff := NewDiffView().
		SetTitle("test.txt").
		SetHunks(hunks).
		WithWidth(60)

	view := diff.View()

	assert.Contains(t, view, "test.txt")
	assert.Contains(t, view, "added")
	assert.Contains(t, view, "removed")
}

func TestDiffView_Scroll(t *testing.T) {
	t.Parallel()

	lines := make([]DiffLine, 20)
	for i := range lines {
		lines[i] = DiffLine{Type: DiffLineContext, Content: "line"}
	}

	diff := NewDiffView().
		SetHunks([]DiffHunk{{Lines: lines}}).
		WithHeight(5)

	// Scroll down
	diff, _ = diff.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.GreaterOrEqual(t, diff.ScrollOffset(), 0)
}

func TestDiffView_Width(t *testing.T) {
	t.Parallel()

	diff := NewDiffView().WithWidth(100)

	assert.Equal(t, 100, diff.Width())
}

func TestDiffView_Height(t *testing.T) {
	t.Parallel()

	diff := NewDiffView().WithHeight(30)

	assert.Equal(t, 30, diff.Height())
}

func TestDiffLine_Types(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DiffLineContext, DiffLineType("context"))
	assert.Equal(t, DiffLineAdd, DiffLineType("add"))
	assert.Equal(t, DiffLineRemove, DiffLineType("remove"))
}

func TestDiffView_Summary(t *testing.T) {
	t.Parallel()

	hunks := []DiffHunk{
		{
			Lines: []DiffLine{
				{Type: DiffLineContext, Content: "context"},
				{Type: DiffLineAdd, Content: "add1"},
				{Type: DiffLineAdd, Content: "add2"},
				{Type: DiffLineRemove, Content: "remove"},
			},
		},
	}

	diff := NewDiffView().SetHunks(hunks)
	summary := diff.Summary()

	assert.Equal(t, 2, summary.Additions)
	assert.Equal(t, 1, summary.Deletions)
}
