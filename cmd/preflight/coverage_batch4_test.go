package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// security.go tests
// ---------------------------------------------------------------------------

func TestBatch4_getScanner_AutoMode(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	scanner, err := getScanner(registry, "auto")
	// On CI or environments without grype/trivy installed, First() returns nil
	if scanner != nil {
		assert.NoError(t, err)
		assert.NotEmpty(t, scanner.Name())
	} else {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no scanners available")
	}
}

func TestBatch4_getScanner_EmptyName(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())

	scanner, err := getScanner(registry, "")
	// Same as auto mode: returns first available or error
	if scanner != nil {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no scanners available")
	}
}

func TestBatch4_getScanner_SpecificNonexistent(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())

	scanner, err := getScanner(registry, "nonexistent")
	assert.Nil(t, scanner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestBatch4_getScanner_EmptyRegistry_Auto(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()

	scanner, err := getScanner(registry, "auto")
	assert.Nil(t, scanner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scanners available")
}

func TestBatch4_listScanners(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	out := captureStdout(t, func() {
		err := listScanners(registry)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Available security scanners")
	assert.Contains(t, out, "grype")
	assert.Contains(t, out, "trivy")
}

func TestBatch4_listScanners_EmptyRegistry(t *testing.T) {
	registry := security.NewScannerRegistry()

	out := captureStdout(t, func() {
		err := listScanners(registry)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Available security scanners")
}

func TestBatch4_shouldFail_WithCritical(t *testing.T) {
	t.Parallel()

	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Severity: security.SeverityCritical},
		},
	}

	assert.True(t, shouldFail(result, security.SeverityCritical))
	assert.True(t, shouldFail(result, security.SeverityHigh))
	assert.True(t, shouldFail(result, security.SeverityMedium))
}

func TestBatch4_shouldFail_WithHigh(t *testing.T) {
	t.Parallel()

	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Severity: security.SeverityHigh},
		},
	}

	assert.False(t, shouldFail(result, security.SeverityCritical))
	assert.True(t, shouldFail(result, security.SeverityHigh))
	assert.True(t, shouldFail(result, security.SeverityMedium))
}

func TestBatch4_shouldFail_NoVulns(t *testing.T) {
	t.Parallel()

	result := &security.ScanResult{}

	assert.False(t, shouldFail(result, security.SeverityCritical))
	assert.False(t, shouldFail(result, security.SeverityHigh))
	assert.False(t, shouldFail(result, security.SeverityMedium))
}

func TestBatch4_outputSecurityJSON_WithError(t *testing.T) {
	out := captureStdout(t, func() {
		outputSecurityJSON(nil, fmt.Errorf("scan failed"))
	})

	assert.Contains(t, out, "scan failed")
	assert.Contains(t, out, `"error"`)
}

func TestBatch4_outputSecurityJSON_WithResult(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-001", Package: "pkg1", Version: "1.0", Severity: security.SeverityHigh, CVSS: 7.5},
		},
		PackagesScanned: 42,
	}

	out := captureStdout(t, func() {
		outputSecurityJSON(result, nil)
	})

	assert.Contains(t, out, "grype")
	assert.Contains(t, out, "CVE-2024-001")
	assert.Contains(t, out, `"summary"`)
}

func TestBatch4_outputSecurityJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputSecurityJSON(nil, nil)
	})

	// Should produce valid JSON with empty fields
	assert.Contains(t, out, "{")
}

