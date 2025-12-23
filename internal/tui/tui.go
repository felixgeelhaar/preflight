// Package tui provides terminal user interface entry points for preflight.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/tui/common"
)

// App represents the main TUI application state.
type App struct {
	width  int
	height int
	styles common.Styles
	keys   common.KeyMap
}

// NewApp creates a new TUI application.
func NewApp() *App {
	return &App{
		width:  80,
		height: 24,
		styles: common.DefaultStyles(),
		keys:   common.DefaultKeyMap(),
	}
}

// Width returns the terminal width.
func (a *App) Width() int {
	return a.width
}

// Height returns the terminal height.
func (a *App) Height() int {
	return a.height
}

// Styles returns the application styles.
func (a *App) Styles() common.Styles {
	return a.styles
}

// Keys returns the key bindings.
func (a *App) Keys() common.KeyMap {
	return a.keys
}

// WithWidth sets the terminal width.
func (a *App) WithWidth(width int) *App {
	a.width = width
	a.styles = a.styles.WithWidth(width)
	return a
}

// WithHeight sets the terminal height.
func (a *App) WithHeight(height int) *App {
	a.height = height
	return a
}

// InitWizardOptions configures the init wizard.
type InitWizardOptions struct {
	SkipWelcome         bool
	PreselectedProvider string
	PreselectedPreset   string
}

// NewInitWizardOptions creates default init wizard options.
func NewInitWizardOptions() InitWizardOptions {
	return InitWizardOptions{}
}

// WithPreselectedProvider sets a pre-selected provider.
func (o InitWizardOptions) WithPreselectedProvider(provider string) InitWizardOptions {
	o.PreselectedProvider = provider
	return o
}

// WithPreselectedPreset sets a pre-selected preset.
func (o InitWizardOptions) WithPreselectedPreset(preset string) InitWizardOptions {
	o.PreselectedPreset = preset
	return o
}

// WithSkipWelcome skips the welcome screen.
func (o InitWizardOptions) WithSkipWelcome(skip bool) InitWizardOptions {
	o.SkipWelcome = skip
	return o
}

// InitWizardResult holds the result of the init wizard.
type InitWizardResult struct {
	ConfigPath     string
	SelectedPreset string
	Cancelled      bool
}

// RunInitWizard runs the interactive init wizard.
func RunInitWizard(ctx context.Context, opts InitWizardOptions) (*InitWizardResult, error) {
	// Create the init wizard model
	model := newInitWizardModel(opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("init wizard failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(initWizardModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return &InitWizardResult{
		ConfigPath:     m.configPath,
		SelectedPreset: m.selectedPreset,
		Cancelled:      m.cancelled,
	}, nil
}

// PlanReviewOptions configures the plan review TUI.
type PlanReviewOptions struct {
	AutoApprove      bool
	ShowExplanations bool
}

// NewPlanReviewOptions creates default plan review options.
func NewPlanReviewOptions() PlanReviewOptions {
	return PlanReviewOptions{
		ShowExplanations: true,
	}
}

// WithAutoApprove enables automatic approval.
func (o PlanReviewOptions) WithAutoApprove(auto bool) PlanReviewOptions {
	o.AutoApprove = auto
	return o
}

// PlanReviewResult holds the result of plan review.
type PlanReviewResult struct {
	Approved  bool
	Cancelled bool
}

// RunPlanReview runs the interactive plan review.
func RunPlanReview(_ context.Context, opts PlanReviewOptions) (*PlanReviewResult, error) {
	// This will be implemented with a full plan review model
	// For now, return a placeholder
	return &PlanReviewResult{
		Approved: opts.AutoApprove,
	}, nil
}

// ApplyProgressOptions configures the apply progress TUI.
type ApplyProgressOptions struct {
	Quiet       bool
	ShowDetails bool
}

// NewApplyProgressOptions creates default apply progress options.
func NewApplyProgressOptions() ApplyProgressOptions {
	return ApplyProgressOptions{
		ShowDetails: true,
	}
}

// WithQuiet enables quiet mode.
func (o ApplyProgressOptions) WithQuiet(quiet bool) ApplyProgressOptions {
	o.Quiet = quiet
	return o
}

// ApplyProgressResult holds the result of apply progress.
type ApplyProgressResult struct {
	Success    bool
	StepsTotal int
	StepsDone  int
	Errors     []error
}

// RunApplyProgress runs the apply progress display.
func RunApplyProgress(_ context.Context, _ ApplyProgressOptions) (*ApplyProgressResult, error) {
	// This will be implemented with a full apply progress model
	return &ApplyProgressResult{
		Success: true,
	}, nil
}

// DoctorReportOptions configures the doctor report TUI.
type DoctorReportOptions struct {
	AutoFix bool
	Verbose bool
}

// NewDoctorReportOptions creates default doctor report options.
func NewDoctorReportOptions() DoctorReportOptions {
	return DoctorReportOptions{
		Verbose: true,
	}
}

// WithAutoFix enables automatic fixing.
func (o DoctorReportOptions) WithAutoFix(fix bool) DoctorReportOptions {
	o.AutoFix = fix
	return o
}

// DoctorReportResult holds the result of doctor report.
type DoctorReportResult struct {
	Issues     int
	FixesFound int
	Fixed      int
}

// RunDoctorReport runs the doctor report display.
func RunDoctorReport(_ context.Context, _ DoctorReportOptions) (*DoctorReportResult, error) {
	// This will be implemented with a full doctor report model
	return &DoctorReportResult{}, nil
}

// CaptureReviewOptions configures the capture review TUI.
type CaptureReviewOptions struct {
	AcceptAll   bool
	Interactive bool
}

// NewCaptureReviewOptions creates default capture review options.
func NewCaptureReviewOptions() CaptureReviewOptions {
	return CaptureReviewOptions{
		Interactive: true,
	}
}

// WithAcceptAll accepts all captured items.
func (o CaptureReviewOptions) WithAcceptAll(all bool) CaptureReviewOptions {
	o.AcceptAll = all
	return o
}

// CaptureReviewResult holds the result of capture review.
type CaptureReviewResult struct {
	AcceptedItems []string
	RejectedItems []string
	Cancelled     bool
}

// RunCaptureReview runs the capture review interface.
func RunCaptureReview(_ context.Context, _ CaptureReviewOptions) (*CaptureReviewResult, error) {
	// This will be implemented with a full capture review model
	return &CaptureReviewResult{}, nil
}
