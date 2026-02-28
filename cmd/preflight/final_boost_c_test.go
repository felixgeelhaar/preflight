package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// outdated.go tests
// ===========================================================================

func TestBoostC_ParseUpdateType_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected security.UpdateType
	}{
		{"major", "major", security.UpdateMajor},
		{"minor", "minor", security.UpdateMinor},
		{"patch", "patch", security.UpdatePatch},
		{"Major_upper", "Major", security.UpdateMajor}, // case insensitive
		{"unknown", "foo", security.UpdateMinor},
		{"empty", "", security.UpdateMinor},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseUpdateType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBoostC_ShouldFailOutdated_WithMajor(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", UpdateType: security.UpdateMajor},
		},
	}

	assert.True(t, shouldFailOutdated(result, security.UpdateMajor))
	assert.True(t, shouldFailOutdated(result, security.UpdateMinor))
}

func TestBoostC_ShouldFailOutdated_OnlyPatch(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "jq", UpdateType: security.UpdatePatch},
		},
	}

	// patch should not fail when threshold is major
	assert.False(t, shouldFailOutdated(result, security.UpdateMajor))
}

func TestBoostC_ShouldFailOutdated_Empty(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{Checker: "brew"}
	assert.False(t, shouldFailOutdated(result, security.UpdateMinor))
}

func TestBoostC_ToOutdatedPackagesJSON(t *testing.T) {
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
			Name:           "node",
			CurrentVersion: "18.0.0",
			LatestVersion:  "20.0.0",
			UpdateType:     security.UpdateMajor,
			Provider:       "brew",
			Pinned:         true,
		},
	}

	result := toOutdatedPackagesJSON(packages)
	require.Len(t, result, 2)

	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "1.21.0", result[0].CurrentVersion)
	assert.Equal(t, "1.22.0", result[0].LatestVersion)
	assert.Equal(t, "minor", result[0].UpdateType)
	assert.False(t, result[0].Pinned)

	assert.Equal(t, "node", result[1].Name)
	assert.True(t, result[1].Pinned)
}

func TestBoostC_FormatUpdateType_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    security.UpdateType
		contains string
	}{
		{"major", security.UpdateMajor, "MAJOR"},
		{"minor", security.UpdateMinor, "MINOR"},
		{"patch", security.UpdatePatch, "PATCH"},
		{"unknown", security.UpdateType("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatUpdateType(tc.input)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestBoostC_OutputOutdatedJSON_WithResult(t *testing.T) {
	t.Parallel()

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

	out := captureStdout(t, func() {
		outputOutdatedJSON(result, nil)
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "brew", parsed["checker"])
	assert.NotNil(t, parsed["packages"])
	assert.NotNil(t, parsed["summary"])
}

func TestBoostC_OutputOutdatedJSON_WithError(t *testing.T) {
	t.Parallel()

	out := captureStdout(t, func() {
		outputOutdatedJSON(nil, fmt.Errorf("check failed"))
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "check failed", parsed["error"])
}

func TestBoostC_OutputOutdatedText_NoPackages(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Checker:  "brew",
		Packages: security.OutdatedPackages{},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, out, "All packages are up to date")
}

func TestBoostC_OutputOutdatedText_WithPackages(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{
				Name:           "go",
				CurrentVersion: "1.21.0",
				LatestVersion:  "2.0.0",
				UpdateType:     security.UpdateMajor,
				Provider:       "brew",
			},
			{
				Name:           "jq",
				CurrentVersion: "1.6",
				LatestVersion:  "1.7",
				UpdateType:     security.UpdateMinor,
				Provider:       "brew",
			},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, out, "Outdated Packages Check (brew)")
	assert.Contains(t, out, "2 packages have updates available")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "jq")
	assert.Contains(t, out, "Recommendations")
}

func TestBoostC_OutputOutdatedText_Quiet(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", UpdateType: security.UpdateMinor, CurrentVersion: "1.21", LatestVersion: "1.22", Provider: "brew"},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, true)
	})

	assert.Contains(t, out, "1 packages have updates available")
	// In quiet mode, the table is not printed
	assert.NotContains(t, out, "PACKAGE")
}

func TestBoostC_PrintUpdateTypeBar(t *testing.T) {
	t.Parallel()

	summary := security.OutdatedSummary{
		Total: 3,
		Major: 1,
		Minor: 1,
		Patch: 1,
	}

	out := captureStdout(t, func() {
		printUpdateTypeBar(summary)
	})

	assert.Contains(t, out, "MAJOR: 1")
	assert.Contains(t, out, "MINOR: 1")
	assert.Contains(t, out, "PATCH: 1")
}

