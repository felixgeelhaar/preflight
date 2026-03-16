package marketplace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
)

// Scanner bridges the security.Scanner with the marketplace install flow.
type Scanner struct {
	registry *security.ScannerRegistry
	policy   ScanPolicy
}

// NewScanner creates a new Scanner.
func NewScanner(registry *security.ScannerRegistry, policy ScanPolicy) *Scanner {
	return &Scanner{
		registry: registry,
		policy:   policy,
	}
}

// ScanPackage scans package data before installation.
// It returns an unscanned result (not an error) when scanning is skipped or no scanner is available.
func (ms *Scanner) ScanPackage(ctx context.Context, pkgID PackageID, version string, data []byte) (*PackageScanResult, error) {
	// Check if scanning should be skipped per policy.
	if !ms.policy.ShouldScan(pkgID, false) {
		return NewPackageScanResult(pkgID, version, ScanStatusUnscanned, "", security.ScanSummary{}), nil
	}

	// Find an available scanner from registry.
	scanner := ms.registry.First()
	if scanner == nil {
		return NewPackageScanResult(pkgID, version, ScanStatusUnscanned, "", security.ScanSummary{}), nil
	}

	// Write data to a temp directory for scanning.
	tmpDir, err := os.MkdirTemp("", "preflight-scan-*")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create temp dir: %w", ErrScanFailed, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pkgFile := filepath.Join(tmpDir, "package.tar.gz")
	if err := os.WriteFile(pkgFile, data, 0o600); err != nil {
		return nil, fmt.Errorf("%w: failed to write package data: %w", ErrScanFailed, err)
	}

	// Run scan.
	target := security.ScanTarget{
		Type: "directory",
		Path: tmpDir,
	}
	opts := security.ScanOptions{
		MinSeverity:    security.SeverityLow,
		FailOnSeverity: ms.policy.BlockOnSeverity,
	}

	scanResult, err := scanner.Scan(ctx, target, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScanFailed, err)
	}

	// Build PackageScanResult from ScanResult.
	summary := scanResult.Summary()
	status := determineScanStatus(scanResult, ms.policy)

	result := NewPackageScanResult(pkgID, version, status, scanner.Name(), summary)

	// Check if result should block installation.
	if ms.policy.ShouldBlock(scanResult) {
		result.Status = ScanStatusBlocked
		result.Blocked = true
		result.BlockReason = fmt.Sprintf(
			"vulnerabilities at or above %s severity found: %d critical, %d high",
			ms.policy.BlockOnSeverity, summary.Critical, summary.High,
		)
	}

	return result, nil
}

// determineScanStatus derives the scan status from the scan result.
func determineScanStatus(result *security.ScanResult, policy ScanPolicy) ScanStatus {
	if !result.HasVulnerabilities() {
		return ScanStatusClean
	}

	if policy.ShouldBlock(result) {
		return ScanStatusBlocked
	}

	return ScanStatusWarning
}
