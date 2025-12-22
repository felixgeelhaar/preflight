package execution

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
)

// configurableMockStep allows configuring Check behavior
type configurableMockStep struct {
	id        compiler.StepID
	deps      []compiler.StepID
	checkFn   func(compiler.RunContext) (compiler.StepStatus, error)
	planFn    func(compiler.RunContext) (compiler.Diff, error)
	applyFn   func(compiler.RunContext) error
	explainFn func(compiler.ExplainContext) compiler.Explanation
}

func newConfigurableStep(id string, deps ...string) *configurableMockStep {
	stepID, _ := compiler.NewStepID(id)
	depIDs := make([]compiler.StepID, len(deps))
	for i, d := range deps {
		depIDs[i], _ = compiler.NewStepID(d)
	}
	return &configurableMockStep{
		id:   stepID,
		deps: depIDs,
		checkFn: func(_ compiler.RunContext) (compiler.StepStatus, error) {
			return compiler.StatusNeedsApply, nil
		},
		planFn: func(_ compiler.RunContext) (compiler.Diff, error) {
			return compiler.NewDiff(compiler.DiffTypeAdd, "test", id, "", "new"), nil
		},
		applyFn: func(_ compiler.RunContext) error {
			return nil
		},
		explainFn: func(_ compiler.ExplainContext) compiler.Explanation {
			return compiler.NewExplanation("Test", "Test step", nil)
		},
	}
}

func (m *configurableMockStep) ID() compiler.StepID          { return m.id }
func (m *configurableMockStep) DependsOn() []compiler.StepID { return m.deps }
func (m *configurableMockStep) Check(ctx compiler.RunContext) (compiler.StepStatus, error) {
	return m.checkFn(ctx)
}
func (m *configurableMockStep) Plan(ctx compiler.RunContext) (compiler.Diff, error) {
	return m.planFn(ctx)
}
func (m *configurableMockStep) Apply(ctx compiler.RunContext) error { return m.applyFn(ctx) }
func (m *configurableMockStep) Explain(ctx compiler.ExplainContext) compiler.Explanation {
	return m.explainFn(ctx)
}

func TestPlanner_EmptyGraph(t *testing.T) {
	graph := compiler.NewStepGraph()
	planner := NewPlanner()

	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !plan.IsEmpty() {
		t.Error("Plan should be empty for empty graph")
	}
}

func TestPlanner_SingleStep_NeedsApply(t *testing.T) {
	graph := compiler.NewStepGraph()
	step := newConfigurableStep("brew:install:git")
	step.checkFn = func(_ compiler.RunContext) (compiler.StepStatus, error) {
		return compiler.StatusNeedsApply, nil
	}
	_ = graph.Add(step)

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.Len() != 1 {
		t.Errorf("Plan.Len() = %d, want 1", plan.Len())
	}

	entries := plan.Entries()
	if entries[0].Status() != compiler.StatusNeedsApply {
		t.Errorf("Entry status = %v, want %v", entries[0].Status(), compiler.StatusNeedsApply)
	}
}

func TestPlanner_SingleStep_Satisfied(t *testing.T) {
	graph := compiler.NewStepGraph()
	step := newConfigurableStep("brew:install:git")
	step.checkFn = func(_ compiler.RunContext) (compiler.StepStatus, error) {
		return compiler.StatusSatisfied, nil
	}
	_ = graph.Add(step)

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	entries := plan.Entries()
	if entries[0].Status() != compiler.StatusSatisfied {
		t.Errorf("Entry status = %v, want %v", entries[0].Status(), compiler.StatusSatisfied)
	}
	if !entries[0].Diff().IsEmpty() {
		t.Error("Satisfied step should have empty diff")
	}
}

func TestPlanner_MultipleSteps(t *testing.T) {
	graph := compiler.NewStepGraph()
	step1 := newConfigurableStep("brew:install:git")
	step2 := newConfigurableStep("brew:install:curl")
	_ = graph.Add(step1)
	_ = graph.Add(step2)

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.Len() != 2 {
		t.Errorf("Plan.Len() = %d, want 2", plan.Len())
	}
}

func TestPlanner_StepsInOrder(t *testing.T) {
	graph := compiler.NewStepGraph()
	step1 := newConfigurableStep("step:first")
	step2 := newConfigurableStep("step:second", "step:first")
	step3 := newConfigurableStep("step:third", "step:second")
	_ = graph.Add(step1)
	_ = graph.Add(step2)
	_ = graph.Add(step3)

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// Verify order
	entries := plan.Entries()
	indices := make(map[string]int)
	for i, e := range entries {
		indices[e.Step().ID().String()] = i
	}

	if indices["step:first"] >= indices["step:second"] {
		t.Error("first should come before second")
	}
	if indices["step:second"] >= indices["step:third"] {
		t.Error("second should come before third")
	}
}

func TestPlanner_CheckError(t *testing.T) {
	graph := compiler.NewStepGraph()
	step := newConfigurableStep("failing:step")
	step.checkFn = func(_ compiler.RunContext) (compiler.StepStatus, error) {
		return compiler.StatusUnknown, errors.New("check failed")
	}
	_ = graph.Add(step)

	planner := NewPlanner()
	_, err := planner.Plan(context.Background(), graph)
	if err == nil {
		t.Error("Plan() should return error when check fails")
	}
}

func TestPlanner_PlanCallsStepPlan(t *testing.T) {
	graph := compiler.NewStepGraph()
	step := newConfigurableStep("brew:install:git")
	step.checkFn = func(_ compiler.RunContext) (compiler.StepStatus, error) {
		return compiler.StatusNeedsApply, nil
	}
	step.planFn = func(_ compiler.RunContext) (compiler.Diff, error) {
		return compiler.NewDiff(compiler.DiffTypeAdd, "package", "git", "", "2.43.0"), nil
	}
	_ = graph.Add(step)

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), graph)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	entries := plan.Entries()
	if entries[0].Diff().Type() != compiler.DiffTypeAdd {
		t.Errorf("Diff type = %v, want %v", entries[0].Diff().Type(), compiler.DiffTypeAdd)
	}
	if entries[0].Diff().Name() != "git" {
		t.Errorf("Diff name = %q, want %q", entries[0].Diff().Name(), "git")
	}
}

func TestPlanner_PlanError(t *testing.T) {
	graph := compiler.NewStepGraph()
	step := newConfigurableStep("failing:step")
	step.checkFn = func(_ compiler.RunContext) (compiler.StepStatus, error) {
		return compiler.StatusNeedsApply, nil
	}
	step.planFn = func(_ compiler.RunContext) (compiler.Diff, error) {
		return compiler.Diff{}, errors.New("plan failed")
	}
	_ = graph.Add(step)

	planner := NewPlanner()
	_, err := planner.Plan(context.Background(), graph)
	if err == nil {
		t.Error("Plan() should return error when step plan fails")
	}
}
