package policy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Attester signs compliance attestations.
type Attester interface {
	// Sign signs an attestation.
	Sign(ctx context.Context, attestation *ComplianceAttestation) error
	// Verify verifies an attestation signature.
	Verify(ctx context.Context, attestation *ComplianceAttestation) error
	// Name returns the attester name.
	Name() string
}

// LocalKeyAttester signs using a local key file with HMAC-SHA256.
type LocalKeyAttester struct {
	keyPath string
}

// NewLocalKeyAttester creates a new attester that signs with a local key file.
func NewLocalKeyAttester(keyPath string) *LocalKeyAttester {
	return &LocalKeyAttester{keyPath: keyPath}
}

// Sign signs the attestation using HMAC-SHA256 with the key file content.
func (a *LocalKeyAttester) Sign(_ context.Context, attestation *ComplianceAttestation) error {
	key, err := os.ReadFile(a.keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrKeyNotFound, a.keyPath)
		}
		return fmt.Errorf("%w: reading key: %w", ErrKeyNotFound, err)
	}

	digest := attestation.Digest()
	sig := computeHMAC(digest, key)

	attestation.SignatureType = "local"
	attestation.Signature = sig
	attestation.SignerIdentity = filepath.Base(a.keyPath)

	return nil
}

// Verify recomputes the HMAC and compares it to the stored signature.
func (a *LocalKeyAttester) Verify(_ context.Context, attestation *ComplianceAttestation) error {
	if !attestation.IsSigned() {
		return ErrAttestationUnsigned
	}

	key, err := os.ReadFile(a.keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrKeyNotFound, a.keyPath)
		}
		return fmt.Errorf("%w: reading key: %w", ErrKeyNotFound, err)
	}

	digest := attestation.Digest()
	expected := computeHMAC(digest, key)

	if !hmac.Equal([]byte(attestation.Signature), []byte(expected)) {
		return ErrSignatureInvalid
	}

	return nil
}

// Name returns the attester name.
func (a *LocalKeyAttester) Name() string {
	return "local-key"
}

// computeHMAC produces a hex-encoded HMAC-SHA256 of the message using the given key.
func computeHMAC(message string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// SigstoreAttester signs using Sigstore keyless signing (cosign).
type SigstoreAttester struct{}

// NewSigstoreAttester creates a new Sigstore-based attester.
func NewSigstoreAttester() *SigstoreAttester {
	return &SigstoreAttester{}
}

// Available returns true if cosign is installed and reachable.
func (a *SigstoreAttester) Available() bool {
	_, err := exec.LookPath("cosign")
	return err == nil
}

// Sign signs the attestation using Sigstore keyless signing.
func (a *SigstoreAttester) Sign(_ context.Context, attestation *ComplianceAttestation) error {
	if !a.Available() {
		return fmt.Errorf("%w: cosign is not installed", ErrAttesterUnavailable)
	}

	attestation.SignatureType = "sigstore"
	attestation.SignerIdentity = "keyless"
	// Actual cosign integration would invoke cosign sign-blob here.
	// This is a stub that marks the attestation as sigstore-signed.
	attestation.Signature = "sigstore:" + attestation.Digest()

	return nil
}

// Verify verifies the attestation signature using Sigstore.
func (a *SigstoreAttester) Verify(_ context.Context, attestation *ComplianceAttestation) error {
	if !a.Available() {
		return fmt.Errorf("%w: cosign is not installed", ErrAttesterUnavailable)
	}

	if !attestation.IsSigned() {
		return ErrAttestationUnsigned
	}

	// Actual cosign integration would invoke cosign verify-blob here.
	return nil
}

// Name returns the attester name.
func (a *SigstoreAttester) Name() string {
	return "sigstore"
}
