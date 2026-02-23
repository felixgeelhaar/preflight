package main

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- agentProvider tests ---

func TestAgentProvider_Status(t *testing.T) {
	t.Parallel()

	cfg := agent.DefaultConfig()
	cfg.Schedule = agent.NewIntervalSchedule(1 * time.Hour)
	ag, err := agent.NewAgent(cfg)
	require.NoError(t, err)

	provider := &agentProvider{agent: ag}

	status := provider.Status()
	assert.Equal(t, agent.StateStopped, status.State)
}

func TestAgentProvider_StatusRunning(t *testing.T) {
	cfg := agent.DefaultConfig()
	cfg.Schedule = agent.NewIntervalSchedule(1 * time.Hour)
	ag, err := agent.NewAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, ag.Start(ctx))
	time.Sleep(150 * time.Millisecond)

	provider := &agentProvider{agent: ag}
	status := provider.Status()
	assert.Equal(t, agent.StateRunning, status.State)

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = ag.Stop(stopCtx)
}

func TestAgentProvider_Stop(t *testing.T) {
	cfg := agent.DefaultConfig()
	cfg.Schedule = agent.NewIntervalSchedule(1 * time.Hour)
	ag, err := agent.NewAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, ag.Start(ctx))
	time.Sleep(150 * time.Millisecond)

	provider := &agentProvider{agent: ag}

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = provider.Stop(stopCtx)
	require.NoError(t, err)
	assert.Equal(t, agent.StateStopped, ag.State())
}

func TestAgentProvider_StopWithoutStart(t *testing.T) {
	t.Parallel()

	cfg := agent.DefaultConfig()
	ag, err := agent.NewAgent(cfg)
	require.NoError(t, err)

	provider := &agentProvider{agent: ag}

	err = provider.Stop(context.Background())
	require.NoError(t, err)
}

func TestAgentProvider_Approve(t *testing.T) {
	t.Parallel()

	cfg := agent.DefaultConfig()
	ag, err := agent.NewAgent(cfg)
	require.NoError(t, err)

	provider := &agentProvider{agent: ag}

	err = provider.Approve("some-request-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approval not yet implemented")
}

// --- reconcileApp fake ---

type fakeReconcileApp struct {
	planResult    *execution.Plan
	planErr       error
	applyResults  []execution.StepResult
	applyErr      error
	planCalled    bool
	applyCalled   bool
	appliedDryRun bool
}

func (f *fakeReconcileApp) Plan(_ context.Context, _, _ string) (*execution.Plan, error) {
	f.planCalled = true
	return f.planResult, f.planErr
}

func (f *fakeReconcileApp) Apply(_ context.Context, _ *execution.Plan, dryRun bool) ([]execution.StepResult, error) {
	f.applyCalled = true
	f.appliedDryRun = dryRun
	return f.applyResults, f.applyErr
}

// --- reconcile tests ---

func TestReconcile_NoDrift(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		newAgentDummyStep("brew:formula:git"),
		compiler.StatusSatisfied,
		compiler.Diff{},
	))

	fake := &fakeReconcileApp{planResult: plan}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationAuto)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, fake.planCalled)
	assert.False(t, fake.applyCalled, "should not apply when no drift")
	assert.False(t, result.DriftDetected)
	assert.Equal(t, 0, result.DriftCount)
	assert.False(t, result.RemediationApplied)
	assert.Equal(t, 0, result.RemediationCount)
	assert.False(t, result.CompletedAt.IsZero())
	assert.Positive(t, result.Duration)
}

func TestReconcile_DriftWithAutoRemediation(t *testing.T) {
	t.Parallel()

	step := newAgentDummyStep("brew:formula:ripgrep")
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		step,
		compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", ""),
	))

	applyResults := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusSatisfied, nil),
	}

	fake := &fakeReconcileApp{planResult: plan, applyResults: applyResults}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationAuto)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, fake.planCalled)
	assert.True(t, fake.applyCalled)
	assert.False(t, fake.appliedDryRun)
	assert.True(t, result.DriftDetected)
	assert.Equal(t, 1, result.DriftCount)
	assert.True(t, result.RemediationApplied)
	assert.Equal(t, 1, result.RemediationCount)
}

