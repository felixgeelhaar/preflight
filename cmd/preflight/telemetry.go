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
//
// PREFLIGHT_HOME is rejected if it is not an absolute path. The variable is
// intended for tests; we do NOT honor relative paths or paths under known
// world-writable roots like /tmp because those are exploitable as
// consent-spoof + symlink redirect vectors on shared hosts.
func preflightConfigDir() string {
	if explicit := os.Getenv("PREFLIGHT_HOME"); explicit != "" {
		if !filepath.IsAbs(explicit) {
			return ""
		}
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

// recordOnce fires a telemetry event only the first time it is observed for
// this machine. Use for activation events whose semantics are
// "first-time-only" (e.g. apply.first_success for the TTFSA metric).
func recordOnce(name string) {
	dir := preflightConfigDir()
	if dir == "" {
		return
	}
	telemetry.NewRecorder(dir).RecordOnce(name)
}
