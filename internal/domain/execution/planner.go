package execution

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// Planner generates an ExecutionPlan from a StepGraph.
// It checks each step's current status and plans necessary changes.
type Planner struct{}

// NewPlanner creates a new Planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Plan generates a Plan by checking each step's status.
// Steps are returned in topological order for correct execution.
func (p *Planner) Plan(ctx context.Context, graph *compiler.StepGraph) (*Plan, error) {
	plan := NewExecutionPlan()

	// Get steps in topological order
	steps, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort steps: %w", err)
	}

	runCtx := compiler.NewRunContext(ctx)

	for _, step := range steps {
		entry, err := p.planStep(step, runCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to plan step %q: %w", step.ID().String(), err)
		}
		plan.Add(entry)
	}

	return plan, nil
}

// planStep checks a single step and generates a PlanEntry.
func (p *Planner) planStep(step compiler.Step, ctx compiler.RunContext) (PlanEntry, error) {
	// Check current status
	status, err := step.Check(ctx)
	if err != nil {
		return PlanEntry{}, fmt.Errorf("check failed: %w", err)
	}

	var diff compiler.Diff

	// Only get diff if step needs to be applied
	if status == compiler.StatusNeedsApply {
		diff, err = step.Plan(ctx)
		if err != nil {
			return PlanEntry{}, fmt.Errorf("plan failed: %w", err)
		}
	}

	return NewPlanEntry(step, status, diff), nil
}
