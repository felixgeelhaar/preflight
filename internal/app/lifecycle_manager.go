package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// LifecycleManager provides file lifecycle management by combining
// snapshot and drift services.
type LifecycleManager struct {
	snapshot *SnapshotService
	drift    *DriftService
}

// NewLifecycleManager creates a new LifecycleManager.
func NewLifecycleManager(snapshot *SnapshotService, drift *DriftService) *LifecycleManager {
	return &LifecycleManager{
		snapshot: snapshot,
		drift:    drift,
	}
}

// DefaultLifecycleManager creates a LifecycleManager using the default preflight directory.
func DefaultLifecycleManager() (*LifecycleManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(home, ".preflight")

	snapshot := NewSnapshotService(baseDir)
	drift := NewDriftService(baseDir)

	return NewLifecycleManager(snapshot, drift), nil
}

// BeforeModify takes a snapshot of the file before modification.
// Returns nil if the file doesn't exist (nothing to snapshot).
func (m *LifecycleManager) BeforeModify(ctx context.Context, path string) error {
	// Expand path
	expandedPath := ports.ExpandPath(path)

	// Check if file exists - if not, nothing to snapshot
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return nil
	}

	// Take snapshot
	_, err := m.snapshot.BeforeApply(ctx, []string{expandedPath})
	return err
}

// AfterApply records that a file was applied by preflight.
// This enables drift detection for the file.
func (m *LifecycleManager) AfterApply(ctx context.Context, path, sourceLayer string) error {
	// Expand path
	expandedPath := ports.ExpandPath(path)

	// Record in drift tracking
	return m.drift.RecordApplied(ctx, expandedPath, sourceLayer)
}

// Snapshot returns the underlying SnapshotService.
func (m *LifecycleManager) Snapshot() *SnapshotService {
	return m.snapshot
}

// Drift returns the underlying DriftService.
func (m *LifecycleManager) Drift() *DriftService {
	return m.drift
}

// Ensure LifecycleManager implements ports.FileLifecycle.
var _ ports.FileLifecycle = (*LifecycleManager)(nil)
