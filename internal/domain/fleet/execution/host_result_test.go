package execution

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
)

func TestHostResult_Duration(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for incomplete result", func(t *testing.T) {
		t.Parallel()
		hr := &HostResult{StartTime: time.Now()}
		assert.Zero(t, hr.Duration())
	})

	t.Run("returns duration for complete result", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		hr := &HostResult{
			StartTime: start,
			EndTime:   start.Add(5 * time.Second),
		}
		assert.Equal(t, 5*time.Second, hr.Duration())
	})
}

func TestHostResult_StepsApplied(t *testing.T) {
	t.Parallel()

	hr := &HostResult{
		StepResults: []StepResult{
			{Applied: true},
			{Applied: false},
			{Applied: true},
		},
	}

	assert.Equal(t, 2, hr.StepsApplied())
}

func TestHostResult_StepsFailed(t *testing.T) {
	t.Parallel()

	hr := &HostResult{
		StepResults: []StepResult{
			{Error: nil},
			{Error: assert.AnError},
			{Error: assert.AnError},
		},
	}

	assert.Equal(t, 2, hr.StepsFailed())
}

func TestFleetResult(t *testing.T) {
	t.Parallel()

	t.Run("new fleet result", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		assert.NotZero(t, result.StartTime)
		assert.Empty(t, result.HostResults)
	})

	t.Run("add host result", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		hr := &HostResult{HostID: fleet.HostID("test")}
		result.AddHostResult(hr)
		assert.Len(t, result.HostResults, 1)
	})

	t.Run("complete sets end time", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		result.Complete()
		assert.NotZero(t, result.EndTime)
	})

	t.Run("duration", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		time.Sleep(10 * time.Millisecond)
		result.Complete()
		assert.GreaterOrEqual(t, result.Duration(), 10*time.Millisecond)
	})
}

func TestFleetResult_HostCounts(t *testing.T) {
	t.Parallel()

	result := NewFleetResult()
	result.AddHostResult(&HostResult{Status: HostStatusSuccess})
	result.AddHostResult(&HostResult{Status: HostStatusSuccess})
	result.AddHostResult(&HostResult{Status: HostStatusFailed})
	result.AddHostResult(&HostResult{Status: HostStatusSkipped})

	assert.Equal(t, 4, result.TotalHosts())
	assert.Equal(t, 2, result.SuccessfulHosts())
	assert.Equal(t, 1, result.FailedHosts())
	assert.Equal(t, 1, result.SkippedHosts())
	assert.False(t, result.AllSuccessful())
}

func TestFleetResult_AllSuccessful(t *testing.T) {
	t.Parallel()

	t.Run("empty result returns false", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		assert.False(t, result.AllSuccessful())
	})

	t.Run("all successful", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		result.AddHostResult(&HostResult{Status: HostStatusSuccess})
		result.AddHostResult(&HostResult{Status: HostStatusSuccess})
		assert.True(t, result.AllSuccessful())
	})

	t.Run("one failure", func(t *testing.T) {
		t.Parallel()
		result := NewFleetResult()
		result.AddHostResult(&HostResult{Status: HostStatusSuccess})
		result.AddHostResult(&HostResult{Status: HostStatusFailed})
		assert.False(t, result.AllSuccessful())
	})
}

func TestFleetResult_Summary(t *testing.T) {
	t.Parallel()

	result := NewFleetResult()
	result.AddHostResult(&HostResult{Status: HostStatusSuccess})
	result.AddHostResult(&HostResult{Status: HostStatusFailed})
	result.AddHostResult(&HostResult{Status: HostStatusSkipped})
	result.Complete()

	summary := result.Summary()

	assert.Equal(t, 3, summary.TotalHosts)
	assert.Equal(t, 1, summary.SuccessfulHosts)
	assert.Equal(t, 1, summary.FailedHosts)
	assert.Equal(t, 1, summary.SkippedHosts)
	assert.NotZero(t, summary.TotalDuration)
}
