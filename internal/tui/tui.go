// Package tui provides terminal user interface entry points for preflight.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
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
func RunPlanReview(ctx context.Context, plan *execution.Plan, opts PlanReviewOptions) (*PlanReviewResult, error) {
	// Handle auto-approve case
	if opts.AutoApprove {
		return &PlanReviewResult{
			Approved: true,
		}, nil
	}

	// Create the plan review model
	model := newPlanReviewModel(plan, opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("plan review failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(planReviewModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return &PlanReviewResult{
		Approved:  m.approved,
		Cancelled: m.cancelled,
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
func RunApplyProgress(ctx context.Context, plan *execution.Plan, opts ApplyProgressOptions) (*ApplyProgressResult, error) {
	// Handle quiet mode - just return success without TUI
	if opts.Quiet {
		return &ApplyProgressResult{
			Success:    true,
			StepsTotal: len(plan.NeedsApply()),
			StepsDone:  len(plan.NeedsApply()),
		}, nil
	}

	// Create the apply progress model
	model := newApplyProgressModel(plan, opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("apply progress failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(applyProgressModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Collect errors from failed steps
	var errors []error
	for _, result := range m.completed {
		if result.Error() != nil {
			errors = append(errors, result.Error())
		}
	}

	return &ApplyProgressResult{
		Success:    m.stepsFailed == 0 && !m.cancelled,
		StepsTotal: m.stepsTotal,
		StepsDone:  m.stepsCompleted,
		Errors:     errors,
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
func RunDoctorReport(ctx context.Context, report *DoctorReport, opts DoctorReportOptions) (*DoctorReportResult, error) {
	// Create the doctor report model
	model := newDoctorReportModel(report, opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("doctor report failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(doctorReportModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return &DoctorReportResult{
		Issues:     len(m.report.Issues),
		FixesFound: m.report.FixableCount(),
		Fixed:      0, // Will be populated when fix functionality is implemented
	}, nil
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
func RunCaptureReview(ctx context.Context, items []CaptureItem, opts CaptureReviewOptions) (*CaptureReviewResult, error) {
	// Handle accept all case
	if opts.AcceptAll {
		accepted := make([]string, 0, len(items))
		for _, item := range items {
			accepted = append(accepted, item.Name)
		}
		return &CaptureReviewResult{
			AcceptedItems: accepted,
			RejectedItems: []string{},
		}, nil
	}

	// Create the capture review model
	model := newCaptureReviewModel(items, opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("capture review failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(captureReviewModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Build result lists
	accepted := make([]string, 0, len(m.accepted))
	for _, item := range m.accepted {
		accepted = append(accepted, item.Name)
	}

	rejected := make([]string, 0, len(m.rejected))
	for _, item := range m.rejected {
		rejected = append(rejected, item.Name)
	}

	return &CaptureReviewResult{
		AcceptedItems: accepted,
		RejectedItems: rejected,
		Cancelled:     m.cancelled,
	}, nil
}
