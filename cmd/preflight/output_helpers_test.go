package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// cleanup.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputCleanupJSON_ErrorOnly(t *testing.T) {
	out := captureStdout(t, func() {
		outputCleanupJSON(nil, nil, fmt.Errorf("brew not available"))
	})
	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "brew not available", parsed.Error)
	assert.Nil(t, parsed.Redundancies)
	assert.Nil(t, parsed.Cleanup)
}

func TestOutputHelpers_OutputCleanupJSON_ResultOnly(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Keep go",
				Remove:         []string{"go@1.24"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupJSON(result, nil, nil)
	})
	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed.Error)
	assert.Len(t, parsed.Redundancies, 1)
	require.NotNil(t, parsed.Summary)
	assert.Equal(t, 1, parsed.Summary.Total)
	assert.Equal(t, 1, parsed.Summary.Duplicates)
}

func TestOutputHelpers_OutputCleanupJSON_CleanupOnly(t *testing.T) {
	cleanup := &security.CleanupResult{
		Removed: []string{"go@1.24"},
		DryRun:  true,
	}
	out := captureStdout(t, func() {
		outputCleanupJSON(nil, cleanup, nil)
	})
	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed.Error)
	require.NotNil(t, parsed.Cleanup)
	assert.True(t, parsed.Cleanup.DryRun)
	assert.Equal(t, []string{"go@1.24"}, parsed.Cleanup.Removed)
}

func TestOutputHelpers_OutputCleanupJSON_ResultAndCleanup(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Type: security.RedundancyDuplicate, Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}},
		},
	}
	cleanup := &security.CleanupResult{Removed: []string{"go@1.24"}, DryRun: false}
	out := captureStdout(t, func() {
		outputCleanupJSON(result, cleanup, nil)
	})
	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed.Error)
	assert.Len(t, parsed.Redundancies, 1)
	require.NotNil(t, parsed.Cleanup)
	assert.False(t, parsed.Cleanup.DryRun)
}

func TestOutputHelpers_OutputCleanupText_NoRedundancies(t *testing.T) {
	result := &security.RedundancyResult{
		Checker:      "brew",
		Redundancies: security.Redundancies{},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, out, "Redundancy Analysis (brew)")
	assert.Contains(t, out, "No redundancies detected")
}

func TestOutputHelpers_OutputCleanupText_WithRedundancies(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Keep go (tracks latest)",
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
	assert.Contains(t, out, "preflight cleanup --remove")
}

func TestOutputHelpers_OutputCleanupText_Quiet(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Keep go",
				Remove:         []string{"go@1.24"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, true)
	})
	assert.Contains(t, out, "1 redundancies found")
	assert.Contains(t, out, "Run 'preflight cleanup --all'")
	// In quiet mode, we do NOT print the table details
	assert.NotContains(t, out, "Version Duplicates")
}

func TestOutputHelpers_PrintRedundancySummaryBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		summary security.RedundancySummary
		expect  []string
	}{
		{
			name:    "all_types",
			summary: security.RedundancySummary{Duplicates: 2, Overlaps: 1, Orphans: 3},
			expect:  []string{"DUPLICATES: 2", "OVERLAPS: 1", "ORPHANS: 3"},
		},
		{
			name:    "duplicates_only",
			summary: security.RedundancySummary{Duplicates: 5},
			expect:  []string{"DUPLICATES: 5"},
		},
		{
			name:    "zeros",
			summary: security.RedundancySummary{},
			expect:  []string{}, // nothing printed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := captureStdout(t, func() {
				printRedundancySummaryBar(tt.summary)
			})
			for _, e := range tt.expect {
				assert.Contains(t, out, e)
			}
		})
	}
}

func TestOutputHelpers_PrintRedundancyTable(t *testing.T) {
	t.Parallel()
	redundancies := security.Redundancies{
		{
			Packages:       []string{"go", "go@1.24"},
			Recommendation: "Keep go",
			Remove:         []string{"go@1.24"},
		},
	}
	out := captureStdout(t, func() {
		printRedundancyTable(redundancies)
	})
	assert.Contains(t, out, "go + go@1.24")
	assert.Contains(t, out, "Keep go")
	assert.Contains(t, out, "Remove: go@1.24")
}

func TestOutputHelpers_PrintOverlapTable(t *testing.T) {
	t.Parallel()
	redundancies := security.Redundancies{
		{
			Category:       "node_package_managers",
			Packages:       []string{"npm", "yarn"},
			Recommendation: "Node.js package managers - consider keeping only one",
			Keep:           []string{"npm"},
			Remove:         []string{"yarn"},
		},
	}
	out := captureStdout(t, func() {
		printOverlapTable(redundancies)
	})
	assert.Contains(t, out, "Node Package Managers")
	assert.Contains(t, out, "npm, yarn")
	assert.Contains(t, out, "Keep: npm, Remove: yarn")
}

func TestOutputHelpers_FormatCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  string
		expect string
	}{
		{"version_duplicate", "Version Duplicate"},
		{"tool_overlap", "Tool Overlap"},
		{"single", "Single"},
		{"", ""},
		{"node_package_managers", "Node Package Managers"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, formatCategory(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// deprecated.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputDeprecatedJSON_Error(t *testing.T) {
	out := captureStdout(t, func() {
		outputDeprecatedJSON(nil, fmt.Errorf("no checkers available"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "no checkers available", parsed["error"])
}

func TestOutputHelpers_OutputDeprecatedJSON_WithResult(t *testing.T) {
	now := time.Now()
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "oldpkg", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated, Date: &now, Alternative: "newpkg"},
		},
	}
	out := captureStdout(t, func() {
		outputDeprecatedJSON(result, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed["error"])
	assert.Equal(t, "brew", parsed["checker"])
	packages := parsed["packages"].([]interface{})
	assert.Len(t, packages, 1)
	pkg := packages[0].(map[string]interface{})
	assert.Equal(t, "oldpkg", pkg["name"])
	assert.Equal(t, "newpkg", pkg["alternative"])
}

func TestOutputHelpers_OutputDeprecatedText_NoPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker:  "brew",
		Packages: security.DeprecatedPackages{},
	}
	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})
	assert.Contains(t, out, "Deprecated Packages Check (brew)")
	assert.Contains(t, out, "No deprecated packages found")
}

