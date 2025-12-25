package catalog

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustLevel_IsAtLeast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		level  TrustLevel
		other  TrustLevel
		expect bool
	}{
		{"builtin >= builtin", TrustLevelBuiltin, TrustLevelBuiltin, true},
		{"builtin >= verified", TrustLevelBuiltin, TrustLevelVerified, true},
		{"builtin >= community", TrustLevelBuiltin, TrustLevelCommunity, true},
		{"builtin >= untrusted", TrustLevelBuiltin, TrustLevelUntrusted, true},
		{"verified >= builtin", TrustLevelVerified, TrustLevelBuiltin, false},
		{"verified >= verified", TrustLevelVerified, TrustLevelVerified, true},
		{"verified >= community", TrustLevelVerified, TrustLevelCommunity, true},
		{"community >= verified", TrustLevelCommunity, TrustLevelVerified, false},
		{"community >= community", TrustLevelCommunity, TrustLevelCommunity, true},
		{"untrusted >= community", TrustLevelUntrusted, TrustLevelCommunity, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, tt.level.IsAtLeast(tt.other))
		})
	}
}

func TestTrustLevelFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		expect  TrustLevel
		wantErr bool
	}{
		{"builtin", TrustLevelBuiltin, false},
		{"BUILTIN", TrustLevelBuiltin, false},
		{"verified", TrustLevelVerified, false},
		{"community", TrustLevelCommunity, false},
		{"untrusted", TrustLevelUntrusted, false},
		{"unknown", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			level, err := TrustLevelFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expect, level)
			}
		})
	}
}

func TestSignature(t *testing.T) {
	t.Parallel()

	publisher := NewPublisher("Test Author", "test@example.com", "key123", SignatureTypeSSH)
	sig := NewSignature(SignatureTypeSSH, "key123", []byte("signature"), publisher)

	assert.Equal(t, SignatureTypeSSH, sig.Type())
	assert.Equal(t, "key123", sig.KeyID())
	assert.Equal(t, []byte("signature"), sig.Value())
	assert.Equal(t, publisher.Name(), sig.Publisher().Name())
	assert.False(t, sig.IsZero())
	assert.False(t, sig.IsExpired())
}

func TestSignature_IsZero(t *testing.T) {
	t.Parallel()

	var sig Signature
	assert.True(t, sig.IsZero())

	sig = NewSignature(SignatureTypeSSH, "key", []byte("sig"), Publisher{})
	assert.False(t, sig.IsZero())
}

func TestPublisher(t *testing.T) {
	t.Parallel()

	pub := NewPublisher("Test Author", "test@example.com", "key123", SignatureTypeGPG)

	assert.Equal(t, "Test Author", pub.Name())
	assert.Equal(t, "test@example.com", pub.Email())
	assert.Equal(t, "key123", pub.KeyID())
	assert.Equal(t, SignatureTypeGPG, pub.KeyType())
	assert.Equal(t, "Test Author <test@example.com>", pub.String())
	assert.False(t, pub.IsZero())
}

func TestPublisher_StringNoEmail(t *testing.T) {
	t.Parallel()

	pub := NewPublisher("Test Author", "", "key123", SignatureTypeGPG)
	assert.Equal(t, "Test Author", pub.String())
}

func TestPublisher_IsZero(t *testing.T) {
	t.Parallel()

	var pub Publisher
	assert.True(t, pub.IsZero())

	pub = NewPublisher("name", "", "", SignatureTypeSSH)
	assert.False(t, pub.IsZero())
}

func TestTrustedKey(t *testing.T) {
	t.Parallel()

	publisher := NewPublisher("Test", "test@example.com", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)

	assert.Equal(t, "key1", key.KeyID())
	assert.Equal(t, SignatureTypeSSH, key.KeyType())
	assert.Equal(t, TrustLevelCommunity, key.TrustLevel())
	assert.Equal(t, "Test", key.Publisher().Name())
	assert.False(t, key.IsExpired())

	// Test setters
	key.SetTrustLevel(TrustLevelVerified)
	assert.Equal(t, TrustLevelVerified, key.TrustLevel())

	key.SetComment("Test key")
	assert.Equal(t, "Test key", key.Comment())

	key.SetFingerprint("SHA256:abc123")
	assert.Equal(t, "SHA256:abc123", key.Fingerprint())

	future := time.Now().Add(time.Hour)
	key.SetExpiresAt(future)
	assert.Equal(t, future, key.ExpiresAt())
	assert.False(t, key.IsExpired())
}

