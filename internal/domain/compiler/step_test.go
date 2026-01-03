package compiler

import (
	"context"
	"errors"
	"testing"
)

// mockStep is a test double for Step interface.
type mockStep struct {
	id        StepID
	deps      []StepID
	checkFn   func(RunContext) (StepStatus, error)
	planFn    func(RunContext) (Diff, error)
	applyFn   func(RunContext) error
	explainFn func(ExplainContext) Explanation
}

func newMockStep(id string, deps ...string) *mockStep {
	stepID, _ := NewStepID(id)
	depIDs := make([]StepID, len(deps))
	for i, d := range deps {
		depIDs[i], _ = NewStepID(d)
	}
	return &mockStep{
		id:   stepID,
		deps: depIDs,
		checkFn: func(RunContext) (StepStatus, error) {
			return StatusNeedsApply, nil
		},
		planFn: func(RunContext) (Diff, error) {
			return NewDiff(DiffTypeAdd, "test", "resource", "", "new"), nil
		},
		applyFn: func(RunContext) error {
			return nil
		},
		explainFn: func(ExplainContext) Explanation {
			return NewExplanation("Test step", "For testing", nil)
		},
	}
}

func (m *mockStep) ID() StepID                               { return m.id }
func (m *mockStep) DependsOn() []StepID                      { return m.deps }
func (m *mockStep) Check(ctx RunContext) (StepStatus, error) { return m.checkFn(ctx) }
func (m *mockStep) Plan(ctx RunContext) (Diff, error)        { return m.planFn(ctx) }
func (m *mockStep) Apply(ctx RunContext) error               { return m.applyFn(ctx) }
func (m *mockStep) Explain(ctx ExplainContext) Explanation   { return m.explainFn(ctx) }

func TestStep_Interface(t *testing.T) {
	step := newMockStep("brew:install:git")

	// Test ID
	if step.ID().String() != "brew:install:git" {
		t.Errorf("ID() = %q, want %q", step.ID().String(), "brew:install:git")
	}

	// Test DependsOn returns empty slice
	if len(step.DependsOn()) != 0 {
		t.Errorf("DependsOn() len = %d, want 0", len(step.DependsOn()))
	}
}

func TestStep_WithDependencies(t *testing.T) {
	step := newMockStep("nvim:install:plugin", "brew:install:nvim")

	deps := step.DependsOn()
	if len(deps) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != "brew:install:nvim" {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), "brew:install:nvim")
	}
}

func TestStep_Check(t *testing.T) {
	step := newMockStep("brew:install:git")
	ctx := NewRunContext(context.Background())

	status, err := step.Check(ctx)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status != StatusNeedsApply {
		t.Errorf("Check() status = %v, want %v", status, StatusNeedsApply)
	}
}

func TestStep_Check_Error(t *testing.T) {
	step := newMockStep("brew:install:git")
	step.checkFn = func(RunContext) (StepStatus, error) {
		return StatusUnknown, errors.New("check failed")
	}

	ctx := NewRunContext(context.Background())
	status, err := step.Check(ctx)
	if err == nil {
		t.Fatal("expected error from Check()")
	}
	if status != StatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, StatusUnknown)
	}
}

func TestStep_Plan(t *testing.T) {
	step := newMockStep("brew:install:git")
	ctx := NewRunContext(context.Background())

	diff, err := step.Plan(ctx)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if diff.Type() != DiffTypeAdd {
		t.Errorf("Plan() diff type = %v, want %v", diff.Type(), DiffTypeAdd)
	}
}

func TestStep_Apply(t *testing.T) {
	applied := false
	step := newMockStep("brew:install:git")
	step.applyFn = func(RunContext) error {
		applied = true
		return nil
	}

	ctx := NewRunContext(context.Background())
	err := step.Apply(ctx)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !applied {
		t.Error("Apply() was not called")
	}
}

func TestStep_Explain(t *testing.T) {
	step := newMockStep("brew:install:git")
	ctx := NewExplainContext()

	explanation := step.Explain(ctx)
	if explanation.Summary() != "Test step" {
		t.Errorf("Explain().Summary() = %q, want %q", explanation.Summary(), "Test step")
	}
}

func TestRunContext_Creation(t *testing.T) {
	ctx := NewRunContext(context.Background())
	if ctx.Context() == nil {
		t.Error("Context() should not be nil")
	}
}

func TestRunContext_WithDryRun(t *testing.T) {
	ctx := NewRunContext(context.Background())
	if ctx.DryRun() {
		t.Error("DryRun() should default to false")
	}

	dryCtx := ctx.WithDryRun(true)
	if !dryCtx.DryRun() {
		t.Error("WithDryRun(true) should set DryRun to true")
	}
	// Original should be unchanged
	if ctx.DryRun() {
		t.Error("original context should be unchanged")
	}
}

