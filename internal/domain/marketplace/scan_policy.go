package marketplace

import (
	"fmt"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
)

// ScanPolicy controls how security scanning behaves for marketplace packages.
type ScanPolicy struct {
	// Enabled controls whether scanning is performed.
	Enabled bool
	// BlockOnSeverity blocks installation if vulnerabilities at or above this severity are found.
	BlockOnSeverity security.Severity
	// SkipPatterns is a list of package ID patterns to skip scanning for.
	SkipPatterns []string
	// TrustVerifiedPackages skips scanning for packages with verified signatures.
	TrustVerifiedPackages bool
}

// validBlockSeverities are the severity levels that can be used as block thresholds.
var validBlockSeverities = map[security.Severity]bool{
	security.SeverityCritical:   true,
	security.SeverityHigh:       true,
	security.SeverityMedium:     true,
	security.SeverityLow:        true,
	security.SeverityNegligible: true,
}

// DefaultScanPolicy returns the default scan policy: enabled, blocks on critical.
func DefaultScanPolicy() ScanPolicy {
	return ScanPolicy{
		Enabled:               true,
		BlockOnSeverity:       security.SeverityCritical,
		SkipPatterns:          nil,
		TrustVerifiedPackages: false,
	}
}

// NewScanPolicy creates a new ScanPolicy with validation.
func NewScanPolicy(enabled bool, blockOnSeverity security.Severity, skipPatterns []string, trustVerified bool) (ScanPolicy, error) {
	if enabled {
		if !validBlockSeverities[blockOnSeverity] {
			return ScanPolicy{}, fmt.Errorf("invalid block severity: %q", blockOnSeverity)
		}
	}

	return ScanPolicy{
		Enabled:               enabled,
		BlockOnSeverity:       blockOnSeverity,
		SkipPatterns:          skipPatterns,
		TrustVerifiedPackages: trustVerified,
	}, nil
}

// ShouldScan returns true if the given package should be scanned.
func (p ScanPolicy) ShouldScan(pkgID PackageID, verified bool) bool {
	if !p.Enabled {
		return false
	}

	if p.TrustVerifiedPackages && verified {
		return false
	}

	for _, pattern := range p.SkipPatterns {
		if matched, _ := filepath.Match(pattern, pkgID.String()); matched {
			return false
		}
	}

	return true
}

// ShouldBlock returns true if the scan result should block installation.
func (p ScanPolicy) ShouldBlock(result *security.ScanResult) bool {
	if result == nil {
		return false
	}

	for _, v := range result.Vulnerabilities {
		if v.Severity.IsAtLeast(p.BlockOnSeverity) {
			return true
		}
	}

	return false
}
