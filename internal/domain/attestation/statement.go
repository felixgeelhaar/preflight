package attestation

import "fmt"

const (
	// StatementType is the in-toto statement type identifier.
	StatementType = "https://in-toto.io/Statement/v1"

	// PredicateTypeSLSAProvenanceV1 is the SLSA v1.0 provenance predicate type.
	PredicateTypeSLSAProvenanceV1 = "https://slsa.dev/provenance/v1"
)

// Subject identifies what the attestation is about.
type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

// Statement represents an in-toto attestation statement (immutable value object).
// See: https://in-toto.io/Statement/v1
type Statement struct {
	statementType string
	predicateType string
	subject       []Subject
	predicate     []byte
}

// NewStatement creates a new validated Statement.
// Returns ErrInvalidStatement if validation fails.
func NewStatement(predicateType string, subjects []Subject, predicate []byte) (Statement, error) {
	s := Statement{
		statementType: StatementType,
		predicateType: predicateType,
		subject:       deepCopySubjects(subjects),
		predicate:     copyBytes(predicate),
	}

	if err := s.Validate(); err != nil {
		return Statement{}, err
	}

	return s, nil
}

// Type returns the in-toto statement type.
func (s Statement) Type() string {
	return s.statementType
}

// PredicateType returns the predicate type URI.
func (s Statement) PredicateType() string {
	return s.predicateType
}

// Subject returns a copy of the subjects.
func (s Statement) Subject() []Subject {
	return deepCopySubjects(s.subject)
}

// Predicate returns a copy of the JSON-encoded predicate.
func (s Statement) Predicate() []byte {
	return copyBytes(s.predicate)
}

// IsZero returns true if the statement is the zero value.
func (s Statement) IsZero() bool {
	return s.statementType == "" && s.predicateType == "" && len(s.subject) == 0
}

// Validate checks that the statement is well-formed.
func (s Statement) Validate() error {
	if s.predicateType == "" {
		return fmt.Errorf("%w: predicate type is required", ErrInvalidStatement)
	}

	if len(s.subject) == 0 {
		return fmt.Errorf("%w: at least one subject is required", ErrInvalidStatement)
	}

	for i, subj := range s.subject {
		if subj.Name == "" {
			return fmt.Errorf("%w: subject[%d]: name is required", ErrInvalidStatement, i)
		}
		if len(subj.Digest) == 0 {
			return fmt.Errorf("%w: subject[%d]: at least one digest is required", ErrInvalidStatement, i)
		}
	}

	return nil
}

// SubjectMatchesDigest returns true if any subject matches the given name,
// algorithm, and digest value.
func (s Statement) SubjectMatchesDigest(name, algorithm, digest string) bool {
	for _, subj := range s.subject {
		if subj.Name != name {
			continue
		}
		if d, ok := subj.Digest[algorithm]; ok && d == digest {
			return true
		}
	}
	return false
}

// deepCopySubjects creates a deep copy of a subject slice.
func deepCopySubjects(subjects []Subject) []Subject {
	if subjects == nil {
		return nil
	}
	out := make([]Subject, len(subjects))
	for i, s := range subjects {
		out[i] = Subject{
			Name:   s.Name,
			Digest: copyDigestMap(s.Digest),
		}
	}
	return out
}

// copyDigestMap creates a copy of a digest map.
func copyDigestMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copyBytes creates a copy of a byte slice.
func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
