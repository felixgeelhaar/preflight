package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/stretchr/testify/assert"
)

func TestApplyProgressModel_Init(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
}

func TestApplyProgressModel_View(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Applying", "should contain applying header")
}

func TestApplyProgressModel_EmptyPlan(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Nothing to apply", "should show nothing to apply message")
}

func TestApplyProgressModel_WindowResize(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(applyProgressModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestApplyProgressModel_StepStart(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	// Simulate step starting
	stepID := mustNewStepID(t, "brew:formula:git")
	newModel, _ := model.Update(StepStartMsg{StepID: stepID})
	m := newModel.(applyProgressModel)

	assert.Equal(t, stepID, m.currentStep)
}

func TestApplyProgressModel_StepComplete(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	// Simulate step completing
	stepID := mustNewStepID(t, "brew:formula:git")
	model.currentStep = stepID

	result := execution.NewStepResult(stepID, compiler.StatusSatisfied, nil)
	newModel, _ := model.Update(StepCompleteMsg{Result: result})
	m := newModel.(applyProgressModel)

	assert.Len(t, m.completed, 1)
	assert.Equal(t, 1, m.stepsCompleted)
}

func TestApplyProgressModel_AllComplete(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24
	model.stepsTotal = 1
	model.stepsCompleted = 0

	// Simulate step completing
	stepID := mustNewStepID(t, "brew:formula:git")
	result := execution.NewStepResult(stepID, compiler.StatusSatisfied, nil)
	newModel, cmd := model.Update(StepCompleteMsg{Result: result})
	m := newModel.(applyProgressModel)

	assert.True(t, m.done)
	assert.NotNil(t, cmd, "should return quit command when all complete")
}

func TestApplyProgressModel_StepFailed(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24
	model.stepsTotal = 1

	// Simulate step failing
	stepID := mustNewStepID(t, "brew:formula:git")
	result := execution.NewStepResult(stepID, compiler.StatusFailed, nil)
	newModel, _ := model.Update(StepCompleteMsg{Result: result})
	m := newModel.(applyProgressModel)

	assert.Equal(t, 1, m.stepsFailed)
}

func TestApplyProgressModel_Cancel(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	// Press Ctrl+C to cancel
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m := newModel.(applyProgressModel)

	assert.True(t, m.cancelled)
	assert.NotNil(t, cmd, "should return quit command")
}

func TestApplyProgressModel_QuietMode(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{Quiet: true})
	model.width = 100
	model.height = 24

	view := model.View()

	// Quiet mode should show minimal output
	assert.NotEmpty(t, view, "should still produce some output")
}

func TestApplyProgressModel_ProgressCalculation(t *testing.T) {
	t.Parallel()

	plan := createTestPlanWithMultipleEntries(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24

	// Initial progress should be 0
	assert.InDelta(t, 0.0, model.progress(), 0.001)

	// Complete one step
	model.stepsCompleted = 1
	model.stepsTotal = 3
	assert.InDelta(t, 0.333, model.progress(), 0.01)

	// Complete all steps
	model.stepsCompleted = 3
	assert.InDelta(t, 1.0, model.progress(), 0.001)
}

func TestApplyProgressModel_ProgressWithZeroTotal(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24
	model.stepsTotal = 0

	// Progress should be 0 when total is 0
	assert.InDelta(t, 0.0, model.progress(), 0.001)
}

func TestApplyProgressModel_FormatResultStatus(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{})
	stepID := mustNewStepID(t, "test:step")

	tests := []struct {
		name     string
		status   compiler.StepStatus
		expected string
	}{
		{"satisfied", compiler.StatusSatisfied, "✓"},
		{"failed", compiler.StatusFailed, "✗"},
		{"skipped", compiler.StatusSkipped, "-"},
		{"needs_apply", compiler.StatusNeedsApply, "?"},
		{"unknown", compiler.StatusUnknown, "?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := execution.NewStepResult(stepID, tt.status, nil)
			status := model.formatResultStatus(result)
			assert.Contains(t, status, tt.expected)
		})
	}
}

func TestApplyProgressModel_View_WithCompletedSteps(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 30

	// Add a completed step
	stepID := mustNewStepID(t, "brew:formula:git")
	result := execution.NewStepResult(stepID, compiler.StatusSatisfied, nil)
	model.completed = append(model.completed, result)
	model.stepsCompleted = 1
	model.stepsTotal = 2

	view := model.View()
	assert.NotEmpty(t, view)
}

func TestApplyProgressModel_View_Done(t *testing.T) {
	t.Parallel()

	plan := createTestPlan(t)
	model := newApplyProgressModel(plan, ApplyProgressOptions{ShowDetails: true})
	model.width = 100
	model.height = 24
	model.done = true
	model.stepsCompleted = 1
	model.stepsTotal = 1

	view := model.View()
	assert.Contains(t, view, "completed successfully")
}
