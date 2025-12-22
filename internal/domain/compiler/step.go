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
