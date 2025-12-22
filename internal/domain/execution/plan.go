package execution

import (
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// PlanEntry represents a single step's planned execution.
type PlanEntry struct {
	step   compiler.Step
	status compiler.StepStatus
	diff   compiler.Diff
}

// NewPlanEntry creates a new PlanEntry.
func NewPlanEntry(step compiler.Step, status compiler.StepStatus, diff compiler.Diff) PlanEntry {
	return PlanEntry{
		step:   step,
		status: status,
		diff:   diff,
	}
}

// Step returns the step to be executed.
func (e PlanEntry) Step() compiler.Step {
	return e.step
}

// Status returns the current status of the step.
func (e PlanEntry) Status() compiler.StepStatus {
	return e.status
}

// Diff returns the planned changes.
func (e PlanEntry) Diff() compiler.Diff {
	return e.diff
}

// PlanSummary provides aggregate statistics about the execution plan.
type PlanSummary struct {
	Total      int
	NeedsApply int
	Satisfied  int
	Unknown    int
	Failed     int
	Skipped    int
}

// Plan represents the full plan for executing all steps.
type Plan struct {
	entries []PlanEntry
}

// NewExecutionPlan creates an empty Plan.
func NewExecutionPlan() *Plan {
	return &Plan{
		entries: make([]PlanEntry, 0),
	}
}

// Add appends a plan entry.
func (p *Plan) Add(entry PlanEntry) {
	p.entries = append(p.entries, entry)
}

// Len returns the number of entries.
func (p *Plan) Len() int {
	return len(p.entries)
}

// IsEmpty returns true if there are no entries.
func (p *Plan) IsEmpty() bool {
	return len(p.entries) == 0
}

// Entries returns all plan entries.
func (p *Plan) Entries() []PlanEntry {
	return p.entries
}

// NeedsApply returns entries that require execution.
func (p *Plan) NeedsApply() []PlanEntry {
	result := make([]PlanEntry, 0)
	for _, e := range p.entries {
		if e.status == compiler.StatusNeedsApply {
			result = append(result, e)
		}
	}
	return result
}

// HasChanges returns true if any steps need to be applied.
func (p *Plan) HasChanges() bool {
	for _, e := range p.entries {
		if e.status == compiler.StatusNeedsApply {
			return true
		}
	}
	return false
}

// Summary returns aggregate statistics.
func (p *Plan) Summary() PlanSummary {
	summary := PlanSummary{Total: len(p.entries)}
	for _, e := range p.entries {
		switch e.status {
		case compiler.StatusNeedsApply:
			summary.NeedsApply++
		case compiler.StatusSatisfied:
			summary.Satisfied++
		case compiler.StatusUnknown:
			summary.Unknown++
		case compiler.StatusFailed:
			summary.Failed++
		case compiler.StatusSkipped:
			summary.Skipped++
		}
	}
	return summary
}
