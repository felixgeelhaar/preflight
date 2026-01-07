package main

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestFormatHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   agent.HealthStatus
		expected string
	}{
		{
			name:     "healthy",
			status:   agent.HealthStatus{Status: agent.HealthHealthy},
			expected: "healthy",
		},
		{
			name:     "degraded",
			status:   agent.HealthStatus{Status: agent.HealthDegraded, Message: "slow"},
			expected: "degraded (slow)",
		},
		{
			name:     "unhealthy",
			status:   agent.HealthStatus{Status: agent.HealthUnhealthy, Message: "offline"},
			expected: "unhealthy (offline)",
		},
		{
			name:     "unknown",
			status:   agent.HealthStatus{Status: "custom"},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatHealth(tt.status))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"negative", -5 * time.Second, "now"},
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours minutes", 90 * time.Minute, "1h 30m"},
		{"days hours", 26 * time.Hour, "1d 2h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatDuration(tt.duration))
		})
	}
}