func TestOutputHelpers_OutputDeprecatedText_WithPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Version: "1.0", Provider: "brew", Reason: security.ReasonDisabled, Message: "removed"},
			{Name: "pkg2", Version: "2.0", Provider: "brew", Reason: security.ReasonDeprecated, Message: "use other"},
		},
	}
	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})
	assert.Contains(t, out, "2 packages require attention")
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "pkg2")
	assert.Contains(t, out, "DISABLED")
}

func TestOutputHelpers_OutputDeprecatedText_Quiet(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated},
		},
	}
	out := captureStdout(t, func() {
		outputDeprecatedText(result, true)
	})
	assert.Contains(t, out, "1 packages require attention")
	// quiet mode skips the table but still shows recommendations
	assert.Contains(t, out, "Recommendations:")
	assert.NotContains(t, out, "STATUS\tPACKAGE")
}

func TestOutputHelpers_ToDeprecatedPackagesJSON(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	packages := security.DeprecatedPackages{
		{Name: "pkg1", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated, Date: &now, Alternative: "newpkg", Message: "old"},
		{Name: "pkg2", Version: "", Provider: "brew", Reason: security.ReasonDisabled, Date: nil, Alternative: ""},
	}
	result := toDeprecatedPackagesJSON(packages)
	assert.Len(t, result, 2)
	assert.Equal(t, "pkg1", result[0].Name)
	assert.Equal(t, "2025-06-15", result[0].Date)
	assert.Equal(t, "newpkg", result[0].Alternative)
	assert.Equal(t, "deprecated", result[0].Reason)
	assert.Equal(t, "pkg2", result[1].Name)
	assert.Empty(t, result[1].Date)
}

func TestOutputHelpers_FormatDeprecationStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		reason security.DeprecationReason
		expect string
	}{
		{security.ReasonDisabled, "DISABLED"},
		{security.ReasonDeprecated, "DEPRECATED"},
		{security.ReasonEOL, "EOL"},
		{security.ReasonUnmaintained, "UNMAINTAINED"},
		{security.DeprecationReason("mystery"), "mystery"},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			got := formatDeprecationStatus(tt.reason)
			assert.Contains(t, got, tt.expect)
		})
	}
}

// ---------------------------------------------------------------------------
// outdated.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputOutdatedJSON_Error(t *testing.T) {
	out := captureStdout(t, func() {
		outputOutdatedJSON(nil, fmt.Errorf("no checkers"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "no checkers", parsed["error"])
}

func TestOutputHelpers_OutputOutdatedJSON_WithResult(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.23", LatestVersion: "1.24", UpdateType: security.UpdateMinor, Provider: "brew"},
		},
	}
	out := captureStdout(t, func() {
		outputOutdatedJSON(result, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "brew", parsed["checker"])
	packages := parsed["packages"].([]interface{})
	assert.Len(t, packages, 1)
}

func TestOutputHelpers_OutputOutdatedText_NoPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker:  "brew",
		Packages: security.OutdatedPackages{},
	}
	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, out, "All packages are up to date")
}

func TestOutputHelpers_OutputOutdatedText_WithPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.23", LatestVersion: "1.24", UpdateType: security.UpdateMinor, Provider: "brew"},
			{Name: "node", CurrentVersion: "18.0", LatestVersion: "20.0", UpdateType: security.UpdateMajor, Provider: "brew"},
			{Name: "curl", CurrentVersion: "8.0.0", LatestVersion: "8.0.1", UpdateType: security.UpdatePatch, Provider: "brew"},
		},
	}
	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, out, "3 packages have updates available")
	assert.Contains(t, out, "MAJOR")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "node")
	assert.Contains(t, out, "Recommendations:")
}

func TestOutputHelpers_OutputOutdatedText_Quiet(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.23", LatestVersion: "1.24", UpdateType: security.UpdateMinor, Provider: "brew"},
		},
	}
	out := captureStdout(t, func() {
		outputOutdatedText(result, true)
	})
	assert.Contains(t, out, "1 packages have updates available")
	// Quiet mode skips the table
	assert.NotContains(t, out, "TYPE\tPACKAGE")
}

func TestOutputHelpers_OutputOutdatedText_WithPinned(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.23", LatestVersion: "1.24", UpdateType: security.UpdateMinor, Provider: "brew", Pinned: true},
		},
	}
	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, out, "Pinned (excluded): 1")
}

func TestOutputHelpers_ToOutdatedPackagesJSON(t *testing.T) {
	t.Parallel()
	packages := security.OutdatedPackages{
		{Name: "go", CurrentVersion: "1.23", LatestVersion: "1.24", UpdateType: security.UpdateMinor, Provider: "brew", Pinned: true},
		{Name: "curl", CurrentVersion: "8.0", LatestVersion: "8.1", UpdateType: security.UpdatePatch, Provider: "brew"},
	}
	result := toOutdatedPackagesJSON(packages)
	assert.Len(t, result, 2)
	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "1.23", result[0].CurrentVersion)
	assert.Equal(t, "minor", result[0].UpdateType)
	assert.True(t, result[0].Pinned)
	assert.Equal(t, "curl", result[1].Name)
	assert.False(t, result[1].Pinned)
}

