package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/stretchr/testify/assert"
)

// Note: TestDoctorCmd_Exists and TestDoctorCmd_HasFlags are in helpers_test.go

func TestDoctorCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"fix default", "fix", "false"},
		{"verbose default", "verbose", "false"},
		{"update-config default", "update-config", "false"},
		{"dry-run default", "dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := doctorCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestDoctorCmd_VerboseShorthand(t *testing.T) {
	t.Parallel()

	f := doctorCmd.Flags().Lookup("verbose")
	assert.NotNil(t, f)
	assert.Equal(t, "v", f.Shorthand)
}

func TestDoctorCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "doctor" {
			found = true
			break
		}
	}
	assert.True(t, found, "doctor should be a subcommand of root")
}

func TestPrintDoctorQuiet_NoIssues(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	report := &app.DoctorReport{}

	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, output, "Doctor Report")
	assert.Contains(t, output, "No issues found")
	assert.Contains(t, output, "Your system is in sync")
}

func TestPrintDoctorQuiet_WithIssues(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Provider:   "brew",
				Severity:   app.SeverityError,
				Message:    "package ripgrep not installed",
				Expected:   "14.1.0",
				Actual:     "not found",
				FixCommand: "brew install ripgrep",
			},
			{
				Provider: "git",
				Severity: app.SeverityWarning,
				Message:  "git user.email not configured",
			},
			{
				Provider: "shell",
				Severity: app.SeverityInfo,
				Message:  "optional plugin not loaded",
			},
		},
	}

	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, output, "Doctor Report")
	assert.Contains(t, output, "Found 3 issue(s)")

	// Verify error severity uses cross mark
	assert.Contains(t, output, "package ripgrep not installed")

	// Verify provider is shown
	assert.Contains(t, output, "Provider: brew")
	assert.Contains(t, output, "Provider: git")

	// Verify expected/actual shown for the error issue
	assert.Contains(t, output, "Expected: 14.1.0")
	assert.Contains(t, output, "Actual: not found")

	// Verify fix command shown
	assert.Contains(t, output, "Fix: brew install ripgrep")

	// Verify warning issue appears
	assert.Contains(t, output, "git user.email not configured")
}

func TestPrintDoctorQuiet_FixableIssues(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Provider:   "brew",
				Severity:   app.SeverityError,
				Message:    "package missing",
				Fixable:    true,
				FixCommand: "brew install pkg",
			},
			{
				Provider: "git",
				Severity: app.SeverityWarning,
				Message:  "config drift",
				Fixable:  true,
			},
			{
				Provider: "shell",
				Severity: app.SeverityError,
				Message:  "manual fix required",
				Fixable:  false,
			},
		},
	}

	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, output, "2 issue(s) can be auto-fixed")
	assert.Contains(t, output, "preflight doctor --fix")
}

func TestPrintDoctorQuiet_WithPatches(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Provider: "brew",
				Severity: app.SeverityWarning,
				Message:  "drift detected",
			},
		},
		SuggestedPatches: []app.ConfigPatch{
			app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpAdd, nil, "ripgrep", "drift"),
			app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpAdd, nil, "fd", "drift"),
		},
	}

	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, output, "2 config patches suggested")
	assert.Contains(t, output, "preflight doctor --update-config")
}
