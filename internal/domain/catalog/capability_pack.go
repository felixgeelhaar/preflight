package catalog

import (
	"errors"
	"fmt"
)

// CapabilityPack errors.
var (
	ErrEmptyPackID = errors.New("pack ID cannot be empty")
)

// CapabilityPack represents a role-based collection of presets and tools.
// For example: "go-developer", "frontend", "devops".
type CapabilityPack struct {
	id       string
	metadata Metadata
	presets  []PresetID
	tools    []string
}

// NewCapabilityPack creates a new CapabilityPack entity.
func NewCapabilityPack(id string, metadata Metadata) (CapabilityPack, error) {
	if id == "" {
		return CapabilityPack{}, ErrEmptyPackID
	}

	if metadata.IsZero() {
		return CapabilityPack{}, ErrInvalidMetadata
	}

	return CapabilityPack{
		id:       id,
		metadata: metadata,
		presets:  []PresetID{},
		tools:    []string{},
	}, nil
}

// ID returns the pack identifier.
func (c CapabilityPack) ID() string {
	return c.id
}

// Metadata returns the pack metadata.
func (c CapabilityPack) Metadata() Metadata {
	return c.metadata
}

// Presets returns the list of preset IDs in this pack.
func (c CapabilityPack) Presets() []PresetID {
	result := make([]PresetID, len(c.presets))
	copy(result, c.presets)
	return result
}

// Tools returns the list of tool names in this pack.
func (c CapabilityPack) Tools() []string {
	result := make([]string, len(c.tools))
	copy(result, c.tools)
	return result
}

// WithPresets returns a new CapabilityPack with presets set.
func (c CapabilityPack) WithPresets(presets []PresetID) CapabilityPack {
	newPresets := make([]PresetID, len(presets))
	copy(newPresets, presets)

	return CapabilityPack{
		id:       c.id,
		metadata: c.metadata,
		presets:  newPresets,
		tools:    c.tools,
	}
}

// WithTools returns a new CapabilityPack with tools set.
func (c CapabilityPack) WithTools(tools []string) CapabilityPack {
	newTools := make([]string, len(tools))
	copy(newTools, tools)

	return CapabilityPack{
		id:       c.id,
		metadata: c.metadata,
		presets:  c.presets,
		tools:    newTools,
	}
}

// HasPreset returns true if this pack includes the given preset.
func (c CapabilityPack) HasPreset(id PresetID) bool {
	for _, preset := range c.presets {
		if preset.Equals(id) {
			return true
		}
	}
	return false
}

// HasTool returns true if this pack includes the given tool.
func (c CapabilityPack) HasTool(name string) bool {
	for _, tool := range c.tools {
		if tool == name {
			return true
		}
	}
	return false
}

// IsZero returns true if this is a zero-value CapabilityPack.
func (c CapabilityPack) IsZero() bool {
	return c.id == ""
}

// String returns a summary string.
func (c CapabilityPack) String() string {
	return fmt.Sprintf("%s (%d presets, %d tools)", c.id, len(c.presets), len(c.tools))
}