func TestBoostC_PrintUpdateTypeBar_OnlyMajor(t *testing.T) {
	t.Parallel()

	summary := security.OutdatedSummary{
		Total: 2,
		Major: 2,
	}

	out := captureStdout(t, func() {
		printUpdateTypeBar(summary)
	})

	assert.Contains(t, out, "MAJOR: 2")
	assert.NotContains(t, out, "MINOR")
	assert.NotContains(t, out, "PATCH")
}

func TestBoostC_PrintOutdatedTable(t *testing.T) {
	t.Parallel()

	packages := security.OutdatedPackages{
		{Name: "go", CurrentVersion: "1.21.0", LatestVersion: "1.22.0", UpdateType: security.UpdateMinor, Provider: "brew"},
	}

	out := captureStdout(t, func() {
		printOutdatedTable(packages)
	})

	assert.Contains(t, out, "PACKAGE")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "1.21.0")
	assert.Contains(t, out, "1.22.0")
}

func TestBoostC_OutputUpgradeJSON_WithResult(t *testing.T) {
	t.Parallel()

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update"},
		},
		Failed: []security.FailedPackage{
			{Name: "rust", Error: "download failed"},
		},
		DryRun: true,
	}

	out := captureStdout(t, func() {
		outputUpgradeJSON(result, nil)
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.True(t, parsed["dry_run"].(bool))
	assert.Len(t, parsed["upgraded"].([]interface{}), 1)
	assert.Len(t, parsed["skipped"].([]interface{}), 1)
	assert.Len(t, parsed["failed"].([]interface{}), 1)
}

func TestBoostC_OutputUpgradeJSON_WithError(t *testing.T) {
	t.Parallel()

	result := &security.UpgradeResult{DryRun: false}

	out := captureStdout(t, func() {
		outputUpgradeJSON(result, fmt.Errorf("upgrade failed"))
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "upgrade failed", parsed["error"])
}

func TestBoostC_OutputUpgradeText_DryRun(t *testing.T) {
	t.Parallel()

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0"},
		},
		DryRun: true,
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "would upgrade")
	assert.Contains(t, out, "Would upgrade 1 package(s)")
}

func TestBoostC_OutputUpgradeText_Actual(t *testing.T) {
	t.Parallel()

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0"},
		},
		DryRun: false,
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "Upgraded 1 package(s)")
}

func TestBoostC_OutputUpgradeText_WithSkippedAndFailed(t *testing.T) {
	t.Parallel()

	// Save and restore package-level var
	savedMajor := outdatedMajor
	defer func() { outdatedMajor = savedMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.21.0", ToVersion: "1.22.0"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update"},
		},
		Failed: []security.FailedPackage{
			{Name: "rust", Error: "download failed"},
		},
		DryRun: false,
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "1 skipped (major updates)")
	assert.Contains(t, out, "1 failed")
	assert.Contains(t, out, "Use --major to include major version upgrades")
}

// ===========================================================================
// deprecated.go tests
// ===========================================================================