func TestOutputHelpers_ParseUpdateType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  string
		expect security.UpdateType
	}{
		{"major", security.UpdateMajor},
		{"minor", security.UpdateMinor},
		{"patch", security.UpdatePatch},
		{"unknown", security.UpdateMinor}, // defaults to minor
		{"MAJOR", security.UpdateMajor},
		{"", security.UpdateMinor},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, parseUpdateType(tt.input))
		})
	}
}

func TestOutputHelpers_ShouldFailOutdated(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		packages security.OutdatedPackages
		failOn   security.UpdateType
		expect   bool
	}{
		{
			name:     "major_above_minor_threshold",
			packages: security.OutdatedPackages{{UpdateType: security.UpdateMajor}},
			failOn:   security.UpdateMinor,
			expect:   true,
		},
		{
			name:     "patch_below_minor_threshold",
			packages: security.OutdatedPackages{{UpdateType: security.UpdatePatch}},
			failOn:   security.UpdateMinor,
			expect:   false,
		},
		{
			name:     "minor_at_minor_threshold",
			packages: security.OutdatedPackages{{UpdateType: security.UpdateMinor}},
			failOn:   security.UpdateMinor,
			expect:   true,
		},
		{
			name:     "empty_packages",
			packages: security.OutdatedPackages{},
			failOn:   security.UpdatePatch,
			expect:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &security.OutdatedResult{Packages: tt.packages}
			assert.Equal(t, tt.expect, shouldFailOutdated(result, tt.failOn))
		})
	}
}

func TestOutputHelpers_FormatUpdateType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  security.UpdateType
		expect string
	}{
		{security.UpdateMajor, "MAJOR"},
		{security.UpdateMinor, "MINOR"},
		{security.UpdatePatch, "PATCH"},
		{security.UpdateUnknown, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()
			got := formatUpdateType(tt.input)
			assert.Contains(t, got, tt.expect)
		})
	}
}

//nolint:tparallel // modifies package-level var outdatedMajor
func TestOutputHelpers_OutputUpgradeJSON_Error(t *testing.T) {
	result := &security.UpgradeResult{DryRun: false}
	out := captureStdout(t, func() {
		outputUpgradeJSON(result, fmt.Errorf("upgrade failed"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "upgrade failed", parsed["error"])
}

func TestOutputHelpers_OutputUpgradeJSON_WithResult(t *testing.T) {
	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.23", ToVersion: "1.24", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update", UpdateType: security.UpdateMajor},
		},
		Failed: []security.FailedPackage{},
		DryRun: true,
	}
	out := captureStdout(t, func() {
		outputUpgradeJSON(result, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.True(t, parsed["dry_run"].(bool))
	upgraded := parsed["upgraded"].([]interface{})
	assert.Len(t, upgraded, 1)
}

//nolint:tparallel // modifies package-level var outdatedMajor
func TestOutputHelpers_OutputUpgradeText_Upgraded(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.23", ToVersion: "1.24"},
		},
		DryRun: false,
	}
	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, out, "go: 1.23")
	assert.Contains(t, out, "1.24")
	assert.Contains(t, out, "Upgraded 1 package(s)")
}

//nolint:tparallel // modifies package-level var outdatedMajor
func TestOutputHelpers_OutputUpgradeText_DryRun(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.23", ToVersion: "1.24"},
		},
		DryRun: true,
	}
	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, out, "would upgrade")
	assert.Contains(t, out, "Would upgrade 1 package(s)")
}

//nolint:tparallel // modifies package-level var outdatedMajor
func TestOutputHelpers_OutputUpgradeText_Skipped(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{},
		Skipped: []security.SkippedPackage{
			{Name: "node", Reason: "major update requires --major flag"},
		},
		DryRun: false,
	}
	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, out, "node: skipped")
	assert.Contains(t, out, "1 skipped")
	assert.Contains(t, out, "Use --major")
}

//nolint:tparallel // modifies package-level var outdatedMajor
func TestOutputHelpers_OutputUpgradeText_Failed(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{},
		Failed: []security.FailedPackage{
			{Name: "broken", Error: "permission denied"},
		},
		DryRun: false,
	}
	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, out, "broken: permission denied")
	assert.Contains(t, out, "1 failed")
}

// ---------------------------------------------------------------------------
// security.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputSecurityJSON_Error(t *testing.T) {
	out := captureStdout(t, func() {
		outputSecurityJSON(nil, fmt.Errorf("scan failed"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "scan failed", parsed["error"])
}

func TestOutputHelpers_OutputSecurityJSON_WithResult(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "grype",
		Version:         "0.75.0",
		PackagesScanned: 100,
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-1234", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, CVSS: 9.8, FixedIn: "1.1.2", Title: "Buffer overflow"},
		},
	}
	out := captureStdout(t, func() {
		outputSecurityJSON(result, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "grype", parsed["scanner"])
	vulns := parsed["vulnerabilities"].([]interface{})
	assert.Len(t, vulns, 1)
	v := vulns[0].(map[string]interface{})
	assert.Equal(t, "CVE-2024-1234", v["id"])
	assert.Equal(t, "critical", v["severity"])
}

func TestOutputHelpers_OutputSecurityText_NoVulns(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "grype",
		Version:         "0.75.0",
		PackagesScanned: 50,
		Vulnerabilities: security.Vulnerabilities{},
	}
	opts := security.ScanOptions{}
	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, out, "No vulnerabilities found")
	assert.Contains(t, out, "Packages scanned: 50")
}

func TestOutputHelpers_OutputSecurityText_WithVulns(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "grype",
		Version:         "0.75.0",
		PackagesScanned: 100,
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-001", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, FixedIn: "1.1.2"},
			{ID: "CVE-2024-002", Package: "curl", Version: "7.0", Severity: security.SeverityHigh},
		},
	}
	opts := security.ScanOptions{Quiet: false}
	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, out, "2 vulnerabilities found")
	assert.Contains(t, out, "CVE-2024-001")
	assert.Contains(t, out, "Recommendations:")
	assert.Contains(t, out, "CRITICAL")
}

