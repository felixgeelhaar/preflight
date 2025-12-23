// Package catalog provides presets and capability packs for the init wizard.
package catalog

import (
	"errors"
	"fmt"
	"strings"
)

// PresetID errors.
var (
	ErrEmptyPresetProvider = errors.New("preset provider cannot be empty")
	ErrEmptyPresetName     = errors.New("preset name cannot be empty")
	ErrInvalidPresetID     = errors.New("invalid preset ID format")
)

// PresetID uniquely identifies a preset within a provider.
// Format: "provider:name" (e.g., "nvim:balanced", "shell:starship").
// It is an immutable value object.
type PresetID struct {
	provider string
	name     string
}

// NewPresetID creates a new PresetID.
// Returns an error if provider or name is empty.
func NewPresetID(provider, name string) (PresetID, error) {
	if provider == "" {
		return PresetID{}, ErrEmptyPresetProvider
	}

	if name == "" {
		return PresetID{}, ErrEmptyPresetName
	}

	return PresetID{
		provider: provider,
		name:     name,
	}, nil
}

// ParsePresetID parses a preset ID string in the format "provider:name".
func ParsePresetID(s string) (PresetID, error) {
	if s == "" {
		return PresetID{}, fmt.Errorf("%w: empty string", ErrInvalidPresetID)
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return PresetID{}, fmt.Errorf("%w: missing colon separator", ErrInvalidPresetID)
	}

	return NewPresetID(parts[0], parts[1])
}

// Provider returns the provider component (e.g., "nvim", "shell").
func (p PresetID) Provider() string {
	return p.provider
}

// Name returns the preset name component (e.g., "balanced", "starship").
func (p PresetID) Name() string {
	return p.name
}

// String returns the preset ID in "provider:name" format.
func (p PresetID) String() string {
	return p.provider + ":" + p.name
}

// Equals returns true if two PresetIDs are equal.
func (p PresetID) Equals(other PresetID) bool {
	return p.provider == other.provider && p.name == other.name
}

// IsZero returns true if this is a zero-value PresetID.
func (p PresetID) IsZero() bool {
	return p.provider == "" && p.name == ""
}

// MatchesProvider returns true if this preset belongs to the given provider.
func (p PresetID) MatchesProvider(provider string) bool {
	return p.provider == provider && provider != ""
}
