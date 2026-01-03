package execution

import (
	"context"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// Executor runs steps from an ExecutionPlan.
type Executor struct {
	dryRun            bool
	rollbackOnFailure bool
}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// WithDryRun returns an Executor that simulates execution without applying.
func (e *Executor) WithDryRun(dryRun bool) *Executor {
	return &Executor{
		dryRun:            dryRun,
		rollbackOnFailure: e.rollbackOnFailure,
	}
}

// WithRollbackOnFailure returns an Executor that rolls back applied steps on failure.
// Only steps that implement RollbackableStep will be rolled back.
func (e *Executor) WithRollbackOnFailure(rollback bool) *Executor {
	return &Executor{
		dryRun:            e.dryRun,
		rollbackOnFailure: rollback,
	}
}

// ExecuteResult contains the results of an execution, including any rollback information.
type ExecuteResult struct {
	Results         []StepResult
	RollbackResults []RollbackResult
	RolledBack      bool
}

// RollbackResult contains the result of rolling back a step.
type RollbackResult struct {
	StepID   compiler.StepID
	Success  bool
	Error    error
	Duration time.Duration
}

// Execute runs all steps in the plan in order.
// Returns results for each step, including failures and skipped steps.
func (e *Executor) Execute(ctx context.Context, plan *Plan) ([]StepResult, error) {
	result := e.ExecuteWithRollback(ctx, plan)
	return result.Results, nil
}

// ExecuteWithRollback runs all steps and returns detailed execution results.
// If rollbackOnFailure is enabled and a step fails, previously applied steps
// will be rolled back in reverse order.
func (e *Executor) ExecuteWithRollback(ctx context.Context, plan *Plan) ExecuteResult {
	results := make([]StepResult, 0, plan.Len())
	appliedSteps := make([]compiler.Step, 0) // Track successfully applied steps for rollback
	failed := make(map[string]bool)          // Track failed step IDs

	runCtx := compiler.NewRunContext(ctx).WithDryRun(e.dryRun)

	var failedResult *StepResult

	for _, entry := range plan.Entries() {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ExecuteResult{Results: results}
		default:
		}

		result := e.executeEntry(entry, runCtx, failed)
		results = append(results, result)

		// Track failures for dependency checking
		if result.Status() == compiler.StatusFailed {
			failed[entry.Step().ID().String()] = true
			failedResult = &result

			// If rollback is enabled, stop executing and rollback
			if e.rollbackOnFailure {
				break
			}
		} else if result.Status() == compiler.StatusSatisfied && !e.dryRun {
			// Track successfully applied steps
			appliedSteps = append(appliedSteps, entry.Step())
		}
	}

	// Perform rollback if enabled and we had a failure
	if e.rollbackOnFailure && failedResult != nil && len(appliedSteps) > 0 {
		rollbackResults := e.rollbackSteps(runCtx, appliedSteps)
		return ExecuteResult{
			Results:         results,
			RollbackResults: rollbackResults,
			RolledBack:      true,
		}
	}

	return ExecuteResult{Results: results}
}

// rollbackSteps rolls back applied steps in reverse order.
func (e *Executor) rollbackSteps(ctx compiler.RunContext, steps []compiler.Step) []RollbackResult {
	results := make([]RollbackResult, 0, len(steps))

	// Rollback in reverse order
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		result := RollbackResult{StepID: step.ID()}

		// Check if step supports rollback
		rollbackable := compiler.AsRollbackable(step)
		if rollbackable == nil {
			// Step doesn't support rollback, skip
			result.Success = false
			result.Error = nil // Not an error, just not supported
			results = append(results, result)
			continue
		}

		// Check if step can be rolled back
		if !rollbackable.CanRollback() {
			result.Success = false
			results = append(results, result)
			continue
		}

		// Perform rollback
		start := time.Now()
		err := rollbackable.Rollback(ctx)
		result.Duration = time.Since(start)

		if err != nil {
			result.Success = false
			result.Error = err
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	return results
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
