package marketplace

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScanner struct {
	name      string
	available bool
	result    *security.ScanResult
	err       error
}

func (m *mockScanner) Name() string                              { return m.name }
func (m *mockScanner) Version(_ context.Context) (string, error) { return "1.0.0", nil }
func (m *mockScanner) Available() bool                           { return m.available }
func (m *mockScanner) Scan(_ context.Context, _ security.ScanTarget, _ security.ScanOptions) (*security.ScanResult, error) {
	return m.result, m.err
}

func TestNewScanner(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	policy := DefaultScanPolicy()

	scanner := NewScanner(registry, policy)
	require.NotNil(t, scanner)
}

func TestScanner_ScanPackage_SkippedByPolicy(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{name: "nox", available: true})

	policy := ScanPolicy{
		Enabled:         false,
		BlockOnSeverity: security.SeverityCritical,
	}

	scanner := NewScanner(registry, policy)
	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("my-pkg"), "1.0.0", []byte("data"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusUnscanned, result.Status)
	assert.False(t, result.Blocked)
}

func TestScanner_ScanPackage_SkippedByPattern(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{name: "nox", available: true})

	policy := ScanPolicy{
		Enabled:         true,
		BlockOnSeverity: security.SeverityCritical,
		SkipPatterns:    []string{"trusted-*"},
	}

	scanner := NewScanner(registry, policy)
	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("trusted-config"), "1.0.0", []byte("data"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusUnscanned, result.Status)
}

func TestScanner_ScanPackage_NoScannerAvailable(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{name: "nox", available: false})

	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("my-pkg"), "1.0.0", []byte("data"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusUnscanned, result.Status)
	assert.False(t, result.Blocked)
}

func TestScanner_ScanPackage_EmptyRegistry(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("my-pkg"), "1.0.0", []byte("data"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusUnscanned, result.Status)
}

func TestScanner_ScanPackage_CleanResult(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "nox",
		available: true,
		result: &security.ScanResult{
			Scanner:         "nox",
			Version:         "0.7.0",
			Vulnerabilities: security.Vulnerabilities{},
			PackagesScanned: 5,
		},
	})

	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("safe-pkg"), "1.0.0", []byte("safe content"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusClean, result.Status)
	assert.Equal(t, "nox", result.Scanner)
	assert.False(t, result.Blocked)
	assert.Equal(t, 0, result.Summary.TotalVulnerabilities)
}

func TestScanner_ScanPackage_WarningResult(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "trivy",
		available: true,
		result: &security.ScanResult{
			Scanner: "trivy",
			Version: "1.0.0",
			Vulnerabilities: security.Vulnerabilities{
				{ID: "CVE-2024-0001", Severity: security.SeverityMedium},
				{ID: "CVE-2024-0002", Severity: security.SeverityLow},
			},
			PackagesScanned: 10,
		},
	})

	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("warn-pkg"), "1.0.0", []byte("content"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusWarning, result.Status)
	assert.False(t, result.Blocked)
	assert.Equal(t, 2, result.Summary.TotalVulnerabilities)
}

func TestScanner_ScanPackage_BlockedResult(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "nox",
		available: true,
		result: &security.ScanResult{
			Scanner: "nox",
			Version: "0.7.0",
			Vulnerabilities: security.Vulnerabilities{
				{ID: "CVE-2024-9999", Severity: security.SeverityCritical, Title: "RCE vulnerability"},
			},
			PackagesScanned: 3,
		},
	})

	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("bad-pkg"), "1.0.0", []byte("bad content"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusBlocked, result.Status)
	assert.True(t, result.Blocked)
	assert.NotEmpty(t, result.BlockReason)
	assert.Equal(t, 1, result.Summary.Critical)
}

func TestScanner_ScanPackage_ScannerError(t *testing.T) {
	t.Parallel()

	scanErr := errors.New("scanner exploded")
	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "nox",
		available: true,
		result:    nil,
		err:       scanErr,
	})

	policy := DefaultScanPolicy()
	scanner := NewScanner(registry, policy)

	_, err := scanner.ScanPackage(context.Background(), MustNewPackageID("my-pkg"), "1.0.0", []byte("data"))

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrScanFailed)
}

func TestScanner_ScanPackage_TrustVerified(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "nox",
		available: true,
		result: &security.ScanResult{
			Scanner:         "nox",
			Version:         "0.7.0",
			Vulnerabilities: security.Vulnerabilities{},
			PackagesScanned: 1,
		},
	})

	policy := ScanPolicy{
		Enabled:               true,
		BlockOnSeverity:       security.SeverityCritical,
		TrustVerifiedPackages: true,
	}

	scanner := NewScanner(registry, policy)

	// ScanPackage does not take a verified param directly; that is checked at the service layer.
	// With TrustVerifiedPackages but verified=false (default), scanning still proceeds.
	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("my-pkg"), "1.0.0", []byte("data"))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusClean, result.Status)
}

func TestScanner_ScanPackage_BlockedOnHigh(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&mockScanner{
		name:      "grype",
		available: true,
		result: &security.ScanResult{
			Scanner: "grype",
			Version: "0.74.0",
			Vulnerabilities: security.Vulnerabilities{
				{ID: "CVE-2024-0010", Severity: security.SeverityHigh},
			},
			PackagesScanned: 5,
		},
	})

	policy := ScanPolicy{
		Enabled:         true,
		BlockOnSeverity: security.SeverityHigh,
	}

	scanner := NewScanner(registry, policy)
	result, err := scanner.ScanPackage(context.Background(), MustNewPackageID("risky-pkg"), "1.0.0", []byte("data"))

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScanStatusBlocked, result.Status)
	assert.True(t, result.Blocked)
}
