package main

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestAgentCmd_Exists(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, agentCmd)
	assert.Equal(t, "agent", agentCmd.Use)
	assert.Contains(t, agentCmd.Short, "background agent")
}

func TestAgentCmd_HasSubcommands(t *testing.T) {
	t.Parallel()

	subcommands := []string{"start", "stop", "status", "install", "uninstall", "approve"}

	for _, name := range subcommands {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			found := false
			for _, cmd := range agentCmd.Commands() {
				if cmd.Use == name || cmd.Name() == name {
					found = true
					break
				}
			}
			assert.True(t, found, "subcommand %s should exist", name)
		})
	}
}

func TestAgentStartCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"foreground", "schedule", "remediation", "target"}

	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := agentStartCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestAgentStartCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"schedule default", "schedule", "30m"},
		{"remediation default", "remediation", "notify"},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := agentStartCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestAgentStopCmd_HasFlags(t *testing.T) {
	t.Parallel()

	f := agentStopCmd.Flags().Lookup("force")
	assert.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestAgentStatusCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"json", "watch"}

	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := agentStatusCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestAgentApproveCmd_RequiresArgs(t *testing.T) {
	t.Parallel()

	// Check that the command requires exactly 1 argument
	err := agentApproveCmd.Args(agentApproveCmd, []string{})
	assert.Error(t, err)

	err = agentApproveCmd.Args(agentApproveCmd, []string{"request-id"})
	assert.NoError(t, err)

	err = agentApproveCmd.Args(agentApproveCmd, []string{"id1", "id2"})
	assert.Error(t, err)
}

func TestFormatHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		health   agent.HealthStatus
		expected string
	}{
		{
			name:     "healthy",
			health:   agent.HealthStatus{Status: agent.HealthHealthy},
			expected: "healthy",
		},
		{
			name:     "degraded",
			health:   agent.HealthStatus{Status: agent.HealthDegraded, Message: "slow"},
			expected: "degraded (slow)",
		},
		{
			name:     "unhealthy",
			health:   agent.HealthStatus{Status: agent.HealthUnhealthy, Message: "failed"},
			expected: "unhealthy (failed)",
		},
		{
			name:     "unknown",
			health:   agent.HealthStatus{Status: "other"},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatHealth(tt.health)
			assert.Equal(t, tt.expected, result)
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
		{
			name:     "negative",
			duration: -5 * time.Second,
			expected: "now",
		},
		{
			name:     "seconds",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes",
			duration: 15 * time.Minute,
			expected: "15m",
		},
		{
			name:     "hours and minutes",
			duration: 3*time.Hour + 30*time.Minute,
			expected: "3h 30m",
		},
		{
			name:     "days and hours",
			duration: 48*time.Hour + 12*time.Hour,
			expected: "2d 12h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
