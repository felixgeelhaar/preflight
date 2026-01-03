package compiler

// Step represents an idempotent unit of execution in the compilation.
// Each step can check its current state, plan changes, and apply them.
type Step interface {
	// ID returns the unique identifier for this step.
	ID() StepID

	// DependsOn returns the IDs of steps that must complete before this one.
	DependsOn() []StepID

	// Check determines the current status of this step.
	// Returns StatusSatisfied if no action needed, StatusNeedsApply if changes required.
	Check(ctx RunContext) (StepStatus, error)

	// Plan returns the diff describing what changes this step will make.
	Plan(ctx RunContext) (Diff, error)

	// Apply executes the step's changes.
	// This should be idempotent - running multiple times produces the same result.
	Apply(ctx RunContext) error

	// Explain returns human-readable context for this step.
	Explain(ctx ExplainContext) Explanation
}

// RollbackableStep extends Step with rollback capability.
// Steps that implement this interface can undo their changes when
// execution fails, enabling transaction-like behavior.
type RollbackableStep interface {
	Step

	// CanRollback returns true if the step can be rolled back in its current state.
	// Some steps may only be rollbackable under certain conditions.
	CanRollback() bool

	// Rollback undoes the changes made by Apply.
	// This should be idempotent - rolling back a non-applied step is a no-op.
	Rollback(ctx RunContext) error
}

// IsRollbackable checks if a step implements the RollbackableStep interface.
func IsRollbackable(step Step) bool {
	_, ok := step.(RollbackableStep)
	return ok
}

// AsRollbackable attempts to cast a step to RollbackableStep.
// Returns nil if the step doesn't implement rollback.
func AsRollbackable(step Step) RollbackableStep {
	if r, ok := step.(RollbackableStep); ok {
		return r
	}
	return nil
}