func TestBoostC_FormatDeprecationStatus_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reason   security.DeprecationReason
		contains string
	}{
		{"disabled", security.ReasonDisabled, "DISABLED"},
		{"deprecated", security.ReasonDeprecated, "DEPRECATED"},
		{"eol", security.ReasonEOL, "EOL"},
		{"unmaintained", security.ReasonUnmaintained, "UNMAINTAINED"},
		{"unknown", security.DeprecationReason("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatDeprecationStatus(tc.reason)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestBoostC_ToDeprecatedPackagesJSON(t *testing.T) {
	t.Parallel()

	now := time.Now()
	packages := security.DeprecatedPackages{
		{
			Name:        "old-tool",
			Version:     "1.0.0",
			Provider:    "brew",
			Reason:      security.ReasonDeprecated,
			Date:        &now,
			Alternative: "new-tool",
			Message:     "Use new-tool instead",
		},
		{
			Name:     "disabled-pkg",
			Provider: "brew",
			Reason:   security.ReasonDisabled,
			Message:  "Disabled upstream",
		},
	}

	result := toDeprecatedPackagesJSON(packages)
	require.Len(t, result, 2)

	assert.Equal(t, "old-tool", result[0].Name)
	assert.Equal(t, "1.0.0", result[0].Version)
	assert.Equal(t, "new-tool", result[0].Alternative)
	assert.NotEmpty(t, result[0].Date)

	assert.Equal(t, "disabled-pkg", result[1].Name)
	assert.Empty(t, result[1].Date) // nil date -> empty string
}

func TestBoostC_OutputDeprecatedJSON_WithResult(t *testing.T) {
	t.Parallel()

	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "old-tool", Provider: "brew", Reason: security.ReasonDeprecated, Message: "deprecated upstream"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedJSON(result, nil)
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "brew", parsed["checker"])
	assert.NotNil(t, parsed["packages"])
	assert.NotNil(t, parsed["summary"])
}

func TestBoostC_OutputDeprecatedJSON_WithError(t *testing.T) {
	t.Parallel()

	out := captureStdout(t, func() {
		outputDeprecatedJSON(nil, fmt.Errorf("no checkers"))
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "no checkers", parsed["error"])
}

func TestBoostC_OutputDeprecatedText_NoPackages(t *testing.T) {
	t.Parallel()

	result := &security.DeprecatedResult{
		Checker:  "brew",
		Packages: security.DeprecatedPackages{},
	}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, out, "No deprecated packages found")
}

func TestBoostC_OutputDeprecatedText_WithPackages(t *testing.T) {
	t.Parallel()

	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "old-tool", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated, Message: "deprecated upstream"},
			{Name: "dead-tool", Provider: "brew", Reason: security.ReasonDisabled, Message: "disabled"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, out, "Deprecated Packages Check (brew)")
	assert.Contains(t, out, "2 packages require attention")
	assert.Contains(t, out, "old-tool")
	assert.Contains(t, out, "Recommendations")
	assert.Contains(t, out, "DISABLED")
	assert.Contains(t, out, "DEPRECATED")
}

func TestBoostC_OutputDeprecatedText_Quiet(t *testing.T) {
	t.Parallel()

	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "old-tool", Provider: "brew", Reason: security.ReasonDeprecated, Message: "deprecated upstream"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, true)
	})

	assert.Contains(t, out, "1 packages require attention")
	// In quiet mode, the table should not be printed (no "STATUS" header)
	assert.NotContains(t, out, "STATUS")
}

func TestBoostC_PrintDeprecationSummaryBar(t *testing.T) {
	t.Parallel()

	summary := security.DeprecatedSummary{
		Total:        4,
		Disabled:     1,
		Deprecated:   1,
		EOL:          1,
		Unmaintained: 1,
	}

	out := captureStdout(t, func() {
		printDeprecationSummaryBar(summary)
	})

	assert.Contains(t, out, "DISABLED: 1")
	assert.Contains(t, out, "DEPRECATED: 1")
	assert.Contains(t, out, "EOL: 1")
	assert.Contains(t, out, "UNMAINTAINED: 1")
}

func TestBoostC_PrintDeprecatedTable(t *testing.T) {
	t.Parallel()

	packages := security.DeprecatedPackages{
		{Name: "old-tool", Version: "1.0", Reason: security.ReasonDeprecated, Message: "deprecated"},
		{Name: "no-version", Reason: security.ReasonDisabled, Message: ""},
		{Name: "long-msg", Version: "2.0", Reason: security.ReasonEOL, Message: "This is a very long deprecation message that exceeds the fifty character limit and should be truncated"},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(packages)
	})

	assert.Contains(t, out, "old-tool")
	assert.Contains(t, out, "no-version")
	assert.Contains(t, out, "-")   // empty version or message replaced with "-"
	assert.Contains(t, out, "...") // truncated long message
}

// ===========================================================================
// cleanup.go tests
// ===========================================================================