func TestOutputHelpers_OutputSecurityText_Quiet(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "0.75.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-001", Package: "openssl", Version: "1.1.1", Severity: security.SeverityHigh},
		},
	}
	opts := security.ScanOptions{Quiet: true}
	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, out, "1 vulnerabilities found")
	// Quiet mode skips the vulnerability table
	assert.NotContains(t, out, "SEVERITY\tID")
}

func TestOutputHelpers_ToVulnerabilitiesJSON(t *testing.T) {
	t.Parallel()
	vulns := security.Vulnerabilities{
		{ID: "CVE-2024-001", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, CVSS: 9.8, FixedIn: "1.1.2", Title: "Buffer overflow", Reference: "https://example.com"},
		{ID: "CVE-2024-002", Package: "curl", Version: "7.0", Severity: security.SeverityLow},
	}
	result := toVulnerabilitiesJSON(vulns)
	assert.Len(t, result, 2)
	assert.Equal(t, "CVE-2024-001", result[0].ID)
	assert.Equal(t, "openssl", result[0].Package)
	assert.Equal(t, string(security.SeverityCritical), result[0].Severity)
	assert.InDelta(t, 9.8, result[0].CVSS, 0.01)
	assert.Equal(t, "1.1.2", result[0].FixedIn)
	assert.Equal(t, "CVE-2024-002", result[1].ID)
	assert.Empty(t, result[1].FixedIn)
}

func TestOutputHelpers_GetScanner_AutoNoScanners(t *testing.T) {
	t.Parallel()
	registry := security.NewScannerRegistry()
	// No scanners registered
	scanner, err := getScanner(registry, "auto")
	assert.Nil(t, scanner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scanners available")
}

func TestOutputHelpers_GetScanner_NameNotFound(t *testing.T) {
	t.Parallel()
	registry := security.NewScannerRegistry()
	scanner, err := getScanner(registry, "nonexistent")
	assert.Nil(t, scanner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestOutputHelpers_ShouldFail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		vulns  security.Vulnerabilities
		failOn security.Severity
		expect bool
	}{
		{
			name:   "critical_above_high",
			vulns:  security.Vulnerabilities{{Severity: security.SeverityCritical}},
			failOn: security.SeverityHigh,
			expect: true,
		},
		{
			name:   "low_below_high",
			vulns:  security.Vulnerabilities{{Severity: security.SeverityLow}},
			failOn: security.SeverityHigh,
			expect: false,
		},
		{
			name:   "empty_vulns",
			vulns:  security.Vulnerabilities{},
			failOn: security.SeverityLow,
			expect: false,
		},
		{
			name:   "high_at_high",
			vulns:  security.Vulnerabilities{{Severity: security.SeverityHigh}},
			failOn: security.SeverityHigh,
			expect: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &security.ScanResult{Vulnerabilities: tt.vulns}
			assert.Equal(t, tt.expect, shouldFail(result, tt.failOn))
		})
	}
}

func TestOutputHelpers_FormatSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input  security.Severity
		expect string
	}{
		{security.SeverityCritical, "CRITICAL"},
		{security.SeverityHigh, "HIGH"},
		{security.SeverityMedium, "MEDIUM"},
		{security.SeverityLow, "LOW"},
		{security.SeverityNegligible, "NEGLIGIBLE"},
		{security.Severity("unknown"), "unknown"},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()
			got := formatSeverity(tt.input)
			assert.Contains(t, got, tt.expect)
		})
	}
}

func TestOutputHelpers_PrintSeverityBar(t *testing.T) {
	t.Parallel()
	summary := security.ScanSummary{
		Critical: 1,
		High:     2,
		Medium:   0,
		Low:      3,
	}
	out := captureStdout(t, func() {
		printSeverityBar(summary)
	})
	assert.Contains(t, out, "CRITICAL: 1")
	assert.Contains(t, out, "HIGH: 2")
	assert.Contains(t, out, "LOW: 3")
	assert.NotContains(t, out, "MEDIUM")
}

func TestOutputHelpers_PrintVulnerabilitiesTable(t *testing.T) {
	t.Parallel()
	vulns := security.Vulnerabilities{
		{ID: "CVE-2024-001", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, FixedIn: "1.1.2"},
		{ID: "CVE-2024-002", Package: "curl", Version: "7.0", Severity: security.SeverityLow, FixedIn: ""},
	}
	out := captureStdout(t, func() {
		printVulnerabilitiesTable(vulns)
	})
	assert.Contains(t, out, "CVE-2024-001")
	assert.Contains(t, out, "openssl")
	assert.Contains(t, out, "1.1.2")
	assert.Contains(t, out, "CVE-2024-002")
	// Empty FixedIn should show "-"
	assert.Contains(t, out, "-")
}

// ---------------------------------------------------------------------------
// compliance.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_CollectEvaluatedItems_Nil(t *testing.T) {
	t.Parallel()
	items := collectEvaluatedItems(nil)
	assert.Nil(t, items)
}

