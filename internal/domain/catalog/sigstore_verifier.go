package catalog

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Sigstore verification errors.
var (
	ErrSigstoreNoCertificate      = errors.New("no certificate in signature")
	ErrSigstoreInvalidCertificate = errors.New("invalid certificate")
	ErrSigstoreIdentityMismatch   = errors.New("OIDC identity mismatch")
	ErrSigstoreIssuerMismatch     = errors.New("OIDC issuer mismatch")
	ErrSigstoreNotYetValid        = errors.New("certificate not yet valid")
	ErrSigstoreCertExpired        = errors.New("certificate expired")
)

// SigstoreIdentity represents an OIDC identity pattern for verification.
type SigstoreIdentity struct {
	// IssuerRegexp is a regex pattern for the OIDC issuer URL.
	// Examples: "https://token.actions.githubusercontent.com", "https://accounts.google.com"
	IssuerRegexp string

	// SubjectRegexp is a regex pattern for the OIDC subject.
	// For GitHub Actions: "https://github.com/owner/repo/.github/workflows/release.yml@refs/tags/.*"
	// For email: "user@example.com"
	SubjectRegexp string
}

// SigstoreVerifierConfig configures the Sigstore verifier.
type SigstoreVerifierConfig struct {
	// TrustedIdentities is a list of trusted OIDC identities.
	TrustedIdentities []SigstoreIdentity

	// AllowExpired allows verification of signatures with expired certificates.
	// This is useful for verifying historical artifacts but should be used carefully.
	AllowExpired bool

	// VerifyTimestamp determines if the signature timestamp should be verified.
	VerifyTimestamp bool
}

// DefaultSigstoreVerifierConfig returns a config with common trusted identities.
func DefaultSigstoreVerifierConfig() SigstoreVerifierConfig {
	return SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			// GitHub Actions
			{
				IssuerRegexp:  `^https://token\.actions\.githubusercontent\.com$`,
				SubjectRegexp: `^https://github\.com/.+/.+/\.github/workflows/.+@refs/.*$`,
			},
			// GitLab CI
			{
				IssuerRegexp:  `^https://gitlab\.com$`,
				SubjectRegexp: `^https://gitlab\.com/.+$`,
			},
			// Google accounts
			{
				IssuerRegexp:  `^https://accounts\.google\.com$`,
				SubjectRegexp: `.+@.+\..+`, // Email pattern
			},
			// Microsoft
			{
				IssuerRegexp:  `^https://login\.microsoftonline\.com/.+/v2\.0$`,
				SubjectRegexp: `.+`,
			},
		},
		AllowExpired:    false,
		VerifyTimestamp: true,
	}
}

// SigstoreSignature contains Sigstore-specific signature data.
type SigstoreSignature struct {
	// Base64-encoded signature
	Signature string `json:"signature"`

	// Base64-encoded certificate chain (PEM format)
	Certificate string `json:"certificate"`

	// Optional bundle from Rekor transparency log
	Bundle *SigstoreBundle `json:"bundle,omitempty"`
}

// SigstoreBundle contains Rekor transparency log entry data.
type SigstoreBundle struct {
	// Log entry UUID
	UUID string `json:"uuid"`

	// Log entry index
	LogIndex int64 `json:"logIndex"`

	// Integrated timestamp from the log
	IntegratedTime int64 `json:"integratedTime"`
}

// SigstoreVerifier verifies Sigstore keyless signatures.
type SigstoreVerifier struct {
	config SigstoreVerifierConfig
}

// NewSigstoreVerifier creates a new Sigstore verifier.
func NewSigstoreVerifier(config SigstoreVerifierConfig) *SigstoreVerifier {
	return &SigstoreVerifier{config: config}
}

// Verify verifies a Sigstore signature.
func (v *SigstoreVerifier) Verify(content []byte, signature Signature) error {
	if signature.Type() != SignatureTypeSigstore {
		return fmt.Errorf("%w: expected sigstore signature", ErrInvalidSignature)
	}

	// Parse the Sigstore-specific signature data
	var sigData SigstoreSignature
	if err := json.Unmarshal(signature.Value(), &sigData); err != nil {
		return fmt.Errorf("failed to parse sigstore signature: %w", err)
	}

	if sigData.Certificate == "" {
		return ErrSigstoreNoCertificate
	}

	// Decode and parse the certificate
	certPEM, err := base64.StdEncoding.DecodeString(sigData.Certificate)
	if err != nil {
		return fmt.Errorf("%w: failed to decode certificate: %w", ErrSigstoreInvalidCertificate, err)
	}

	cert, err := parseCertificateFromPEM(certPEM)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSigstoreInvalidCertificate, err)
	}

	// Verify certificate validity period
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return ErrSigstoreNotYetValid
	}
	if !v.config.AllowExpired && now.After(cert.NotAfter) {
		return ErrSigstoreCertExpired
	}

	// Extract and verify OIDC claims
	issuer, subject, err := extractOIDCClaims(cert)
	if err != nil {
		return err
	}

	if err := v.verifyIdentity(issuer, subject); err != nil {
		return err
	}

	// Verify the signature
	sigBytes, err := base64.StdEncoding.DecodeString(sigData.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := sha256.Sum256(content)
	return verifyECDSASignature(cert.PublicKey, hash[:], sigBytes)
}

// SupportsType returns true for Sigstore signature type.
func (v *SigstoreVerifier) SupportsType(sigType SignatureType) bool {
	return sigType == SignatureTypeSigstore
}

