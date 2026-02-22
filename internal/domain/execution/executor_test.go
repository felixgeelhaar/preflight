package execution

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

func TestExecutor_EmptyPlan(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	results, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results len = %d, want 0", len(results))
	}
}

func TestExecutor_SingleStep_Apply(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	applied := false
	step := newConfigurableStep("brew:install:git")
	step.applyFn = func(_ compiler.RunContext) error {
		applied = true
		return nil
	}

	plan.Add(NewPlanEntry(step, compiler.StatusNeedsApply, compiler.Diff{}))

	results, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !applied {
		t.Error("Step was not applied")
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}

	if !results[0].Success() {
		t.Error("Result should be success")
	}
}

func TestExecutor_SingleStep_Satisfied(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	applied := false
	step := newConfigurableStep("brew:install:git")
	step.applyFn = func(_ compiler.RunContext) error {
		applied = true
		return nil
	}

	// Already satisfied, should not apply
	plan.Add(NewPlanEntry(step, compiler.StatusSatisfied, compiler.Diff{}))

	results, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if applied {
		t.Error("Satisfied step should not be applied")
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}

	if !results[0].Success() {
		t.Error("Satisfied step should report success")
	}
}

func TestExecutor_ApplyError(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	step := newConfigurableStep("failing:step")
	step.applyFn = func(_ compiler.RunContext) error {
		return errors.New("apply failed")
	}

	plan.Add(NewPlanEntry(step, compiler.StatusNeedsApply, compiler.Diff{}))

	results, err := executor.Execute(context.Background(), plan)
	// Execute should not fail, but the step result should indicate failure
	if err != nil {
		t.Fatalf("Execute() should not return error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}

	if results[0].Success() {
		t.Error("Failed step should not report success")
	}
	if results[0].Status() != compiler.StatusFailed {
		t.Errorf("Status = %v, want %v", results[0].Status(), compiler.StatusFailed)
	}
	if results[0].Error() == nil {
		t.Error("Failed step should have error")
	}
}

func TestExecutor_ExecutesInOrder(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	var mu sync.Mutex
	order := make([]string, 0)

	step1 := newConfigurableStep("step:first")
	step1.applyFn = func(_ compiler.RunContext) error {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return nil
	}

	step2 := newConfigurableStep("step:second")
	step2.applyFn = func(_ compiler.RunContext) error {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return nil
	}

	step3 := newConfigurableStep("step:third")
	step3.applyFn = func(_ compiler.RunContext) error {
		mu.Lock()
		order = append(order, "third")
		mu.Unlock()
		return nil
	}

	// Add in plan order (which should be topological order from Planner)
	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step3, compiler.StatusNeedsApply, compiler.Diff{}))

	_, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify sequential execution order
	if len(order) != 3 {
		t.Fatalf("order len = %d, want 3", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "third" {
		t.Errorf("order = %v, want [first, second, third]", order)
	}
}

func TestExecutor_SkipsAfterFailure(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	step1 := newConfigurableStep("step:first")
	step1.applyFn = func(_ compiler.RunContext) error {
		return errors.New("first failed")
	}

	step2 := newConfigurableStep("step:second", "step:first")
	applied := false
	step2.applyFn = func(_ compiler.RunContext) error {
		applied = true
		return nil
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	results, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if applied {
		t.Error("Dependent step should be skipped when dependency fails")
	}

	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}

	// First should be failed
	if results[0].Status() != compiler.StatusFailed {
		t.Errorf("First status = %v, want %v", results[0].Status(), compiler.StatusFailed)
	}

	// Second should be skipped
	if results[1].Status() != compiler.StatusSkipped {
		t.Errorf("Second status = %v, want %v", results[1].Status(), compiler.StatusSkipped)
	}
}

func TestExecutor_DryRun(t *testing.T) {
	executor := NewExecutor().WithDryRun(true)
	plan := NewExecutionPlan()

	applied := false
	step := newConfigurableStep("brew:install:git")
	step.applyFn = func(_ compiler.RunContext) error {
		applied = true
		return nil
	}

	plan.Add(NewPlanEntry(step, compiler.StatusNeedsApply, compiler.Diff{}))

	results, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if applied {
		t.Error("Dry run should not apply steps")
	}

	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}

	// In dry run, we report what would happen
	if results[0].Status() != compiler.StatusNeedsApply {
		t.Errorf("Dry run status = %v, want %v", results[0].Status(), compiler.StatusNeedsApply)
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	executor := NewExecutor()
	plan := NewExecutionPlan()

	step := newConfigurableStep("slow:step")
	// This step would block forever, but context is already cancelled

	plan.Add(NewPlanEntry(step, compiler.StatusNeedsApply, compiler.Diff{}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := executor.ExecuteWithRollback(ctx, plan)
	if len(result.Results) != 0 {
		t.Error("Cancelled context should return empty results")
	}
}

// rollbackableMockStep extends configurableMockStep with rollback capability.
type rollbackableMockStep struct {
	*configurableMockStep
	canRollback bool
	rollbackFn  func(compiler.RunContext) error
}

func newRollbackableStep(id string, deps ...string) *rollbackableMockStep {
	return &rollbackableMockStep{
		configurableMockStep: newConfigurableStep(id, deps...),
		canRollback:          true,
		rollbackFn: func(_ compiler.RunContext) error {
			return nil
		},
	}
}

func (m *rollbackableMockStep) CanRollback() bool {
	return m.canRollback
}

func (m *rollbackableMockStep) Rollback(ctx compiler.RunContext) error {
	return m.rollbackFn(ctx)
}

func TestExecutor_WithRollbackOnFailure(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	if executor == nil {
		t.Fatal("WithRollbackOnFailure should return executor")
	}
}

func TestExecutor_RollbackOnFailure_RollsBackAppliedSteps(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	var mu sync.Mutex
	appliedSteps := make([]string, 0)
	rolledBackSteps := make([]string, 0)

	// Step 1: succeeds and is rollbackable
	step1 := newRollbackableStep("step:first")
	step1.applyFn = func(_ compiler.RunContext) error {
		mu.Lock()
		appliedSteps = append(appliedSteps, "first")
		mu.Unlock()
		return nil
	}
	step1.rollbackFn = func(_ compiler.RunContext) error {
		mu.Lock()
		rolledBackSteps = append(rolledBackSteps, "first")
		mu.Unlock()
		return nil
	}

	// Step 2: succeeds and is rollbackable
	step2 := newRollbackableStep("step:second", "step:first")
	step2.applyFn = func(_ compiler.RunContext) error {
		mu.Lock()
		appliedSteps = append(appliedSteps, "second")
		mu.Unlock()
		return nil
	}
	step2.rollbackFn = func(_ compiler.RunContext) error {
		mu.Lock()
		rolledBackSteps = append(rolledBackSteps, "second")
		mu.Unlock()
		return nil
	}

	// Step 3: fails
	step3 := newRollbackableStep("step:third", "step:second")
	step3.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step3, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	// Check that steps were applied in order
	if len(appliedSteps) != 2 {
		t.Errorf("applied steps = %v, want [first, second]", appliedSteps)
	}

	// Check that rollback occurred
	if !result.RolledBack {
		t.Error("RolledBack should be true")
	}

	// Check that steps were rolled back in reverse order
	if len(rolledBackSteps) != 2 {
		t.Fatalf("rolled back steps = %d, want 2", len(rolledBackSteps))
	}
	if rolledBackSteps[0] != "second" || rolledBackSteps[1] != "first" {
		t.Errorf("rollback order = %v, want [second, first]", rolledBackSteps)
	}

	// Check rollback results
	if len(result.RollbackResults) != 2 {
		t.Fatalf("rollback results = %d, want 2", len(result.RollbackResults))
	}
	for _, rr := range result.RollbackResults {
		if !rr.Success {
			t.Errorf("rollback of %s should succeed", rr.StepID)
		}
	}
}

func TestExecutor_RollbackOnFailure_SkipsNonRollbackableSteps(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	// Step 1: succeeds but is NOT rollbackable (regular step)
	step1 := newConfigurableStep("step:first")
	step1.applyFn = func(_ compiler.RunContext) error {
		return nil
	}

	// Step 2: succeeds and is rollbackable
	step2 := newRollbackableStep("step:second", "step:first")
	rolledBack := false
	step2.rollbackFn = func(_ compiler.RunContext) error {
		rolledBack = true
		return nil
	}

	// Step 3: fails
	step3 := newConfigurableStep("step:third", "step:second")
	step3.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step3, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	if !result.RolledBack {
		t.Error("RolledBack should be true")
	}

	// Step 2 should be rolled back
	if !rolledBack {
		t.Error("Rollbackable step should be rolled back")
	}

	// Rollback results should include both steps (one skipped, one successful)
	if len(result.RollbackResults) != 2 {
		t.Fatalf("rollback results = %d, want 2", len(result.RollbackResults))
	}

	// First in rollback order is step2 (rolled back successfully)
	// Second in rollback order is step1 (not rollbackable, should be marked as not successful but no error)
	foundNonRollbackable := false
	for _, rr := range result.RollbackResults {
		if rr.StepID.String() == "step:first" {
			if rr.Success {
				t.Error("Non-rollbackable step should not report success")
			}
			if rr.Error != nil {
				t.Error("Non-rollbackable step should have nil error (not supported, not failed)")
			}
			foundNonRollbackable = true
		}
	}
	if !foundNonRollbackable {
		t.Error("Non-rollbackable step should be in rollback results")
	}
}

func TestExecutor_RollbackOnFailure_CanRollbackFalse(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	// Step 1: rollbackable but CanRollback returns false
	step1 := newRollbackableStep("step:first")
	step1.canRollback = false
	rolledBack := false
	step1.rollbackFn = func(_ compiler.RunContext) error {
		rolledBack = true
		return nil
	}

	// Step 2: fails
	step2 := newConfigurableStep("step:second", "step:first")
	step2.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	if !result.RolledBack {
		t.Error("RolledBack should be true")
	}

	// Step 1 should NOT be rolled back because CanRollback is false
	if rolledBack {
		t.Error("Step with CanRollback=false should not be rolled back")
	}
}

func TestExecutor_RollbackOnFailure_RollbackError(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	// Step 1: rollback fails
	step1 := newRollbackableStep("step:first")
	step1.rollbackFn = func(_ compiler.RunContext) error {
		return errors.New("rollback failed")
	}

	// Step 2: fails
	step2 := newConfigurableStep("step:second", "step:first")
	step2.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	if !result.RolledBack {
		t.Error("RolledBack should be true")
	}

	// Check that rollback error is captured
	found := false
	for _, rr := range result.RollbackResults {
		if rr.StepID.String() == "step:first" {
			if rr.Success {
				t.Error("Failed rollback should not report success")
			}
			if rr.Error == nil {
				t.Error("Failed rollback should have error")
			}
			found = true
		}
	}
	if !found {
		t.Error("Step:first should be in rollback results")
	}
}

func TestExecutor_NoRollbackWhenDisabled(t *testing.T) {
	// Default executor has rollback disabled
	executor := NewExecutor()
	plan := NewExecutionPlan()

	step1 := newRollbackableStep("step:first")
	rolledBack := false
	step1.rollbackFn = func(_ compiler.RunContext) error {
		rolledBack = true
		return nil
	}

	step2 := newConfigurableStep("step:second", "step:first")
	step2.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	// Should not rollback when disabled
	if result.RolledBack {
		t.Error("RolledBack should be false when rollback is disabled")
	}
	if rolledBack {
		t.Error("Should not rollback when rollback is disabled")
	}
	if len(result.RollbackResults) != 0 {
		t.Error("Should have no rollback results when disabled")
	}
}

func TestExecutor_NoRollbackWhenNoFailure(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	step1 := newRollbackableStep("step:first")
	rolledBack := false
	step1.rollbackFn = func(_ compiler.RunContext) error {
		rolledBack = true
		return nil
	}

	step2 := newRollbackableStep("step:second", "step:first")

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	// Should not rollback when all steps succeed
	if result.RolledBack {
		t.Error("RolledBack should be false when all steps succeed")
	}
	if rolledBack {
		t.Error("Should not rollback when all steps succeed")
	}
}

func TestExecutor_RollbackDuration(t *testing.T) {
	executor := NewExecutor().WithRollbackOnFailure(true)
	plan := NewExecutionPlan()

	step1 := newRollbackableStep("step:first")

	step2 := newConfigurableStep("step:second", "step:first")
	step2.applyFn = func(_ compiler.RunContext) error {
		return errors.New("step failed")
	}

	plan.Add(NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.Diff{}))

	result := executor.ExecuteWithRollback(context.Background(), plan)

	// Check that duration is captured (may be zero on fast machines)
	for _, rr := range result.RollbackResults {
		if rr.Success && rr.Duration < 0 {
			t.Error("Successful rollback should have non-negative duration")
		}
	}
}
