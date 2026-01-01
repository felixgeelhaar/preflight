package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
)

func TestSecurityCommand_Flags(t *testing.T) {
	t.Parallel()

	// Verify command exists and has expected flags
	assert.NotNil(t, securityCmd)
	assert.Equal(t, "security", securityCmd.Use)

	// Check flags exist
	flags := securityCmd.Flags()
	assert.NotNil(t, flags.Lookup("path"))
	assert.NotNil(t, flags.Lookup("scanner"))
	assert.NotNil(t, flags.Lookup("severity"))
	assert.NotNil(t, flags.Lookup("fail-on"))
	assert.NotNil(t, flags.Lookup("ignore"))
	assert.NotNil(t, flags.Lookup("json"))
	assert.NotNil(t, flags.Lookup("quiet"))
	assert.NotNil(t, flags.Lookup("list-scanners"))
}

func TestGetScanner_Auto(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	// Auto should return first available scanner (or error if none)
	scanner, err := getScanner(registry, "auto")

	// This test may pass or fail depending on whether grype/trivy is installed
	if err != nil {
		assert.Contains(t, err.Error(), "no scanners available")
	} else {
		assert.NotNil(t, scanner)
	}
}

func TestGetScanner_Specific(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())

	// Try to get a scanner that's not registered
	scanner, err := getScanner(registry, "unknown")
	assert.Error(t, err)
	assert.Nil(t, scanner)
	assert.Contains(t, err.Error(), "not available")
}

func TestShouldFail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		vulns      security.Vulnerabilities
		failOn     security.Severity
		shouldFail bool
	}{
		{
			name:       "no vulnerabilities",
			vulns:      security.Vulnerabilities{},
			failOn:     security.SeverityHigh,
			shouldFail: false,
		},
		{
			name: "critical when fail-on high",
			vulns: security.Vulnerabilities{
				{ID: "CVE-1", Severity: security.SeverityCritical},
			},
			failOn:     security.SeverityHigh,
			shouldFail: true,
		},
		{
			name: "high when fail-on high",
			vulns: security.Vulnerabilities{
				{ID: "CVE-1", Severity: security.SeverityHigh},
			},
			failOn:     security.SeverityHigh,
			shouldFail: true,
		},
		{
			name: "medium when fail-on high",
			vulns: security.Vulnerabilities{
				{ID: "CVE-1", Severity: security.SeverityMedium},
			},
			failOn:     security.SeverityHigh,
			shouldFail: false,
		},
		{
			name: "low when fail-on critical",
			vulns: security.Vulnerabilities{
				{ID: "CVE-1", Severity: security.SeverityLow},
			},
			failOn:     security.SeverityCritical,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &security.ScanResult{
				Vulnerabilities: tt.vulns,
			}

			got := shouldFail(result, tt.failOn)
			assert.Equal(t, tt.shouldFail, got)
		})
	}
}

func TestToVulnerabilitiesJSON(t *testing.T) {
	t.Parallel()

	vulns := security.Vulnerabilities{
		{
			ID:        "CVE-2024-1234",
			Package:   "openssl",
			Version:   "3.0.0",
			Severity:  security.SeverityCritical,
			CVSS:      9.8,
			FixedIn:   "3.0.1",
			Title:     "Critical OpenSSL vulnerability",
			Reference: "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
		},
	}

	result := toVulnerabilitiesJSON(vulns)

	assert.Len(t, result, 1)
	assert.Equal(t, "CVE-2024-1234", result[0].ID)
	assert.Equal(t, "openssl", result[0].Package)
	assert.Equal(t, "3.0.0", result[0].Version)
	assert.Equal(t, "critical", result[0].Severity)
	assert.InDelta(t, 9.8, result[0].CVSS, 0.001)
	assert.Equal(t, "3.0.1", result[0].FixedIn)
}

func TestFormatSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity security.Severity
		contains string
	}{
		{security.SeverityCritical, "CRITICAL"},
		{security.SeverityHigh, "HIGH"},
		{security.SeverityMedium, "MEDIUM"},
		{security.SeverityLow, "LOW"},
		{security.SeverityNegligible, "NEGLIGIBLE"},
		{security.SeverityUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			t.Parallel()

			result := formatSeverity(tt.severity)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestListScanners(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	// Just verify it doesn't panic
	err := listScanners(registry)
	assert.NoError(t, err)
}
