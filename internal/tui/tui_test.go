package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	t.Parallel()

	app := NewApp()

	assert.NotNil(t, app)
	assert.NotNil(t, app.Styles())
	assert.NotNil(t, app.Keys())
}

func TestApp_WithDimensions(t *testing.T) {
	t.Parallel()

	app := NewApp().
		WithWidth(100).
		WithHeight(40)

	assert.Equal(t, 100, app.Width())
	assert.Equal(t, 40, app.Height())
}

func TestNewInitWizardOptions(t *testing.T) {
	t.Parallel()

	opts := NewInitWizardOptions()

	assert.False(t, opts.SkipWelcome)
	assert.Empty(t, opts.PreselectedProvider)
}

func TestInitWizardOptions_WithPreselection(t *testing.T) {
	t.Parallel()

	opts := NewInitWizardOptions().
		WithPreselectedProvider("nvim")

	assert.Equal(t, "nvim", opts.PreselectedProvider)
}

func TestNewPlanReviewOptions(t *testing.T) {
	t.Parallel()

	opts := NewPlanReviewOptions()

	assert.False(t, opts.AutoApprove)
	assert.True(t, opts.ShowExplanations)
}

func TestNewApplyProgressOptions(t *testing.T) {
	t.Parallel()

	opts := NewApplyProgressOptions()

	assert.False(t, opts.Quiet)
	assert.True(t, opts.ShowDetails)
}

func TestNewDoctorReportOptions(t *testing.T) {
	t.Parallel()

	opts := NewDoctorReportOptions()

	assert.False(t, opts.AutoFix)
	assert.True(t, opts.Verbose)
}

func TestNewCaptureReviewOptions(t *testing.T) {
	t.Parallel()

	opts := NewCaptureReviewOptions()

	assert.False(t, opts.AcceptAll)
	assert.True(t, opts.Interactive)
}

func TestInitWizardOptions_WithPreselectedPreset(t *testing.T) {
	t.Parallel()

	opts := NewInitWizardOptions().
		WithPreselectedPreset("nvim:balanced")

	assert.Equal(t, "nvim:balanced", opts.PreselectedPreset)
}

func TestInitWizardOptions_WithSkipWelcome(t *testing.T) {
	t.Parallel()

	opts := NewInitWizardOptions().
		WithSkipWelcome(true)

	assert.True(t, opts.SkipWelcome)
}

func TestPlanReviewOptions_WithAutoApprove(t *testing.T) {
	t.Parallel()

	opts := NewPlanReviewOptions().
		WithAutoApprove(true)

	assert.True(t, opts.AutoApprove)
}

func TestApplyProgressOptions_WithQuiet(t *testing.T) {
	t.Parallel()

	opts := NewApplyProgressOptions().
		WithQuiet(true)

	assert.True(t, opts.Quiet)
}

func TestDoctorReportOptions_WithAutoFix(t *testing.T) {
	t.Parallel()

	opts := NewDoctorReportOptions().
		WithAutoFix(true)

	assert.True(t, opts.AutoFix)
}

func TestCaptureReviewOptions_WithAcceptAll(t *testing.T) {
	t.Parallel()

	opts := NewCaptureReviewOptions().
		WithAcceptAll(true)

	assert.True(t, opts.AcceptAll)
}

func TestRunCaptureReview_AcceptAllBypassesTUI(t *testing.T) {
	t.Parallel()

	// Create test items
	items := []CaptureItem{
		{Name: "git", Category: "brew", Type: CaptureTypeFormula, Details: "version control"},
		{Name: "nvim", Category: "brew", Type: CaptureTypeFormula, Details: "editor"},
	}

	// With AcceptAll, RunCaptureReview should return immediately without TUI
	opts := NewCaptureReviewOptions().WithAcceptAll(true)
	result, err := RunCaptureReview(t.Context(), items, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Cancelled)
	assert.Len(t, result.AcceptedItems, 2)
	assert.Empty(t, result.RejectedItems)

	// Verify items are returned correctly
	assert.Equal(t, "git", result.AcceptedItems[0].Name)
	assert.Equal(t, "nvim", result.AcceptedItems[1].Name)
}
