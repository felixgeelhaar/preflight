package app

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/stretchr/testify/require"
)

type dummyStep struct {
	id compiler.StepID
}

func newDummyStep(id string) *dummyStep {
	return &dummyStep{id: compiler.MustNewStepID(id)}
}

func (d *dummyStep) ID() compiler.StepID { return d.id }
func (d *dummyStep) DependsOn() []compiler.StepID {
	return nil
}
func (d *dummyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusSatisfied, nil
}
func (d *dummyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.Diff{}, nil
}
func (d *dummyStep) Apply(_ compiler.RunContext) error { return nil }
func (d *dummyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.Explanation{}
}

func TestIsBootstrapStep(t *testing.T) {
	require.True(t, IsBootstrapStep("brew:install"))
	require.True(t, IsBootstrapStep("apt:update"))
	require.True(t, IsBootstrapStep("winget:ready"))
	require.True(t, IsBootstrapStep("bootstrap:tool:node"))
	require.False(t, IsBootstrapStep("npm:package:eslint"))
}

func TestBootstrapSteps(t *testing.T) {
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(newDummyStep("brew:install"), compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(execution.NewPlanEntry(newDummyStep("bootstrap:tool:node"), compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(execution.NewPlanEntry(newDummyStep("npm:package:eslint"), compiler.StatusNeedsApply, compiler.Diff{}))
	plan.Add(execution.NewPlanEntry(newDummyStep("apt:update"), compiler.StatusSatisfied, compiler.Diff{}))

	steps := BootstrapSteps(plan)
	require.Equal(t, []string{"bootstrap:tool:node", "brew:install"}, steps)
}
