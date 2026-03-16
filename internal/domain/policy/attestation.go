package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrAttestationInvalid indicates the attestation is malformed.
	ErrAttestationInvalid = errors.New("invalid compliance attestation")
	// ErrAttestationUnsigned indicates the attestation has not been signed.
	ErrAttestationUnsigned = errors.New("attestation is not signed")
	// ErrSignatureInvalid indicates the attestation signature does not match.
	ErrSignatureInvalid = errors.New("attestation signature is invalid")
	// ErrAttesterUnavailable indicates the attester tooling is not installed.
	ErrAttesterUnavailable = errors.New("attester is not available")
	// ErrKeyNotFound indicates the signing key file was not found.
	ErrKeyNotFound = errors.New("signing key not found")
)

// ComplianceAttestation wraps a ComplianceReport with a cryptographic signature,
// providing proof that a machine was compliant at a point in time.
type ComplianceAttestation struct {
	// Report is the compliance report being attested.
	Report *ComplianceReport `json:"report"`
	// MachineID identifies the machine.
	MachineID string `json:"machine_id"`
	// Hostname is the machine's hostname.
	Hostname string `json:"hostname"`
	// AttestedAt is when the attestation was created.
	AttestedAt time.Time `json:"attested_at"`
	// SignatureType is "local" for key-based or "sigstore" for keyless.
	SignatureType string `json:"signature_type,omitempty"`
	// Signature is the base64-encoded signature.
	Signature string `json:"signature,omitempty"`
	// SignerIdentity is the signer (key ID or OIDC identity).
	SignerIdentity string `json:"signer_identity,omitempty"`
	// ContentDigest is the SHA256 of the report content (for verification).
	ContentDigest string `json:"content_digest"`
}

// digestContent is the structure hashed to produce the content digest.
type digestContent struct {
	Report     *ComplianceReport `json:"report"`
	MachineID  string            `json:"machine_id"`
	Hostname   string            `json:"hostname"`
	AttestedAt time.Time         `json:"attested_at"`
}

// NewComplianceAttestation creates a new unsigned attestation for a report.
func NewComplianceAttestation(report *ComplianceReport, machineID, hostname string) (*ComplianceAttestation, error) {
	if report == nil {
		return nil, fmt.Errorf("%w: report is required", ErrAttestationInvalid)
	}
	if machineID == "" {
		return nil, fmt.Errorf("%w: machine ID is required", ErrAttestationInvalid)
	}

	att := &ComplianceAttestation{
		Report:     report,
		MachineID:  machineID,
		Hostname:   hostname,
		AttestedAt: time.Now(),
	}
	att.ContentDigest = att.Digest()

	return att, nil
}

// Digest computes the SHA256 of the attestation content (report + machine info).
func (a *ComplianceAttestation) Digest() string {
	content := digestContent{
		Report:     a.Report,
		MachineID:  a.MachineID,
		Hostname:   a.Hostname,
		AttestedAt: a.AttestedAt,
	}

	data, err := json.Marshal(content)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// IsSigned returns true if the attestation has been signed.
func (a *ComplianceAttestation) IsSigned() bool {
	return a.Signature != ""
}

// Validate checks that the attestation is well-formed.
func (a *ComplianceAttestation) Validate() error {
	if a.Report == nil {
		return fmt.Errorf("%w: report is required", ErrAttestationInvalid)
	}
	if a.MachineID == "" {
		return fmt.Errorf("%w: machine ID is required", ErrAttestationInvalid)
	}
	if a.AttestedAt.IsZero() {
		return fmt.Errorf("%w: attested_at must not be zero", ErrAttestationInvalid)
	}
	return nil
}
