package attestation

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Verifier defines the interface for attestation verification.
type Verifier interface {
	// Name returns the verifier name.
	Name() string
	// Verify verifies an attestation bundle.
	Verify(ctx context.Context, bundle *Bundle) (*VerificationResult, error)
	// Available returns true if the verifier is operational.
	Available() bool
}

// Bundle represents a signed attestation package (Sigstore bundle format).
type Bundle struct {
	MediaType   string    `json:"mediaType"`
	Content     []byte    `json:"content"`
	Signature   []byte    `json:"signature"`
	Certificate []byte    `json:"certificate,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
}

// BundleOption configures optional Bundle fields.
type BundleOption func(*Bundle)

// WithCertificate sets the certificate on a Bundle.
func WithCertificate(cert []byte) BundleOption {
	return func(b *Bundle) {
		b.Certificate = copyBytes(cert)
	}
}

// WithTimestamp sets the timestamp on a Bundle.
func WithTimestamp(ts time.Time) BundleOption {
	return func(b *Bundle) {
		b.Timestamp = ts
	}
}

// NewBundle creates a new validated Bundle.
func NewBundle(mediaType string, content, signature []byte, opts ...BundleOption) (*Bundle, error) {
	if mediaType == "" {
		return nil, fmt.Errorf("%w: media type is required", ErrInvalidBundle)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("%w: content is required", ErrInvalidBundle)
	}
	if len(signature) == 0 {
		return nil, fmt.Errorf("%w: signature is required", ErrInvalidBundle)
	}

	b := &Bundle{
		MediaType: mediaType,
		Content:   copyBytes(content),
		Signature: copyBytes(signature),
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

// VerificationResult contains the outcome of attestation verification.
type VerificationResult struct {
	Verified       bool
	SignerIdentity string
	Issuer         string
	Timestamp      time.Time
	BundleDigest   string
	Errors         []string
}

// NewVerificationResult creates a new VerificationResult.
func NewVerificationResult(verified bool, signerIdentity, issuer string) *VerificationResult {
	return &VerificationResult{
		Verified:       verified,
		SignerIdentity: signerIdentity,
		Issuer:         issuer,
	}
}

// SigstoreVerifier verifies attestations using the cosign CLI.
type SigstoreVerifier struct {
	cosignPath string
}

// NewSigstoreVerifier creates a new SigstoreVerifier.
func NewSigstoreVerifier() *SigstoreVerifier {
	return &SigstoreVerifier{}
}

// Name returns the verifier name.
func (v *SigstoreVerifier) Name() string {
	return "sigstore"
}

// Available returns true if cosign is installed and in PATH.
func (v *SigstoreVerifier) Available() bool {
	path, err := exec.LookPath("cosign")
	if err != nil {
		return false
	}
	v.cosignPath = path
	return true
}

// Verify verifies an attestation bundle using cosign.
func (v *SigstoreVerifier) Verify(ctx context.Context, bundle *Bundle) (*VerificationResult, error) {
	if bundle == nil {
		return nil, fmt.Errorf("%w: bundle is required", ErrInvalidBundle)
	}

	if !v.Available() {
		return nil, fmt.Errorf("%w: cosign not found in PATH (install: https://docs.sigstore.dev/cosign/installation/)",
			ErrVerificationFailed)
	}

	// Write bundle to temp file for cosign verification.
	tmpDir, err := createTempDir("preflight-attest-*")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrVerificationFailed, err)
	}
	defer cleanupTempDir(tmpDir)

	bundleFile, err := writeTempFile(tmpDir, "bundle.json", bundle.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrVerificationFailed, err)
	}

	args := []string{
		"verify-blob",
		"--bundle", bundleFile,
		"--new-bundle-format",
	}

	cmd := exec.CommandContext(ctx, v.cosignPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &VerificationResult{
			Verified: false,
			Errors:   []string{fmt.Sprintf("cosign verification failed: %s", stderr.String())},
		}, nil
	}

	return &VerificationResult{
		Verified:  true,
		Timestamp: time.Now(),
	}, nil
}

// Ensure SigstoreVerifier implements Verifier.
var _ Verifier = (*SigstoreVerifier)(nil)