func TestOutputHelpers_CollectEvaluatedItems_Empty(t *testing.T) {
	t.Parallel()
	result := &app.ValidationResult{
		Info:   []string{},
		Errors: []string{},
	}
	items := collectEvaluatedItems(result)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestOutputHelpers_CollectEvaluatedItems_WithItems(t *testing.T) {
	t.Parallel()
	result := &app.ValidationResult{
		Info:   []string{"brew.formulae validated", "git.config validated"},
		Errors: []string{"ssh.config missing"},
	}
	items := collectEvaluatedItems(result)
	assert.Len(t, items, 3)
	assert.Contains(t, items, "brew.formulae validated")
	assert.Contains(t, items, "ssh.config missing")
}

func TestOutputHelpers_OutputComplianceError(t *testing.T) {
	out := captureStdout(t, func() {
		outputComplianceError(fmt.Errorf("cannot load config"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "cannot load config", parsed["error"])
}

// ---------------------------------------------------------------------------
// rollback.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_FormatAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		offset time.Duration
		expect string
	}{
		{"just_now", 10 * time.Second, "just now"},
		{"1_min", 1 * time.Minute, "1 min ago"},
		{"5_mins", 5 * time.Minute, "5 mins ago"},
		{"1_hour", 1 * time.Hour, "1 hour ago"},
		{"3_hours", 3 * time.Hour, "3 hours ago"},
		{"1_day", 24 * time.Hour, "1 day ago"},
		{"3_days", 3 * 24 * time.Hour, "3 days ago"},
		{"1_week", 7 * 24 * time.Hour, "1 week ago"},
		{"3_weeks", 21 * 24 * time.Hour, "3 weeks ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := time.Now().Add(-tt.offset)
			got := formatAge(ts)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestOutputHelpers_ListSnapshots(t *testing.T) {
	sets := []snapshot.Set{
		{
			ID:        "abcdef1234567890",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Reason:    "apply",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.zshrc"},
				{Path: "/home/user/.gitconfig"},
			},
		},
		{
			ID:        "12345678abcdefgh",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			Reason:    "",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.bashrc"},
			},
		},
	}

	out := captureStdout(t, func() {
		_ = listSnapshots(context.Background(), nil, sets)
	})
	assert.Contains(t, out, "Available Snapshots")
	assert.Contains(t, out, "abcdef12")
	assert.Contains(t, out, "2 files")
	assert.Contains(t, out, "12345678")
	assert.Contains(t, out, "1 files")
	assert.Contains(t, out, "Reason: apply")
	assert.Contains(t, out, "preflight rollback --to")
}

// ---------------------------------------------------------------------------
// clean.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_FindOrphans_BrewOrphans(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "curl"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "curl", "htop"},
		},
	}
	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "htop", orphans[0].Name)
	assert.Equal(t, "brew", orphans[0].Provider)
	assert.Equal(t, "formula", orphans[0].Type)
}

func TestOutputHelpers_FindOrphans_VSCodeOrphans(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.Go"},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.Go", "ms-python.python"},
		},
	}
	orphans := findOrphans(config, systemState, nil, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "ms-python.python", orphans[0].Name)
	assert.Equal(t, "vscode", orphans[0].Provider)
}

func TestOutputHelpers_FindOrphans_ProviderFilter(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"htop"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"some.ext"},
		},
	}
	// Only check vscode
	orphans := findOrphans(config, systemState, []string{"vscode"}, nil)
	for _, o := range orphans {
		assert.Equal(t, "vscode", o.Provider, "only vscode should be checked")
	}
}

func TestOutputHelpers_FindOrphans_IgnoreList(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"htop", "curl"},
		},
	}
	orphans := findOrphans(config, systemState, nil, []string{"htop"})
	assert.Len(t, orphans, 1)
	assert.Equal(t, "curl", orphans[0].Name)
}

func TestOutputHelpers_ShouldCheckProvider(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filter   []string
		provider string
		expect   bool
	}{
		{"empty_filter_allows_all", nil, "brew", true},
		{"matching_provider", []string{"brew", "vscode"}, "brew", true},
		{"non_matching", []string{"brew"}, "vscode", false},
		{"empty_list_allows_all", []string{}, "anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, shouldCheckProvider(tt.filter, tt.provider))
		})
	}
}

func TestOutputHelpers_IsIgnored(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		pkg        string
		ignoreList []string
		expect     bool
	}{
		{"found", "htop", []string{"htop", "curl"}, true},
		{"not_found", "go", []string{"htop", "curl"}, false},
		{"empty_list", "go", nil, false},
		{"empty_list_slice", "go", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, isIgnored(tt.pkg, tt.ignoreList))
		})
	}
}

func TestOutputHelpers_FindBrewOrphans(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go"},
			"casks":    []interface{}{"firefox"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "htop"},
			"casks":    []interface{}{"firefox", "chrome"},
		},
	}
	orphans := findBrewOrphans(config, systemState, nil)
	assert.Len(t, orphans, 2)

	names := make([]string, len(orphans))
	for i, o := range orphans {
		names[i] = o.Name
	}
	assert.Contains(t, names, "htop")
	assert.Contains(t, names, "chrome")
}

func TestOutputHelpers_FindVSCodeOrphans(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.Go"},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.Go", "ms-python.Python"},
		},
	}
	orphans := findVSCodeOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "ms-python.Python", orphans[0].Name)
	assert.Equal(t, "extension", orphans[0].Type)
}

func TestOutputHelpers_OutputOrphansText(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "vscode", Type: "extension", Name: "ms-python.python"},
	}
	out := captureStdout(t, func() {
		outputOrphansText(orphans)
	})
	assert.Contains(t, out, "Found 2 orphaned items")
	assert.Contains(t, out, "brew")
	assert.Contains(t, out, "htop")
	assert.Contains(t, out, "vscode")
	assert.Contains(t, out, "ms-python.python")
}

func TestOutputHelpers_RemoveOrphans(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "brew", Type: "cask", Name: "chrome"},
		{Provider: "vscode", Type: "extension", Name: "ms-python.python"},
	}
	out := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 3, removed)
		assert.Equal(t, 0, failed)
	})
	assert.Contains(t, out, "brew uninstall htop")
	assert.Contains(t, out, "brew uninstall --cask chrome")
	assert.Contains(t, out, "code --uninstall-extension ms-python.python")
}

