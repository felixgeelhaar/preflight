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