func TestBatch4_toVulnerabilitiesJSON(t *testing.T) {
	t.Parallel()

	vulns := security.Vulnerabilities{
		{
			ID:        "CVE-1",
			Package:   "pkg1",
			Version:   "1.0",
			Severity:  security.SeverityHigh,
			CVSS:      7.5,
			FixedIn:   "1.1",
			Title:     "Test vuln",
			Reference: "http://example.com",
		},
		{
			ID:       "CVE-2",
			Package:  "pkg2",
			Version:  "2.0",
			Severity: security.SeverityLow,
		},
	}

	result := toVulnerabilitiesJSON(vulns)
	require.Len(t, result, 2)

	assert.Equal(t, "CVE-1", result[0].ID)
	assert.Equal(t, "pkg1", result[0].Package)
	assert.Equal(t, "1.0", result[0].Version)
	assert.Equal(t, "high", result[0].Severity)
	assert.Equal(t, 7.5, result[0].CVSS)
	assert.Equal(t, "1.1", result[0].FixedIn)
	assert.Equal(t, "Test vuln", result[0].Title)
	assert.Equal(t, "http://example.com", result[0].Reference)

	assert.Equal(t, "CVE-2", result[1].ID)
	assert.Equal(t, "pkg2", result[1].Package)
	assert.Equal(t, "low", result[1].Severity)
	assert.Empty(t, result[1].FixedIn)
}

func TestBatch4_toVulnerabilitiesJSON_Empty(t *testing.T) {
	t.Parallel()

	result := toVulnerabilitiesJSON(security.Vulnerabilities{})
	assert.Empty(t, result)
}

func TestBatch4_outputSecurityText_NoVulns(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0.0",
	}
	opts := security.ScanOptions{}

	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})

	assert.Contains(t, out, "No vulnerabilities found")
	assert.Contains(t, out, "grype")
}

func TestBatch4_outputSecurityText_WithVulns(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Severity: security.SeverityCritical, Package: "pkg1", Version: "1.0", FixedIn: "1.1"},
			{ID: "CVE-2", Severity: security.SeverityHigh, Package: "pkg2", Version: "2.0"},
		},
		PackagesScanned: 100,
	}
	opts := security.ScanOptions{}

	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})

	assert.Contains(t, out, "vulnerabilities found")
	assert.Contains(t, out, "CRITICAL")
	assert.Contains(t, out, "CVE-1")
	assert.Contains(t, out, "CVE-2")
	assert.Contains(t, out, "Recommendations")
}

func TestBatch4_outputSecurityText_QuietMode(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Severity: security.SeverityHigh, Package: "pkg1", Version: "1.0"},
		},
		PackagesScanned: 50,
	}
	opts := security.ScanOptions{Quiet: true}

	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})

	assert.Contains(t, out, "vulnerabilities found")
	// In quiet mode, the vulnerabilities table should not be printed
	// but the summary still shows
	assert.Contains(t, out, "HIGH")
}

func TestBatch4_outputSecurityText_WithFixable(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "trivy",
		Version: "2.0.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Severity: security.SeverityCritical, Package: "pkg1", Version: "1.0", FixedIn: "1.1"},
		},
		PackagesScanned: 10,
	}
	opts := security.ScanOptions{}

	out := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})

	assert.Contains(t, out, "Fixable")
	assert.Contains(t, out, "fixes available")
}

func TestBatch4_printSeverityBar(t *testing.T) {
	summary := security.ScanSummary{
		Critical: 1,
		High:     2,
		Medium:   3,
		Low:      4,
	}

	out := captureStdout(t, func() {
		printSeverityBar(summary)
	})

	assert.Contains(t, out, "CRITICAL: 1")
	assert.Contains(t, out, "HIGH: 2")
	assert.Contains(t, out, "MEDIUM: 3")
	assert.Contains(t, out, "LOW: 4")
}

func TestBatch4_printSeverityBar_ZeroCounts(t *testing.T) {
	summary := security.ScanSummary{
		Critical: 0,
		High:     0,
		Medium:   5,
		Low:      0,
	}

	out := captureStdout(t, func() {
		printSeverityBar(summary)
	})

	// Zero counts should not be printed
	assert.NotContains(t, out, "CRITICAL")
	assert.NotContains(t, out, "HIGH")
	assert.Contains(t, out, "MEDIUM: 5")
	assert.NotContains(t, out, "LOW")
}