func TestExplainContext_Creation(t *testing.T) {
	ctx := NewExplainContext()
	// ExplainContext is a value type, just verify it's usable
	if ctx.Verbose() {
		t.Error("Verbose() should default to false")
	}
}

func TestExplainContext_WithVerbose(t *testing.T) {
	ctx := NewExplainContext()
	verboseCtx := ctx.WithVerbose(true)
	if !verboseCtx.Verbose() {
		t.Error("WithVerbose(true) should set Verbose to true")
	}
	// Original should be unchanged (immutable)
	if ctx.Verbose() {
		t.Error("original context should be unchanged")
	}
}

func TestExplainContext_Provenance(t *testing.T) {
	ctx := NewExplainContext()
	if ctx.Provenance() != "" {
		t.Error("Provenance() should default to empty string")
	}
}

func TestExplainContext_WithProvenance(t *testing.T) {
	ctx := NewExplainContext()
	layerPath := "layers/identity.work.yaml"
	ctxWithProv := ctx.WithProvenance(layerPath)

	if ctxWithProv.Provenance() != layerPath {
		t.Errorf("Provenance() = %q, want %q", ctxWithProv.Provenance(), layerPath)
	}
	// Original should be unchanged (immutable)
	if ctx.Provenance() != "" {
		t.Error("original context should be unchanged")
	}
}

// rollbackableMockStep implements RollbackableStep interface for testing.
type rollbackableMockStep struct {
	*mockStep
	canRollback bool
	rollbackFn  func(RunContext) error
}

func newRollbackableMockStep(id string, deps ...string) *rollbackableMockStep {
	return &rollbackableMockStep{
		mockStep:    newMockStep(id, deps...),
		canRollback: true,
		rollbackFn: func(_ RunContext) error {
			return nil
		},
	}
}

func (r *rollbackableMockStep) CanRollback() bool {
	return r.canRollback
}

func (r *rollbackableMockStep) Rollback(ctx RunContext) error {
	return r.rollbackFn(ctx)
}

func TestIsRollbackable_RegularStep(t *testing.T) {
	step := newMockStep("brew:install:git")
	if IsRollbackable(step) {
		t.Error("IsRollbackable should return false for regular step")
	}
}

func TestIsRollbackable_RollbackableStep(t *testing.T) {
	step := newRollbackableMockStep("brew:install:git", "brew:tap:homebrew/core")
	if !IsRollbackable(step) {
		t.Error("IsRollbackable should return true for rollbackable step")
	}
}

func TestAsRollbackable_RegularStep(t *testing.T) {
	step := newMockStep("brew:install:git")
	if AsRollbackable(step) != nil {
		t.Error("AsRollbackable should return nil for regular step")
	}
}

func TestAsRollbackable_RollbackableStep(t *testing.T) {
	step := newRollbackableMockStep("brew:install:git")
	rollbackable := AsRollbackable(step)
	if rollbackable == nil {
		t.Fatal("AsRollbackable should return non-nil for rollbackable step")
	}
	if rollbackable.ID().String() != "brew:install:git" {
		t.Errorf("ID() = %q, want %q", rollbackable.ID().String(), "brew:install:git")
	}
}

func TestRollbackableStep_CanRollback(t *testing.T) {
	step := newRollbackableMockStep("brew:install:git")
	rollbackable := AsRollbackable(step)

	if !rollbackable.CanRollback() {
		t.Error("CanRollback() should return true by default")
	}

	step.canRollback = false
	if rollbackable.CanRollback() {
		t.Error("CanRollback() should return false after setting")
	}
}

func TestRollbackableStep_Rollback(t *testing.T) {
	step := newRollbackableMockStep("brew:install:curl")
	rolledBack := false
	step.rollbackFn = func(_ RunContext) error {
		rolledBack = true
		return nil
	}

	rollbackable := AsRollbackable(step)
	ctx := NewRunContext(context.Background())
	err := rollbackable.Rollback(ctx)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if !rolledBack {
		t.Error("Rollback() was not called")
	}
}

func TestRollbackableStep_Rollback_Error(t *testing.T) {
	step := newRollbackableMockStep("brew:install:wget")
	step.rollbackFn = func(_ RunContext) error {
		return errors.New("rollback failed")
	}

	rollbackable := AsRollbackable(step)
	ctx := NewRunContext(context.Background())
	err := rollbackable.Rollback(ctx)
	if err == nil {
		t.Fatal("expected error from Rollback()")
	}
	if err.Error() != "rollback failed" {
		t.Errorf("error = %q, want %q", err.Error(), "rollback failed")
	}
}
