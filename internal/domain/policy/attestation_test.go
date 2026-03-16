package policy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComplianceAttestation(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	assert.Equal(t, report, att.Report)
	assert.Equal(t, "machine-001", att.MachineID)
	assert.Equal(t, "dev-laptop", att.Hostname)
	assert.False(t, att.AttestedAt.IsZero())
	assert.NotEmpty(t, att.ContentDigest)
	assert.Empty(t, att.Signature)
	assert.Empty(t, att.SignatureType)
	assert.Empty(t, att.SignerIdentity)
}

func TestNewComplianceAttestation_NilReport(t *testing.T) {
	t.Parallel()

	att, err := NewComplianceAttestation(nil, "machine-001", "dev-laptop")
	assert.Nil(t, att)
	assert.ErrorIs(t, err, ErrAttestationInvalid)
}

func TestNewComplianceAttestation_EmptyMachineID(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{PolicyName: "test"}
	att, err := NewComplianceAttestation(report, "", "dev-laptop")
	assert.Nil(t, att)
	assert.ErrorIs(t, err, ErrAttestationInvalid)
}

func TestComplianceAttestation_Digest(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	digest := att.Digest()
	assert.NotEmpty(t, digest)
	assert.Len(t, digest, 64) // SHA256 hex is 64 chars

	// Digest should be deterministic
	assert.Equal(t, digest, att.Digest())
}

func TestComplianceAttestation_Digest_DifferentInputs(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att1, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	att2, err := NewComplianceAttestation(report, "machine-002", "dev-laptop")
	require.NoError(t, err)

	assert.NotEqual(t, att1.Digest(), att2.Digest())
}

func TestComplianceAttestation_IsSigned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{
			name:      "unsigned",
			signature: "",
			want:      false,
		},
		{
			name:      "signed",
			signature: "abc123",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			att := &ComplianceAttestation{
				Signature: tt.signature,
			}
			assert.Equal(t, tt.want, att.IsSigned())
		})
	}
}

func TestComplianceAttestation_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		att     *ComplianceAttestation
		wantErr error
	}{
		{
			name: "valid attestation",
			att: &ComplianceAttestation{
				Report:     &ComplianceReport{PolicyName: "test"},
				MachineID:  "machine-001",
				Hostname:   "dev-laptop",
				AttestedAt: time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "nil report",
			att: &ComplianceAttestation{
				Report:     nil,
				MachineID:  "machine-001",
				AttestedAt: time.Now(),
			},
			wantErr: ErrAttestationInvalid,
		},
		{
			name: "empty machine ID",
			att: &ComplianceAttestation{
				Report:     &ComplianceReport{PolicyName: "test"},
				MachineID:  "",
				AttestedAt: time.Now(),
			},
			wantErr: ErrAttestationInvalid,
		},
		{
			name: "zero attested at",
			att: &ComplianceAttestation{
				Report:    &ComplianceReport{PolicyName: "test"},
				MachineID: "machine-001",
			},
			wantErr: ErrAttestationInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.att.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComplianceAttestation_ContentDigest_SetOnCreation(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
	}

	att, err := NewComplianceAttestation(report, "machine-001", "dev-laptop")
	require.NoError(t, err)

	// ContentDigest should match Digest()
	assert.Equal(t, att.Digest(), att.ContentDigest)
}
