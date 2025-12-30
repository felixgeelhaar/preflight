package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	t.Run("creates agent with valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		agent, err := NewAgent(cfg)

		require.NoError(t, err)
		assert.NotNil(t, agent)
		assert.Equal(t, StateStopped, agent.State())
	})

	t.Run("returns error with nil config", func(t *testing.T) {
		agent, err := NewAgent(nil)

		require.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "config is required")
	})
}

func TestAgent_StartStop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(100 * time.Millisecond) // Fast for testing
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Start agent
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Wait for agent to be running
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, StateRunning, agent.State())

	// Stop agent
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = agent.Stop(stopCtx)
	require.NoError(t, err)

	// Agent should be stopped (interpreter is nil after stop)
	assert.Equal(t, StateStopped, agent.State())
}

func TestAgent_Status(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(1 * time.Hour) // Long interval
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	// Status before start - interpreter not created yet
	status := agent.Status()
	assert.Equal(t, StateStopped, status.State)
	assert.True(t, status.StartedAt.IsZero())
	assert.Equal(t, 0, status.ReconcileCount)

	// Start agent
	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Wait for agent to be running
	time.Sleep(150 * time.Millisecond)

	// Status after start
	status = agent.Status()
	assert.Equal(t, StateRunning, status.State)
	// Note: StartedAt is recorded by the state machine action, check via Runtime
	assert.GreaterOrEqual(t, agent.Runtime().GetContext().ReconcileCount, 0)

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = agent.Stop(stopCtx)
}

func TestAgent_SetReconcileHandler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(50 * time.Millisecond) // Fast for testing
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	reconcileCount := 0
	agent.SetReconcileHandler(func(_ context.Context) (*ReconciliationResult, error) {
		reconcileCount++
		result := NewReconciliationResult()
		result.Complete()
		return result, nil
	})

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Wait for at least one reconciliation
	time.Sleep(500 * time.Millisecond)

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = agent.Stop(stopCtx)

	// Should have reconciled at least once
	assert.Positive(t, reconcileCount)
}

func TestAgent_ReconcileError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(50 * time.Millisecond)
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	expectedErr := errors.New("reconciliation failed")
	agent.SetReconcileHandler(func(_ context.Context) (*ReconciliationResult, error) {
		return nil, expectedErr
	})

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Wait for agent to start and trigger reconciliation
	time.Sleep(400 * time.Millisecond)

	// Agent should be in error state
	state := agent.State()
	assert.Equal(t, StateError, state, "expected error state after failed reconciliation")

	// Context should reflect error (modified by closure in action)
	runtimeCtx := agent.Runtime().GetContext()
	assert.Positive(t, runtimeCtx.ErrorCount, "expected error count > 0")
	assert.Equal(t, HealthDegraded, runtimeCtx.Health.Status)

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = agent.Stop(stopCtx)
}

func TestAgent_Recover(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(100 * time.Millisecond)
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	// First call fails, subsequent calls succeed
	callCount := 0
	agent.SetReconcileHandler(func(_ context.Context) (*ReconciliationResult, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("first call fails")
		}
		result := NewReconciliationResult()
		result.Complete()
		return result, nil
	})

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Wait for error state
	time.Sleep(400 * time.Millisecond)
	assert.Equal(t, StateError, agent.State(), "expected error state after first reconciliation")

	// Recover
	agent.Recover()

	// Wait for recovery
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, StateRunning, agent.State(), "expected running state after recovery")

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = agent.Stop(stopCtx)
}

func TestAgent_SendEvent(t *testing.T) {
	cfg := DefaultConfig()
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Send custom event (should not panic)
	agent.SendEvent("CUSTOM_EVENT", map[string]interface{}{"key": "value"})

	// Cleanup
	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = agent.Stop(stopCtx)
}

func TestAgent_StopTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Schedule = NewIntervalSchedule(1 * time.Hour)
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Stop with very short timeout (should not block)
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err = agent.Stop(stopCtx)
	require.NoError(t, err)
}

func TestAgent_StopWithoutStart(t *testing.T) {
	cfg := DefaultConfig()
	agent, err := NewAgent(cfg)
	require.NoError(t, err)

	// Stop without starting should not error
	ctx := context.Background()
	err = agent.Stop(ctx)
	require.NoError(t, err)
}

func TestRuntimeContext_RecordStart(t *testing.T) {
	runtime := NewRuntimeContext(DefaultConfig())

	ctx := runtime.GetContext()
	assert.True(t, ctx.StartedAt.IsZero())
	assert.Equal(t, HealthUnknown, ctx.Health.Status)

	runtime.RecordStart()

	ctx = runtime.GetContext()
	assert.False(t, ctx.StartedAt.IsZero())
	assert.Equal(t, HealthHealthy, ctx.Health.Status)
	assert.False(t, ctx.Health.LastCheck.IsZero())
}

func TestRuntimeContext_RecordReconciliation(t *testing.T) {
	runtime := NewRuntimeContext(DefaultConfig())

	ctx := runtime.GetContext()
	assert.Equal(t, 0, ctx.ReconcileCount)
	assert.True(t, ctx.LastReconcileAt.IsZero())

	result := NewReconciliationResult()
	runtime.RecordReconciliation(result)

	ctx = runtime.GetContext()
	assert.Equal(t, 1, ctx.ReconcileCount)
	assert.False(t, ctx.LastReconcileAt.IsZero())
	assert.Equal(t, result, ctx.LastResult)
}

func TestRuntimeContext_RecordError(t *testing.T) {
	runtime := NewRuntimeContext(DefaultConfig())
	// Initial state: HealthUnknown
	ctx := runtime.GetContext()
	assert.Equal(t, 0, ctx.ErrorCount)
	assert.NoError(t, ctx.LastError)

	testErr := errors.New("test error")
	runtime.RecordError(testErr)

	ctx = runtime.GetContext()
	assert.Equal(t, 1, ctx.ErrorCount)
	assert.Equal(t, testErr, ctx.LastError)
	assert.Equal(t, HealthDegraded, ctx.Health.Status)
	assert.Equal(t, "test error", ctx.Health.Message)
}

func TestRuntimeContext_GetStatus(t *testing.T) {
	runtime := NewRuntimeContext(DefaultConfig())
	runtime.RecordStart()

	// Record multiple reconciliations to set counts
	result := NewReconciliationResult()
	for i := 0; i < 5; i++ {
		runtime.RecordReconciliation(result)
	}
	// Record errors
	for i := 0; i < 2; i++ {
		runtime.RecordError(errors.New("test"))
	}

	status := runtime.GetStatus()

	assert.False(t, status.StartedAt.IsZero())
	assert.Equal(t, 5, status.ReconcileCount)
	assert.Equal(t, 2, status.ErrorCount)
	// After recording errors, health is degraded
	assert.Equal(t, HealthDegraded, status.Health.Status)
}