func TestOutputHelpers_RunBrewUninstall_Formula(t *testing.T) {
	out := captureStdout(t, func() {
		err := runBrewUninstall("htop", false)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "brew uninstall htop")
	assert.NotContains(t, out, "--cask")
}

func TestOutputHelpers_RunBrewUninstall_Cask(t *testing.T) {
	out := captureStdout(t, func() {
		err := runBrewUninstall("firefox", true)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "brew uninstall --cask firefox")
}

func TestOutputHelpers_RunVSCodeUninstall(t *testing.T) {
	out := captureStdout(t, func() {
		err := runVSCodeUninstall("ms-python.python")
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "code --uninstall-extension ms-python.python")
}

// ---------------------------------------------------------------------------
// discover.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_GetPatternIcon(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patternType discover.PatternType
		expect      string
	}{
		{discover.PatternTypeShell, "\U0001f41a"},      // crab emoji
		{discover.PatternTypeEditor, "\U0001f4dd"},     // memo emoji
		{discover.PatternTypeGit, "\U0001f4e6"},        // package emoji
		{discover.PatternTypeSSH, "\U0001f510"},        // lock emoji
		{discover.PatternTypeTmux, "\U0001f5a5\ufe0f"}, // desktop emoji
		{discover.PatternTypePackageManager, "\U0001f4e6"},
		{discover.PatternType("other"), "\u2022"}, // bullet
	}
	for _, tt := range tests {
		t.Run(string(tt.patternType), func(t *testing.T) {
			t.Parallel()
			got := getPatternIcon(tt.patternType)
			assert.Equal(t, tt.expect, got)
		})
	}
}

// ---------------------------------------------------------------------------
// export.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_ExportToNix(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"ripgrep", "fzf"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
		"shell": map[string]interface{}{
			"shell":   "zsh",
			"plugins": []interface{}{"git", "docker"},
		},
	}
	output, err := exportToNix(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "Generated by preflight export")
	assert.Contains(t, s, "{ config, pkgs, ... }:")
	assert.Contains(t, s, "home.packages = with pkgs;")
	assert.Contains(t, s, "ripgrep")
	assert.Contains(t, s, "fzf")
	assert.Contains(t, s, "programs.git")
	assert.Contains(t, s, `userName = "Test User"`)
	assert.Contains(t, s, `userEmail = "test@example.com"`)
	assert.Contains(t, s, "programs.zsh")
	assert.Contains(t, s, `name = "git"`)
}

func TestOutputHelpers_ExportToBrewfile(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"ripgrep", "fzf"},
			"casks":    []interface{}{"firefox", "alacritty"},
		},
	}
	output, err := exportToBrewfile(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, `tap "homebrew/cask"`)
	assert.Contains(t, s, `brew "ripgrep"`)
	assert.Contains(t, s, `brew "fzf"`)
	assert.Contains(t, s, `cask "firefox"`)
	assert.Contains(t, s, `cask "alacritty"`)
}

func TestOutputHelpers_ExportToShell(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"ripgrep", "fzf"},
			"casks":    []interface{}{"firefox"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	}
	output, err := exportToShell(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	assert.Contains(t, s, "set -euo pipefail")
	assert.Contains(t, s, "brew tap homebrew/cask")
	assert.Contains(t, s, "brew install")
	assert.Contains(t, s, "ripgrep")
	assert.Contains(t, s, "brew install --cask")
	assert.Contains(t, s, "firefox")
	assert.Contains(t, s, `git config --global user.name "Test User"`)
	assert.Contains(t, s, `git config --global user.email "test@example.com"`)
	assert.Contains(t, s, "Setup complete!")
}

// ---------------------------------------------------------------------------
// init.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_GenerateManifestForPreset(t *testing.T) {
	t.Parallel()
	manifest := generateManifestForPreset("balanced")
	assert.Contains(t, manifest, "Generated by preflight init --preset balanced")
	assert.Contains(t, manifest, "defaults:")
	assert.Contains(t, manifest, "mode: intent")
	assert.Contains(t, manifest, "targets:")
	assert.Contains(t, manifest, "- base")
}

func TestOutputHelpers_GenerateLayerForPreset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		preset string
		expect []string
	}{
		{"nvim:minimal", []string{"preset: minimal", "ensure_install: true"}},
		{"balanced", []string{"preset: kickstart", "shell:", "git:"}},
		{"nvim:balanced", []string{"preset: kickstart"}},
		{"nvim:maximal", []string{"preset: astronvim", "starship:"}},
		{"maximal", []string{"preset: astronvim"}},
		{"shell:minimal", []string{"default: zsh"}},
		{"shell:balanced", []string{"oh-my-zsh", "plugins:"}},
		{"git:minimal", []string{"editor: vim"}},
		{"brew:minimal", []string{"ripgrep", "fzf"}},
		{"unknown-preset", []string{"name: base", "Add your configuration here"}},
	}
	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			t.Parallel()
			layer := generateLayerForPreset(tt.preset)
			for _, e := range tt.expect {
				assert.Contains(t, layer, e, "preset %s should contain %q", tt.preset, e)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// agent.go
// ---------------------------------------------------------------------------

func TestOutputHelpers_FormatHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		health agent.HealthStatus
		expect string
	}{
		{
			name:   "healthy",
			health: agent.HealthStatus{Status: agent.HealthHealthy},
			expect: "healthy",
		},
		{
			name:   "degraded",
			health: agent.HealthStatus{Status: agent.HealthDegraded, Message: "high load"},
			expect: "degraded (high load)",
		},
		{
			name:   "unhealthy",
			health: agent.HealthStatus{Status: agent.HealthUnhealthy, Message: "disk full"},
			expect: "unhealthy (disk full)",
		},
		{
			name:   "unknown",
			health: agent.HealthStatus{Status: agent.Health("something-else")},
			expect: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatHealth(tt.health)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestOutputHelpers_FormatDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration time.Duration
		expect   string
	}{
		{"negative", -5 * time.Second, "now"},
		{"seconds", 45 * time.Second, "45s"},
		{"one_minute", 90 * time.Second, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours_and_minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"days_and_hours", 50 * time.Hour, "2d 2h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatDuration(tt.duration)
			assert.Equal(t, tt.expect, got)
		})
	}
}

