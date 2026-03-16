package attestation

import (
	"fmt"
	"path/filepath"
	"time"
)

// Policy defines requirements for attestation verification.
type Policy struct {
	// RequiredLevel is the minimum SLSA level required.
	RequiredLevel SLSALevel
	// TrustedBuilders is a list of trusted builder IDs (glob patterns).
	TrustedBuilders []string
	// TrustedIdentities is a list of trusted signer identities (glob patterns).
	TrustedIdentities []string
	// RequireTimestamp requires attestations to include a timestamp.
	RequireTimestamp bool
	// MaxAge is the maximum age of an attestation (0 = no limit).
	MaxAge time.Duration
}

// PolicyResult contains the outcome of a policy evaluation.
type PolicyResult struct {
	Passed     bool
	Violations []string
}

// DefaultPolicy returns a default policy requiring SLSA Level 1.
func DefaultPolicy() Policy {
	return Policy{
		RequiredLevel: SLSALevel1,
	}
}

// StrictPolicy returns a strict policy requiring SLSA Level 2 with timestamps.
func StrictPolicy() Policy {
	return Policy{
		RequiredLevel:    SLSALevel2,
		RequireTimestamp: true,
	}
}

// Evaluate checks a provenance and verification result against this policy.
// Returns a PolicyResult with any violations found.
func (p Policy) Evaluate(prov *Provenance, result *VerificationResult) (*PolicyResult, error) {
	if prov == nil {
		return nil, fmt.Errorf("%w: provenance is required", ErrNoProvenance)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: verification result is required", ErrVerificationFailed)
	}

	pr := &PolicyResult{Passed: true}

	// Check verification status.
	if !result.Verified {
		pr.addViolation("attestation is not verified")
	}

	// Check SLSA level.
	if prov.SLSALevel() < p.RequiredLevel {
		pr.addViolation(fmt.Sprintf(
			"SLSA level %d does not meet required level %d",
			prov.SLSALevel(), p.RequiredLevel,
		))
	}

	// Check trusted builders.
	if len(p.TrustedBuilders) > 0 {
		builderID := prov.RunDetails().Builder.ID
		if !matchesAnyGlob(builderID, p.TrustedBuilders) {
			pr.addViolation(fmt.Sprintf(
				"builder %q is not in the trusted builders list",
				builderID,
			))
		}
	}

	// Check trusted identities.
	if len(p.TrustedIdentities) > 0 {
		identity := result.SignerIdentity
		if !matchesAnyGlob(identity, p.TrustedIdentities) {
			pr.addViolation(fmt.Sprintf(
				"signer identity %q is not in the trusted identities list",
				identity,
			))
		}
	}

	// Check timestamp requirement.
	if p.RequireTimestamp && result.Timestamp.IsZero() {
		pr.addViolation("timestamp is required but not present")
	}

	// Check max age.
	if p.MaxAge > 0 && !result.Timestamp.IsZero() {
		age := time.Since(result.Timestamp)
		if age > p.MaxAge {
			pr.addViolation(fmt.Sprintf(
				"attestation has expired: age %s exceeds maximum %s",
				age.Truncate(time.Second), p.MaxAge,
			))
		}
	}

	return pr, nil
}

// addViolation records a policy violation and marks the result as failed.
func (pr *PolicyResult) addViolation(msg string) {
	pr.Passed = false
	pr.Violations = append(pr.Violations, msg)
}

// matchesAnyGlob returns true if value matches any of the glob patterns.
func matchesAnyGlob(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := filepath.Match(pattern, value); err == nil && matched {
			return true
		}
	}
	return false
}
