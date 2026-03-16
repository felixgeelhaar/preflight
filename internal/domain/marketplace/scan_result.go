package marketplace

import (
	"fmt"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
)

// ScanStatus represents the scan state of a package.
type ScanStatus string

const (
	// ScanStatusUnscanned means the package has not been scanned.
	ScanStatusUnscanned ScanStatus = "unscanned"
	// ScanStatusClean means the scan found no vulnerabilities.
	ScanStatusClean ScanStatus = "clean"
	// ScanStatusWarning means vulnerabilities were found but below the block threshold.
	ScanStatusWarning ScanStatus = "warning"
	// ScanStatusBlocked means the scan found vulnerabilities that block installation.
	ScanStatusBlocked ScanStatus = "blocked"
)

// PackageScanResult caches scan results per package version.
type PackageScanResult struct {
	// PackageID identifies the scanned package.
	PackageID PackageID
	// Version is the package version that was scanned.
	Version string
	// Status is the overall scan status.
	Status ScanStatus
	// Scanner is the name of the scanner used.
	Scanner string
	// ScannedAt is when the scan was performed.
	ScannedAt time.Time
	// Summary contains aggregated vulnerability counts.
	Summary security.ScanSummary
	// Blocked indicates whether installation was blocked.
	Blocked bool
	// BlockReason explains why installation was blocked.
	BlockReason string
}

// NewPackageScanResult creates a new PackageScanResult with the current time.
func NewPackageScanResult(pkgID PackageID, version string, status ScanStatus, scanner string, summary security.ScanSummary) *PackageScanResult {
	return &PackageScanResult{
		PackageID: pkgID,
		Version:   version,
		Status:    status,
		Scanner:   scanner,
		ScannedAt: time.Now(),
		Summary:   summary,
	}
}

// IsClean returns true if the scan found no issues.
func (r *PackageScanResult) IsClean() bool {
	return r.Status == ScanStatusClean
}

// String returns a human-readable representation of the scan result.
func (r *PackageScanResult) String() string {
	return fmt.Sprintf("%s@%s: %s (scanner: %s, vulnerabilities: %d)",
		r.PackageID, r.Version, r.Status, r.Scanner, r.Summary.TotalVulnerabilities)
}
