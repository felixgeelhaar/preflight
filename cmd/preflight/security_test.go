package main

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
)

func TestOutputSecurityJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputSecurityJSON(nil, errors.New("scan failed"))
	})

	assert.Contains(t, output, `"error"`)
	assert.Contains(t, output, "scan failed")
}

func TestOutputSecurityJSON_WithResult(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "grype",
		Version:         "0.104.3",
		PackagesScanned: 1,
		Vulnerabilities: security.Vulnerabilities{
			{
				ID:       "CVE-2024-0001",
				Package:  "pkg",
				Version:  "1.0.0",
				Severity: security.SeverityHigh,
			},
		},
	}

	output := captureStdout(t, func() {
		outputSecurityJSON(result, nil)
	})

	assert.Contains(t, output, `"scanner": "grype"`)
	assert.Contains(t, output, `"TotalVulnerabilities": 1`)
}

func TestOutputSecurityText_NoVulnerabilities(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "trivy",
		Version:         "1.2.3",
		PackagesScanned: 5,
	}

	output := captureStdout(t, func() {
		outputSecurityText(result, security.ScanOptions{Quiet: true})
	})

	assert.Contains(t, output, "No vulnerabilities found")
	assert.Contains(t, output, "Packages scanned: 5")
}

func TestOutputSecurityText_WithVulnerabilities(t *testing.T) {
	result := &security.ScanResult{
		Scanner:         "grype",
		Version:         "0.104.3",
		PackagesScanned: 2,
		Vulnerabilities: security.Vulnerabilities{
			{
				ID:       "CVE-2024-9999",
				Package:  "pkg",
				Version:  "1.0.0",
				Severity: security.SeverityCritical,
				FixedIn:  "1.0.2",
			},
		},
	}

	output := captureStdout(t, func() {
		outputSecurityText(result, security.ScanOptions{Quiet: false})
	})

	assert.Contains(t, output, "Security Scan Results")
	assert.Contains(t, output, "Summary: 1 vulnerabilities found")
	assert.Contains(t, output, "CRITICAL")
	assert.Contains(t, output, "Fixable")
	assert.Contains(t, output, "Recommendations")
}

func TestFormatSeverity(t *testing.T) {
	cases := []struct {
		severity security.Severity
		contains string
	}{
		{severity: security.SeverityCritical, contains: "CRITICAL"},
		{severity: security.SeverityHigh, contains: "HIGH"},
		{severity: security.SeverityMedium, contains: "MEDIUM"},
		{severity: security.SeverityLow, contains: "LOW"},
		{severity: security.SeverityNegligible, contains: "NEGLIGIBLE"},
		{severity: security.SeverityUnknown, contains: "unknown"},
	}

	for _, tc := range cases {
		got := formatSeverity(tc.severity)
		assert.Contains(t, got, tc.contains)
	}
}

func TestGetScanner_Auto(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "test", version: "1.0", available: true})

	scanner, err := getScanner(registry, "auto")
	assert.NoError(t, err)
	assert.Equal(t, "test", scanner.Name())
}

func TestGetScanner_NotAvailable(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "other", available: false})

	_, err := getScanner(registry, "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestListScanners(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "scan", version: "1", available: true})

	output := captureStdout(t, func() {
		err := listScanners(registry)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Available security scanners")
	assert.Contains(t, output, "scan")
	assert.Contains(t, output, "available (v1)")
}

func TestShouldFail(t *testing.T) {
	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{
			{Severity: security.SeverityHigh},
		},
	}

	assert.True(t, shouldFail(result, security.SeverityMedium))
	assert.False(t, shouldFail(result, security.SeverityCritical))
}

type fakeScanner struct {
	name      string
	version   string
	available bool
}

func (s *fakeScanner) Name() string {
	return s.name
}

func (s *fakeScanner) Version(_ context.Context) (string, error) {
	if s.version == "" {
		return "", errors.New("no version")
	}
	return s.version, nil
}

func (s *fakeScanner) Available() bool {
	return s.available
}

func (s *fakeScanner) Scan(context.Context, security.ScanTarget, security.ScanOptions) (*security.ScanResult, error) {
	return nil, errors.New("not implemented")
}