func TestBatch4_printVulnerabilitiesTable(t *testing.T) {
	vulns := security.Vulnerabilities{
		{ID: "CVE-1", Severity: security.SeverityHigh, Package: "pkg1", Version: "1.0", FixedIn: "1.1"},
		{ID: "CVE-2", Severity: security.SeverityLow, Package: "pkg2", Version: "2.0"},
	}

	out := captureStdout(t, func() {
		printVulnerabilitiesTable(vulns)
	})

	assert.Contains(t, out, "CVE-1")
	assert.Contains(t, out, "CVE-2")
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "pkg2")
	assert.Contains(t, out, "1.1")
	assert.Contains(t, out, "-") // No fix for CVE-2
	assert.Contains(t, out, "SEVERITY")
	assert.Contains(t, out, "PACKAGE")
}

func TestBatch4_printVulnerabilitiesTable_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		printVulnerabilitiesTable(security.Vulnerabilities{})
	})

	// Should still have headers
	assert.Contains(t, out, "SEVERITY")
}

func TestBatch4_formatSeverity_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity security.Severity
		expected string
	}{
		{"critical", security.SeverityCritical, "CRITICAL"},
		{"high", security.SeverityHigh, "HIGH"},
		{"medium", security.SeverityMedium, "MEDIUM"},
		{"low", security.SeverityLow, "LOW"},
		{"negligible", security.SeverityNegligible, "NEGLIGIBLE"},
		{"unknown", security.Severity("unknown"), "unknown"},
		{"custom", security.Severity("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatSeverity(tc.severity)
			assert.Contains(t, result, tc.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// outdated.go tests
// ---------------------------------------------------------------------------

func TestBatch4_parseUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected security.UpdateType
	}{
		{"major", "major", security.UpdateMajor},
		{"minor", "minor", security.UpdateMinor},
		{"patch", "patch", security.UpdatePatch},
		{"MAJOR_uppercase", "MAJOR", security.UpdateMajor},
		{"Minor_mixed", "Minor", security.UpdateMinor},
		{"PATCH_uppercase", "PATCH", security.UpdatePatch},
		{"unknown_default", "unknown", security.UpdateMinor},
		{"empty_default", "", security.UpdateMinor},
		{"gibberish", "foobar", security.UpdateMinor},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, parseUpdateType(tc.input))
		})
	}
}

func TestBatch4_shouldFailOutdated_WithMajor(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Packages: security.OutdatedPackages{
			{Name: "pkg1", UpdateType: security.UpdateMajor},
		},
	}

	assert.True(t, shouldFailOutdated(result, security.UpdateMajor))
	assert.True(t, shouldFailOutdated(result, security.UpdateMinor))
	assert.True(t, shouldFailOutdated(result, security.UpdatePatch))
}

func TestBatch4_shouldFailOutdated_WithMinor(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{
		Packages: security.OutdatedPackages{
			{Name: "pkg1", UpdateType: security.UpdateMinor},
		},
	}

	assert.False(t, shouldFailOutdated(result, security.UpdateMajor))
	assert.True(t, shouldFailOutdated(result, security.UpdateMinor))
	assert.True(t, shouldFailOutdated(result, security.UpdatePatch))
}

func TestBatch4_shouldFailOutdated_NoPackages(t *testing.T) {
	t.Parallel()

	result := &security.OutdatedResult{}

	assert.False(t, shouldFailOutdated(result, security.UpdateMajor))
	assert.False(t, shouldFailOutdated(result, security.UpdateMinor))
	assert.False(t, shouldFailOutdated(result, security.UpdatePatch))
}

func TestBatch4_outputOutdatedJSON_WithError(t *testing.T) {
	out := captureStdout(t, func() {
		outputOutdatedJSON(nil, fmt.Errorf("check failed"))
	})

	assert.Contains(t, out, "check failed")
	assert.Contains(t, out, `"error"`)
}

func TestBatch4_outputOutdatedJSON_WithResult(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.20", LatestVersion: "1.21", UpdateType: security.UpdateMinor, Provider: "brew"},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedJSON(result, nil)
	})

	assert.Contains(t, out, "brew")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "1.20")
	assert.Contains(t, out, "1.21")
	assert.Contains(t, out, `"summary"`)
}

