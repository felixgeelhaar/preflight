package attestation

import (
	"context"
	"fmt"
	"time"
)

// SLSALevel represents the SLSA build level.
type SLSALevel int

const (
	// SLSALevel0 provides no guarantees.
	SLSALevel0 SLSALevel = 0
	// SLSALevel1 requires provenance exists.
	SLSALevel1 SLSALevel = 1
	// SLSALevel2 requires hosted build and signed provenance.
	SLSALevel2 SLSALevel = 2
	// SLSALevel3 requires hardened builds.
	SLSALevel3 SLSALevel = 3
	// SLSALevel4 requires two-person reviewed (future).
	SLSALevel4 SLSALevel = 4
)

// String returns a human-readable SLSA level description.
func (l SLSALevel) String() string {
	switch l {
	case SLSALevel0:
		return "SLSA Level 0"
	case SLSALevel1:
		return "SLSA Level 1"
	case SLSALevel2:
		return "SLSA Level 2"
	case SLSALevel3:
		return "SLSA Level 3"
	case SLSALevel4:
		return "SLSA Level 4"
	default:
		return "SLSA Level unknown"
	}
}

// BuildDefinition describes what was built.
type BuildDefinition struct {
	BuildType            string               `json:"buildType"`
	ExternalParameters   map[string]string    `json:"externalParameters,omitempty"`
	InternalParameters   map[string]string    `json:"internalParameters,omitempty"`
	ResolvedDependencies []ResourceDescriptor `json:"resolvedDependencies,omitempty"`
}

// ResourceDescriptor describes a resource (material or output).
type ResourceDescriptor struct {
	URI    string            `json:"uri"`
	Digest map[string]string `json:"digest"`
	Name   string            `json:"name,omitempty"`
}

// RunDetails describes who built it.
type RunDetails struct {
	Builder  BuilderID     `json:"builder"`
	Metadata BuildMetadata `json:"metadata,omitempty"`
}

// BuilderID identifies the build system.
type BuilderID struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// BuildMetadata captures build timestamps and invocation info.
type BuildMetadata struct {
	InvocationID string    `json:"invocationId,omitempty"`
	StartedOn    time.Time `json:"startedOn,omitempty"`
	FinishedOn   time.Time `json:"finishedOn,omitempty"`
}

// Provenance is the SLSA v1.0 provenance aggregate root.
type Provenance struct {
	statement       Statement
	buildDefinition BuildDefinition
	runDetails      RunDetails
	slsaLevel       SLSALevel
	verified        bool
	verifiedAt      time.Time
}

// NewProvenance creates a new validated Provenance aggregate.
func NewProvenance(statement Statement, buildDef BuildDefinition, runDet RunDetails, level SLSALevel) (*Provenance, error) {
	if statement.IsZero() {
		return nil, fmt.Errorf("%w: statement is required", ErrInvalidProvenance)
	}

	if buildDef.BuildType == "" {
		return nil, fmt.Errorf("%w: build type is required", ErrInvalidProvenance)
	}

	if runDet.Builder.ID == "" {
		return nil, fmt.Errorf("%w: builder ID is required", ErrInvalidProvenance)
	}

	if level < SLSALevel0 || level > SLSALevel4 {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedLevel, level)
	}

	return &Provenance{
		statement:       statement,
		buildDefinition: buildDef,
		runDetails:      runDet,
		slsaLevel:       level,
	}, nil
}

// Statement returns the in-toto statement.
func (p *Provenance) Statement() Statement {
	return p.statement
}

// BuildDefinition returns the build definition.
func (p *Provenance) BuildDefinition() BuildDefinition {
	return p.buildDefinition
}

// RunDetails returns the run details.
func (p *Provenance) RunDetails() RunDetails {
	return p.runDetails
}

// SLSALevel returns the SLSA build level.
func (p *Provenance) SLSALevel() SLSALevel {
	return p.slsaLevel
}

// IsVerified returns true if the provenance has been verified.
func (p *Provenance) IsVerified() bool {
	return p.verified
}

// VerifiedAt returns when the provenance was verified.
func (p *Provenance) VerifiedAt() time.Time {
	return p.verifiedAt
}

// Verify verifies the provenance using the given verifier and bundle.
// On success, marks the provenance as verified.
func (p *Provenance) Verify(ctx context.Context, verifier Verifier, bundle *Bundle) error {
	if !verifier.Available() {
		return fmt.Errorf("%w: verifier %q is not available", ErrVerificationFailed, verifier.Name())
	}

	result, err := verifier.Verify(ctx, bundle)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrVerificationFailed, err)
	}

	if !result.Verified {
		return fmt.Errorf("%w: verification reported not verified", ErrVerificationFailed)
	}

	p.verified = true
	p.verifiedAt = time.Now()
	return nil
}

// MatchesMaterial returns true if any resolved dependency matches the given
// URI, algorithm, and digest.
func (p *Provenance) MatchesMaterial(uri, algorithm, digest string) bool {
	for _, dep := range p.buildDefinition.ResolvedDependencies {
		if dep.URI != uri {
			continue
		}
		if d, ok := dep.Digest[algorithm]; ok && d == digest {
			return true
		}
	}
	return false
}
