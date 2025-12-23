package catalog

import (
	"errors"
	"fmt"
	"strings"
)

// Preset errors.
var (
	ErrInvalidMetadata   = errors.New("metadata is invalid")
	ErrInvalidDifficulty = errors.New("invalid difficulty level")
)

// DifficultyLevel indicates the complexity/expertise required.
type DifficultyLevel string

const (
	DifficultyBeginner     DifficultyLevel = "beginner"
	DifficultyIntermediate DifficultyLevel = "intermediate"
	DifficultyAdvanced     DifficultyLevel = "advanced"
)

// String returns the difficulty level as a string.
func (d DifficultyLevel) String() string {
	return string(d)
}

// IsValid returns true if this is a known difficulty level.
func (d DifficultyLevel) IsValid() bool {
	switch d {
	case DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced:
		return true
	default:
		return false
	}
}

// ParseDifficultyLevel parses a string into a DifficultyLevel.
func ParseDifficultyLevel(s string) (DifficultyLevel, error) {
	level := DifficultyLevel(strings.ToLower(s))
	if !level.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidDifficulty, s)
	}
	return level, nil
}

// Preset represents a pre-configured bundle for a specific tool.
// It is an entity identified by its PresetID.
type Preset struct {
	id         PresetID
	metadata   Metadata
	difficulty DifficultyLevel
	config     map[string]interface{}
	requires   []PresetID
	conflicts  []PresetID
}

// NewPreset creates a new Preset entity.
func NewPreset(id PresetID, metadata Metadata, difficulty DifficultyLevel, config map[string]interface{}) (Preset, error) {
	if id.IsZero() {
		return Preset{}, ErrInvalidPresetID
	}

	if metadata.IsZero() {
		return Preset{}, ErrInvalidMetadata
	}

	// Copy config to ensure immutability
	configCopy := make(map[string]interface{}, len(config))
	for k, v := range config {
		configCopy[k] = v
	}

	return Preset{
		id:         id,
		metadata:   metadata,
		difficulty: difficulty,
		config:     configCopy,
		requires:   []PresetID{},
		conflicts:  []PresetID{},
	}, nil
}

// ID returns the preset identifier.
func (p Preset) ID() PresetID {
	return p.id
}

// Metadata returns the preset metadata.
func (p Preset) Metadata() Metadata {
	return p.metadata
}

// Difficulty returns the difficulty level.
func (p Preset) Difficulty() DifficultyLevel {
	return p.difficulty
}

// Config returns the preset configuration.
func (p Preset) Config() map[string]interface{} {
	result := make(map[string]interface{}, len(p.config))
	for k, v := range p.config {
		result[k] = v
	}
	return result
}

// Requires returns the list of required preset IDs.
func (p Preset) Requires() []PresetID {
	result := make([]PresetID, len(p.requires))
	copy(result, p.requires)
	return result
}

// Conflicts returns the list of conflicting preset IDs.
func (p Preset) Conflicts() []PresetID {
	result := make([]PresetID, len(p.conflicts))
	copy(result, p.conflicts)
	return result
}

// WithRequires returns a new Preset with required presets set.
func (p Preset) WithRequires(requires []PresetID) Preset {
	newRequires := make([]PresetID, len(requires))
	copy(newRequires, requires)

	return Preset{
		id:         p.id,
		metadata:   p.metadata,
		difficulty: p.difficulty,
		config:     p.config,
		requires:   newRequires,
		conflicts:  p.conflicts,
	}
}

// WithConflicts returns a new Preset with conflicting presets set.
func (p Preset) WithConflicts(conflicts []PresetID) Preset {
	newConflicts := make([]PresetID, len(conflicts))
	copy(newConflicts, conflicts)

	return Preset{
		id:         p.id,
		metadata:   p.metadata,
		difficulty: p.difficulty,
		config:     p.config,
		requires:   p.requires,
		conflicts:  newConflicts,
	}
}

// RequiresPreset returns true if this preset requires the given preset.
func (p Preset) RequiresPreset(id PresetID) bool {
	for _, req := range p.requires {
		if req.Equals(id) {
			return true
		}
	}
	return false
}

// ConflictsWith returns true if this preset conflicts with the given preset.
func (p Preset) ConflictsWith(id PresetID) bool {
	for _, conflict := range p.conflicts {
		if conflict.Equals(id) {
			return true
		}
	}
	return false
}

// IsZero returns true if this is a zero-value Preset.
func (p Preset) IsZero() bool {
	return p.id.IsZero()
}

// String returns a summary string.
func (p Preset) String() string {
	return fmt.Sprintf("%s (%s)", p.id.String(), p.difficulty)
}