func TestBatch4_outputOutdatedJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputOutdatedJSON(nil, nil)
	})

	assert.Contains(t, out, "{")
}

func TestBatch4_toOutdatedPackagesJSON(t *testing.T) {
	t.Parallel()

	pkgs := security.OutdatedPackages{
		{Name: "go", CurrentVersion: "1.20", LatestVersion: "1.21", UpdateType: security.UpdateMinor, Provider: "brew", Pinned: true},
		{Name: "vim", CurrentVersion: "9.0", LatestVersion: "10.0", UpdateType: security.UpdateMajor, Provider: "brew"},
	}

	result := toOutdatedPackagesJSON(pkgs)
	require.Len(t, result, 2)

	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "1.20", result[0].CurrentVersion)
	assert.Equal(t, "1.21", result[0].LatestVersion)
	assert.Equal(t, "minor", result[0].UpdateType)
	assert.Equal(t, "brew", result[0].Provider)
	assert.True(t, result[0].Pinned)

	assert.Equal(t, "vim", result[1].Name)
	assert.False(t, result[1].Pinned)
}

func TestBatch4_toOutdatedPackagesJSON_Empty(t *testing.T) {
	t.Parallel()

	result := toOutdatedPackagesJSON(security.OutdatedPackages{})
	assert.Empty(t, result)
}

func TestBatch4_outputOutdatedText_NoPackages(t *testing.T) {
	result := &security.OutdatedResult{Checker: "brew"}

	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, out, "All packages are up to date")
	assert.Contains(t, out, "brew")
}

func TestBatch4_outputOutdatedText_WithPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.20", LatestVersion: "2.0", UpdateType: security.UpdateMajor, Provider: "brew"},
			{Name: "vim", CurrentVersion: "9.0", LatestVersion: "9.1", UpdateType: security.UpdateMinor, Provider: "brew"},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, out, "updates available")
	assert.Contains(t, out, "MAJOR")
	assert.Contains(t, out, "go")
	assert.Contains(t, out, "vim")
	assert.Contains(t, out, "Recommendations")
}

func TestBatch4_outputOutdatedText_QuietMode(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.20", LatestVersion: "2.0", UpdateType: security.UpdateMajor, Provider: "brew"},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, true)
	})

	// In quiet mode, the table is not printed but the summary is
	assert.Contains(t, out, "updates available")
	assert.Contains(t, out, "MAJOR")
	// The table rows (go, 1.20) should not appear in quiet mode
	// but the MAJOR count in the summary bar will
}

func TestBatch4_outputOutdatedText_WithPinned(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.20", LatestVersion: "1.21", UpdateType: security.UpdateMinor, Provider: "brew", Pinned: true},
		},
	}

	out := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})

	assert.Contains(t, out, "Pinned")
}

func TestBatch4_printUpdateTypeBar(t *testing.T) {
	summary := security.OutdatedSummary{Major: 1, Minor: 2, Patch: 3}

	out := captureStdout(t, func() {
		printUpdateTypeBar(summary)
	})

	assert.Contains(t, out, "MAJOR: 1")
	assert.Contains(t, out, "MINOR: 2")
	assert.Contains(t, out, "PATCH: 3")
}

func TestBatch4_printUpdateTypeBar_ZeroCounts(t *testing.T) {
	summary := security.OutdatedSummary{Major: 0, Minor: 5, Patch: 0}

	out := captureStdout(t, func() {
		printUpdateTypeBar(summary)
	})

	assert.NotContains(t, out, "MAJOR")
	assert.Contains(t, out, "MINOR: 5")
	assert.NotContains(t, out, "PATCH")
}