// ---------------------------------------------------------------------------
// cleanup.go - additional overlaps/orphans text output tests
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputCleanupText_WithOverlaps(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyOverlap,
				Category:       "node_package_managers",
				Packages:       []string{"npm", "yarn", "pnpm"},
				Recommendation: "Node.js package managers - consider keeping only one",
				Keep:           []string{"npm"},
				Remove:         []string{"yarn", "pnpm"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, out, "Overlapping Tools (1)")
	assert.Contains(t, out, "Node Package Managers")
}

func TestOutputHelpers_OutputCleanupText_WithOrphans(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:     security.RedundancyOrphan,
				Packages: []string{"libfoo", "libbar"},
				Action:   "preflight cleanup --autoremove",
				Remove:   []string{"libfoo", "libbar"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, out, "Orphaned Dependencies")
	assert.Contains(t, out, "libfoo, libbar")
	assert.Contains(t, out, "preflight cleanup --autoremove")
}

// ---------------------------------------------------------------------------
// clean.go - edge cases
// ---------------------------------------------------------------------------

func TestOutputHelpers_FindOrphans_NoBrewConfig(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"htop"},
		},
	}
	orphans := findBrewOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "htop", orphans[0].Name)
}

func TestOutputHelpers_FindOrphans_NoSystemState(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go"},
		},
	}
	systemState := map[string]interface{}{}
	orphans := findBrewOrphans(config, systemState, nil)
	assert.Empty(t, orphans)
}

func TestOutputHelpers_FindVSCodeOrphans_CaseInsensitive(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.Go"},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.go"}, // lowercase in system
		},
	}
	orphans := findVSCodeOrphans(config, systemState, nil)
	// "golang.go" matches "golang.Go" case-insensitively
	assert.Empty(t, orphans)
}

// ---------------------------------------------------------------------------
// export.go - edge cases
// ---------------------------------------------------------------------------

func TestOutputHelpers_ExportToNix_EmptyConfig(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToNix(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "{ config, pkgs, ... }:")
	assert.Contains(t, s, "}")
}

func TestOutputHelpers_ExportToBrewfile_EmptyConfig(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToBrewfile(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "Generated by preflight export")
	assert.NotContains(t, s, "tap")
	assert.NotContains(t, s, "brew")
}

func TestOutputHelpers_ExportToShell_EmptyConfig(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToShell(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	assert.Contains(t, s, "Setup complete!")
}

// ---------------------------------------------------------------------------
// security.go - edge cases
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputSecurityText_WithFixable(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "trivy",
		Version:         "0.50.0",
		PackagesScanned: 200,
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-001", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, FixedIn: "1.1.2"},
		},
	}
	opts := security.ScanOptions{Quiet: false}
	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, out, "Fixable: 1")
	assert.Contains(t, out, "vulnerabilities have fixes available")
}

// ---------------------------------------------------------------------------
// outdated.go - edge cases
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputOutdatedText_WithMajorRecommendations(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "node", CurrentVersion: "18.0", LatestVersion: "20.0", UpdateType: security.UpdateMajor, Provider: "brew"},
		},
	}
	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, out, "MAJOR updates")
	assert.Contains(t, out, "breaking changes")
}

// ---------------------------------------------------------------------------
// cleanup.go - outputCleanupText with mixed types
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputCleanupText_MixedTypes(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Recommendation: "Keep go",
				Remove:         []string{"go@1.24"},
			},
			{
				Type:           security.RedundancyOverlap,
				Category:       "editors",
				Packages:       []string{"vim", "neovim"},
				Recommendation: "Terminal editors - typically used together",
				Keep:           []string{"vim", "neovim"},
			},
			{
				Type:     security.RedundancyOrphan,
				Packages: []string{"libfoo"},
				Action:   "preflight cleanup --autoremove",
				Remove:   []string{"libfoo"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, out, "3 redundancies found")
	assert.Contains(t, out, "Version Duplicates (1)")
	assert.Contains(t, out, "Overlapping Tools (1)")
	assert.Contains(t, out, "Orphaned Dependencies")
	assert.Contains(t, out, "preflight cleanup --remove")
	assert.Contains(t, out, "preflight cleanup --autoremove")
	assert.Contains(t, out, "preflight cleanup --all")
}

// ---------------------------------------------------------------------------
// redundancy summary bar with only some types
// ---------------------------------------------------------------------------

func TestOutputHelpers_PrintRedundancySummaryBar_OrphansOnly(t *testing.T) {
	t.Parallel()
	summary := security.RedundancySummary{Orphans: 7}
	out := captureStdout(t, func() {
		printRedundancySummaryBar(summary)
	})
	assert.Contains(t, out, "ORPHANS: 7")
	assert.NotContains(t, out, "DUPLICATES")
	assert.NotContains(t, out, "OVERLAPS")
}

// ---------------------------------------------------------------------------
// printOverlapTable - without keep/remove (keepAll scenario)
// ---------------------------------------------------------------------------

func TestOutputHelpers_PrintOverlapTable_NoKeepRemove(t *testing.T) {
	t.Parallel()
	redundancies := security.Redundancies{
		{
			Category:       "terminal_emulators",
			Packages:       []string{"alacritty", "wezterm"},
			Recommendation: "Terminal emulators - typically used together",
			Keep:           nil,
			Remove:         nil,
		},
	}
	out := captureStdout(t, func() {
		printOverlapTable(redundancies)
	})
	assert.Contains(t, out, "Terminal Emulators")
	assert.Contains(t, out, "alacritty, wezterm")
	// Should NOT print keep/remove line when both are empty
	assert.NotContains(t, out, "Keep:")
}

