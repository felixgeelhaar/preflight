package agent

import (
	"fmt"
	"time"
)

// RemediationPolicy defines how the agent handles detected drift.
type RemediationPolicy string

const (
	// RemediationNotify only notifies about drift, doesn't fix it.
	RemediationNotify RemediationPolicy = "notify"
	// RemediationAuto automatically applies fixes without confirmation.
	RemediationAuto RemediationPolicy = "auto"
	// RemediationApproved requires explicit approval before applying fixes.
	RemediationApproved RemediationPolicy = "approved"
	// RemediationSafe only applies safe, reversible fixes automatically.
	RemediationSafe RemediationPolicy = "safe"
)

// IsValid checks if the remediation policy is valid.
func (p RemediationPolicy) IsValid() bool {
	switch p {
	case RemediationNotify, RemediationAuto, RemediationApproved, RemediationSafe:
		return true
	default:
		return false
	}
}

// ParseRemediationPolicy parses a string into a RemediationPolicy.
func ParseRemediationPolicy(s string) (RemediationPolicy, error) {
	policy := RemediationPolicy(s)
	if !policy.IsValid() {
		return "", fmt.Errorf("invalid remediation policy: %q (valid: notify, auto, approved, safe)", s)
	}
	return policy, nil
}

// Config holds the agent configuration.
type Config struct {
	// Enabled indicates whether the agent should run.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Schedule defines when reconciliation runs.
	Schedule Schedule `yaml:"schedule" json:"schedule"`

	// Remediation defines how drift is handled.
	Remediation RemediationPolicy `yaml:"remediation" json:"remediation"`

	// Target is the preflight target to apply.
	Target string `yaml:"target" json:"target"`

	// ConfigPath is the path to the preflight.yaml config file.
	ConfigPath string `yaml:"config_path" json:"config_path"`

	// Notifications configures notification settings.
	Notifications NotificationConfig `yaml:"notifications" json:"notifications"`

	// Timeouts configures various timeout values.
	Timeouts TimeoutConfig `yaml:"timeouts" json:"timeouts"`
}

// NotificationConfig defines notification settings.
type NotificationConfig struct {
	// OnDrift enables notifications when drift is detected.
	OnDrift bool `yaml:"on_drift" json:"on_drift"`
	// OnRemediation enables notifications when remediation is applied.
	OnRemediation bool `yaml:"on_remediation" json:"on_remediation"`
	// OnError enables notifications on errors.
	OnError bool `yaml:"on_error" json:"on_error"`
}

// TimeoutConfig defines timeout values.
type TimeoutConfig struct {
	// Reconciliation is the max time for a reconciliation cycle.
	Reconciliation time.Duration `yaml:"reconciliation" json:"reconciliation"`
	// Shutdown is the max time to wait for graceful shutdown.
	Shutdown time.Duration `yaml:"shutdown" json:"shutdown"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		Schedule:    Schedule{interval: 30 * time.Minute},
		Remediation: RemediationNotify,
		Target:      "default",
		ConfigPath:  "preflight.yaml",
		Notifications: NotificationConfig{
			OnDrift:       true,
			OnRemediation: true,
			OnError:       true,
		},
		Timeouts: TimeoutConfig{
			Reconciliation: 5 * time.Minute,
			Shutdown:       30 * time.Second,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Schedule.Interval() < time.Minute {
		return fmt.Errorf("schedule interval must be at least 1 minute")
	}
	if !c.Remediation.IsValid() {
		return fmt.Errorf("invalid remediation policy: %s", c.Remediation)
	}
	if c.Target == "" {
		return fmt.Errorf("target is required")
	}
	if c.Timeouts.Reconciliation <= 0 {
		return fmt.Errorf("reconciliation timeout must be positive")
	}
	return nil
}

// WithSchedule returns a copy with the given schedule.
func (c *Config) WithSchedule(s Schedule) *Config {
	cfg := *c
	cfg.Schedule = s
	return &cfg
}

// WithRemediation returns a copy with the given remediation policy.
func (c *Config) WithRemediation(p RemediationPolicy) *Config {
	cfg := *c
	cfg.Remediation = p
	return &cfg
}

// WithTarget returns a copy with the given target.
func (c *Config) WithTarget(t string) *Config {
	cfg := *c
	cfg.Target = t
	return &cfg
}