func TestBatch4_printOutdatedTable(t *testing.T) {
	pkgs := security.OutdatedPackages{
		{Name: "go", CurrentVersion: "1.20", LatestVersion: "1.21", UpdateType: security.UpdateMinor, Provider: "brew"},
		{Name: "vim", CurrentVersion: "9.0", LatestVersion: "10.0", UpdateType: security.UpdateMajor, Provider: "brew"},
	}

	out := captureStdout(t, func() {
		printOutdatedTable(pkgs)
	})

	assert.Contains(t, out, "go")
	assert.Contains(t, out, "1.20")
	assert.Contains(t, out, "1.21")
	assert.Contains(t, out, "vim")
	assert.Contains(t, out, "TYPE")
	assert.Contains(t, out, "PACKAGE")
	assert.Contains(t, out, "PROVIDER")
}

func TestBatch4_printOutdatedTable_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		printOutdatedTable(security.OutdatedPackages{})
	})

	assert.Contains(t, out, "TYPE")
}

func TestBatch4_formatUpdateType_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		updateType security.UpdateType
		expected   string
	}{
		{"major", security.UpdateMajor, "MAJOR"},
		{"minor", security.UpdateMinor, "MINOR"},
		{"patch", security.UpdatePatch, "PATCH"},
		{"unknown", security.UpdateType("unknown"), "unknown"},
		{"empty", security.UpdateType(""), ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatUpdateType(tc.updateType)
			assert.Contains(t, result, tc.expected)
		})
	}
}

func TestBatch4_outputUpgradeJSON_WithError(t *testing.T) {
	out := captureStdout(t, func() {
		outputUpgradeJSON(nil, fmt.Errorf("upgrade failed"))
	})

	assert.Contains(t, out, "upgrade failed")
	assert.Contains(t, out, `"error"`)
}

func TestBatch4_outputUpgradeJSON_WithResult(t *testing.T) {
	result := &security.UpgradeResult{
		DryRun: true,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.21", Provider: "brew"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "vim", Reason: "major update", UpdateType: security.UpdateMajor},
		},
		Failed: []security.FailedPackage{
			{Name: "gcc", Error: "conflict"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeJSON(result, nil)
	})

	assert.Contains(t, out, "go")
	assert.Contains(t, out, "1.20")
	assert.Contains(t, out, "1.21")
	assert.Contains(t, out, "vim")
	assert.Contains(t, out, "gcc")
	assert.Contains(t, out, "dry_run")
}

func TestBatch4_outputUpgradeJSON_NilResultWithError(t *testing.T) {
	out := captureStdout(t, func() {
		outputUpgradeJSON(nil, fmt.Errorf("failed"))
	})

	assert.Contains(t, out, "failed")
	// DryRun should be false when result is nil
	assert.Contains(t, out, `"dry_run": false`)
}

func TestBatch4_outputUpgradeText_DryRun(t *testing.T) {
	// Save and restore global flags
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		DryRun: true,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.21"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "vim", Reason: "major update"},
		},
		Failed: []security.FailedPackage{
			{Name: "gcc", Error: "conflict"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "would upgrade")
	assert.Contains(t, out, "skipped")
	assert.Contains(t, out, "gcc")
	assert.Contains(t, out, "Use --major")
}

func TestBatch4_outputUpgradeText_RealUpgrade(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		DryRun: false,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.21"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "Upgraded 1")
	assert.NotContains(t, out, "would upgrade")
}

