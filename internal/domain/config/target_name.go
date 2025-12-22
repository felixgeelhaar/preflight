package config

import (
	"errors"
	"regexp"
	"strings"
)

// TargetName is a validated target/profile identifier.
// Target names are simpler than layer names: alphanumeric, hyphens, underscores only.
type TargetName struct {
	value string
}

// Errors for TargetName validation.
var (
	ErrEmptyTargetName   = errors.New("target name cannot be empty")
	ErrInvalidTargetName = errors.New("target name contains invalid characters")
)

// validTargetNamePattern matches valid target names.
// Allowed: alphanumeric, hyphens, underscores (no dots - those are for layers).
var validTargetNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// NewTargetName creates a new TargetName from a string.
// Returns an error if the name is empty or contains invalid characters.
func NewTargetName(s string) (TargetName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return TargetName{}, ErrEmptyTargetName
	}

	if !validTargetNamePattern.MatchString(trimmed) {
		return TargetName{}, ErrInvalidTargetName
	}

	return TargetName{value: trimmed}, nil
}

// String returns the target name as a string.
func (n TargetName) String() string {
	return n.value
}

// IsZero returns true if the TargetName is the zero value.
func (n TargetName) IsZero() bool {
	return n.value == ""
}
