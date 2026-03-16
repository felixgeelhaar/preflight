package attestation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBundle_Valid(t *testing.T) {
	t.Parallel()

	bundle, err := NewBundle(
		"application/vnd.dev.sigstore.bundle+json;version=0.1",
		[]byte(`{"payloadType":"application/vnd.in-toto+json"}`),
		[]byte("signature-bytes"),
	)
	require.NoError(t, err)
	assert.Equal(t, "application/vnd.dev.sigstore.bundle+json;version=0.1", bundle.MediaType)
	assert.NotEmpty(t, bundle.Content)
	assert.NotEmpty(t, bundle.Signature)
}

func TestNewBundle_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediaType string
		content   []byte
		signature []byte
		wantErr   string
	}{
		{
			name:      "empty media type",
			mediaType: "",
			content:   []byte("content"),
			signature: []byte("sig"),
			wantErr:   "media type is required",
		},
		{
			name:      "nil content",
			mediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
			content:   nil,
			signature: []byte("sig"),
			wantErr:   "content is required",
		},
		{
			name:      "empty content",
			mediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
			content:   []byte{},
			signature: []byte("sig"),
			wantErr:   "content is required",
		},
		{
			name:      "nil signature",
			mediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
			content:   []byte("content"),
			signature: nil,
			wantErr:   "signature is required",
		},
		{
			name:      "empty signature",
			mediaType: "application/vnd.dev.sigstore.bundle+json;version=0.1",
			content:   []byte("content"),
			signature: []byte{},
			wantErr:   "signature is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewBundle(tt.mediaType, tt.content, tt.signature)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidBundle)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewBundle_WithOptions(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	cert := []byte("cert-data")

	bundle, err := NewBundle(
		"application/vnd.dev.sigstore.bundle+json;version=0.1",
		[]byte("content"),
		[]byte("sig"),
		WithCertificate(cert),
		WithTimestamp(ts),
	)
	require.NoError(t, err)
	assert.Equal(t, cert, bundle.Certificate)
	assert.Equal(t, ts, bundle.Timestamp)
}

func TestNewVerificationResult(t *testing.T) {
	t.Parallel()

	result := NewVerificationResult(true, "user@example.com", "https://accounts.google.com")
	assert.True(t, result.Verified)
	assert.Equal(t, "user@example.com", result.SignerIdentity)
	assert.Equal(t, "https://accounts.google.com", result.Issuer)
	assert.Empty(t, result.Errors)
}

func TestNewVerificationResult_Failed(t *testing.T) {
	t.Parallel()

	result := NewVerificationResult(false, "", "")
	result.Errors = []string{"bad signature", "expired certificate"}
	assert.False(t, result.Verified)
	assert.Len(t, result.Errors, 2)
}

func TestSigstoreVerifier_Name(t *testing.T) {
	t.Parallel()

	v := NewSigstoreVerifier()
	assert.Equal(t, "sigstore", v.Name())
}

func TestSigstoreVerifier_Available(t *testing.T) {
	t.Parallel()

	v := NewSigstoreVerifier()
	// The result depends on whether cosign is installed.
	// We just verify the method does not panic and returns a bool.
	_ = v.Available()
}

func TestVerifierInterface_Compliance(t *testing.T) {
	t.Parallel()

	// Verify SigstoreVerifier implements the Verifier interface.
	var _ Verifier = (*SigstoreVerifier)(nil)
}