func TestBatch4_outputUpgradeText_WithMajorFlag(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = true

	result := &security.UpgradeResult{
		DryRun: false,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.21"},
		},
		Skipped: []security.SkippedPackage{
			{Name: "vim", Reason: "major update"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	// When --major is set, the hint should not appear
	assert.NotContains(t, out, "Use --major")
}

func TestBatch4_outputUpgradeText_NoSkipped(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		DryRun: false,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.21"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.NotContains(t, out, "skipped")
	assert.NotContains(t, out, "Use --major")
}

func TestBatch4_outputUpgradeText_AllFailed(t *testing.T) {
	oldMajor := outdatedMajor
	defer func() { outdatedMajor = oldMajor }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		DryRun: false,
		Failed: []security.FailedPackage{
			{Name: "gcc", Error: "conflict"},
			{Name: "cmake", Error: "permission denied"},
		},
	}

	out := captureStdout(t, func() {
		outputUpgradeText(result)
	})

	assert.Contains(t, out, "gcc")
	assert.Contains(t, out, "cmake")
	assert.Contains(t, out, "2 failed")
}

// ---------------------------------------------------------------------------
// deprecated.go tests
// ---------------------------------------------------------------------------

func TestBatch4_outputDeprecatedJSON_WithError(t *testing.T) {
	out := captureStdout(t, func() {
		outputDeprecatedJSON(nil, fmt.Errorf("check failed"))
	})

	assert.Contains(t, out, "check failed")
	assert.Contains(t, out, `"error"`)
}

func TestBatch4_outputDeprecatedJSON_WithResult(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated, Message: "use pkg2"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedJSON(result, nil)
	})

	assert.Contains(t, out, "brew")
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, `"summary"`)
}

func TestBatch4_outputDeprecatedJSON_NilResultNilError(t *testing.T) {
	out := captureStdout(t, func() {
		outputDeprecatedJSON(nil, nil)
	})

	assert.Contains(t, out, "{")
}

func TestBatch4_toDeprecatedPackagesJSON_WithDate(t *testing.T) {
	t.Parallel()

	now := time.Now()
	pkgs := security.DeprecatedPackages{
		{
			Name:        "pkg1",
			Version:     "1.0",
			Provider:    "brew",
			Reason:      security.ReasonDeprecated,
			Date:        &now,
			Alternative: "pkg2",
			Message:     "deprecated in favor of pkg2",
		},
	}

	result := toDeprecatedPackagesJSON(pkgs)
	require.Len(t, result, 1)

	assert.Equal(t, "pkg1", result[0].Name)
	assert.Equal(t, "1.0", result[0].Version)
	assert.Equal(t, "brew", result[0].Provider)
	assert.Equal(t, "deprecated", result[0].Reason)
	assert.NotEmpty(t, result[0].Date)
	assert.Equal(t, "pkg2", result[0].Alternative)
	assert.Equal(t, "deprecated in favor of pkg2", result[0].Message)
}

func TestBatch4_toDeprecatedPackagesJSON_WithoutDate(t *testing.T) {
	t.Parallel()

	pkgs := security.DeprecatedPackages{
		{Name: "pkg2", Provider: "brew", Reason: security.ReasonDisabled},
	}

	result := toDeprecatedPackagesJSON(pkgs)
	require.Len(t, result, 1)

	assert.Equal(t, "pkg2", result[0].Name)
	assert.Empty(t, result[0].Date)
	assert.Equal(t, "disabled", result[0].Reason)
}

func TestBatch4_toDeprecatedPackagesJSON_Empty(t *testing.T) {
	t.Parallel()

	result := toDeprecatedPackagesJSON(security.DeprecatedPackages{})
	assert.Empty(t, result)
}

func TestBatch4_outputDeprecatedText_NoPackages(t *testing.T) {
	result := &security.DeprecatedResult{Checker: "brew"}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, out, "No deprecated packages found")
	assert.Contains(t, out, "brew")
}

func TestBatch4_outputDeprecatedText_WithPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Reason: security.ReasonDisabled, Version: "1.0", Message: "disabled by brew"},
			{Name: "pkg2", Reason: security.ReasonDeprecated, Message: "old"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, out, "require attention")
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "pkg2")
	assert.Contains(t, out, "DISABLED")
	assert.Contains(t, out, "DEPRECATED")
	assert.Contains(t, out, "Recommendations")
}

func TestBatch4_outputDeprecatedText_QuietMode(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "pkg1", Reason: security.ReasonDisabled, Message: "disabled"},
		},
	}

	out := captureStdout(t, func() {
		outputDeprecatedText(result, true)
	})

	// In quiet mode, the table should not be printed
	assert.Contains(t, out, "require attention")
	assert.Contains(t, out, "DISABLED: 1")
}