func TestTrustedKey_Expired(t *testing.T) {
	t.Parallel()

	publisher := NewPublisher("Test", "", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)

	// Set expiration in the past
	past := time.Now().Add(-time.Hour)
	key.SetExpiresAt(past)
	assert.True(t, key.IsExpired())
}

func TestED25519Verifier(t *testing.T) {
	t.Parallel()

	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	verifier := NewED25519Verifier()
	verifier.AddKey("test-key", publicKey)

	// Sign content
	content := []byte("test content to sign")
	hash := sha256.Sum256(content)
	sigValue := ed25519.Sign(privateKey, hash[:])

	publisher := NewPublisher("Test", "", "test-key", SignatureTypeSSH)
	sig := NewSignature(SignatureTypeSSH, "test-key", sigValue, publisher)

	// Verify
	err = verifier.Verify(content, sig)
	assert.NoError(t, err)
}

func TestED25519Verifier_InvalidSignature(t *testing.T) {
	t.Parallel()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	verifier := NewED25519Verifier()
	verifier.AddKey("test-key", publicKey)

	content := []byte("test content")
	publisher := NewPublisher("Test", "", "test-key", SignatureTypeSSH)
	sig := NewSignature(SignatureTypeSSH, "test-key", []byte("invalid signature"), publisher)

	err = verifier.Verify(content, sig)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestED25519Verifier_UntrustedKey(t *testing.T) {
	t.Parallel()

	verifier := NewED25519Verifier()

	content := []byte("test content")
	publisher := NewPublisher("Test", "", "unknown-key", SignatureTypeSSH)
	sig := NewSignature(SignatureTypeSSH, "unknown-key", []byte("sig"), publisher)

	err := verifier.Verify(content, sig)
	assert.ErrorIs(t, err, ErrUntrustedPublisher)
}

func TestED25519Verifier_SupportsType(t *testing.T) {
	t.Parallel()

	verifier := NewED25519Verifier()

	assert.True(t, verifier.SupportsType(SignatureTypeSSH))
	assert.False(t, verifier.SupportsType(SignatureTypeGPG))
	assert.False(t, verifier.SupportsType(SignatureTypeSigstore))
}

func TestSignedManifest(t *testing.T) {
	t.Parallel()

	manifest, _ := NewManifestBuilder("test").Build()
	publisher := NewPublisher("Test", "", "key1", SignatureTypeSSH)
	sig := NewSignature(SignatureTypeSSH, "key1", []byte("sig"), publisher)

	sm := NewSignedManifest(manifest, sig)

	assert.Equal(t, "test", sm.Manifest().Name())
	assert.Equal(t, "key1", sm.Signature().KeyID())
	assert.True(t, sm.IsSigned())
}

func TestSignedManifest_Unsigned(t *testing.T) {
	t.Parallel()

	manifest, _ := NewManifestBuilder("test").Build()
	sm := NewSignedManifest(manifest, Signature{})

	assert.False(t, sm.IsSigned())
}

func TestVerificationResult(t *testing.T) {
	t.Parallel()

	publisher := NewPublisher("Test", "", "key1", SignatureTypeSSH)
	result := NewVerificationResult(true, TrustLevelVerified, publisher, "key1")

	assert.True(t, result.Verified)
	assert.Equal(t, TrustLevelVerified, result.TrustLevel)
	assert.Equal(t, "Test", result.Publisher.Name())
	assert.Equal(t, "key1", result.KeyID)
	assert.NoError(t, result.Error)

	// With error
	result = result.WithError(ErrInvalidSignature)
	assert.Equal(t, ErrInvalidSignature, result.Error)
}

func TestComputeKeyFingerprint(t *testing.T) {
	t.Parallel()

	publicKey := []byte("test public key data")
	fingerprint := ComputeKeyFingerprint(publicKey)

	assert.NotEmpty(t, fingerprint)
	assert.Contains(t, fingerprint, "SHA256:")
}