// ---------------------------------------------------------------------------
// removeOrphans with unknown provider (does nothing special)
// ---------------------------------------------------------------------------

func TestOutputHelpers_RemoveOrphans_UnknownProvider(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "unknown", Type: "thing", Name: "foo"},
	}
	out := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		// Unknown provider does nothing, but doesn't error either
		assert.Equal(t, 1, removed)
		assert.Equal(t, 0, failed)
	})
	assert.Contains(t, out, "Removed unknown foo")
}

// ---------------------------------------------------------------------------
// formatCategory - edge cases
// ---------------------------------------------------------------------------

func TestOutputHelpers_FormatCategory_AlreadyCapitalized(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Already Good", formatCategory("Already_Good"))
}

func TestOutputHelpers_FormatCategory_SingleLetter(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "A B C", formatCategory("a_b_c"))
}

// ---------------------------------------------------------------------------
// deprecated.go - additional tests for coverage of all reason paths
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputDeprecatedText_AllReasonTypes(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Provider: "brew", Reason: security.ReasonDisabled},
			{Name: "pkg2", Provider: "brew", Reason: security.ReasonDeprecated},
			{Name: "pkg3", Provider: "brew", Reason: security.ReasonEOL},
			{Name: "pkg4", Provider: "brew", Reason: security.ReasonUnmaintained},
		},
	}
	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})
	assert.Contains(t, out, "DISABLED")
	assert.Contains(t, out, "packages are DISABLED")
	assert.Contains(t, out, "packages are DEPRECATED")
}

// ---------------------------------------------------------------------------
// security JSON with nil result (no error, no result)
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputSecurityJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputSecurityJSON(nil, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	// Should be a valid JSON object with no error and no data
	assert.Empty(t, parsed["error"])
	assert.Nil(t, parsed["scanner"])
}

// ---------------------------------------------------------------------------
// outdated JSON with nil result (no error, no result)
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputOutdatedJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputOutdatedJSON(nil, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed["error"])
	assert.Nil(t, parsed["checker"])
}

// ---------------------------------------------------------------------------
// deprecated JSON with nil result (no error, no result)
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputDeprecatedJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputDeprecatedJSON(nil, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Empty(t, parsed["error"])
}

// ---------------------------------------------------------------------------
// upgrade JSON with nil result
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputUpgradeJSON_NilResult(t *testing.T) {
	// When result is nil, DryRun should be false
	out := captureStdout(t, func() {
		outputUpgradeJSON(nil, fmt.Errorf("checker missing"))
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "checker missing", parsed["error"])
	assert.False(t, parsed["dry_run"].(bool))
}

// ---------------------------------------------------------------------------
// formatAge edge: exactly at boundaries
// ---------------------------------------------------------------------------

func TestOutputHelpers_FormatAge_ExactBoundaries(t *testing.T) {
	t.Parallel()

	// Exactly 59 seconds should be "just now"
	ts59s := time.Now().Add(-59 * time.Second)
	got := formatAge(ts59s)
	assert.Equal(t, "just now", got)

	// Exactly 60 seconds should be "1 min ago"
	ts60s := time.Now().Add(-60 * time.Second)
	got = formatAge(ts60s)
	assert.Equal(t, "1 min ago", got)
}

// ---------------------------------------------------------------------------
// findOrphans with files provider (returns nil - stub)
// ---------------------------------------------------------------------------

func TestOutputHelpers_FindOrphans_FilesProvider(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	systemState := map[string]interface{}{}
	// Only check files provider
	orphans := findOrphans(config, systemState, []string{"files"}, nil)
	assert.Empty(t, orphans)
}

// ---------------------------------------------------------------------------
// export.go - Nix without shell section
// ---------------------------------------------------------------------------

func TestOutputHelpers_ExportToNix_NoShell(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"ripgrep"},
		},
	}
	output, err := exportToNix(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "ripgrep")
	assert.NotContains(t, s, "programs.zsh")
}

// ---------------------------------------------------------------------------
// export.go - Shell without git section
// ---------------------------------------------------------------------------

func TestOutputHelpers_ExportToShell_NoGit(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"curl"},
		},
	}
	output, err := exportToShell(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "curl")
	assert.NotContains(t, s, "git config")
}

// ---------------------------------------------------------------------------
// printRedundancyTable with no remove entries
// ---------------------------------------------------------------------------

func TestOutputHelpers_PrintRedundancyTable_NoRemove(t *testing.T) {
	t.Parallel()
	redundancies := security.Redundancies{
		{
			Packages:       []string{"vim", "neovim"},
			Recommendation: "Both are fine",
			Remove:         nil,
		},
	}
	out := captureStdout(t, func() {
		printRedundancyTable(redundancies)
	})
	assert.Contains(t, out, "vim + neovim")
	assert.Contains(t, out, "Both are fine")
	assert.NotContains(t, out, "Remove:")
}

// ---------------------------------------------------------------------------
// Cleanup text - verify actions section with duplicates but no orphans
// ---------------------------------------------------------------------------

func TestOutputHelpers_OutputCleanupText_DuplicatesOnly_ActionSection(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"python", "python@3.12"},
				Recommendation: "Keep python",
				Remove:         []string{"python@3.12"},
			},
		},
	}
	out := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, out, "Actions:")
	assert.Contains(t, out, "preflight cleanup --remove python@3.12")
	// No orphans, so no --autoremove action
	lines := strings.Split(out, "\n")
	hasAutoremove := false
	for _, line := range lines {
		if strings.Contains(line, "--autoremove") {
			hasAutoremove = true
		}
	}
	assert.False(t, hasAutoremove, "should not contain --autoremove when no orphans")
}
