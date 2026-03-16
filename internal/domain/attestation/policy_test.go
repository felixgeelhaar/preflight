package attestation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPolicy(t *testing.T) {
	t.Parallel()

	p := DefaultPolicy()
	assert.Equal(t, SLSALevel1, p.RequiredLevel)
	assert.Empty(t, p.TrustedBuilders)
	assert.Empty(t, p.TrustedIdentities)
	assert.False(t, p.RequireTimestamp)
	assert.Equal(t, time.Duration(0), p.MaxAge)
}

func TestStrictPolicy(t *testing.T) {
	t.Parallel()

	p := StrictPolicy()
	assert.Equal(t, SLSALevel2, p.RequiredLevel)
	assert.True(t, p.RequireTimestamp)
}

func TestPolicy_Evaluate_PassesDefault(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	result := &VerificationResult{
		Verified:       true,
		SignerIdentity: "user@example.com",
		Issuer:         "https://accounts.google.com",
		Timestamp:      time.Now(),
	}

	policy := DefaultPolicy()
	policyResult, err := policy.Evaluate(prov, result)
	require.NoError(t, err)
	assert.True(t, policyResult.Passed)
	assert.Empty(t, policyResult.Violations)
}

func TestPolicy_Evaluate_FailsInsufficientLevel(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel0)
	require.NoError(t, err)

	result := &VerificationResult{
		Verified: true,
	}

	policy := DefaultPolicy() // Requires L1
	policyResult, err := policy.Evaluate(prov, result)
	require.NoError(t, err)
	assert.False(t, policyResult.Passed)
	assert.NotEmpty(t, policyResult.Violations)
	assert.Contains(t, policyResult.Violations[0], "SLSA level")
}

func TestPolicy_Evaluate_FailsUnverified(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel2)
	require.NoError(t, err)

	result := &VerificationResult{
		Verified: false,
		Errors:   []string{"bad sig"},
	}

	policy := DefaultPolicy()
	policyResult, err := policy.Evaluate(prov, result)
	require.NoError(t, err)
	assert.False(t, policyResult.Passed)
	assert.Contains(t, policyResult.Violations[0], "not verified")
}

func TestPolicy_Evaluate_TrustedBuilders(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel2)
	require.NoError(t, err)

	result := &VerificationResult{
		Verified:       true,
		SignerIdentity: "user@example.com",
		Timestamp:      time.Now(),
	}

	tests := []struct {
		name            string
		trustedBuilders []string
		wantPass        bool
	}{
		{
			name:            "matching glob pattern",
			trustedBuilders: []string{"https://github.com/actions/*"},
			wantPass:        true,
		},
		{
			name:            "exact match",
			trustedBuilders: []string{"https://github.com/actions/runner"},
			wantPass:        true,
		},
		{
			name:            "no match",
			trustedBuilders: []string{"https://gitlab.com/*"},
			wantPass:        false,
		},
		{
			name:            "empty list trusts all",
			trustedBuilders: nil,
			wantPass:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			policy := Policy{
				RequiredLevel:   SLSALevel1,
				TrustedBuilders: tt.trustedBuilders,
			}

			policyResult, err := policy.Evaluate(prov, result)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPass, policyResult.Passed, "violations: %v", policyResult.Violations)
		})
	}
}

func TestPolicy_Evaluate_TrustedIdentities(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel2)
	require.NoError(t, err)

	tests := []struct {
		name              string
		trustedIdentities []string
		signerIdentity    string
		wantPass          bool
	}{
		{
			name:              "matching glob pattern",
			trustedIdentities: []string{"*@example.com"},
			signerIdentity:    "user@example.com",
			wantPass:          true,
		},
		{
			name:              "exact match",
			trustedIdentities: []string{"user@example.com"},
			signerIdentity:    "user@example.com",
			wantPass:          true,
		},
		{
			name:              "no match",
			trustedIdentities: []string{"admin@example.com"},
			signerIdentity:    "user@example.com",
			wantPass:          false,
		},
		{
			name:              "empty list trusts all",
			trustedIdentities: nil,
			signerIdentity:    "user@example.com",
			wantPass:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &VerificationResult{
				Verified:       true,
				SignerIdentity: tt.signerIdentity,
				Timestamp:      time.Now(),
			}

			policy := Policy{
				RequiredLevel:     SLSALevel1,
				TrustedIdentities: tt.trustedIdentities,
			}

			policyResult, err := policy.Evaluate(prov, result)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPass, policyResult.Passed, "violations: %v", policyResult.Violations)
		})
	}
}

func TestPolicy_Evaluate_RequireTimestamp(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel2)
	require.NoError(t, err)

	policy := Policy{
		RequiredLevel:    SLSALevel1,
		RequireTimestamp: true,
	}

	t.Run("with timestamp", func(t *testing.T) {
		t.Parallel()

		result := &VerificationResult{
			Verified:  true,
			Timestamp: time.Now(),
		}
		policyResult, err := policy.Evaluate(prov, result)
		require.NoError(t, err)
		assert.True(t, policyResult.Passed)
	})

	t.Run("without timestamp", func(t *testing.T) {
		t.Parallel()

		result := &VerificationResult{
			Verified: true,
		}
		policyResult, err := policy.Evaluate(prov, result)
		require.NoError(t, err)
		assert.False(t, policyResult.Passed)
		assert.Contains(t, policyResult.Violations[0], "timestamp")
	})
}

func TestPolicy_Evaluate_MaxAge(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel2)
	require.NoError(t, err)

	policy := Policy{
		RequiredLevel: SLSALevel1,
		MaxAge:        24 * time.Hour,
	}

	t.Run("fresh attestation", func(t *testing.T) {
		t.Parallel()

		result := &VerificationResult{
			Verified:  true,
			Timestamp: time.Now(),
		}
		policyResult, err := policy.Evaluate(prov, result)
		require.NoError(t, err)
		assert.True(t, policyResult.Passed)
	})

	t.Run("expired attestation", func(t *testing.T) {
		t.Parallel()

		result := &VerificationResult{
			Verified:  true,
			Timestamp: time.Now().Add(-48 * time.Hour),
		}
		policyResult, err := policy.Evaluate(prov, result)
		require.NoError(t, err)
		assert.False(t, policyResult.Passed)
		assert.Contains(t, policyResult.Violations[0], "expired")
	})
}

func TestPolicy_Evaluate_NilProvenance(t *testing.T) {
	t.Parallel()

	policy := DefaultPolicy()
	result := &VerificationResult{Verified: true}

	_, err := policy.Evaluate(nil, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoProvenance)
}

func TestPolicy_Evaluate_NilResult(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	policy := DefaultPolicy()
	_, err = policy.Evaluate(prov, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVerificationFailed)
}

func TestPolicy_Evaluate_MultipleViolations(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel0)
	require.NoError(t, err)

	policy := Policy{
		RequiredLevel:     SLSALevel2,
		TrustedBuilders:   []string{"https://gitlab.com/*"},
		TrustedIdentities: []string{"admin@example.com"},
		RequireTimestamp:  true,
	}

	result := &VerificationResult{
		Verified:       false,
		SignerIdentity: "user@example.com",
	}

	policyResult, err := policy.Evaluate(prov, result)
	require.NoError(t, err)
	assert.False(t, policyResult.Passed)
	// Should have multiple violations: unverified, SLSA level, builder, identity, timestamp
	assert.GreaterOrEqual(t, len(policyResult.Violations), 4)
}
