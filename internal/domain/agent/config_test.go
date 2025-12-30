package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemediationPolicy_IsValid(t *testing.T) {
	tests := []struct {
		policy RemediationPolicy
		valid  bool
	}{
		{RemediationNotify, true},
		{RemediationAuto, true},
		{RemediationApproved, true},
		{RemediationSafe, true},
		{RemediationPolicy("invalid"), false},
		{RemediationPolicy(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.policy), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.policy.IsValid())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 30*time.Minute, cfg.Schedule.Interval())
	assert.Equal(t, RemediationNotify, cfg.Remediation)
	assert.Equal(t, "default", cfg.Target)
	assert.Equal(t, "preflight.yaml", cfg.ConfigPath)

	// Notifications
	assert.True(t, cfg.Notifications.OnDrift)
	assert.True(t, cfg.Notifications.OnRemediation)
	assert.True(t, cfg.Notifications.OnError)

	// Timeouts
	assert.Equal(t, 5*time.Minute, cfg.Timeouts.Reconciliation)
	assert.Equal(t, 30*time.Second, cfg.Timeouts.Shutdown)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name:   "valid default config",
			modify: func(_ *Config) {},
		},
		{
			name: "schedule too short",
			modify: func(c *Config) {
				c.Schedule = NewIntervalSchedule(30 * time.Second)
			},
			wantErr: "schedule interval must be at least 1 minute",
		},
		{
			name: "invalid remediation policy",
			modify: func(c *Config) {
				c.Remediation = RemediationPolicy("unknown")
			},
			wantErr: "invalid remediation policy",
		},
		{
			name: "empty target",
			modify: func(c *Config) {
				c.Target = ""
			},
			wantErr: "target is required",
		},
		{
			name: "zero reconciliation timeout",
			modify: func(c *Config) {
				c.Timeouts.Reconciliation = 0
			},
			wantErr: "reconciliation timeout must be positive",
		},
		{
			name: "negative reconciliation timeout",
			modify: func(c *Config) {
				c.Timeouts.Reconciliation = -1
			},
			wantErr: "reconciliation timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfig_WithSchedule(t *testing.T) {
	cfg := DefaultConfig()
	newSchedule := NewIntervalSchedule(1 * time.Hour)

	newCfg := cfg.WithSchedule(newSchedule)

	assert.Equal(t, 1*time.Hour, newCfg.Schedule.Interval())
	assert.Equal(t, 30*time.Minute, cfg.Schedule.Interval()) // Original unchanged
}

func TestConfig_WithRemediation(t *testing.T) {
	cfg := DefaultConfig()

	newCfg := cfg.WithRemediation(RemediationAuto)

	assert.Equal(t, RemediationAuto, newCfg.Remediation)
	assert.Equal(t, RemediationNotify, cfg.Remediation) // Original unchanged
}

func TestConfig_WithTarget(t *testing.T) {
	cfg := DefaultConfig()

	newCfg := cfg.WithTarget("production")

	assert.Equal(t, "production", newCfg.Target)
	assert.Equal(t, "default", cfg.Target) // Original unchanged
}
