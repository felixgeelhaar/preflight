package compiler

import (
	"errors"
	"regexp"
	"strings"
)

// StepID uniquely identifies a step within the compilation.
// Format: provider:action:resource (e.g., "brew:install:git")
type StepID struct {
	value string
}

// Errors for StepID validation.
var (
	ErrEmptyStepID   = errors.New("step ID cannot be empty")
	ErrInvalidStepID = errors.New("step ID format invalid: must be alphanumeric with colons, hyphens, underscores, or slashes")
)

// stepIDPattern validates step ID format.
// Allows: alphanumeric, hyphens, underscores, slashes, separated by colons.
// Must not start or end with colon, no spaces.
var stepIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_/-]*(?::[a-zA-Z0-9][a-zA-Z0-9_/-]*)*$`)

// NewStepID creates a new StepID from a string.
func NewStepID(value string) (StepID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return StepID{}, ErrEmptyStepID
	}

	if !stepIDPattern.MatchString(trimmed) {
		return StepID{}, ErrInvalidStepID
	}

	return StepID{value: trimmed}, nil
}

// MustNewStepID creates a new StepID from a string, panicking on error.
// Use this for compile-time known values that should never fail validation.
func MustNewStepID(value string) StepID {
	id, err := NewStepID(value)
	if err != nil {
		panic("invalid step ID: " + value + ": " + err.Error())
	}
	return id
}

// String returns the string representation.
func (id StepID) String() string {
	return id.value
}

// Equals checks equality with another StepID.
func (id StepID) Equals(other StepID) bool {
	return id.value == other.value
}

// Provider extracts the provider name (first segment).
func (id StepID) Provider() string {
	parts := strings.SplitN(id.value, ":", 2)
	return parts[0]
}

// IsZero returns true if this is a zero-value StepID.
func (id StepID) IsZero() bool {
	return id.value == ""
}
