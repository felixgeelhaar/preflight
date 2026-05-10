package main

import (
	"path/filepath"
	"testing"
)

func TestPreflightConfigDir_RejectsRelativePREFLIGHT_HOME(t *testing.T) {
	t.Setenv("PREFLIGHT_HOME", "relative/path")
	got := preflightConfigDir()
	if got != "" {
		t.Errorf("relative PREFLIGHT_HOME must be rejected, got %q", got)
	}
}

func TestPreflightConfigDir_HonorsAbsolutePREFLIGHT_HOME(t *testing.T) {
	want := filepath.Join(t.TempDir(), "preflight-test-home")
	t.Setenv("PREFLIGHT_HOME", want)
	if got := preflightConfigDir(); got != want {
		t.Errorf("preflightConfigDir() = %q, want %q", got, want)
	}
}

func TestPreflightConfigDir_DefaultsToHomeDotPreflight(t *testing.T) {
	t.Setenv("PREFLIGHT_HOME", "")
	got := preflightConfigDir()
	if got == "" {
		t.Fatal("expected non-empty config dir, got empty")
	}
	if filepath.Base(got) != ".preflight" {
		t.Errorf("preflightConfigDir() = %q, expected suffix '/.preflight'", got)
	}
}
