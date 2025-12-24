// Package snapshot provides automatic backup functionality before file modifications.
package snapshot

import (
	"time"

	"github.com/google/uuid"
)

// Reason describes why a snapshot was created.
type Reason string

// Snapshot reason constants.
const (
	ReasonApply    Reason = "apply"
	ReasonFix      Reason = "fix"
	ReasonRollback Reason = "rollback"
)

// IsValid checks if the reason is a known valid reason.
func (r Reason) IsValid() bool {
	switch r {
	case ReasonApply, ReasonFix, ReasonRollback:
		return true
	default:
		return false
	}
}

// Snapshot represents a backup of a file at a point in time.
type Snapshot struct {
	ID        string
	Path      string
	Hash      string
	CreatedAt time.Time
	Size      int64
}

// NewSnapshot creates a new snapshot with a generated ID.
func NewSnapshot(path, hash string, size int64, createdAt time.Time) Snapshot {
	return Snapshot{
		ID:        uuid.New().String(),
		Path:      path,
		Hash:      hash,
		Size:      size,
		CreatedAt: createdAt,
	}
}

// IsExpired checks if the snapshot is older than the given duration.
func (s Snapshot) IsExpired(maxAge time.Duration) bool {
	return time.Since(s.CreatedAt) > maxAge
}

// Set represents a group of snapshots created together.
type Set struct {
	ID        string
	Snapshots []Snapshot
	CreatedAt time.Time
	Reason    string
}

// NewSet creates a new snapshot set with a generated ID.
func NewSet(reason string, snapshots []Snapshot, createdAt time.Time) Set {
	if snapshots == nil {
		snapshots = []Snapshot{}
	}
	return Set{
		ID:        uuid.New().String(),
		Snapshots: snapshots,
		CreatedAt: createdAt,
		Reason:    reason,
	}
}

// GetSnapshot returns a snapshot by path if it exists in the set.
func (s Set) GetSnapshot(path string) (Snapshot, bool) {
	for _, snap := range s.Snapshots {
		if snap.Path == path {
			return snap, true
		}
	}
	return Snapshot{}, false
}

// Paths returns all file paths in this snapshot set.
func (s Set) Paths() []string {
	paths := make([]string, 0, len(s.Snapshots))
	for _, snap := range s.Snapshots {
		paths = append(paths, snap.Path)
	}
	return paths
}