// verifyIdentity checks if the OIDC identity matches any trusted pattern.
func (v *SigstoreVerifier) verifyIdentity(issuer, subject string) error {
	for _, identity := range v.config.TrustedIdentities {
		issuerMatch, err := regexp.MatchString(identity.IssuerRegexp, issuer)
		if err != nil {
			continue
		}
		if !issuerMatch {
			continue
		}

		subjectMatch, err := regexp.MatchString(identity.SubjectRegexp, subject)
		if err != nil {
			continue
		}
		if subjectMatch {
			return nil // Identity verified
		}
	}

	return fmt.Errorf("%w: issuer=%s subject=%s", ErrSigstoreIdentityMismatch, issuer, subject)
}

// AddTrustedIdentity adds a trusted OIDC identity pattern.
func (v *SigstoreVerifier) AddTrustedIdentity(identity SigstoreIdentity) {
	v.config.TrustedIdentities = append(v.config.TrustedIdentities, identity)
}

// parseCertificateFromPEM parses a certificate from PEM data.
func parseCertificateFromPEM(pemData []byte) (*x509.Certificate, error) {
	// Handle both raw PEM and base64-encoded PEM
	data := pemData

	// Check if it's base64 encoded (no PEM header)
	if !strings.Contains(string(pemData), "-----BEGIN") {
		decoded, err := base64.StdEncoding.DecodeString(string(pemData))
		if err == nil {
			data = decoded
		}
	}

	// Try parsing as PEM
	pemStr := string(data)
	beginIdx := strings.Index(pemStr, "-----BEGIN CERTIFICATE-----")
	if beginIdx >= 0 {
		endIdx := strings.Index(pemStr, "-----END CERTIFICATE-----")
		if endIdx > beginIdx {
			certPEM := pemStr[beginIdx+27 : endIdx]
			certPEM = strings.ReplaceAll(certPEM, "\n", "")
			certPEM = strings.ReplaceAll(certPEM, "\r", "")
			certDER, err := base64.StdEncoding.DecodeString(certPEM)
			if err != nil {
				return nil, fmt.Errorf("failed to decode certificate base64: %w", err)
			}
			return x509.ParseCertificate(certDER)
		}
	}

	// Try parsing as raw DER
	return x509.ParseCertificate(data)
}

// extractOIDCClaims extracts OIDC issuer and subject from a Fulcio certificate.
// Fulcio certificates store OIDC claims in certificate extensions.
func extractOIDCClaims(cert *x509.Certificate) (issuer, subject string, err error) {
	// Fulcio OIDC extension OIDs
	// 1.3.6.1.4.1.57264.1.1 = OIDC Issuer
	// 1.3.6.1.4.1.57264.1.8 = OIDC Issuer (v2)
	oidcIssuerOID := []int{1, 3, 6, 1, 4, 1, 57264, 1, 1}
	oidcIssuerV2OID := []int{1, 3, 6, 1, 4, 1, 57264, 1, 8}

	for _, ext := range cert.Extensions {
		if equalOID(ext.Id, oidcIssuerOID) || equalOID(ext.Id, oidcIssuerV2OID) {
			issuer = string(ext.Value)
			// Clean up potential ASN.1 string prefix
			if len(issuer) > 2 && (issuer[0] == 0x0c || issuer[0] == 0x13) {
				issuer = issuer[2:]
			}
		}
	}

	// Subject is typically in the Subject Alternative Name (SAN) extension
	// or in the Subject field for email identities
	switch {
	case len(cert.EmailAddresses) > 0:
		subject = cert.EmailAddresses[0]
	case len(cert.URIs) > 0:
		subject = cert.URIs[0].String()
	case cert.Subject.CommonName != "":
		subject = cert.Subject.CommonName
	}

	if issuer == "" {
		// Fallback: try to get issuer from Issuer field
		if cert.Issuer.CommonName != "" {
			issuer = cert.Issuer.CommonName
		} else {
			return "", "", fmt.Errorf("could not extract OIDC issuer from certificate")
		}
	}

	if subject == "" {
		return "", "", fmt.Errorf("could not extract OIDC subject from certificate")
	}

	return issuer, subject, nil
}

// equalOID compares two OID slices.
func equalOID(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// verifyECDSASignature verifies an ECDSA signature.
func verifyECDSASignature(publicKey crypto.PublicKey, hash, signature []byte) error {
	ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: expected ECDSA public key", ErrInvalidSignature)
	}

	if !ecdsa.VerifyASN1(ecdsaKey, hash, signature) {
		return ErrInvalidSignature
	}

	return nil
}

// SigstorePublisher creates a Publisher from Sigstore certificate claims.
func SigstorePublisher(issuer, subject string) Publisher {
	// Determine name from subject
	name := subject
	email := ""

	// If subject looks like an email, use it
	if strings.Contains(subject, "@") && !strings.HasPrefix(subject, "https://") {
		email = subject
		// Extract name from email
		parts := strings.Split(email, "@")
		if len(parts) > 0 {
			name = parts[0]
		}
	} else if strings.HasPrefix(subject, "https://github.com/") {
		// Extract repo owner from GitHub Actions subject
		parts := strings.Split(subject, "/")
		if len(parts) >= 4 {
			name = parts[3] // owner name
		}
	}

	return Publisher{
		name:    name,
		email:   email,
		keyID:   issuer,
		keyType: SignatureTypeSigstore,
	}
}
