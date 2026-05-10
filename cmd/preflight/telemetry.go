package main

import (
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/telemetry"
)

// preflightConfigDir returns the per-user config directory used for opt-in
// telemetry, machine ID, and consent state. Honors PREFLIGHT_HOME for tests
// and unusual deployments. Returns "" if the home directory cannot be
// resolved — callers should treat this as "telemetry disabled".
func preflightConfigDir() string {
	if explicit := os.Getenv("PREFLIGHT_HOME"); explicit != "" {
		return explicit
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".preflight")
}

// recordEvent fires a telemetry event if the user has opted in. Safe to call
// from any code path: the recorder is a no-op when disabled, and a missing
// home directory is treated as "telemetry off". Never returns an error
// because telemetry must not interfere with the user's workflow.
func recordEvent(name string) {
	dir := preflightConfigDir()
	if dir == "" {
		return
	}
	telemetry.NewRecorder(dir).Record(name)
}