func TestReconcile_DriftWithSafeRemediation(t *testing.T) {
	t.Parallel()

	step := newAgentDummyStep("files:link:bashrc")
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		step,
		compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeModify, "files", "link", "", ""),
	))

	applyResults := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusSatisfied, nil),
	}

	fake := &fakeReconcileApp{planResult: plan, applyResults: applyResults}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationSafe)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, fake.applyCalled, "safe remediation should trigger apply")
	assert.True(t, result.DriftDetected)
	assert.True(t, result.RemediationApplied)
}

func TestReconcile_DriftWithNotifyPolicy(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		newAgentDummyStep("brew:formula:htop"),
		compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", ""),
	))

	fake := &fakeReconcileApp{planResult: plan}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationNotify)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, fake.planCalled)
	assert.False(t, fake.applyCalled, "notify policy should not apply")
	assert.True(t, result.DriftDetected)
	assert.Equal(t, 1, result.DriftCount)
	assert.False(t, result.RemediationApplied)
	assert.Equal(t, 0, result.RemediationCount)
}

func TestReconcile_DriftWithApprovedPolicy(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		newAgentDummyStep("apt:package:curl"),
		compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "apt", "package", "", ""),
	))

	fake := &fakeReconcileApp{planResult: plan}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationApproved)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.False(t, fake.applyCalled, "approved policy should not auto-apply")
	assert.True(t, result.DriftDetected)
	assert.False(t, result.RemediationApplied)
}

func TestReconcile_PlanError(t *testing.T) {
	t.Parallel()

	fake := &fakeReconcileApp{planErr: errors.New("config not found")}
	cfg := agent.DefaultConfig()

	result, err := reconcile(context.Background(), fake, cfg)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "plan failed")
	assert.Contains(t, err.Error(), "config not found")
}

func TestReconcile_ApplyError(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(
		newAgentDummyStep("brew:formula:node"),
		compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", ""),
	))

	fake := &fakeReconcileApp{
		planResult: plan,
		applyErr:   errors.New("permission denied"),
	}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationAuto)

	result, err := reconcile(context.Background(), fake, cfg)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "apply failed")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestReconcile_PartialApplySuccess(t *testing.T) {
	t.Parallel()

	step1 := newAgentDummyStep("brew:formula:git")
	step2 := newAgentDummyStep("brew:formula:curl")
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(step1, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", "")))
	plan.Add(execution.NewPlanEntry(step2, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", "")))

	applyResults := []execution.StepResult{
		execution.NewStepResult(step1.ID(), compiler.StatusSatisfied, nil),
		execution.NewStepResult(step2.ID(), compiler.StatusFailed, errors.New("install failed")),
	}

	fake := &fakeReconcileApp{planResult: plan, applyResults: applyResults}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationAuto)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, result.DriftDetected)
	assert.Equal(t, 2, result.DriftCount)
	assert.True(t, result.RemediationApplied)
	assert.Equal(t, 1, result.RemediationCount, "only one step succeeded")
}

func TestReconcile_AllApplyFailed(t *testing.T) {
	t.Parallel()

	step := newAgentDummyStep("brew:formula:node")
	plan := execution.NewExecutionPlan()
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "brew", "formula", "", "")))

	applyResults := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusFailed, errors.New("failed")),
	}

	fake := &fakeReconcileApp{planResult: plan, applyResults: applyResults}
	cfg := agent.DefaultConfig().WithRemediation(agent.RemediationAuto)

	result, err := reconcile(context.Background(), fake, cfg)
	require.NoError(t, err)

	assert.True(t, result.DriftDetected)
	assert.False(t, result.RemediationApplied, "no steps succeeded")
	assert.Equal(t, 0, result.RemediationCount)
}

