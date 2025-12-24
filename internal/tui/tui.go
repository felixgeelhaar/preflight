// Package tui provides terminal user interface entry points for preflight.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/tui/ui"
)

// CatalogServiceInterface defines the catalog service interface for the TUI.
// This allows for dependency injection and testing.
type CatalogServiceInterface interface {
	GetProviders() []string
	GetPresetsForProvider(provider string) []PresetItem
	GetCapabilityPacks() []PackItem
	GetPreset(id string) (PresetItem, bool)
}

// PresetItem represents a preset for display in the TUI.
type PresetItem struct {
	ID          string
	Title       string
	Description string
	Difficulty  string
}

// PackItem represents a capability pack for display in the TUI.
type PackItem struct {
	ID          string
	Title       string
	Description string
}

// App represents the main TUI application state.
type App struct {
	width  int
	height int
	styles ui.Styles
	keys   ui.KeyMap
}

// NewApp creates a new TUI application.
func NewApp() *App {
	return &App{
		width:  80,
		height: 24,
		styles: ui.DefaultStyles(),
		keys:   ui.DefaultKeyMap(),
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
func (a *App) Styles() ui.Styles {
	return a.styles
}

// Keys returns the key bindings.
func (a *App) Keys() ui.KeyMap {
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
	CatalogService      CatalogServiceInterface
	TargetDir           string
	Advisor             advisor.AIProvider
	SkipInterview       bool
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

// WithCatalogService sets the catalog service.
func (o InitWizardOptions) WithCatalogService(service CatalogServiceInterface) InitWizardOptions {
	o.CatalogService = service
	return o
}

// WithTargetDir sets the target directory for generated config files.
func (o InitWizardOptions) WithTargetDir(dir string) InitWizardOptions {
	o.TargetDir = dir
	return o
}

// WithAdvisor sets the AI advisor for the interview.
func (o InitWizardOptions) WithAdvisor(adv advisor.AIProvider) InitWizardOptions {
	o.Advisor = adv
	return o
}

// WithSkipInterview skips the AI interview step.
func (o InitWizardOptions) WithSkipInterview(skip bool) InitWizardOptions {
	o.SkipInterview = skip
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
	AcceptAll       bool
	Interactive     bool
	AvailableLayers []string // Available layers for reassignment
}

// DefaultLayers returns the default available layers.
func DefaultLayers() []string {
	return []string{
		"base",
		"identity.work",
		"identity.personal",
		"role.dev",
		"device.laptop",
		"captured",
	}
}

// NewCaptureReviewOptions creates default capture review options.
func NewCaptureReviewOptions() CaptureReviewOptions {
	return CaptureReviewOptions{
		Interactive:     true,
		AvailableLayers: DefaultLayers(),
	}
}

// WithAcceptAll accepts all captured items.
func (o CaptureReviewOptions) WithAcceptAll(all bool) CaptureReviewOptions {
	o.AcceptAll = all
	return o
}

// WithAvailableLayers sets the available layers for reassignment.
func (o CaptureReviewOptions) WithAvailableLayers(layers []string) CaptureReviewOptions {
	o.AvailableLayers = layers
	return o
}

// CaptureItemResult holds rich information about a reviewed capture item.
type CaptureItemResult struct {
	Name     string
	Category string
	Type     CaptureType
	Layer    string
	Value    string
}

// CaptureReviewResult holds the result of capture review.
type CaptureReviewResult struct {
	AcceptedItems []CaptureItemResult
	RejectedItems []CaptureItemResult
	Cancelled     bool
}

// ToCaptureItemResult converts a CaptureItem to a CaptureItemResult.
func ToCaptureItemResult(item CaptureItem) CaptureItemResult {
	layer := item.Layer
	if layer == "" {
		layer = "captured"
	}
	return CaptureItemResult{
		Name:     item.Name,
		Category: item.Category,
		Type:     item.Type,
		Layer:    layer,
		Value:    item.Value,
	}
}

// ToCaptureItemResults converts a slice of CaptureItems to CaptureItemResults.
func ToCaptureItemResults(items []CaptureItem) []CaptureItemResult {
	if items == nil {
		return []CaptureItemResult{}
	}
	results := make([]CaptureItemResult, 0, len(items))
	for _, item := range items {
		results = append(results, ToCaptureItemResult(item))
	}
	return results
}

// NewLayerPreviewOptions creates default layer preview options.
func NewLayerPreviewOptions() LayerPreviewOptions {
	return LayerPreviewOptions{
		Title:        "Layer Preview",
		ShowLineNums: false,
	}
}

// WithTitle sets a custom title for the preview.
func (o LayerPreviewOptions) WithTitle(title string) LayerPreviewOptions {
	o.Title = title
	return o
}

// WithLineNumbers enables line number display.
func (o LayerPreviewOptions) WithLineNumbers(show bool) LayerPreviewOptions {
	o.ShowLineNums = show
	return o
}

// RunLayerPreview runs the layer preview interface.
func RunLayerPreview(ctx context.Context, files []PreviewFile, opts LayerPreviewOptions) (*LayerPreviewResult, error) {
	if len(files) == 0 {
		return &LayerPreviewResult{Confirmed: true}, nil
	}

	// Create the layer preview model
	model := newLayerPreviewModel(files, opts)

	// Run the program
	p := tea.NewProgram(model, tea.WithContext(ctx))
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("layer preview failed: %w", err)
	}

	// Extract result from final model
	m, ok := finalModel.(layerPreviewModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return &LayerPreviewResult{
		Confirmed: m.confirmed,
		Cancelled: m.cancelled,
	}, nil
}

// RunCaptureReview runs the capture review interface.
func RunCaptureReview(ctx context.Context, items []CaptureItem, opts CaptureReviewOptions) (*CaptureReviewResult, error) {
	// Handle accept all case
	if opts.AcceptAll {
		return &CaptureReviewResult{
			AcceptedItems: ToCaptureItemResults(items),
			RejectedItems: []CaptureItemResult{},
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

	return &CaptureReviewResult{
		AcceptedItems: ToCaptureItemResults(m.accepted),
		RejectedItems: ToCaptureItemResults(m.rejected),
		Cancelled:     m.cancelled,
	}, nil
}
