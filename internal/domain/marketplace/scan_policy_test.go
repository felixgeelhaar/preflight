package marketplace

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultScanPolicy(t *testing.T) {
	t.Parallel()

	policy := DefaultScanPolicy()
	assert.True(t, policy.Enabled)
	assert.Equal(t, security.SeverityCritical, policy.BlockOnSeverity)
	assert.Empty(t, policy.SkipPatterns)
	assert.False(t, policy.TrustVerifiedPackages)
}

func TestNewScanPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		enabled         bool
		blockOnSeverity security.Severity
		skipPatterns    []string
		trustVerified   bool
		wantErr         bool
	}{
		{
			name:            "valid policy blocking on critical",
			enabled:         true,
			blockOnSeverity: security.SeverityCritical,
			wantErr:         false,
		},
		{
			name:            "valid policy blocking on high",
			enabled:         true,
			blockOnSeverity: security.SeverityHigh,
			wantErr:         false,
		},
		{
			name:            "valid disabled policy",
			enabled:         false,
			blockOnSeverity: security.SeverityCritical,
			wantErr:         false,
		},
		{
			name:            "valid with skip patterns",
			enabled:         true,
			blockOnSeverity: security.SeverityCritical,
			skipPatterns:    []string{"trusted-*", "internal-*"},
			wantErr:         false,
		},
		{
			name:            "valid with trust verified",
			enabled:         true,
			blockOnSeverity: security.SeverityCritical,
			trustVerified:   true,
			wantErr:         false,
		},
		{
			name:            "invalid empty severity",
			enabled:         true,
			blockOnSeverity: security.Severity(""),
			wantErr:         true,
		},
		{
			name:            "invalid unknown severity",
			enabled:         true,
			blockOnSeverity: security.Severity("invalid"),
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policy, err := NewScanPolicy(tt.enabled, tt.blockOnSeverity, tt.skipPatterns, tt.trustVerified)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.enabled, policy.Enabled)
				assert.Equal(t, tt.blockOnSeverity, policy.BlockOnSeverity)
				assert.Equal(t, tt.skipPatterns, policy.SkipPatterns)
				assert.Equal(t, tt.trustVerified, policy.TrustVerifiedPackages)
			}
		})
	}
}

func TestScanPolicy_ShouldScan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		policy   ScanPolicy
		pkgID    PackageID
		verified bool
		want     bool
	}{
		{
			name:   "enabled policy scans normal package",
			policy: DefaultScanPolicy(),
			pkgID:  MustNewPackageID("my-package"),
			want:   true,
		},
		{
			name: "disabled policy skips all",
			policy: ScanPolicy{
				Enabled:         false,
				BlockOnSeverity: security.SeverityCritical,
			},
			pkgID: MustNewPackageID("my-package"),
			want:  false,
		},
		{
			name: "skip pattern matches package",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityCritical,
				SkipPatterns:    []string{"trusted-*"},
			},
			pkgID: MustNewPackageID("trusted-config"),
			want:  false,
		},
		{
			name: "skip pattern does not match",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityCritical,
				SkipPatterns:    []string{"trusted-*"},
			},
			pkgID: MustNewPackageID("untrusted-config"),
			want:  true,
		},
		{
			name: "trust verified skips verified package",
			policy: ScanPolicy{
				Enabled:               true,
				BlockOnSeverity:       security.SeverityCritical,
				TrustVerifiedPackages: true,
			},
			pkgID:    MustNewPackageID("my-package"),
			verified: true,
			want:     false,
		},
		{
			name: "trust verified does not skip unverified",
			policy: ScanPolicy{
				Enabled:               true,
				BlockOnSeverity:       security.SeverityCritical,
				TrustVerifiedPackages: true,
			},
			pkgID:    MustNewPackageID("my-package"),
			verified: false,
			want:     true,
		},
		{
			name: "multiple skip patterns",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityCritical,
				SkipPatterns:    []string{"internal-*", "test-*"},
			},
			pkgID: MustNewPackageID("test-package"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.policy.ShouldScan(tt.pkgID, tt.verified)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScanPolicy_ShouldBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy ScanPolicy
		result *security.ScanResult
		want   bool
	}{
		{
			name:   "nil result does not block",
			policy: DefaultScanPolicy(),
			result: nil,
			want:   false,
		},
		{
			name:   "no vulnerabilities does not block",
			policy: DefaultScanPolicy(),
			result: &security.ScanResult{
				Scanner:         "test",
				Vulnerabilities: security.Vulnerabilities{},
			},
			want: false,
		},
		{
			name:   "critical vulnerability blocks with default policy",
			policy: DefaultScanPolicy(),
			result: &security.ScanResult{
				Scanner: "test",
				Vulnerabilities: security.Vulnerabilities{
					{ID: "CVE-2024-0001", Severity: security.SeverityCritical},
				},
			},
			want: true,
		},
		{
			name:   "high vulnerability does not block with default policy (blocks on critical)",
			policy: DefaultScanPolicy(),
			result: &security.ScanResult{
				Scanner: "test",
				Vulnerabilities: security.Vulnerabilities{
					{ID: "CVE-2024-0002", Severity: security.SeverityHigh},
				},
			},
			want: false,
		},
		{
			name: "high vulnerability blocks when policy blocks on high",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityHigh,
			},
			result: &security.ScanResult{
				Scanner: "test",
				Vulnerabilities: security.Vulnerabilities{
					{ID: "CVE-2024-0003", Severity: security.SeverityHigh},
				},
			},
			want: true,
		},
		{
			name: "medium vulnerability blocks when policy blocks on medium",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityMedium,
			},
			result: &security.ScanResult{
				Scanner: "test",
				Vulnerabilities: security.Vulnerabilities{
					{ID: "CVE-2024-0004", Severity: security.SeverityMedium},
				},
			},
			want: true,
		},
		{
			name: "low vulnerability does not block when policy blocks on high",
			policy: ScanPolicy{
				Enabled:         true,
				BlockOnSeverity: security.SeverityHigh,
			},
			result: &security.ScanResult{
				Scanner: "test",
				Vulnerabilities: security.Vulnerabilities{
					{ID: "CVE-2024-0005", Severity: security.SeverityLow},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.policy.ShouldBlock(tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}
