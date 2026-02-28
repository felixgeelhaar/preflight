package main

import (
	"encoding/json"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutdatedCommand_Flags(t *testing.T) {
	t.Parallel()

	// Verify command exists and has expected flags
	assert.NotNil(t, outdatedCmd)
	assert.Equal(t, "outdated [packages...]", outdatedCmd.Use)

	// Check flags exist
	flags := outdatedCmd.Flags()
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("fail-on"))
	assert.NotNil(t, flags.Lookup("ignore"))
	assert.NotNil(t, flags.Lookup("json"))
	assert.NotNil(t, flags.Lookup("quiet"))
}

func TestParseUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected security.UpdateType
	}{
		{"major", security.UpdateMajor},
		{"MAJOR", security.UpdateMajor},
		{"Minor", security.UpdateMinor},
		{"minor", security.UpdateMinor},
		{"patch", security.UpdatePatch},
		{"PATCH", security.UpdatePatch},
		{"invalid", security.UpdateMinor}, // Default to minor
		{"", security.UpdateMinor},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := parseUpdateType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldFailOutdated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		packages   security.OutdatedPackages
		failOn     security.UpdateType
		shouldFail bool
	}{
		{
			name:       "no packages",
			packages:   security.OutdatedPackages{},
			failOn:     security.UpdateMajor,
			shouldFail: false,
		},
		{
			name: "major when fail-on major",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMajor},
			},
			failOn:     security.UpdateMajor,
			shouldFail: true,
		},
		{
			name: "minor when fail-on major",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMinor},
			},
			failOn:     security.UpdateMajor,
			shouldFail: false,
		},
		{
			name: "minor when fail-on minor",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMinor},
			},
			failOn:     security.UpdateMinor,
			shouldFail: true,
		},
		{
			name: "patch when fail-on minor",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdatePatch},
			},
			failOn:     security.UpdateMinor,
			shouldFail: false,
		},
		{
			name: "patch when fail-on patch",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdatePatch},
			},
			failOn:     security.UpdatePatch,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &security.OutdatedResult{
				Packages: tt.packages,
			}

			got := shouldFailOutdated(result, tt.failOn)
			assert.Equal(t, tt.shouldFail, got)
		})
	}
}

func TestToOutdatedPackagesJSON(t *testing.T) {
	t.Parallel()

	packages := security.OutdatedPackages{
		{
			Name:           "go",
			CurrentVersion: "1.21.0",
			LatestVersion:  "1.22.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
			Pinned:         false,
		},
		{
			Name:           "python",
			CurrentVersion: "3.11.0",
			LatestVersion:  "3.12.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
			Pinned:         true,
		},
	}

	result := toOutdatedPackagesJSON(packages)

	assert.Len(t, result, 2)

	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "1.21.0", result[0].CurrentVersion)
	assert.Equal(t, "1.22.0", result[0].LatestVersion)
	assert.Equal(t, "minor", result[0].UpdateType)
	assert.Equal(t, "brew", result[0].Provider)
	assert.False(t, result[0].Pinned)

	assert.Equal(t, "python", result[1].Name)
	assert.True(t, result[1].Pinned)
}

func TestFormatUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		updateType security.UpdateType
		contains   string
	}{
		{security.UpdateMajor, "MAJOR"},
		{security.UpdateMinor, "MINOR"},
		{security.UpdatePatch, "PATCH"},
		{security.UpdateUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.updateType), func(t *testing.T) {
			t.Parallel()

			result := formatUpdateType(tt.updateType)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestOutputOutdatedJSON_WithResults(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{
				Name:           "go",
				CurrentVersion: "1.21.0",
				LatestVersion:  "1.22.0",
				UpdateType:     security.UpdateMinor,
				Provider:       "brew",
			},
			{
				Name:           "node",
				CurrentVersion: "18.0.0",
				LatestVersion:  "20.0.0",
				UpdateType:     security.UpdateMajor,
				Provider:       "brew",
				Pinned:         true,
			},
		},
	}

	output := captureStdout(t, func() {
		outputOutdatedJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "brew", parsed["checker"])
	assert.Empty(t, parsed["error"])

	packages, ok := parsed["packages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, packages, 2)

	pkg0 := packages[0].(map[string]interface{})
	assert.Equal(t, "go", pkg0["name"])
	assert.Equal(t, "1.21.0", pkg0["current_version"])
	assert.Equal(t, "1.22.0", pkg0["latest_version"])
	assert.Equal(t, "minor", pkg0["update_type"])
	assert.Equal(t, "brew", pkg0["provider"])

	summary, ok := parsed["summary"].(map[string]interface{})
	require.True(t, ok)
	assert.InDelta(t, float64(2), summary["total"], 0)
	assert.InDelta(t, float64(1), summary["major"], 0)
	assert.InDelta(t, float64(1), summary["minor"], 0)
}

func TestOutputOutdatedJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputOutdatedJSON(nil, assert.AnError)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, assert.AnError.Error(), parsed["error"])
	assert.Nil(t, parsed["checker"])
	assert.Nil(t, parsed["packages"])
	assert.Nil(t, parsed["summary"])
}

func TestOutputOutdatedText_NoPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker:  "brew",
		Packages: security.OutdatedPackages{},
	}

	output := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, output, "up to date")
	assert.Contains(t, output, "brew")
}

