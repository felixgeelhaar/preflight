package main

import (
	"context"
	"io"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: TestApplyCmd_Exists and TestApplyCmd_HasFlags are in helpers_test.go

func TestApplyCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
		{"dry-run default", "dry-run", "false"},
		{"update-lock default", "update-lock", "false"},
		{"rollback-on-error default", "rollback-on-error", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := applyCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestApplyCmd_ConfigShorthand(t *testing.T) {
	t.Parallel()

	f := applyCmd.Flags().Lookup("config")
	assert.NotNil(t, f)
	assert.Equal(t, "c", f.Shorthand)
}

func TestApplyCmd_TargetShorthand(t *testing.T) {
	t.Parallel()

	f := applyCmd.Flags().Lookup("target")
	assert.NotNil(t, f)
	assert.Equal(t, "t", f.Shorthand)
}

func TestApplyCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "apply" {
			found = true
			break
		}
	}
	assert.True(t, found, "apply should be a subcommand of root")
}

func TestRunApply_NoChangesSkipsApply(t *testing.T) {

	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(newDummyStep("files:link:bashrc"), compiler.StatusSatisfied, compiler.Diff{}))

	fake := newFakePreflightClient(plan, nil, nil)
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, false, false, false)
	defer reset()

	err := runApply(&cobra.Command{}, nil)
	require.NoError(t, err)
	assert.True(t, fake.printPlanCalled)
	assert.False(t, fake.applyCalled)
	assert.False(t, fake.updateLockCalled)
}

func TestRunApply_AppliesAndUpdatesLock(t *testing.T) {

	plan := execution.NewExecutionPlan()
	step := newDummyStep("files:link:bashrc")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "files", "link", "", "")))

	results := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusSatisfied, nil),
	}

	fake := newFakePreflightClient(plan, results, nil)
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, false, true, false)
	defer reset()

	err := runApply(&cobra.Command{}, nil)
	require.NoError(t, err)
	assert.True(t, fake.applyCalled)
	assert.True(t, fake.printResultsCalled)
	assert.True(t, fake.updateLockCalled)
}

func overrideNewPreflight(client *fakePreflightClient) func() {
	prev := newPreflight
	newPreflight = func(_ io.Writer) preflightClient { return client }
	return func() { newPreflight = prev }
}

func setApplyFlags(t *testing.T, dryRun, updateLock, rollback bool) func() {
	t.Helper()
	prevDryRun := applyDryRun
	prevUpdateLock := applyUpdateLock
	prevRollback := applyRollback
	applyDryRun = dryRun
	applyUpdateLock = updateLock
	applyRollback = rollback
	return func() {
		applyDryRun = prevDryRun
		applyUpdateLock = prevUpdateLock
		applyRollback = prevRollback
	}
}

type fakePreflightClient struct {
	planResult         *execution.Plan
	planErr            error
	results            []execution.StepResult
	applyErr           error
	printPlanCalled    bool
	printResultsCalled bool
	applyCalled        bool
	updateLockCalled   bool
}

func newFakePreflightClient(plan *execution.Plan, results []execution.StepResult, applyErr error) *fakePreflightClient {
	return &fakePreflightClient{
		planResult: plan,
		results:    results,
		applyErr:   applyErr,
	}
}

func (f *fakePreflightClient) Plan(_ context.Context, _ string, _ string) (*execution.Plan, error) {
	return f.planResult, f.planErr
}

func (f *fakePreflightClient) PrintPlan(plan *execution.Plan) {
	if plan == nil {
		return
	}
	f.printPlanCalled = true
}

func (f *fakePreflightClient) Apply(_ context.Context, _ *execution.Plan, _ bool) ([]execution.StepResult, error) {
	f.applyCalled = true
	return f.results, f.applyErr
}

func (f *fakePreflightClient) PrintResults(results []execution.StepResult) {
	if len(results) > 0 {
		f.printResultsCalled = true
	}
}

func (f *fakePreflightClient) UpdateLockFromPlan(_ context.Context, _ string, _ *execution.Plan) error {
	f.updateLockCalled = true
	return nil
}

func (f *fakePreflightClient) WithMode(config.ReproducibilityMode) preflightClient {
	return f
}

func (f *fakePreflightClient) WithRollbackOnFailure(bool) preflightClient {
	return f
}

type dummyStep struct {
	id compiler.StepID
}

func newDummyStep(id string) *dummyStep {
	stepID, _ := compiler.NewStepID(id)
	return &dummyStep{id: stepID}
}

func (d *dummyStep) ID() compiler.StepID {
	return d.id
}

func (d *dummyStep) DependsOn() []compiler.StepID {
	return nil
}

func (d *dummyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusSatisfied, nil
}

func (d *dummyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.Diff{}, nil
}

func (d *dummyStep) Apply(_ compiler.RunContext) error {
	return nil
}

func (d *dummyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.Explanation{}
}
