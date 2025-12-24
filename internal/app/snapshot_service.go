package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
)

// SnapshotService provides high-level snapshot operations.
type SnapshotService struct {
	manager *snapshot.Manager
	baseDir string
}

// NewSnapshotService creates a new SnapshotService.
func NewSnapshotService(baseDir string) *SnapshotService {
	snapshotDir := filepath.Join(baseDir, "snapshots")
	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)

	return &SnapshotService{
		manager: manager,
		baseDir: baseDir,
	}
}

// DefaultSnapshotService creates a SnapshotService using the default preflight directory.
func DefaultSnapshotService() (*SnapshotService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(home, ".preflight")
	return NewSnapshotService(baseDir), nil
}

// BeforeApply creates snapshots for the given paths before applying changes.
func (s *SnapshotService) BeforeApply(ctx context.Context, paths []string) (*snapshot.Set, error) {
	return s.manager.BeforeApply(ctx, paths)
}

// Restore restores files from a snapshot set.
func (s *SnapshotService) Restore(ctx context.Context, snapshotSetID string) error {
	return s.manager.Restore(ctx, snapshotSetID)
}

// GetSnapshotSet retrieves a snapshot set by ID.
func (s *SnapshotService) GetSnapshotSet(ctx context.Context, id string) (*snapshot.Set, error) {
	return s.manager.GetSet(ctx, id)
}

// ListSnapshotSets returns all snapshot sets.
func (s *SnapshotService) ListSnapshotSets(ctx context.Context) ([]snapshot.Set, error) {
	return s.manager.ListSets(ctx)
}
