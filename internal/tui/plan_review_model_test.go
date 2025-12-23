package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanReviewModel_Init(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
}

func TestPlanReviewModel_View(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Plan Review", "should contain header")
	assert.Contains(t, view, "brew:formula:git", "should show step ID")
}

func TestPlanReviewModel_EmptyPlan(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "No changes", "should show no changes message")
}

func TestPlanReviewModel_Navigation(t *testing.T) {
	t.Parallel()

	plan := createTestPlanWithMultipleEntries(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	// Initial cursor should be at 0
	assert.Equal(t, 0, model.Cursor())

	// Navigate down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(planReviewModel)
	assert.Equal(t, 1, m.Cursor())
}

func TestPlanReviewModel_Approve(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	// Press 'a' to approve
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m := newModel.(planReviewModel)

	assert.True(t, m.approved, "should be approved")
	require.NotNil(t, cmd, "should return quit command")
}

func TestPlanReviewModel_Cancel(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	// Press 'q' to cancel
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(planReviewModel)

	assert.True(t, m.cancelled, "should be cancelled")
	require.NotNil(t, cmd, "should return quit command")
}

func TestPlanReviewModel_EscapeCancel(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	// Press Escape to cancel
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m := newModel.(planReviewModel)

	assert.True(t, m.cancelled, "should be cancelled")
	require.NotNil(t, cmd, "should return quit command")
}

func TestPlanReviewModel_EnterApprove(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	// Press Enter to approve
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(planReviewModel)

	assert.True(t, m.approved, "should be approved")
	require.NotNil(t, cmd, "should return quit command")
}

func TestPlanReviewModel_WindowResize(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})

	// Send window size message
	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(planReviewModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestPlanReviewModel_StatusIndicators(t *testing.T) {
	t.Parallel()

	plan := createTestPlanWithStatuses(t)
	model := newPlanReviewModel(plan, PlanReviewOptions{ShowExplanations: true})
	model.width = 100
	model.height = 24

	view := model.View()

	// Should show different indicators for different statuses
	assert.Contains(t, view, "+", "should show + for needs-apply")
	assert.Contains(t, view, "✓", "should show ✓ for satisfied")
}

// Helper functions to create test plans

func mustNewStepID(t *testing.T, value string) compiler.StepID {
	t.Helper()
	id, err := compiler.NewStepID(value)
	if err != nil {
		t.Fatalf("failed to create step ID %q: %v", value, err)
	}
	return id
}

func createTestPlan(t *testing.T) *execution.Plan {
	t.Helper()

	plan := execution.NewExecutionPlan()
	step := &mockStep{
		id:      mustNewStepID(t, "brew:formula:git"),
		deps:    []compiler.StepID{},
		status:  compiler.StatusNeedsApply,
		diff:    compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.42.0"),
		explain: compiler.NewExplanation("Install git", "Git is a distributed version control system.", []string{"https://git-scm.com"}),
	}
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, step.diff))
	return plan
}

func createTestPlanWithMultipleEntries(t *testing.T) *execution.Plan {
	t.Helper()

	plan := execution.NewExecutionPlan()

	steps := []struct {
		id   string
		name string
	}{
		{"brew:formula:git", "git"},
		{"brew:formula:curl", "curl"},
		{"brew:formula:jq", "jq"},
	}

	for _, s := range steps {
		step := &mockStep{
			id:      mustNewStepID(t, s.id),
			deps:    []compiler.StepID{},
			status:  compiler.StatusNeedsApply,
			diff:    compiler.NewDiff(compiler.DiffTypeAdd, "package", s.name, "", "latest"),
			explain: compiler.NewExplanation("Install "+s.name, "Install package "+s.name, nil),
		}
		plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, step.diff))
	}

	return plan
}

func createTestPlanWithStatuses(t *testing.T) *execution.Plan {
	t.Helper()

	plan := execution.NewExecutionPlan()

	// NeedsApply step
	needsApplyStep := &mockStep{
		id:      mustNewStepID(t, "brew:formula:git"),
		deps:    []compiler.StepID{},
		status:  compiler.StatusNeedsApply,
		diff:    compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.42.0"),
		explain: compiler.NewExplanation("Install git", "Git version control", nil),
	}
	plan.Add(execution.NewPlanEntry(needsApplyStep, compiler.StatusNeedsApply, needsApplyStep.diff))

	// Satisfied step
	satisfiedStep := &mockStep{
		id:      mustNewStepID(t, "brew:formula:curl"),
		deps:    []compiler.StepID{},
		status:  compiler.StatusSatisfied,
		diff:    compiler.NewDiff(compiler.DiffTypeNone, "package", "curl", "8.0.0", "8.0.0"),
		explain: compiler.NewExplanation("Check curl", "Curl is already installed", nil),
	}
	plan.Add(execution.NewPlanEntry(satisfiedStep, compiler.StatusSatisfied, satisfiedStep.diff))

	return plan
}

// mockStep implements compiler.Step for testing.
type mockStep struct {
	id      compiler.StepID
	deps    []compiler.StepID
	status  compiler.StepStatus
	diff    compiler.Diff
	explain compiler.Explanation
}

func (m *mockStep) ID() compiler.StepID {
	return m.id
}

func (m *mockStep) DependsOn() []compiler.StepID {
	return m.deps
}

func (m *mockStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return m.status, nil
}

func (m *mockStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return m.diff, nil
}

func (m *mockStep) Apply(_ compiler.RunContext) error {
	return nil
}

func (m *mockStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return m.explain
}