func TestReconcile_TimestampsAndDuration(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	fake := &fakeReconcileApp{planResult: plan}
	cfg := agent.DefaultConfig()

	before := time.Now()
	result, err := reconcile(context.Background(), fake, cfg)
	after := time.Now()
	require.NoError(t, err)

	assert.False(t, result.StartedAt.IsZero())
	assert.False(t, result.CompletedAt.IsZero())
	assert.True(t, result.StartedAt.After(before) || result.StartedAt.Equal(before))
	assert.True(t, result.CompletedAt.Before(after) || result.CompletedAt.Equal(after))
	assert.True(t, result.CompletedAt.After(result.StartedAt) || result.CompletedAt.Equal(result.StartedAt))
	assert.GreaterOrEqual(t, result.Duration.Nanoseconds(), int64(0))
}

func TestReconcile_UsesConfigPathAndTarget(t *testing.T) {
	t.Parallel()

	plan := execution.NewExecutionPlan()
	var capturedConfigPath, capturedTarget string
	fake := &fakeReconcileApp{planResult: plan}
	fake2 := &configCapturingReconcileApp{
		fakeReconcileApp: fake,
		onPlan: func(configPath, target string) {
			capturedConfigPath = configPath
			capturedTarget = target
		},
	}

	cfg := agent.DefaultConfig().WithTarget("work-laptop")
	cfg.ConfigPath = "/home/user/preflight.yaml"

	_, err := reconcile(context.Background(), fake2, cfg)
	require.NoError(t, err)
	assert.Equal(t, "/home/user/preflight.yaml", capturedConfigPath)
	assert.Equal(t, "work-laptop", capturedTarget)
}

// configCapturingReconcileApp wraps fakeReconcileApp to capture Plan arguments.
type configCapturingReconcileApp struct {
	*fakeReconcileApp
	onPlan func(configPath, target string)
}

func (f *configCapturingReconcileApp) Plan(ctx context.Context, configPath, target string) (*execution.Plan, error) {
	if f.onPlan != nil {
		f.onPlan(configPath, target)
	}
	return f.fakeReconcileApp.Plan(ctx, configPath, target)
}

// --- daemonProcAttr tests ---

func TestDaemonProcAttr(t *testing.T) {
	t.Parallel()

	attr := daemonProcAttr()
	require.NotNil(t, attr)
	assert.IsType(t, &syscall.SysProcAttr{}, attr)
}

// --- agentStartCmd flag tests ---

func TestAgentStartCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"foreground default", "foreground", "false"},
		{"schedule default", "schedule", "30m"},
		{"remediation default", "remediation", "notify"},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := agentStartCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %q should exist", tt.flag)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestAgentCmd_SubcommandRegistration(t *testing.T) {
	t.Parallel()

	expected := []string{"start", "stop", "status", "install", "uninstall", "approve"}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			found := false
			for _, cmd := range agentCmd.Commands() {
				if cmd.Use == name || cmd.Name() == name {
					found = true
					break
				}
			}
			assert.True(t, found, "agent should have %q subcommand", name)
		})
	}
}

// --- helper ---

type agentDummyStep struct {
	id compiler.StepID
}

func newAgentDummyStep(id string) *agentDummyStep {
	stepID, _ := compiler.NewStepID(id)
	return &agentDummyStep{id: stepID}
}

func (d *agentDummyStep) ID() compiler.StepID          { return d.id }
func (d *agentDummyStep) DependsOn() []compiler.StepID { return nil }
func (d *agentDummyStep) Check(_ compiler.RunContext) (compiler.StepStatus, error) {
	return compiler.StatusSatisfied, nil
}
func (d *agentDummyStep) Plan(_ compiler.RunContext) (compiler.Diff, error) {
	return compiler.Diff{}, nil
}
func (d *agentDummyStep) Apply(_ compiler.RunContext) error { return nil }
func (d *agentDummyStep) Explain(_ compiler.ExplainContext) compiler.Explanation {
	return compiler.Explanation{}
}
