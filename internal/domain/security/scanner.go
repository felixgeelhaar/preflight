package security

import (
	"context"
	"errors"
)

// Scanner errors.
var (
	ErrScannerNotAvailable = errors.New("scanner is not available")
	ErrScanFailed          = errors.New("scan failed")
	ErrNoScannersFound     = errors.New("no security scanners found")
)

// ScanTarget represents what to scan.
type ScanTarget struct {
	// Type is the target type (e.g., "directory", "sbom", "image").
	Type string

	// Path is the path to scan (directory, file, or image reference).
	Path string

	// Packages is a list of package names to scan (optional).
	Packages []string

	// Provider filters results to a specific package manager.
	Provider string
}

// ScanOptions configures the scan behavior.
type ScanOptions struct {
	// MinSeverity filters vulnerabilities below this severity.
	MinSeverity Severity

	// IgnoreIDs is a list of vulnerability IDs to ignore.
	IgnoreIDs []string

	// FailOnSeverity causes the scan to fail if vulnerabilities of this severity or higher are found.
	FailOnSeverity Severity

	// Quiet reduces output verbosity.
	Quiet bool

	// JSON outputs results in JSON format.
	JSON bool
}

// DefaultScanOptions returns default scan options.
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		MinSeverity:    SeverityMedium,
		FailOnSeverity: SeverityCritical,
	}
}

// Scanner defines the interface for vulnerability scanners.
type Scanner interface {
	// Name returns the scanner name (e.g., "grype", "trivy").
	Name() string

	// Version returns the scanner version.
	Version(ctx context.Context) (string, error)

	// Available returns true if the scanner is installed and available.
	Available() bool

	// Scan performs a vulnerability scan on the target.
	Scan(ctx context.Context, target ScanTarget, opts ScanOptions) (*ScanResult, error)
}

// ScanResult contains the results of a vulnerability scan.
type ScanResult struct {
	// Scanner is the name of the scanner used.
	Scanner string

	// Version is the scanner version.
	Version string

	// Vulnerabilities found during the scan.
	Vulnerabilities Vulnerabilities

	// PackagesScanned is the number of packages scanned.
	PackagesScanned int

	// Error contains any error message from the scan.
	Error string
}

// Summary returns a summary of the scan result.
func (r *ScanResult) Summary() ScanSummary {
	counts := r.Vulnerabilities.CountBySeverity()
	return ScanSummary{
		TotalVulnerabilities: len(r.Vulnerabilities),
		Critical:             counts[SeverityCritical],
		High:                 counts[SeverityHigh],
		Medium:               counts[SeverityMedium],
		Low:                  counts[SeverityLow],
		Negligible:           counts[SeverityNegligible],
		PackagesScanned:      r.PackagesScanned,
		FixableCount:         len(r.Vulnerabilities.Fixable()),
	}
}

// HasVulnerabilities returns true if vulnerabilities were found.
func (r *ScanResult) HasVulnerabilities() bool {
	return len(r.Vulnerabilities) > 0
}

// HasCritical returns true if critical vulnerabilities were found.
func (r *ScanResult) HasCritical() bool {
	return r.Vulnerabilities.HasCritical()
}

// ScanSummary provides a summary of scan results.
type ScanSummary struct {
	TotalVulnerabilities int
	Critical             int
	High                 int
	Medium               int
	Low                  int
	Negligible           int
	PackagesScanned      int
	FixableCount         int
}

// String returns a string representation of the summary.
func (s ScanSummary) String() string {
	if s.TotalVulnerabilities == 0 {
		return "No vulnerabilities found"
	}
	return ""
}

// ScannerRegistry manages available scanners.
type ScannerRegistry struct {
	scanners []Scanner
}

// NewScannerRegistry creates a new scanner registry.
func NewScannerRegistry() *ScannerRegistry {
	return &ScannerRegistry{
		scanners: make([]Scanner, 0),
	}
}

// Register adds a scanner to the registry.
func (r *ScannerRegistry) Register(s Scanner) {
	r.scanners = append(r.scanners, s)
}

// Available returns all available scanners.
func (r *ScannerRegistry) Available() []Scanner {
	available := make([]Scanner, 0)
	for _, s := range r.scanners {
		if s.Available() {
			available = append(available, s)
		}
	}
	return available
}

// Get returns a scanner by name, or nil if not found/available.
func (r *ScannerRegistry) Get(name string) Scanner {
	for _, s := range r.scanners {
		if s.Name() == name && s.Available() {
			return s
		}
	}
	return nil
}

// First returns the first available scanner.
func (r *ScannerRegistry) First() Scanner {
	available := r.Available()
	if len(available) == 0 {
		return nil
	}
	return available[0]
}

// Names returns the names of all registered scanners.
func (r *ScannerRegistry) Names() []string {
	names := make([]string, len(r.scanners))
	for i, s := range r.scanners {
		names[i] = s.Name()
	}
	return names
}

// AvailableNames returns the names of available scanners.
func (r *ScannerRegistry) AvailableNames() []string {
	available := r.Available()
	names := make([]string, len(available))
	for i, s := range available {
		names[i] = s.Name()
	}
	return names
}