func TestBatch4_printDeprecationSummaryBar(t *testing.T) {
	summary := security.DeprecatedSummary{
		Disabled:     1,
		Deprecated:   2,
		EOL:          1,
		Unmaintained: 1,
	}

	out := captureStdout(t, func() {
		printDeprecationSummaryBar(summary)
	})

	assert.Contains(t, out, "DISABLED: 1")
	assert.Contains(t, out, "DEPRECATED: 2")
	assert.Contains(t, out, "EOL: 1")
	assert.Contains(t, out, "UNMAINTAINED: 1")
}

func TestBatch4_printDeprecationSummaryBar_ZeroCounts(t *testing.T) {
	summary := security.DeprecatedSummary{
		Disabled:     0,
		Deprecated:   3,
		EOL:          0,
		Unmaintained: 0,
	}

	out := captureStdout(t, func() {
		printDeprecationSummaryBar(summary)
	})

	assert.NotContains(t, out, "DISABLED")
	assert.Contains(t, out, "DEPRECATED: 3")
	assert.NotContains(t, out, "EOL")
	assert.NotContains(t, out, "UNMAINTAINED")
}

func TestBatch4_printDeprecatedTable_WithVersionAndMessage(t *testing.T) {
	pkgs := security.DeprecatedPackages{
		{Name: "pkg1", Reason: security.ReasonDisabled, Version: "1.0", Message: "disabled"},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(pkgs)
	})

	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "1.0")
	assert.Contains(t, out, "disabled")
}

func TestBatch4_printDeprecatedTable_EmptyVersionAndMessage(t *testing.T) {
	pkgs := security.DeprecatedPackages{
		{Name: "pkg2", Reason: security.ReasonDeprecated},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(pkgs)
	})

	assert.Contains(t, out, "pkg2")
	assert.Contains(t, out, "-") // Both version and message are empty => "-"
}

func TestBatch4_printDeprecatedTable_LongMessageTruncation(t *testing.T) {
	longMsg := strings.Repeat("x", 60)
	pkgs := security.DeprecatedPackages{
		{Name: "pkg3", Reason: security.ReasonEOL, Message: longMsg},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(pkgs)
	})

	assert.Contains(t, out, "pkg3")
	assert.Contains(t, out, "...")
	// The full 60-char message should not appear
	assert.NotContains(t, out, longMsg)
}

func TestBatch4_printDeprecatedTable_MultiplePackages(t *testing.T) {
	pkgs := security.DeprecatedPackages{
		{Name: "pkg1", Reason: security.ReasonDisabled, Version: "1.0", Message: "disabled"},
		{Name: "pkg2", Reason: security.ReasonDeprecated},
		{Name: "pkg3", Reason: security.ReasonEOL, Message: strings.Repeat("y", 60)},
		{Name: "pkg4", Reason: security.ReasonUnmaintained, Version: "4.0", Message: "no maintainer"},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(pkgs)
	})

	assert.Contains(t, out, "STATUS")
	assert.Contains(t, out, "PACKAGE")
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "pkg2")
	assert.Contains(t, out, "pkg3")
	assert.Contains(t, out, "pkg4")
}

