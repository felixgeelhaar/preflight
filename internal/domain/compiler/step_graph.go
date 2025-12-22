package compiler

import (
	"errors"
	"fmt"
)

// Errors for StepGraph operations.
var (
	ErrDuplicateStep    = errors.New("step with this ID already exists")
	ErrCyclicDependency = errors.New("cyclic dependency detected")
	ErrMissingDep       = errors.New("step depends on nonexistent step")
)

// StepGraph represents a directed acyclic graph of steps.
// It tracks dependencies and provides topological sorting for execution order.
type StepGraph struct {
	steps      map[string]Step
	dependsOn  map[string][]string // step ID -> list of dependency IDs
	dependedBy map[string][]string // step ID -> list of steps that depend on it
}

// NewStepGraph creates an empty StepGraph.
func NewStepGraph() *StepGraph {
	return &StepGraph{
		steps:      make(map[string]Step),
		dependsOn:  make(map[string][]string),
		dependedBy: make(map[string][]string),
	}
}

// Len returns the number of steps in the graph.
func (g *StepGraph) Len() int {
	return len(g.steps)
}

// Add adds a step to the graph.
// Returns ErrDuplicateStep if a step with the same ID already exists.
func (g *StepGraph) Add(step Step) error {
	id := step.ID().String()

	if _, exists := g.steps[id]; exists {
		return ErrDuplicateStep
	}

	g.steps[id] = step

	// Record dependencies
	deps := step.DependsOn()
	depIDs := make([]string, len(deps))
	for i, dep := range deps {
		depID := dep.String()
		depIDs[i] = depID
		// Track reverse dependency
		g.dependedBy[depID] = append(g.dependedBy[depID], id)
	}
	g.dependsOn[id] = depIDs

	return nil
}

// Get retrieves a step by ID.
func (g *StepGraph) Get(id StepID) (Step, bool) {
	step, ok := g.steps[id.String()]
	return step, ok
}

// Steps returns all steps in the graph (in no particular order).
func (g *StepGraph) Steps() []Step {
	steps := make([]Step, 0, len(g.steps))
	for _, step := range g.steps {
		steps = append(steps, step)
	}
	return steps
}

// Validate checks that all dependencies exist.
func (g *StepGraph) Validate() error {
	for id, deps := range g.dependsOn {
		for _, depID := range deps {
			if _, exists := g.steps[depID]; !exists {
				return fmt.Errorf("%w: step %q depends on %q", ErrMissingDep, id, depID)
			}
		}
	}
	return nil
}

// Roots returns steps that have no dependencies.
func (g *StepGraph) Roots() []Step {
	roots := make([]Step, 0)
	for id, step := range g.steps {
		if len(g.dependsOn[id]) == 0 {
			roots = append(roots, step)
		}
	}
	return roots
}

// Leaves returns steps that nothing depends on.
func (g *StepGraph) Leaves() []Step {
	leaves := make([]Step, 0)
	for id, step := range g.steps {
		if len(g.dependedBy[id]) == 0 {
			leaves = append(leaves, step)
		}
	}
	return leaves
}

// TopologicalSort returns steps in dependency order.
// Steps with no dependencies come first.
// Returns ErrCyclicDependency if the graph contains a cycle.
func (g *StepGraph) TopologicalSort() ([]Step, error) {
	// Kahn's algorithm for topological sorting

	// Calculate in-degree (number of dependencies) for each step
	inDegree := make(map[string]int)
	for id := range g.steps {
		inDegree[id] = 0
	}
	for id := range g.steps {
		for _, depID := range g.dependsOn[id] {
			// Only count dependencies that exist in the graph
			if _, exists := g.steps[depID]; exists {
				inDegree[id]++
			}
		}
	}

	// Start with steps that have no dependencies
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	sorted := make([]Step, 0, len(g.steps))

	for len(queue) > 0 {
		// Pop from queue
		id := queue[0]
		queue = queue[1:]

		sorted = append(sorted, g.steps[id])

		// Reduce in-degree for all dependents
		for _, dependentID := range g.dependedBy[id] {
			if _, exists := g.steps[dependentID]; !exists {
				continue
			}
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				queue = append(queue, dependentID)
			}
		}
	}

	// If we didn't process all steps, there's a cycle
	if len(sorted) != len(g.steps) {
		return nil, ErrCyclicDependency
	}

	return sorted, nil
}
