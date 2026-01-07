package main

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: TestPlanCmd_Exists and TestPlanCmd_HasFlags are in helpers_test.go

func TestPlanCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := planCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestPlanCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "plan" {
			found = true
			break
		}
	}
	assert.True(t, found, "plan should be a subcommand of root")
}

func TestRunPlan_Success(t *testing.T) {
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(newDummyStep("example:step"), compiler.StatusSatisfied, compiler.Diff{}))

	fake := newFakePlanPreflightClient(plan, nil)
	restore := overrideNewPlanPreflight(fake)
	defer restore()

	reset := setPlanFlags(t, "custom.yaml", "canary")
	defer reset()

	cmd := &cobra.Command{}

	err := runPlan(cmd, nil)
	require.NoError(t, err)
	assert.True(t, fake.printPlanCalled)
	assert.Equal(t, "custom.yaml", fake.configPath)
	assert.Equal(t, "canary", fake.target)
}

func TestRunPlan_PlanError(t *testing.T) {
	fake := newFakePlanPreflightClient(nil, errTestPlan)
	restore := overrideNewPlanPreflight(fake)
	defer restore()

	reset := setPlanFlags(t, "preflight.yaml", "default")
	defer reset()

	cmd := &cobra.Command{}

	err := runPlan(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan failed")
}

func TestRunPlan_ModeOverride(t *testing.T) {
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(newDummyStep("mode:step"), compiler.StatusSatisfied, compiler.Diff{}))

	fake := newFakePlanPreflightClient(plan, nil)
	restore := overrideNewPlanPreflight(fake)
	defer restore()

	reset := setPlanFlags(t, "intent.yaml", "default")
	defer reset()

	cmd := &cobra.Command{}
	cmd.Flags().String("mode", "intent", "")
	require.NoError(t, cmd.Flags().Set("mode", "locked"))

	err := runPlan(cmd, nil)
	require.NoError(t, err)
	require.NotNil(t, fake.modeOverride)
	assert.Equal(t, config.ModeLocked, *fake.modeOverride)
}

func overrideNewPlanPreflight(client *fakePlanPreflightClient) func() {
	prev := newPlanPreflight
	newPlanPreflight = func(_ io.Writer) preflightClient { return client }
	return func() { newPlanPreflight = prev }
}

func setPlanFlags(t *testing.T, config, target string) func() {
	t.Helper()
	prevConfig := planConfigPath
	prevTarget := planTarget
	planConfigPath = config
	planTarget = target
	return func() {
		planConfigPath = prevConfig
		planTarget = prevTarget
	}
}

type fakePlanPreflightClient struct {
	planResult      *execution.Plan
	planErr         error
	configPath      string
	target          string
	printPlanCalled bool
	modeOverride    *config.ReproducibilityMode
}

var errTestPlan = fmt.Errorf("plan error")

func newFakePlanPreflightClient(plan *execution.Plan, err error) *fakePlanPreflightClient {
	if plan == nil {
		plan = execution.NewExecutionPlan()
	}
	return &fakePlanPreflightClient{
		planResult: plan,
		planErr:    err,
	}
}

func (f *fakePlanPreflightClient) Plan(_ context.Context, configPath, target string) (*execution.Plan, error) {
	f.configPath = configPath
	f.target = target
	return f.planResult, f.planErr
}

func (f *fakePlanPreflightClient) PrintPlan(plan *execution.Plan) {
	if plan != nil {
		f.printPlanCalled = true
	}
}

func (f *fakePlanPreflightClient) Apply(context.Context, *execution.Plan, bool) ([]execution.StepResult, error) {
	return nil, nil
}

func (f *fakePlanPreflightClient) PrintResults([]execution.StepResult) {}

func (f *fakePlanPreflightClient) UpdateLockFromPlan(context.Context, string, *execution.Plan) error {
	return nil
}

func (f *fakePlanPreflightClient) WithMode(mode config.ReproducibilityMode) preflightClient {
	f.modeOverride = &mode
	return f
}

func (f *fakePlanPreflightClient) WithRollbackOnFailure(bool) preflightClient {
	return f
}
