package execution

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// mockStep for testing
type mockStep struct {
	id   compiler.StepID
	deps []compiler.StepID
	diff compiler.Diff
}

func newMockStep(id string) *mockStep {
	stepID, _ := compiler.NewStepID(id)
	return &mockStep{
		id:   stepID,
		deps: nil,
		diff: compiler.NewDiff(compiler.DiffTypeAdd, "test", id, "", "new"),
	}
}

func (m *mockStep) ID() compiler.StepID          { return m.id }
func (m *mockStep) DependsOn() []compiler.StepID { return m.deps }
func (m *mockStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusNeedsApply, nil
}
func (m *mockStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return m.diff, nil
}
func (m *mockStep) Apply(_ compiler.RunContext) error {
	return nil
}
func (m *mockStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.NewExplanation("Test", "Test step", nil)
}

func TestExecutionPlan_Empty(t *testing.T) {
	plan := NewExecutionPlan()

	if plan.Len() != 0 {
		t.Errorf("Len() = %d, want 0", plan.Len())
	}
	if !plan.IsEmpty() {
		t.Error("IsEmpty() should be true")
	}
}

func TestExecutionPlan_AddEntry(t *testing.T) {
	plan := NewExecutionPlan()
	step := newMockStep("brew:install:git")
	diff := compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.43.0")

	plan.Add(NewPlanEntry(step, compiler.StatusNeedsApply, diff))

	if plan.Len() != 1 {
		t.Errorf("Len() = %d, want 1", plan.Len())
	}
}

func TestExecutionPlan_Entries(t *testing.T) {
	plan := NewExecutionPlan()
	step1 := newMockStep("brew:install:git")
	step2 := newMockStep("brew:install:curl")

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusSatisfied, compiler.Diff{}))

	entries := plan.Entries()
	if len(entries) != 2 {
		t.Errorf("Entries() len = %d, want 2", len(entries))
	}
}

func TestExecutionPlan_NeedsApply(t *testing.T) {
	plan := NewExecutionPlan()
	step1 := newMockStep("brew:install:git")
	step2 := newMockStep("brew:install:curl")
	step3 := newMockStep("apt:install:vim")

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusSatisfied, compiler.Diff{}))
	plan.Add(NewPlanEntry(step3, compiler.StatusNeedsApply, compiler.Diff{}))

	needsApply := plan.NeedsApply()
	if len(needsApply) != 2 {
		t.Errorf("NeedsApply() len = %d, want 2", len(needsApply))
	}
}

func TestExecutionPlan_HasChanges(t *testing.T) {
	plan := NewExecutionPlan()
	step := newMockStep("brew:install:git")

	// No changes initially
	if plan.HasChanges() {
		t.Error("HasChanges() should be false for empty plan")
	}

	// Add satisfied step (no change needed)
	plan.Add(NewPlanEntry(step, compiler.StatusSatisfied, compiler.Diff{}))
	if plan.HasChanges() {
		t.Error("HasChanges() should be false when all satisfied")
	}

	// Add step that needs apply
	step2 := newMockStep("brew:install:curl")
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))
	if !plan.HasChanges() {
		t.Error("HasChanges() should be true when steps need apply")
	}
}

func TestPlanEntry_Fields(t *testing.T) {
	step := newMockStep("brew:install:git")
	diff := compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.43.0")
	entry := NewPlanEntry(step, compiler.StatusNeedsApply, diff)

	if entry.Step().ID().String() != "brew:install:git" {
		t.Errorf("Step().ID() = %q, want %q", entry.Step().ID().String(), "brew:install:git")
	}
	if entry.Status() != compiler.StatusNeedsApply {
		t.Errorf("Status() = %v, want %v", entry.Status(), compiler.StatusNeedsApply)
	}
	if entry.Diff().Type() != compiler.DiffTypeAdd {
		t.Errorf("Diff().Type() = %v, want %v", entry.Diff().Type(), compiler.DiffTypeAdd)
	}
}

func TestExecutionPlan_Summary(t *testing.T) {
	plan := NewExecutionPlan()
	step1 := newMockStep("brew:install:git")
	step2 := newMockStep("brew:install:curl")
	step3 := newMockStep("apt:install:vim")

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusSatisfied, compiler.Diff{}))
	plan.Add(NewPlanEntry(step3, compiler.StatusNeedsApply, compiler.Diff{}))

	summary := plan.Summary()
	if summary.Total != 3 {
		t.Errorf("Summary().Total = %d, want 3", summary.Total)
	}
	if summary.NeedsApply != 2 {
		t.Errorf("Summary().NeedsApply = %d, want 2", summary.NeedsApply)
	}
	if summary.Satisfied != 1 {
		t.Errorf("Summary().Satisfied = %d, want 1", summary.Satisfied)
	}
}