func TestBoostC_OutputCleanupJSON_WithError(t *testing.T) {
	t.Parallel()

	out := captureStdout(t, func() {
		outputCleanupJSON(nil, nil, fmt.Errorf("brew not available"))
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "brew not available", parsed.Error)
}

func TestBoostC_OutputCleanupJSON_WithResult(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Remove older version",
				Remove:         []string{"go@1.24"},
			},
		},
	}

	out := captureStdout(t, func() {
		outputCleanupJSON(result, nil, nil)
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.Len(t, parsed.Redundancies, 1)
	assert.Equal(t, security.RedundancyDuplicate, parsed.Redundancies[0].Type)
	assert.NotNil(t, parsed.Summary)
}

func TestBoostC_OutputCleanupJSON_WithCleanup(t *testing.T) {
	t.Parallel()

	cleanup := &security.CleanupResult{
		Removed: []string{"go@1.24", "node@18"},
		DryRun:  true,
	}

	out := captureStdout(t, func() {
		outputCleanupJSON(nil, cleanup, nil)
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.NotNil(t, parsed.Cleanup)
	assert.True(t, parsed.Cleanup.DryRun)
	assert.Len(t, parsed.Cleanup.Removed, 2)
}

func TestBoostC_OutputCleanupText_NoRedundancies(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker:      "brew",
		Redundancies: security.Redundancies{},
	}

	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, out, "No redundancies detected")
}

func TestBoostC_OutputCleanupText_WithDuplicates(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Remove older version",
				Remove:         []string{"go@1.24"},
			},
		},
	}

	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, out, "Redundancy Analysis (brew)")
	assert.Contains(t, out, "1 redundancies found")
	assert.Contains(t, out, "Version Duplicates")
	assert.Contains(t, out, "go + go@1.24")
	assert.Contains(t, out, "Remove: go@1.24")
	assert.Contains(t, out, "Actions:")
}

func TestBoostC_OutputCleanupText_Quiet(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Remove older version",
				Remove:         []string{"go@1.24"},
			},
		},
	}

	out := captureStdout(t, func() {
		outputCleanupText(result, true)
	})

	assert.Contains(t, out, "1 redundancies found")
	assert.Contains(t, out, "preflight cleanup --all")
	// quiet mode should NOT print detailed tables
	assert.NotContains(t, out, "Version Duplicates")
}

func TestBoostC_OutputCleanupText_WithOverlaps(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyOverlap,
				Category:       "version_manager",
				Packages:       []string{"rtx", "asdf"},
				Recommendation: "Choose one version manager",
				Keep:           []string{"rtx"},
				Remove:         []string{"asdf"},
			},
		},
	}

	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, out, "Overlapping Tools")
	assert.Contains(t, out, "Version Manager")
	assert.Contains(t, out, "Keep: rtx")
}

func TestBoostC_OutputCleanupText_WithOrphans(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:     security.RedundancyOrphan,
				Packages: []string{"libfoo", "libbar"},
				Action:   "brew autoremove",
			},
		},
	}

	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, out, "Orphaned Dependencies")
	assert.Contains(t, out, "libfoo, libbar")
	assert.Contains(t, out, "brew autoremove")
	assert.Contains(t, out, "preflight cleanup --autoremove")
}

func TestBoostC_PrintRedundancySummaryBar(t *testing.T) {
	t.Parallel()

	summary := security.RedundancySummary{
		Total:      3,
		Duplicates: 1,
		Overlaps:   1,
		Orphans:    1,
	}

	out := captureStdout(t, func() {
		printRedundancySummaryBar(summary)
	})

	assert.Contains(t, out, "DUPLICATES: 1")
	assert.Contains(t, out, "OVERLAPS: 1")
	assert.Contains(t, out, "ORPHANS: 1")
}

func TestBoostC_PrintRedundancySummaryBar_PartialCounts(t *testing.T) {
	t.Parallel()

	summary := security.RedundancySummary{
		Total:      2,
		Duplicates: 2,
	}

	out := captureStdout(t, func() {
		printRedundancySummaryBar(summary)
	})

	assert.Contains(t, out, "DUPLICATES: 2")
	assert.NotContains(t, out, "OVERLAPS")
	assert.NotContains(t, out, "ORPHANS")
}

func TestBoostC_PrintRedundancyTable(t *testing.T) {
	t.Parallel()

	redundancies := security.Redundancies{
		{
			Packages:       []string{"go", "go@1.24"},
			Recommendation: "Remove older version",
			Remove:         []string{"go@1.24"},
		},
	}

	out := captureStdout(t, func() {
		printRedundancyTable(redundancies)
	})

	assert.Contains(t, out, "go + go@1.24")
	assert.Contains(t, out, "Remove older version")
	assert.Contains(t, out, "Remove: go@1.24")
}

func TestBoostC_PrintOverlapTable(t *testing.T) {
	t.Parallel()

	redundancies := security.Redundancies{
		{
			Category:       "version_manager",
			Packages:       []string{"rtx", "asdf"},
			Recommendation: "Choose one",
			Keep:           []string{"rtx"},
			Remove:         []string{"asdf"},
		},
	}

	out := captureStdout(t, func() {
		printOverlapTable(redundancies)
	})

	assert.Contains(t, out, "Version Manager")
	assert.Contains(t, out, "rtx, asdf")
	assert.Contains(t, out, "Keep: rtx, Remove: asdf")
}

