package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth_Predicates(t *testing.T) {
	tests := []struct {
		status      Health
		isHealthy   bool
		isDegraded  bool
		isUnhealthy bool
	}{
		{HealthHealthy, true, false, false},
		{HealthDegraded, false, true, false},
		{HealthUnhealthy, false, false, true},
		{HealthUnknown, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			hs := HealthStatus{Status: tt.status}
			assert.Equal(t, tt.isHealthy, hs.IsHealthy())
			assert.Equal(t, tt.isDegraded, hs.IsDegraded())
			assert.Equal(t, tt.isUnhealthy, hs.IsUnhealthy())
		})
	}
}

func TestNewHealthStatus(t *testing.T) {
	hs := NewHealthStatus()

	assert.Equal(t, HealthHealthy, hs.Status)
	assert.False(t, hs.LastCheck.IsZero())
	assert.Empty(t, hs.Checks)
}

func TestNewHealthCheck(t *testing.T) {
	check := NewHealthCheck("database", HealthHealthy, "connected")

	assert.Equal(t, "database", check.Name)
	assert.Equal(t, HealthHealthy, check.Status)
	assert.Equal(t, "connected", check.Message)
}

func TestHealthStatus_AddCheck(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		hs := NewHealthStatus()
		hs.AddCheck(NewHealthCheck("db", HealthHealthy, "ok"))
		hs.AddCheck(NewHealthCheck("cache", HealthHealthy, "ok"))

		assert.Equal(t, HealthHealthy, hs.Status)
		assert.Len(t, hs.Checks, 2)
	})

	t.Run("one degraded", func(t *testing.T) {
		hs := NewHealthStatus()
		hs.AddCheck(NewHealthCheck("db", HealthHealthy, "ok"))
		hs.AddCheck(NewHealthCheck("cache", HealthDegraded, "high latency"))

		assert.Equal(t, HealthDegraded, hs.Status)
	})

	t.Run("one unhealthy", func(t *testing.T) {
		hs := NewHealthStatus()
		hs.AddCheck(NewHealthCheck("db", HealthUnhealthy, "connection failed"))
		hs.AddCheck(NewHealthCheck("cache", HealthHealthy, "ok"))

		assert.Equal(t, HealthUnhealthy, hs.Status)
	})

	t.Run("unhealthy takes precedence over degraded", func(t *testing.T) {
		hs := NewHealthStatus()
		hs.AddCheck(NewHealthCheck("db", HealthDegraded, "slow"))
		hs.AddCheck(NewHealthCheck("cache", HealthUnhealthy, "down"))

		assert.Equal(t, HealthUnhealthy, hs.Status)
	})
}

func TestHealthStatus_UpdateOverallStatus_NoChecks(t *testing.T) {
	hs := HealthStatus{Status: HealthDegraded}
	hs.updateOverallStatus()

	// Should remain unchanged when no checks
	assert.Equal(t, HealthDegraded, hs.Status)
}
