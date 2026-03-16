// Package attestation provides in-toto attestation and SLSA provenance
// verification for supply chain security.
package attestation

import "errors"

// Sentinel errors for programmatic error handling.
var (
	// ErrInvalidStatement indicates an attestation statement is malformed.
	ErrInvalidStatement = errors.New("invalid attestation statement")
	// ErrInvalidBundle indicates an attestation bundle is malformed.
	ErrInvalidBundle = errors.New("invalid attestation bundle")
	// ErrVerificationFailed indicates attestation verification failed.
	ErrVerificationFailed = errors.New("attestation verification failed")
	// ErrPolicyViolation indicates an attestation policy was violated.
	ErrPolicyViolation = errors.New("attestation policy violation")
	// ErrNoProvenance indicates no provenance was found.
	ErrNoProvenance = errors.New("no provenance found")
	// ErrInvalidProvenance indicates provenance data is malformed.
	ErrInvalidProvenance = errors.New("invalid provenance")
	// ErrUnsupportedLevel indicates an unsupported SLSA level.
	ErrUnsupportedLevel = errors.New("unsupported SLSA level")
	// ErrBuilderNotTrusted indicates the builder is not in the trusted list.
	ErrBuilderNotTrusted = errors.New("builder not trusted")
	// ErrIdentityNotTrusted indicates the signer identity is not trusted.
	ErrIdentityNotTrusted = errors.New("signer identity not trusted")
	// ErrAttestationExpired indicates the attestation has exceeded its maximum age.
	ErrAttestationExpired = errors.New("attestation has expired")
)