func TestBatch4_formatDeprecationStatus_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reason   security.DeprecationReason
		expected string
	}{
		{"disabled", security.ReasonDisabled, "DISABLED"},
		{"deprecated", security.ReasonDeprecated, "DEPRECATED"},
		{"eol", security.ReasonEOL, "EOL"},
		{"unmaintained", security.ReasonUnmaintained, "UNMAINTAINED"},
		{"unknown", security.DeprecationReason("unknown"), "unknown"},
		{"custom", security.DeprecationReason("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatDeprecationStatus(tc.reason)
			assert.Contains(t, result, tc.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// compliance.go tests
// ---------------------------------------------------------------------------

func TestBatch4_collectEvaluatedItems_NilResult(t *testing.T) {
	t.Parallel()

	items := collectEvaluatedItems(nil)
	assert.Nil(t, items)
}

func TestBatch4_collectEvaluatedItems_EmptyResult(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{}
	items := collectEvaluatedItems(result)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestBatch4_collectEvaluatedItems_WithInfoAndErrors(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{
		Info:   []string{"info1", "info2"},
		Errors: []string{"err1"},
	}

	items := collectEvaluatedItems(result)
	assert.Len(t, items, 3)
	assert.Contains(t, items, "info1")
	assert.Contains(t, items, "info2")
	assert.Contains(t, items, "err1")
}

func TestBatch4_collectEvaluatedItems_OnlyInfo(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{
		Info: []string{"item1", "item2", "item3"},
	}

	items := collectEvaluatedItems(result)
	assert.Len(t, items, 3)
}

func TestBatch4_collectEvaluatedItems_OnlyErrors(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{
		Errors: []string{"error1"},
	}

	items := collectEvaluatedItems(result)
	assert.Len(t, items, 1)
	assert.Equal(t, "error1", items[0])
}

func TestBatch4_outputComplianceError(t *testing.T) {
	out := captureStdout(t, func() {
		outputComplianceError(fmt.Errorf("test error"))
	})

	assert.Contains(t, out, "test error")
	assert.Contains(t, out, `"error"`)
}

func TestBatch4_outputComplianceError_LongMessage(t *testing.T) {
	out := captureStdout(t, func() {
		outputComplianceError(fmt.Errorf("detailed error: config file not found at /path/to/preflight.yaml"))
	})

	assert.Contains(t, out, "config file not found")
}

func TestBatch4_outputComplianceJSON_CompliantReport(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     10,
			PassedChecks:    10,
			ComplianceScore: 100,
		},
	}

	out := captureStdout(t, func() {
		outputComplianceJSON(report)
	})

	assert.Contains(t, out, "test-policy")
	assert.Contains(t, out, "compliant")
	assert.Contains(t, out, "100")
}

func TestBatch4_outputComplianceJSON_NonCompliantReport(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName:  "strict-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusNonCompliant,
			TotalChecks:     10,
			PassedChecks:    5,
			ViolationCount:  3,
			WarningCount:    2,
			ComplianceScore: 50,
		},
		Violations: []policy.ViolationDetail{
			{Type: "missing_required", Pattern: "git:*", Message: "Git config required"},
		},
	}

	out := captureStdout(t, func() {
		outputComplianceJSON(report)
	})

	assert.Contains(t, out, "strict-policy")
	assert.Contains(t, out, "non_compliant")
	assert.Contains(t, out, "missing_required")
}

func TestBatch4_outputComplianceText_CompliantReport(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     10,
			PassedChecks:    10,
			ComplianceScore: 100,
		},
	}

	out := captureStdout(t, func() {
		outputComplianceText(report)
	})

	assert.NotEmpty(t, out)
	assert.Contains(t, out, "test-policy")
	assert.Contains(t, out, "COMPLIANCE REPORT")
}

func TestBatch4_outputComplianceText_WithExpiringOverrides(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusWarning,
			TotalChecks:     10,
			PassedChecks:    8,
			OverrideCount:   1,
			ComplianceScore: 80,
		},
		Overrides: []policy.OverrideDetail{
			{
				Pattern:         "forbidden:*",
				Justification:   "legacy system",
				ApprovedBy:      "admin",
				ExpiresAt:       time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
				DaysUntilExpiry: 3,
			},
		},
	}

	out := captureStdout(t, func() {
		outputComplianceText(report)
	})

	assert.Contains(t, out, "override")
	assert.Contains(t, out, "expiring")
}

func TestBatch4_outputComplianceText_NoExpiringOverrides(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     10,
			PassedChecks:    10,
			ComplianceScore: 100,
		},
		Overrides: []policy.OverrideDetail{
			{
				Pattern:         "forbidden:*",
				Justification:   "legacy system",
				DaysUntilExpiry: -1, // No expiry
			},
		},
	}

	out := captureStdout(t, func() {
		outputComplianceText(report)
	})

	assert.NotContains(t, out, "expiring")
}
