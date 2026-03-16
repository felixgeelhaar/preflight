package attestation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validStatement(t *testing.T) Statement {
	t.Helper()

	subjects := []Subject{
		{Name: "pkg:brew/git@2.40.0", Digest: map[string]string{"sha256": "abc123"}},
	}
	stmt, err := NewStatement(PredicateTypeSLSAProvenanceV1, subjects, []byte(`{}`))
	require.NoError(t, err)
	return stmt
}

func validBuildDefinition() BuildDefinition {
	return BuildDefinition{
		BuildType: "https://github.com/felixgeelhaar/preflight/build/v1",
		ExternalParameters: map[string]string{
			"repository": "https://github.com/felixgeelhaar/preflight",
		},
		ResolvedDependencies: []ResourceDescriptor{
			{
				URI:    "git+https://github.com/felixgeelhaar/preflight@refs/heads/main",
				Digest: map[string]string{"sha256": "dep123"},
				Name:   "source",
			},
		},
	}
}

func validRunDetails() RunDetails {
	return RunDetails{
		Builder: BuilderID{
			ID:      "https://github.com/actions/runner",
			Version: "2.304.0",
		},
		Metadata: BuildMetadata{
			InvocationID: "run-12345",
			StartedOn:    time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			FinishedOn:   time.Date(2026, 1, 15, 10, 5, 0, 0, time.UTC),
		},
	}
}

func TestNewProvenance_Valid(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	buildDef := validBuildDefinition()
	runDet := validRunDetails()

	prov, err := NewProvenance(stmt, buildDef, runDet, SLSALevel1)
	require.NoError(t, err)

	assert.Equal(t, stmt, prov.Statement())
	assert.Equal(t, buildDef, prov.BuildDefinition())
	assert.Equal(t, runDet, prov.RunDetails())
	assert.Equal(t, SLSALevel1, prov.SLSALevel())
	assert.False(t, prov.IsVerified())
	assert.True(t, prov.VerifiedAt().IsZero())
}

func TestNewProvenance_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		statement Statement
		buildDef  BuildDefinition
		runDet    RunDetails
		level     SLSALevel
		wantErr   string
	}{
		{
			name:      "zero statement",
			statement: Statement{},
			buildDef:  validBuildDefinition(),
			runDet:    validRunDetails(),
			level:     SLSALevel1,
			wantErr:   "statement is required",
		},
		{
			name:      "empty build type",
			statement: validStatement(t),
			buildDef:  BuildDefinition{},
			runDet:    validRunDetails(),
			level:     SLSALevel1,
			wantErr:   "build type is required",
		},
		{
			name:      "empty builder ID",
			statement: validStatement(t),
			buildDef:  validBuildDefinition(),
			runDet:    RunDetails{Builder: BuilderID{}},
			level:     SLSALevel1,
			wantErr:   "builder ID is required",
		},
		{
			name:      "invalid SLSA level negative",
			statement: validStatement(t),
			buildDef:  validBuildDefinition(),
			runDet:    validRunDetails(),
			level:     SLSALevel(-1),
			wantErr:   "unsupported SLSA level",
		},
		{
			name:      "invalid SLSA level too high",
			statement: validStatement(t),
			buildDef:  validBuildDefinition(),
			runDet:    validRunDetails(),
			level:     SLSALevel(5),
			wantErr:   "unsupported SLSA level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewProvenance(tt.statement, tt.buildDef, tt.runDet, tt.level)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestProvenance_MatchesMaterial(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	buildDef := BuildDefinition{
		BuildType: "https://github.com/felixgeelhaar/preflight/build/v1",
		ResolvedDependencies: []ResourceDescriptor{
			{
				URI:    "git+https://github.com/felixgeelhaar/preflight@refs/heads/main",
				Digest: map[string]string{"sha256": "dep123"},
				Name:   "source",
			},
			{
				URI:    "https://registry.npmjs.org/pkg/-/pkg-1.0.0.tgz",
				Digest: map[string]string{"sha256": "npm456", "sha512": "longhash"},
			},
		},
	}

	prov, err := NewProvenance(stmt, buildDef, validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	tests := []struct {
		name      string
		uri       string
		algorithm string
		digest    string
		want      bool
	}{
		{
			name:      "matches first dependency",
			uri:       "git+https://github.com/felixgeelhaar/preflight@refs/heads/main",
			algorithm: "sha256",
			digest:    "dep123",
			want:      true,
		},
		{
			name:      "matches second dependency sha512",
			uri:       "https://registry.npmjs.org/pkg/-/pkg-1.0.0.tgz",
			algorithm: "sha512",
			digest:    "longhash",
			want:      true,
		},
		{
			name:      "wrong URI",
			uri:       "https://example.com/other",
			algorithm: "sha256",
			digest:    "dep123",
			want:      false,
		},
		{
			name:      "wrong digest",
			uri:       "git+https://github.com/felixgeelhaar/preflight@refs/heads/main",
			algorithm: "sha256",
			digest:    "wrong",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, prov.MatchesMaterial(tt.uri, tt.algorithm, tt.digest))
		})
	}
}

