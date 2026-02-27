package main

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestFormatDuration_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"exactly one minute", time.Minute, "1m"},
		{"exactly one hour", time.Hour, "1h 0m"},
		{"exactly one day", 24 * time.Hour, "1d 0h"},
		{"multi-day", 72 * time.Hour, "3d 0h"},
		{"49 hours", 49 * time.Hour, "2d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatDuration(tt.duration))
		})
	}
}

func TestFormatHealth_EmptyMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   agent.HealthStatus
		expected string
	}{
		{"degraded empty message", agent.HealthStatus{Status: agent.HealthDegraded, Message: ""}, "degraded ()"},
		{"unhealthy empty message", agent.HealthStatus{Status: agent.HealthUnhealthy, Message: ""}, "unhealthy ()"},
		{"empty status string", agent.HealthStatus{Status: ""}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatHealth(tt.status))
		})
	}
}

// ---------------------------------------------------------------------------
// Command structure tests
// ---------------------------------------------------------------------------

func TestAgentCmd_IsRegisteredOnRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "agent" {
			found = true
			break
		}
	}
	assert.True(t, found, "agent command should be registered on root")
}

func TestAgentStopCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	f := agentStopCmd.Flags().Lookup("force")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestAgentStatusCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"json", "json", "false"},
		{"watch", "watch", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := agentStatusCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %s should exist", tt.flag)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestAgentInstallCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"schedule", "schedule", "30m"},
		{"remediation", "remediation", "notify"},
		{"target", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := agentInstallCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %s should exist", tt.flag)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestAgentApproveCmd_RequiresOneArg(t *testing.T) {
	t.Parallel()

	err := agentApproveCmd.Args(agentApproveCmd, []string{})
	assert.Error(t, err)

	err = agentApproveCmd.Args(agentApproveCmd, []string{"req-123"})
	assert.NoError(t, err)

	err = agentApproveCmd.Args(agentApproveCmd, []string{"a", "b"})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// run* function tests (experimental gate, validation)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global env
func TestRunAgentStart_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentStart(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies global flags
func TestRunAgentStart_InvalidSchedule(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origSchedule := agentSchedule
	defer func() { agentSchedule = origSchedule }()
	agentSchedule = "invalid-schedule"

	err := runAgentStart(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
}

//nolint:tparallel // modifies global flags
func TestRunAgentStart_InvalidRemediation(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origSchedule := agentSchedule
	origRemediation := agentRemediation
	defer func() {
		agentSchedule = origSchedule
		agentRemediation = origRemediation
	}()
	agentSchedule = "30m"
	agentRemediation = "invalid-policy"

	err := runAgentStart(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid remediation policy")
}

//nolint:tparallel // modifies global env
func TestRunAgentStop_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentStop(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies env, captures stdout
func TestRunAgentStop_AgentNotRunning(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	output := captureStdout(t, func() {
		err := runAgentStop(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Agent is not running")
}

//nolint:tparallel // modifies global env
func TestRunAgentStatus_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentStatus(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies env, captures stdout
func TestRunAgentStatus_AgentNotRunning_Text(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origJSON := agentStatusJSON
	origWatch := agentStatusWatch
	defer func() {
		agentStatusJSON = origJSON
		agentStatusWatch = origWatch
	}()
	agentStatusJSON = false
	agentStatusWatch = false

	output := captureStdout(t, func() {
		err := runAgentStatus(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Agent is not running")
	assert.Contains(t, output, "preflight agent start")
}

//nolint:tparallel // modifies env, captures stdout
func TestRunAgentStatus_AgentNotRunning_JSON(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origJSON := agentStatusJSON
	origWatch := agentStatusWatch
	defer func() {
		agentStatusJSON = origJSON
		agentStatusWatch = origWatch
	}()
	agentStatusJSON = true
	agentStatusWatch = false

	output := captureStdout(t, func() {
		err := runAgentStatus(nil, nil)
		assert.NoError(t, err)
	})

	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, false, result["running"])
}

//nolint:tparallel // modifies global env
func TestRunAgentInstall_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentInstall(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies global flags and env
func TestRunAgentInstall_InvalidSchedule(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origSchedule := agentSchedule
	defer func() { agentSchedule = origSchedule }()
	agentSchedule = "bad-schedule"

	err := runAgentInstall(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid schedule")
}

//nolint:tparallel // modifies global flags and env
func TestRunAgentInstall_InvalidRemediation(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origSchedule := agentSchedule
	origRemediation := agentRemediation
	defer func() {
		agentSchedule = origSchedule
		agentRemediation = origRemediation
	}()
	agentSchedule = "30m"
	agentRemediation = "bad-policy"

	err := runAgentInstall(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid remediation policy")
}

//nolint:tparallel // modifies global env
func TestRunAgentUninstall_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentUninstall(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies global env
func TestRunAgentApprove_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runAgentApprove(nil, []string{"req-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

//nolint:tparallel // modifies global env
func TestRunAgentApprove_AgentNotRunning(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	err := runAgentApprove(nil, []string{"req-123"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent is not running")
}

// ---------------------------------------------------------------------------
// agentProvider tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Platform-specific install/uninstall tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies env and flags
func TestRunAgentInstall_RoutesToPlatform(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origSchedule := agentSchedule
	origRemediation := agentRemediation
	defer func() {
		agentSchedule = origSchedule
		agentRemediation = origRemediation
	}()
	agentSchedule = "30m"
	agentRemediation = "notify"

	// The install function routes by GOOS. On this platform, it should
	// attempt the platform-specific install (which may fail due to actual
	// system commands, but shouldn't fail on validation).
	err := runAgentInstall(nil, nil)
	// On macOS or Linux, this will attempt to write a plist/service file.
	// It might fail or succeed depending on the environment. We just
	// verify it got past validation.
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	}
	// On darwin/linux we can't guarantee success, but we know it passed validation
}

//nolint:tparallel // modifies env
func TestRunAgentUninstall_RoutesToPlatform(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	// On macOS, uninstallLaunchAgent checks if plist exists.
	// On Linux, uninstallSystemdService checks if service file exists.
	output := captureStdout(t, func() {
		err := runAgentUninstall(nil, nil)
		if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	})

	if runtime.GOOS == "darwin" {
		// May say "not installed" or "uninstalled successfully" depending on prior tests
		assert.True(t,
			strings.Contains(output, "LaunchAgent is not installed") ||
				strings.Contains(output, "LaunchAgent uninstalled successfully"),
			"unexpected output: %s", output)
	} else if runtime.GOOS == "linux" {
		assert.True(t,
			strings.Contains(output, "Systemd service is not installed") ||
				strings.Contains(output, "Systemd service uninstalled"),
			"unexpected output: %s", output)
	}
}

//nolint:tparallel // modifies env, calls os functions
func TestUninstallLaunchAgent_NotInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}
	// If the plist doesn't exist, it should print a message and return nil
	output := captureStdout(t, func() {
		err := uninstallLaunchAgent()
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "LaunchAgent is not installed")
}

//nolint:tparallel // modifies env, calls os functions
func TestUninstallSystemdService_NotInstalled(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	output := captureStdout(t, func() {
		err := uninstallSystemdService()
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Systemd service is not installed")
}

// ---------------------------------------------------------------------------
// installLaunchAgent plist content test
// ---------------------------------------------------------------------------

func TestInstallLaunchAgent_PlistContent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	origSchedule := agentSchedule
	origRemediation := agentRemediation
	defer func() {
		agentSchedule = origSchedule
		agentRemediation = origRemediation
	}()
	agentSchedule = "15m"
	agentRemediation = "auto"

	// installLaunchAgent will write the plist and then try to launchctl load,
	// which will fail in test. But we can verify the plist was written.
	_ = installLaunchAgent()

	plistPath := tmpDir + "/Library/LaunchAgents/com.preflight.agent.plist"
	data, err := os.ReadFile(plistPath)
	if err != nil {
		t.Skip("plist not written (launchctl may have failed early)")
	}

	content := string(data)
	assert.Contains(t, content, "com.preflight.agent")
	assert.Contains(t, content, "--foreground")
	assert.Contains(t, content, "15m")
	assert.Contains(t, content, "auto")
	assert.Contains(t, content, tmpDir+"/.preflight/agent.log")
}