func TestBoostC_FormatCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", "version_manager", "Version Manager"},
		{"single_word", "editor", "Editor"},
		{"multi_words", "package_manager_tool", "Package Manager Tool"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatCategory(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ===========================================================================
// sync_conflicts.go - printConflicts
// ===========================================================================

func TestBoostC_PrintConflicts(t *testing.T) {
	t.Parallel()

	machineA, _ := sync.ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	machineB, _ := sync.ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	vectorA := sync.NewVersionVector().Increment(machineA)
	vectorB := sync.NewVersionVector().Increment(machineB)

	local := sync.NewPackageLockInfo("14.0.0", sync.NewPackageProvenance(machineA, vectorA))
	remote := sync.NewPackageLockInfo("14.1.0", sync.NewPackageProvenance(machineB, vectorB))
	base := sync.NewPackageLockInfo("13.0.0", sync.NewPackageProvenance(machineA, sync.NewVersionVector()))

	conflicts := []sync.LockConflict{
		sync.NewLockConflict("brew:ripgrep", sync.BothModified, local, remote, base),
	}

	out := captureStdout(t, func() {
		printConflicts(conflicts)
	})

	assert.Contains(t, out, "PACKAGE")
	assert.Contains(t, out, "brew:ripgrep")
	assert.Contains(t, out, "14.0.0")
	assert.Contains(t, out, "14.1.0")
}

// ===========================================================================
// compliance.go - collectEvaluatedItems
// ===========================================================================

func TestBoostC_CollectEvaluatedItems_NilResult(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(nil)
	assert.Nil(t, result)
}

func TestBoostC_CollectEvaluatedItems_WithItems(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{
		Info:   []string{"brew:go validated", "brew:jq validated"},
		Errors: []string{"missing required package"},
	})

	assert.Len(t, result, 3)
	assert.Contains(t, result, "brew:go validated")
	assert.Contains(t, result, "missing required package")
}

func TestBoostC_CollectEvaluatedItems_Empty(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{})
	assert.Empty(t, result)
}

// ===========================================================================
// doctor.go - printDoctorQuiet
// ===========================================================================

func TestBoostC_PrintDoctorQuiet_NoIssues(t *testing.T) {
	t.Parallel()

	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{},
	}

	out := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, out, "Doctor Report")
	assert.Contains(t, out, "No issues found")
}

func TestBoostC_PrintDoctorQuiet_WithIssues(t *testing.T) {
	t.Parallel()

	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityError,
				Message:    "package go is missing",
				Provider:   "brew",
				Expected:   "installed",
				Actual:     "not found",
				FixCommand: "brew install go",
				Fixable:    true,
			},
			{
				Severity: app.SeverityWarning,
				Message:  "drift detected in .gitconfig",
			},
		},
	}

	out := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, out, "Found 2 issue(s)")
	assert.Contains(t, out, "package go is missing")
	assert.Contains(t, out, "Provider: brew")
	assert.Contains(t, out, "Expected: installed")
	assert.Contains(t, out, "Actual: not found")
	assert.Contains(t, out, "Fix: brew install go")
	assert.Contains(t, out, "drift detected")
	assert.Contains(t, out, "can be auto-fixed")
}

func TestBoostC_PrintDoctorQuiet_WithPatches(t *testing.T) {
	t.Parallel()

	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{Severity: app.SeverityWarning, Message: "test issue"},
		},
		SuggestedPatches: []app.ConfigPatch{
			{LayerPath: "base.yaml", YAMLPath: "shell.env.EDITOR", NewValue: "nvim"},
		},
	}

	out := captureStdout(t, func() {
		printDoctorQuiet(report)
	})

	assert.Contains(t, out, "config patches suggested")
	assert.Contains(t, out, "preflight doctor --update-config")
}

// ===========================================================================
// compliance.go - outputComplianceJSON
// ===========================================================================

func TestBoostC_OutputComplianceJSON(t *testing.T) {
	t.Parallel()

	// outputComplianceJSON calls report.ToJSON() then prints.
	// If report.ToJSON() returns valid JSON, it should print it.
	// We test by providing a simple compliant report.
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     3,
			PassedChecks:    3,
			ComplianceScore: 100,
		},
	}

	out := captureStdout(t, func() {
		outputComplianceJSON(report)
	})

	assert.Contains(t, out, "test-policy")
	assert.Contains(t, out, "compliant")
}
