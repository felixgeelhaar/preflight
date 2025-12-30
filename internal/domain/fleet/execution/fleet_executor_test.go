package execution

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/felixgeelhaar/preflight/internal/domain/fleet/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestHosts(t *testing.T, count int) []*fleet.Host {
	t.Helper()
	hosts := make([]*fleet.Host, count)
	for i := 0; i < count; i++ {
		id, _ := fleet.NewHostID("host" + string(rune('a'+i)))
		host, _ := fleet.NewHost(id, fleet.SSHConfig{Hostname: "localhost"})
		hosts[i] = host
	}
	return hosts
}

func TestDefaultExecutorConfig(t *testing.T) {
	t.Parallel()

	config := DefaultExecutorConfig()

	assert.Equal(t, StrategyParallel, config.Strategy)
	assert.Equal(t, 10, config.MaxParallel)
	assert.Equal(t, 5, config.BatchSize)
	assert.False(t, config.StopOnError)
	assert.Equal(t, 5*time.Minute, config.Timeout)
	assert.False(t, config.DryRun)
}

func TestNewFleetExecutor(t *testing.T) {
	t.Parallel()

	t.Run("with defaults", func(t *testing.T) {
		t.Parallel()
		tr := transport.NewLocalTransport()
		config := DefaultExecutorConfig()
		executor := NewFleetExecutor(tr, config)
		assert.NotNil(t, executor)
	})

	t.Run("fixes invalid config", func(t *testing.T) {
		t.Parallel()
		tr := transport.NewLocalTransport()
		config := ExecutorConfig{
			MaxParallel: 0,
			BatchSize:   0,
			Timeout:     0,
		}
		executor := NewFleetExecutor(tr, config)
		assert.NotNil(t, executor)
	})
}

func TestFleetExecutor_Execute_EmptyHosts(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	executor := NewFleetExecutor(tr, DefaultExecutorConfig())

	result := executor.Execute(context.Background(), nil, nil)

	assert.NotZero(t, result.StartTime)
	assert.NotZero(t, result.EndTime)
	assert.Empty(t, result.HostResults)
}

func TestFleetExecutor_Execute_Parallel(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.Strategy = StrategyParallel
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 3)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "true"),
		NewRemoteStep("step2", "true"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.Equal(t, 3, result.TotalHosts())
	assert.True(t, result.AllSuccessful())
}

func TestFleetExecutor_Execute_Rolling(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.Strategy = StrategyRolling
	config.BatchSize = 2
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 5)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "true"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.Equal(t, 5, result.TotalHosts())
	assert.True(t, result.AllSuccessful())
}

func TestFleetExecutor_Execute_Canary(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.Strategy = StrategyCanary
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 3)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "true"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.Equal(t, 3, result.TotalHosts())
	assert.True(t, result.AllSuccessful())
}

func TestFleetExecutor_Execute_CanaryFailure(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.Strategy = StrategyCanary
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 3)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "exit 1"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.Equal(t, 3, result.TotalHosts())
	assert.Equal(t, 1, result.FailedHosts())
	assert.Equal(t, 2, result.SkippedHosts())
}

func TestFleetExecutor_Execute_StopOnError(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.Strategy = StrategyRolling
	config.BatchSize = 1
	config.StopOnError = true
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 3)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "exit 1"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	// First host fails, rest are skipped
	assert.Equal(t, 3, result.TotalHosts())
	assert.Equal(t, 1, result.FailedHosts())
	assert.Equal(t, 2, result.SkippedHosts())
}

func TestFleetExecutor_Execute_DryRun(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	config.DryRun = true
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 1)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "touch /tmp/test-dryrun"),
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.True(t, result.AllSuccessful())
	// Step should not have been applied
	assert.Equal(t, 0, result.HostResults[0].StepsApplied())
}

func TestFleetExecutor_Execute_StepCheck(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	config := DefaultExecutorConfig()
	executor := NewFleetExecutor(tr, config)

	hosts := createTestHosts(t, 1)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "echo applied").
			WithCheck("true"), // Check passes, step is satisfied
	}

	result := executor.Execute(context.Background(), hosts, steps)

	assert.True(t, result.AllSuccessful())
	// Step should be satisfied, not applied
	assert.Equal(t, 0, result.HostResults[0].StepsApplied())
	assert.Equal(t, StepStatusSatisfied, result.HostResults[0].StepResults[0].Status)
}

func TestFleetExecutor_Plan(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	executor := NewFleetExecutor(tr, DefaultExecutorConfig())

	hosts := createTestHosts(t, 2)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "echo hello").
			WithCheck("false").
			WithDescription("Test step"),
		NewRemoteStep("step2", "echo world").
			WithCheck("true"),
	}

	plan, err := executor.Plan(context.Background(), hosts, steps)
	require.NoError(t, err)

	assert.Len(t, plan.Hosts, 2)
	assert.Equal(t, 2, plan.HostsWithChanges()) // Both hosts have step1 that needs changes

	for _, hp := range plan.Hosts {
		assert.Len(t, hp.Steps, 2)
		assert.True(t, hp.HasChanges())
		assert.Equal(t, StepStatusNeeds, hp.Steps[0].Status)
		assert.Equal(t, StepStatusSatisfied, hp.Steps[1].Status)
	}

	// Total changes: 2 hosts * 1 step that needs changes
	assert.Equal(t, 2, plan.TotalChanges())
}

func TestFleetPlan_NoChanges(t *testing.T) {
	t.Parallel()

	tr := transport.NewLocalTransport()
	executor := NewFleetExecutor(tr, DefaultExecutorConfig())

	hosts := createTestHosts(t, 1)
	steps := []*RemoteStep{
		NewRemoteStep("step1", "echo hello").
			WithCheck("true"), // Already satisfied
	}

	plan, err := executor.Plan(context.Background(), hosts, steps)
	require.NoError(t, err)

	assert.Equal(t, 0, plan.TotalChanges())
	assert.Equal(t, 0, plan.HostsWithChanges())
}

func TestHostPlan_HasChanges(t *testing.T) {
	t.Parallel()

	t.Run("with changes", func(t *testing.T) {
		t.Parallel()
		hp := &HostPlan{
			Steps: []StepPlan{
				{Status: StepStatusSatisfied},
				{Status: StepStatusNeeds},
			},
		}
		assert.True(t, hp.HasChanges())
	})

	t.Run("without changes", func(t *testing.T) {
		t.Parallel()
		hp := &HostPlan{
			Steps: []StepPlan{
				{Status: StepStatusSatisfied},
				{Status: StepStatusSatisfied},
			},
		}
		assert.False(t, hp.HasChanges())
	})
}