func TestOutputOutdatedText_WithPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{
				Name:           "go",
				CurrentVersion: "1.21.0",
				LatestVersion:  "1.22.0",
				UpdateType:     security.UpdateMinor,
				Provider:       "brew",
			},
			{
				Name:           "node",
				CurrentVersion: "18.0.0",
				LatestVersion:  "20.0.0",
				UpdateType:     security.UpdateMajor,
				Provider:       "brew",
			},
		},
	}

	output := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, output, "Outdated Packages Check (brew)")
	assert.Contains(t, output, "2 packages have updates available")
	// Table headers
	assert.Contains(t, output, "PACKAGE")
	assert.Contains(t, output, "CURRENT")
	assert.Contains(t, output, "LATEST")
	assert.Contains(t, output, "PROVIDER")
	// Package data
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "1.21.0")
	assert.Contains(t, output, "1.22.0")
	assert.Contains(t, output, "node")
	// Recommendations for major updates
	assert.Contains(t, output, "Recommendations")
	assert.Contains(t, output, "MAJOR")
}

func TestOutputOutdatedText_Quiet(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{
				Name:           "go",
				CurrentVersion: "1.21.0",
				LatestVersion:  "1.22.0",
				UpdateType:     security.UpdateMinor,
				Provider:       "brew",
			},
		},
	}

	output := captureStdout(t, func() {
		outputOutdatedText(result, true)
	})

	assert.Contains(t, output, "1 packages have updates available")
	// In quiet mode, the table should not be printed
	assert.NotContains(t, output, "PACKAGE")
	assert.NotContains(t, output, "CURRENT")
}

func TestPrintUpdateTypeBar(t *testing.T) {
	summary := security.OutdatedSummary{
		Total:  5,
		Major:  2,
		Minor:  2,
		Patch:  1,
		Pinned: 0,
	}

	output := captureStdout(t, func() {
		printUpdateTypeBar(summary)
	})

	assert.Contains(t, output, "MAJOR: 2")
	assert.Contains(t, output, "MINOR: 2")
	assert.Contains(t, output, "PATCH: 1")
}

func TestPrintOutdatedTable(t *testing.T) {
	packages := security.OutdatedPackages{
		{
			Name:           "go",
			CurrentVersion: "1.21.0",
			LatestVersion:  "1.22.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
		},
		{
			Name:           "python",
			CurrentVersion: "3.11.0",
			LatestVersion:  "3.12.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
		},
	}

	output := captureStdout(t, func() {
		printOutdatedTable(packages)
	})

	// Verify column headers
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "PACKAGE")
	assert.Contains(t, output, "CURRENT")
	assert.Contains(t, output, "LATEST")
	assert.Contains(t, output, "PROVIDER")
	// Verify data rows
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "1.21.0")
	assert.Contains(t, output, "1.22.0")
	assert.Contains(t, output, "python")
	assert.Contains(t, output, "3.11.0")
	assert.Contains(t, output, "3.12.0")
	assert.Contains(t, output, "brew")
}

func TestOutputUpgradeJSON_Success(t *testing.T) {
	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update requires --major flag", UpdateType: security.UpdateMajor},
		},
		Failed: []security.FailedPackage{
			{Name: "ruby", Error: "permission denied"},
		},
		DryRun: false,
	}

	output := captureStdout(t, func() {
		outputUpgradeJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.False(t, parsed["dry_run"].(bool))
	assert.Empty(t, parsed["error"])

	upgraded := parsed["upgraded"].([]interface{})
	assert.Len(t, upgraded, 1)
	up0 := upgraded[0].(map[string]interface{})
	assert.Equal(t, "go", up0["name"])
	assert.Equal(t, "1.21.0", up0["from_version"])
	assert.Equal(t, "1.22.0", up0["to_version"])

	skipped := parsed["skipped"].([]interface{})
	assert.Len(t, skipped, 1)

	failed := parsed["failed"].([]interface{})
	assert.Len(t, failed, 1)
}

func TestOutputUpgradeJSON_Error(t *testing.T) {
	output := captureStdout(t, func() {
		outputUpgradeJSON(&security.UpgradeResult{DryRun: false}, assert.AnError)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, assert.AnError.Error(), parsed["error"])
}

func TestOutputUpgradeText_Upgraded(t *testing.T) {
	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0", Provider: "brew"},
			{Name: "python", FromVersion: "3.11.0", ToVersion: "3.12.0", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{},
		Failed:  []security.FailedPackage{},
		DryRun:  false,
	}

	output := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, output, "go")
	assert.Contains(t, output, "1.21.0")
	assert.Contains(t, output, "1.22.0")
	assert.Contains(t, output, "python")
	assert.Contains(t, output, "Upgraded 2 package(s)")
	// Should not contain dry-run language
	assert.NotContains(t, output, "Would upgrade")
}

func TestOutputUpgradeText_DryRun(t *testing.T) {
	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{},
		Failed:  []security.FailedPackage{},
		DryRun:  true,
	}

	output := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, output, "would upgrade")
	assert.Contains(t, output, "Would upgrade 1 package(s)")
}

func TestOutputUpgradeText_WithSkippedAndFailed(t *testing.T) {
	// Save and restore the package-level flag to avoid side effects
	prevMajor := outdatedMajor
	outdatedMajor = false
	defer func() { outdatedMajor = prevMajor }()

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update requires --major flag", UpdateType: security.UpdateMajor},
		},
		Failed: []security.FailedPackage{
			{Name: "ruby", Error: "permission denied"},
		},
		DryRun: false,
	}

	output := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	// Upgraded
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "1.21.0")

	// Skipped
	assert.Contains(t, output, "node")
	assert.Contains(t, output, "skipped")

	// Failed
	assert.Contains(t, output, "ruby")
	assert.Contains(t, output, "permission denied")

	// Summary
	assert.Contains(t, output, "Upgraded 1 package(s)")
	assert.Contains(t, output, "1 skipped")
	assert.Contains(t, output, "1 failed")

	// Hint for major upgrades
	assert.Contains(t, output, "--major")
}