func TestProvenance_Verify(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	// Create a mock verifier that succeeds.
	mockVerifier := &mockVerifierImpl{
		name:      "mock",
		available: true,
		result: &VerificationResult{
			Verified:       true,
			SignerIdentity: "test@example.com",
			Issuer:         "https://accounts.google.com",
			Timestamp:      time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			BundleDigest:   "sha256:bundledigest",
		},
	}

	bundle := &Bundle{
		MediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
		Content:   []byte(`{"payloadType":"application/vnd.in-toto+json"}`),
		Signature: []byte("signature-bytes"),
		Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	err = prov.Verify(context.Background(), mockVerifier, bundle)
	require.NoError(t, err)
	assert.True(t, prov.IsVerified())
	assert.False(t, prov.VerifiedAt().IsZero())
}

func TestProvenance_Verify_Failure(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	mockVerifier := &mockVerifierImpl{
		name:      "mock",
		available: true,
		result: &VerificationResult{
			Verified: false,
			Errors:   []string{"bad signature"},
		},
	}

	bundle := &Bundle{
		MediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
		Content:   []byte(`{"payloadType":"application/vnd.in-toto+json"}`),
		Signature: []byte("bad-sig"),
		Timestamp: time.Now(),
	}

	err = prov.Verify(context.Background(), mockVerifier, bundle)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVerificationFailed)
	assert.False(t, prov.IsVerified())
}

func TestProvenance_Verify_UnavailableVerifier(t *testing.T) {
	t.Parallel()

	stmt := validStatement(t)
	prov, err := NewProvenance(stmt, validBuildDefinition(), validRunDetails(), SLSALevel1)
	require.NoError(t, err)

	mockVerifier := &mockVerifierImpl{
		name:      "unavailable",
		available: false,
	}

	bundle := &Bundle{
		MediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
		Content:   []byte(`{"payloadType":"application/vnd.in-toto+json"}`),
		Signature: []byte("sig"),
		Timestamp: time.Now(),
	}

	err = prov.Verify(context.Background(), mockVerifier, bundle)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestSLSALevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level SLSALevel
		want  string
	}{
		{SLSALevel0, "SLSA Level 0"},
		{SLSALevel1, "SLSA Level 1"},
		{SLSALevel2, "SLSA Level 2"},
		{SLSALevel3, "SLSA Level 3"},
		{SLSALevel4, "SLSA Level 4"},
		{SLSALevel(-1), "SLSA Level unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

// mockVerifierImpl is a test double for the Verifier interface.
type mockVerifierImpl struct {
	name      string
	available bool
	result    *VerificationResult
	err       error
}

func (m *mockVerifierImpl) Name() string { return m.name }

func (m *mockVerifierImpl) Verify(_ context.Context, _ *Bundle) (*VerificationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *mockVerifierImpl) Available() bool { return m.available }
