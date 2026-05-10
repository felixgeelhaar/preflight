// Package execution handles step orchestration and runtime execution.
package execution

import (
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// StepResult captures the outcome of executing a single step.
type StepResult struct {
	stepID   compiler.StepID
	status   compiler.StepStatus
	err      error
	duration time.Duration
	diff     compiler.Diff
	applied  bool
}

// NewStepResult creates a new StepResult.
func NewStepResult(stepID compiler.StepID, status compiler.StepStatus, err error) StepResult {
	return StepResult{
		stepID: stepID,
		status: status,
		err:    err,
	}
}

// StepID returns the ID of the step that was executed.
func (r StepResult) StepID() compiler.StepID {
	return r.stepID
}

// Status returns the final status of the step.
func (r StepResult) Status() compiler.StepStatus {
	return r.status
}

// Error returns any error that occurred during execution.
func (r StepResult) Error() error {
	return r.err
}

// Duration returns how long the step took to execute.
func (r StepResult) Duration() time.Duration {
	return r.duration
}

// Diff returns the diff that was applied (if any).
func (r StepResult) Diff() compiler.Diff {
	return r.diff
}

// Success returns true if the step completed successfully.
func (r StepResult) Success() bool {
	return r.status == compiler.StatusSatisfied
}

// Skipped returns true if the step was skipped.
func (r StepResult) Skipped() bool {
	return r.status == compiler.StatusSkipped
}

// WithDuration returns a new StepResult with duration set.
func (r StepResult) WithDuration(d time.Duration) StepResult {
	r.duration = d
	return r
}

// WithDiff returns a new StepResult with diff set.
func (r StepResult) WithDiff(d compiler.Diff) StepResult {
	r.diff = d
	return r
}

// Applied reports whether this step actually mutated the system in this run.
// True only when Apply was invoked and returned without error; false for
// already-satisfied, dry-run, skipped, or failed results. Used to gate rollback
// so pre-existing satisfied steps are not rolled back alongside newly applied ones.
func (r StepResult) Applied() bool {
	return r.applied
}

// WithApplied returns a new StepResult marking whether the step mutated the system.
func (r StepResult) WithApplied(applied bool) StepResult {
	r.applied = applied
	return r
}
