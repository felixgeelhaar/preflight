// Package drift provides drift detection functionality for tracking
// files that have been modified outside of Preflight.
package drift

import (
	"fmt"
	"time"
)

// Type describes the source of drift.
type Type string

const (
	// TypeManual indicates the file was manually modified by the user.
	TypeManual Type = "manual"
	// TypeExternal indicates the file was modified by an external tool.
	TypeExternal Type = "external"
	// TypeUnknown indicates the drift source is unknown.
	TypeUnknown Type = "unknown"
	// TypeNone indicates no drift was detected.
	TypeNone Type = "none"
)

// Drift represents a detected drift in a managed file.
type Drift struct {
	Path         string
	CurrentHash  string
	ExpectedHash string
	LastApplied  time.Time
	Type         Type
}

// NewDrift creates a new Drift instance.
func NewDrift(path, currentHash, expectedHash string, lastApplied time.Time, driftType Type) Drift {
	return Drift{
		Path:         path,
		CurrentHash:  currentHash,
		ExpectedHash: expectedHash,
		LastApplied:  lastApplied,
		Type:         driftType,
	}
}

// HasDrift returns true if the current hash differs from the expected hash.
func (d Drift) HasDrift() bool {
	return d.CurrentHash != d.ExpectedHash
}

// Description returns a human-readable description of the drift.
func (d Drift) Description() string {
	switch d.Type {
	case TypeManual:
		return fmt.Sprintf("%s was manually modified", d.Path)
	case TypeExternal:
		return fmt.Sprintf("%s was modified by an external tool", d.Path)
	case TypeUnknown:
		return fmt.Sprintf("%s was modified by an unknown source", d.Path)
	case TypeNone:
		return fmt.Sprintf("%s has no drift", d.Path)
	default:
		return fmt.Sprintf("%s has drift of type %s", d.Path, d.Type)
	}
}

// FileState tracks the state of a file that was applied by Preflight.
type FileState struct {
	Path        string    `json:"path"`
	AppliedHash string    `json:"applied_hash"`
	AppliedAt   time.Time `json:"applied_at"`
	SourceLayer string    `json:"source_layer"`
}

// AppliedState tracks all files that have been applied by Preflight.
type AppliedState struct {
	Files map[string]FileState `json:"files"`
}

// NewAppliedState creates a new AppliedState instance.
func NewAppliedState() *AppliedState {
	return &AppliedState{
		Files: make(map[string]FileState),
	}
}

// SetFile records that a file was applied.
func (s *AppliedState) SetFile(path, hash, sourceLayer string, appliedAt time.Time) {
	s.Files[path] = FileState{
		Path:        path,
		AppliedHash: hash,
		AppliedAt:   appliedAt,
		SourceLayer: sourceLayer,
	}
}

// GetFile retrieves the state of a file.
func (s *AppliedState) GetFile(path string) (FileState, bool) {
	state, exists := s.Files[path]
	return state, exists
}

// RemoveFile removes a file from tracking.
func (s *AppliedState) RemoveFile(path string) {
	delete(s.Files, path)
}

// ListFiles returns all tracked files.
func (s *AppliedState) ListFiles() []FileState {
	files := make([]FileState, 0, len(s.Files))
	for _, state := range s.Files {
		files = append(files, state)
	}
	return files
}

// Clear removes all tracked files.
func (s *AppliedState) Clear() {
	s.Files = make(map[string]FileState)
}
