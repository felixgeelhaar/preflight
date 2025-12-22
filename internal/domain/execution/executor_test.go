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

	_, err := executor.Execute(ctx, plan)
	if err == nil {
		t.Error("Execute() should return error when context is cancelled")
	}
}
