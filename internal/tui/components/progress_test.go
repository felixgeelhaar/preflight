package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProgress(t *testing.T) {
	t.Parallel()

	progress := NewProgress()

	assert.Equal(t, 0.0, progress.Percent())
	assert.Empty(t, progress.Message())
}

func TestProgress_SetPercent(t *testing.T) {
	t.Parallel()

	progress := NewProgress().SetPercent(0.5)

	assert.Equal(t, 0.5, progress.Percent())
}

func TestProgress_SetPercent_Clamps(t *testing.T) {
	t.Parallel()

	// Test upper bound
	progress := NewProgress().SetPercent(1.5)
	assert.Equal(t, 1.0, progress.Percent())

	// Test lower bound
	progress = NewProgress().SetPercent(-0.5)
	assert.Equal(t, 0.0, progress.Percent())
}

func TestProgress_SetMessage(t *testing.T) {
	t.Parallel()

	progress := NewProgress().SetMessage("Installing packages...")

	assert.Equal(t, "Installing packages...", progress.Message())
}

func TestProgress_SetCurrent(t *testing.T) {
	t.Parallel()

	progress := NewProgress().SetTotal(10).SetCurrent(5)

	assert.Equal(t, 5, progress.Current())
	assert.Equal(t, 10, progress.Total())
	assert.Equal(t, 0.5, progress.Percent())
}

func TestProgress_IncrementCurrent(t *testing.T) {
	t.Parallel()

	progress := NewProgress().SetTotal(10).SetCurrent(3)
	progress = progress.IncrementCurrent()

	assert.Equal(t, 4, progress.Current())
}

func TestProgress_IncrementCurrent_ClampsToTotal(t *testing.T) {
	t.Parallel()

	progress := NewProgress().SetTotal(10).SetCurrent(10)
	progress = progress.IncrementCurrent()

	assert.Equal(t, 10, progress.Current())
}

func TestProgress_Width(t *testing.T) {
	t.Parallel()

	progress := NewProgress().WithWidth(60)

	assert.Equal(t, 60, progress.Width())
}

func TestProgress_View(t *testing.T) {
	t.Parallel()

	progress := NewProgress().
		SetPercent(0.5).
		SetMessage("Processing...").
		WithWidth(40)

	view := progress.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Processing...")
}

func TestProgress_View_WithItems(t *testing.T) {
	t.Parallel()

	progress := NewProgress().
		SetTotal(10).
		SetCurrent(5).
		SetMessage("Step 5 of 10").
		WithWidth(40)

	view := progress.View()

	assert.NotEmpty(t, view)
}

func TestNewSpinner(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner()

	assert.NotNil(t, spinner)
	assert.Empty(t, spinner.Message())
}

func TestSpinner_SetMessage(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner().SetMessage("Loading...")

	assert.Equal(t, "Loading...", spinner.Message())
}

func TestSpinner_View(t *testing.T) {
	t.Parallel()

	spinner := NewSpinner().SetMessage("Working...")

	view := spinner.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Working...")
}
