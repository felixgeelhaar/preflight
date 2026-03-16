package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalKeyAttester(t *testing.T) {
	t.Parallel()

	attester := NewLocalKeyAttester("/path/to/key")
	assert.NotNil(t, attester)
	assert.Equal(t, "local-key", attester.Name())
}

func TestLocalKeyAttester_Sign(t *testing.T) {
	t.Parallel()

	// Create a temporary key file
	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "signing.key")
	err := os.WriteFile(keyPath, []byte("test-secret-key-content"), 0600)
	require.NoError(t, err)

	attester := NewLocalKeyAttester(keyPath)

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

	ctx := context.Background()
	err = attester.Sign(ctx, att)
	require.NoError(t, err)

	assert.True(t, att.IsSigned())
	assert.Equal(t, "local", att.SignatureType)
	assert.NotEmpty(t, att.Signature)
	assert.NotEmpty(t, att.SignerIdentity)
}

func TestLocalKeyAttester_Sign_KeyNotFound(t *testing.T) {
	t.Parallel()

	attester := NewLocalKeyAttester("/nonexistent/key")

	att := &ComplianceAttestation{
		Report:        &ComplianceReport{PolicyName: "test"},
		MachineID:     "machine-001",
		AttestedAt:    time.Now(),
		ContentDigest: "abc123",
	}

	ctx := context.Background()
	err := attester.Sign(ctx, att)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestLocalKeyAttester_Verify_ValidSignature(t *testing.T) {
	t.Parallel()

	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "signing.key")
	err := os.WriteFile(keyPath, []byte("test-secret-key-content"), 0600)
	require.NoError(t, err)

	attester := NewLocalKeyAttester(keyPath)

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

	ctx := context.Background()
	err = attester.Sign(ctx, att)
	require.NoError(t, err)

	err = attester.Verify(ctx, att)
	assert.NoError(t, err)
}

func TestLocalKeyAttester_Verify_InvalidSignature(t *testing.T) {
	t.Parallel()

	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "signing.key")
	err := os.WriteFile(keyPath, []byte("test-secret-key-content"), 0600)
	require.NoError(t, err)

	attester := NewLocalKeyAttester(keyPath)

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

	att.Signature = "tampered-signature"
	att.SignatureType = "local"

	ctx := context.Background()
	err = attester.Verify(ctx, att)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestLocalKeyAttester_Verify_UnsignedAttestation(t *testing.T) {
	t.Parallel()

	keyDir := t.TempDir()
	keyPath := filepath.Join(keyDir, "signing.key")
	err := os.WriteFile(keyPath, []byte("test-secret-key-content"), 0600)
	require.NoError(t, err)

	attester := NewLocalKeyAttester(keyPath)

	att := &ComplianceAttestation{
		Report:        &ComplianceReport{PolicyName: "test"},
		MachineID:     "machine-001",
		AttestedAt:    time.Now(),
		ContentDigest: "abc123",
	}

	ctx := context.Background()
	err = attester.Verify(ctx, att)
	assert.ErrorIs(t, err, ErrAttestationUnsigned)
}

func TestNewSigstoreAttester(t *testing.T) {
	t.Parallel()

	attester := NewSigstoreAttester()
	assert.NotNil(t, attester)
	assert.Equal(t, "sigstore", attester.Name())
}

func TestSigstoreAttester_Sign_Unavailable(t *testing.T) {
	t.Parallel()

	attester := NewSigstoreAttester()

	// If cosign is not installed, Sign should fail with ErrAttesterUnavailable
	if !attester.Available() {
		att := &ComplianceAttestation{
			Report:        &ComplianceReport{PolicyName: "test"},
			MachineID:     "machine-001",
			AttestedAt:    time.Now(),
			ContentDigest: "abc123",
		}

		ctx := context.Background()
		err := attester.Sign(ctx, att)
		assert.ErrorIs(t, err, ErrAttesterUnavailable)
	}
}

func TestSigstoreAttester_Verify_Unavailable(t *testing.T) {
	t.Parallel()

	attester := NewSigstoreAttester()

	if !attester.Available() {
		att := &ComplianceAttestation{
			Report:        &ComplianceReport{PolicyName: "test"},
			MachineID:     "machine-001",
			AttestedAt:    time.Now(),
			Signature:     "some-sig",
			SignatureType: "sigstore",
			ContentDigest: "abc123",
		}

		ctx := context.Background()
		err := attester.Verify(ctx, att)
		assert.ErrorIs(t, err, ErrAttesterUnavailable)
	}
}

func TestSigstoreAttester_Available(t *testing.T) {
	t.Parallel()

	attester := NewSigstoreAttester()
	// Just verify it returns a boolean without panicking
	_ = attester.Available()
}
