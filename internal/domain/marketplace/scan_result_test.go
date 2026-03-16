package marketplace

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanStatus_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ScanStatusUnscanned, ScanStatus("unscanned"))
	assert.Equal(t, ScanStatusClean, ScanStatus("clean"))
	assert.Equal(t, ScanStatusWarning, ScanStatus("warning"))
	assert.Equal(t, ScanStatusBlocked, ScanStatus("blocked"))
}

func TestNewPackageScanResult(t *testing.T) {
	t.Parallel()

	pkgID := MustNewPackageID("test-package")
	version := "1.0.0"
	scannerName := "grype"
	summary := security.ScanSummary{
		TotalVulnerabilities: 0,
		PackagesScanned:      10,
	}

	result := NewPackageScanResult(pkgID, version, ScanStatusClean, scannerName, summary)

	require.NotNil(t, result)
	assert.True(t, pkgID.Equals(result.PackageID))
	assert.Equal(t, version, result.Version)
	assert.Equal(t, ScanStatusClean, result.Status)
	assert.Equal(t, scannerName, result.Scanner)
	assert.False(t, result.ScannedAt.IsZero())
	assert.Equal(t, summary, result.Summary)
	assert.False(t, result.Blocked)
	assert.Empty(t, result.BlockReason)
}

func TestNewPackageScanResult_Blocked(t *testing.T) {
	t.Parallel()

	pkgID := MustNewPackageID("unsafe-package")
	summary := security.ScanSummary{
		TotalVulnerabilities: 3,
		Critical:             1,
		High:                 2,
		PackagesScanned:      5,
	}

	result := NewPackageScanResult(pkgID, "2.0.0", ScanStatusBlocked, "trivy", summary)
	result.Blocked = true
	result.BlockReason = "critical vulnerabilities found"

	assert.Equal(t, ScanStatusBlocked, result.Status)
	assert.True(t, result.Blocked)
	assert.Equal(t, "critical vulnerabilities found", result.BlockReason)
}

func TestPackageScanResult_IsClean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status ScanStatus
		want   bool
	}{
		{"clean status", ScanStatusClean, true},
		{"unscanned status", ScanStatusUnscanned, false},
		{"warning status", ScanStatusWarning, false},
		{"blocked status", ScanStatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := &PackageScanResult{
				PackageID: MustNewPackageID("test"),
				Status:    tt.status,
				ScannedAt: time.Now(),
			}
			assert.Equal(t, tt.want, result.IsClean())
		})
	}
}

func TestPackageScanResult_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   *PackageScanResult
		contains []string
	}{
		{
			name: "clean result",
			result: &PackageScanResult{
				PackageID: MustNewPackageID("my-pkg"),
				Version:   "1.0.0",
				Status:    ScanStatusClean,
				Scanner:   "grype",
			},
			contains: []string{"my-pkg", "1.0.0", "clean"},
		},
		{
			name: "blocked result",
			result: &PackageScanResult{
				PackageID:   MustNewPackageID("bad-pkg"),
				Version:     "2.0.0",
				Status:      ScanStatusBlocked,
				Scanner:     "trivy",
				Blocked:     true,
				BlockReason: "critical vulnerabilities",
			},
			contains: []string{"bad-pkg", "2.0.0", "blocked"},
		},
		{
			name: "unscanned result",
			result: &PackageScanResult{
				PackageID: MustNewPackageID("new-pkg"),
				Version:   "1.0.0",
				Status:    ScanStatusUnscanned,
			},
			contains: []string{"new-pkg", "1.0.0", "unscanned"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			str := tt.result.String()
			for _, substr := range tt.contains {
				assert.Contains(t, str, substr)
			}
		})
	}
}
