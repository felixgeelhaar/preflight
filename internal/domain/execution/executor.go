package execution

import (
	"context"
	"errors"
	"fmt"
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
// If any step failed, the returned error is the joined set of step errors so
// callers cannot mistake a partial run for a clean one.
func (e *Executor) Execute(ctx context.Context, plan *Plan) ([]StepResult, error) {
	result := e.ExecuteWithRollback(ctx, plan)
	var errs []error
	for _, r := range result.Results {
		if r.Status() == compiler.StatusFailed {
			err := r.Error()
			if err == nil {
				err = fmt.Errorf("step %s failed", r.StepID())
			} else {
				err = fmt.Errorf("step %s: %w", r.StepID(), err)
			}
			errs = append(errs, err)
		}
	}
	return result.Results, errors.Join(errs...)
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
		} else if result.Applied() {
			// Track only steps that actually mutated the system in this run.
			// Already-satisfied (no-op) steps must not be rolled back.
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

	// Apply the step. We wrap Apply in a goroutine and select on ctx.Done()
	// so cancellation is honored even by steps that don't internally observe
	// the context. The step's goroutine continues until Apply returns; the
	// executor returns control to the caller immediately on cancel.
	start := time.Now()
	err := applyWithCancellation(ctx, step)
	duration := time.Since(start)

	if err != nil {
		return NewStepResult(stepID, compiler.StatusFailed, err).WithDuration(duration)
	}

	return NewStepResult(stepID, compiler.StatusSatisfied, nil).
		WithDuration(duration).
		WithDiff(entry.Diff()).
		WithApplied(true)
}

// applyWithCancellation runs step.Apply and races it against the run context's
// Done channel. If the context is cancelled before Apply returns, the
// cancellation error is returned even if the step itself doesn't honor ctx.
//
// The step's goroutine is leaked until Apply finishes — that is intentional:
// the alternative (force-killing) would leave the system in an inconsistent
// state. Steps remain responsible for honoring ctx for prompt termination;
// this wrapper ensures the executor's caller is unblocked regardless.
func applyWithCancellation(ctx compiler.RunContext, step compiler.Step) error {
	done := make(chan error, 1)
	go func() {
		done <- step.Apply(ctx)
	}()

	select {
	case <-ctx.Context().Done():
		return ctx.Context().Err()
	case err := <-done:
		return err
	}
}
