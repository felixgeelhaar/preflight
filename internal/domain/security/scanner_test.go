package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultScanOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultScanOptions()
	assert.Equal(t, SeverityMedium, opts.MinSeverity)
	assert.Equal(t, SeverityCritical, opts.FailOnSeverity)
	assert.Empty(t, opts.IgnoreIDs)
	assert.False(t, opts.Quiet)
	assert.False(t, opts.JSON)
}

func TestScanResult_Summary(t *testing.T) {
	t.Parallel()

	result := &ScanResult{
		Scanner: "grype",
		Version: "0.74.0",
		Vulnerabilities: Vulnerabilities{
			NewVulnerability("CVE-1", "pkg1").WithSeverity(SeverityCritical).Build(),
			NewVulnerability("CVE-2", "pkg2").WithSeverity(SeverityCritical).Build(),
			NewVulnerability("CVE-3", "pkg3").WithSeverity(SeverityHigh).Build(),
			NewVulnerability("CVE-4", "pkg4").WithSeverity(SeverityMedium).WithFixedIn("1.0.1").Build(),
			NewVulnerability("CVE-5", "pkg5").WithSeverity(SeverityLow).Build(),
		},
		PackagesScanned: 100,
	}

	summary := result.Summary()
	assert.Equal(t, 5, summary.TotalVulnerabilities)
	assert.Equal(t, 2, summary.Critical)
	assert.Equal(t, 1, summary.High)
	assert.Equal(t, 1, summary.Medium)
	assert.Equal(t, 1, summary.Low)
	assert.Equal(t, 0, summary.Negligible)
	assert.Equal(t, 100, summary.PackagesScanned)
	assert.Equal(t, 1, summary.FixableCount)
}

func TestScanResult_HasVulnerabilities(t *testing.T) {
	t.Parallel()

	withVulns := &ScanResult{
		Vulnerabilities: Vulnerabilities{
			NewVulnerability("CVE-1", "pkg1").Build(),
		},
	}
	assert.True(t, withVulns.HasVulnerabilities())

	withoutVulns := &ScanResult{
		Vulnerabilities: Vulnerabilities{},
	}
	assert.False(t, withoutVulns.HasVulnerabilities())
}

func TestScanResult_HasCritical(t *testing.T) {
	t.Parallel()

	withCritical := &ScanResult{
		Vulnerabilities: Vulnerabilities{
			NewVulnerability("CVE-1", "pkg1").WithSeverity(SeverityCritical).Build(),
		},
	}
	assert.True(t, withCritical.HasCritical())

	withoutCritical := &ScanResult{
		Vulnerabilities: Vulnerabilities{
			NewVulnerability("CVE-1", "pkg1").WithSeverity(SeverityHigh).Build(),
		},
	}
	assert.False(t, withoutCritical.HasCritical())
}

func TestScannerRegistry(t *testing.T) {
	t.Parallel()

	registry := NewScannerRegistry()
	assert.Empty(t, registry.Names())

	// Create mock scanners
	mockAvailable := &mockScanner{name: "available", available: true}
	mockUnavailable := &mockScanner{name: "unavailable", available: false}

	registry.Register(mockAvailable)
	registry.Register(mockUnavailable)

	// Test Names
	names := registry.Names()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "available")
	assert.Contains(t, names, "unavailable")

	// Test Available
	available := registry.Available()
	assert.Len(t, available, 1)
	assert.Equal(t, "available", available[0].Name())

	// Test AvailableNames
	availableNames := registry.AvailableNames()
	assert.Len(t, availableNames, 1)
	assert.Equal(t, "available", availableNames[0])

	// Test Get
	assert.NotNil(t, registry.Get("available"))
	assert.Nil(t, registry.Get("unavailable"))
	assert.Nil(t, registry.Get("nonexistent"))

	// Test First
	first := registry.First()
	assert.NotNil(t, first)
	assert.Equal(t, "available", first.Name())
}

func TestScannerRegistry_Empty(t *testing.T) {
	t.Parallel()

	registry := NewScannerRegistry()

	assert.Empty(t, registry.Available())
	assert.Empty(t, registry.AvailableNames())
	assert.Nil(t, registry.Get("anything"))
	assert.Nil(t, registry.First())
}

func TestScanSummary_String(t *testing.T) {
	t.Parallel()

	noVulns := ScanSummary{TotalVulnerabilities: 0}
	assert.Equal(t, "No vulnerabilities found", noVulns.String())

	withVulns := ScanSummary{TotalVulnerabilities: 5}
	assert.Equal(t, "", withVulns.String())
}

// mockScanner is a test double for Scanner interface.
type mockScanner struct {
	name      string
	available bool
}

func (m *mockScanner) Name() string {
	return m.name
}

func (m *mockScanner) Version(_ context.Context) (string, error) {
	return "1.0.0", nil
}

func (m *mockScanner) Available() bool {
	return m.available
}

func (m *mockScanner) Scan(_ context.Context, _ ScanTarget, _ ScanOptions) (*ScanResult, error) {
	return &ScanResult{}, nil
}
