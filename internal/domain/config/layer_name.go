package config

import (
	"errors"
	"regexp"
	"strings"
)

// LayerName is a validated layer identifier.
// Layer names follow the pattern: base, identity.work, role.go-developer, device.macbook-pro
type LayerName struct {
	value string
}

// Errors for LayerName validation.
var (
	ErrEmptyLayerName   = errors.New("layer name cannot be empty")
	ErrInvalidLayerName = errors.New("layer name contains invalid characters")
)

// validLayerNamePattern matches valid layer names.
// Allowed: alphanumeric, dots, hyphens, underscores.
var validLayerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// NewLayerName creates a new LayerName from a string.
// Returns an error if the name is empty or contains invalid characters.
func NewLayerName(s string) (LayerName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return LayerName{}, ErrEmptyLayerName
	}

	if !validLayerNamePattern.MatchString(trimmed) {
		return LayerName{}, ErrInvalidLayerName
	}

	return LayerName{value: trimmed}, nil
}

// String returns the layer name as a string.
func (n LayerName) String() string {
	return n.value
}

// IsZero returns true if the LayerName is the zero value.
func (n LayerName) IsZero() bool {
	return n.value == ""
}
