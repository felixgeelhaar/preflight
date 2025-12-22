package compiler

// StepStatus represents the current state of a step.
type StepStatus string

const (
	// StatusSatisfied indicates the step's desired state is already met.
	StatusSatisfied StepStatus = "satisfied"
	// StatusNeedsApply indicates the step needs to be applied.
	StatusNeedsApply StepStatus = "needs-apply"
	// StatusUnknown indicates the step's state could not be determined.
	StatusUnknown StepStatus = "unknown"
	// StatusFailed indicates the step failed during check or apply.
	StatusFailed StepStatus = "failed"
	// StatusSkipped indicates the step was skipped (e.g., dependency failed).
	StatusSkipped StepStatus = "skipped"
)

// String returns the string representation of the status.
func (s StepStatus) String() string {
	return string(s)
}

// NeedsAction returns true if this status requires user attention or execution.
func (s StepStatus) NeedsAction() bool {
	switch s {
	case StatusNeedsApply, StatusUnknown, StatusFailed:
		return true
	case StatusSatisfied, StatusSkipped:
		return false
	}
	return false
}

// IsTerminal returns true if this status represents a final state.
func (s StepStatus) IsTerminal() bool {
	switch s {
	case StatusSatisfied, StatusFailed, StatusSkipped:
		return true
	case StatusNeedsApply, StatusUnknown:
		return false
	}
	return false
}
