package execution

import (
	"context"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// Executor runs steps from an ExecutionPlan.
type Executor struct {
	dryRun bool
}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// WithDryRun returns an Executor that simulates execution without applying.
func (e *Executor) WithDryRun(dryRun bool) *Executor {
	return &Executor{dryRun: dryRun}
}

// Execute runs all steps in the plan in order.
// Returns results for each step, including failures and skipped steps.
func (e *Executor) Execute(ctx context.Context, plan *Plan) ([]StepResult, error) {
	results := make([]StepResult, 0, plan.Len())
	failed := make(map[string]bool) // Track failed step IDs

	runCtx := compiler.NewRunContext(ctx).WithDryRun(e.dryRun)

	for _, entry := range plan.Entries() {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := e.executeEntry(entry, runCtx, failed)
		results = append(results, result)

		// Track failures for dependency checking
		if result.Status() == compiler.StatusFailed {
			failed[entry.Step().ID().String()] = true
		}
	}

	return results, nil
}

// executeEntry executes a single plan entry.
func (e *Executor) executeEntry(entry PlanEntry, ctx compiler.RunContext, failed map[string]bool) StepResult {
	step := entry.Step()
	stepID := step.ID()

	// Check if any dependency failed
	for _, depID := range step.DependsOn() {
		if failed[depID.String()] {
			return NewStepResult(stepID, compiler.StatusSkipped, nil)
		}
	}

	// If already satisfied, report success without applying
	if entry.Status() == compiler.StatusSatisfied {
		return NewStepResult(stepID, compiler.StatusSatisfied, nil)
	}

	// If dry run, report what would happen
	if ctx.DryRun() {
		return NewStepResult(stepID, entry.Status(), nil).WithDiff(entry.Diff())
	}

	// Apply the step
	start := time.Now()
	err := step.Apply(ctx)
	duration := time.Since(start)

	if err != nil {
		return NewStepResult(stepID, compiler.StatusFailed, err).WithDuration(duration)
	}

	return NewStepResult(stepID, compiler.StatusSatisfied, nil).
		WithDuration(duration).
		WithDiff(entry.Diff())
}
