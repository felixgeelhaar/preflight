package app

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/felixgeelhaar/preflight/internal/domain/drift"
)

// DriftService provides high-level drift detection operations.
type DriftService struct {
	store   *drift.StateStore
	baseDir string
	fs      *filesystem.RealFileSystem
}

// NewDriftService creates a new DriftService.
func NewDriftService(baseDir string) *DriftService {
	statePath := filepath.Join(baseDir, "state.json")
	store := drift.NewStateStore(statePath)

	return &DriftService{
		store:   store,
		baseDir: baseDir,
		fs:      filesystem.NewRealFileSystem(),
	}
}

// DefaultDriftService creates a DriftService using the default preflight directory.
func DefaultDriftService() (*DriftService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(home, ".preflight")
	return NewDriftService(baseDir), nil
}

// RecordApplied records that a file was applied by preflight.
func (s *DriftService) RecordApplied(ctx context.Context, path, sourceLayer string) error {
	hash, err := s.fs.FileHash(path)
	if err != nil {
		return err
	}

	return s.store.UpdateFile(ctx, path, hash, sourceLayer, time.Now())
}

// CheckDrift checks if a file has drifted from its applied state.
func (s *DriftService) CheckDrift(ctx context.Context, path string) (drift.Drift, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return drift.Drift{}, err
	}

	detector := drift.NewDetector(s.fs, state)
	return detector.Detect(ctx, path)
}

// CheckAll checks all tracked files for drift.
func (s *DriftService) CheckAll(ctx context.Context) ([]drift.Drift, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	detector := drift.NewDetector(s.fs, state)
	return detector.DetectAll(ctx)
}

// CheckPaths checks specific paths for drift.
func (s *DriftService) CheckPaths(ctx context.Context, paths []string) ([]drift.Drift, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	detector := drift.NewDetector(s.fs, state)
	return detector.DetectPaths(ctx, paths)
}

// RemoveTracking removes a file from drift tracking.
func (s *DriftService) RemoveTracking(ctx context.Context, path string) error {
	return s.store.RemoveFile(ctx, path)
}

// ListTrackedFiles returns all files being tracked for drift.
func (s *DriftService) ListTrackedFiles(ctx context.Context) ([]drift.FileState, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	return state.ListFiles(), nil
}
