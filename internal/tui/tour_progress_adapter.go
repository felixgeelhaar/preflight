package tui

import (
	"github.com/felixgeelhaar/preflight/internal/domain/tour"
)

// tourProgressAdapter adapts tour.Progress to TourProgress interface.
type tourProgressAdapter struct {
	*tour.Progress
}

// Ensure tourProgressAdapter implements TourProgress.
var _ TourProgress = (*tourProgressAdapter)(nil)

// tourProgressStoreAdapter adapts tour.ProgressStore to TourProgressStore interface.
type tourProgressStoreAdapter struct {
	*tour.ProgressStore
}

// Ensure tourProgressStoreAdapter implements TourProgressStore.
var _ TourProgressStore = (*tourProgressStoreAdapter)(nil)

// NewTourProgressStore creates a new progress store adapter with the default path.
func NewTourProgressStore() (TourProgressStore, error) {
	store, err := tour.NewProgressStore()
	if err != nil {
		return nil, err
	}
	return &tourProgressStoreAdapter{ProgressStore: store}, nil
}

// NewTourProgressStoreWithPath creates a new progress store adapter with a custom path.
func NewTourProgressStoreWithPath(path string) TourProgressStore {
	store := tour.NewProgressStoreWithPath(path)
	return &tourProgressStoreAdapter{ProgressStore: store}
}

// Load implements TourProgressStore.
func (s *tourProgressStoreAdapter) Load() (TourProgress, error) {
	progress, err := s.ProgressStore.Load()
	if err != nil {
		return nil, err
	}
	return &tourProgressAdapter{Progress: progress}, nil
}

// Save implements TourProgressStore.
func (s *tourProgressStoreAdapter) Save(progress TourProgress) error {
	// Type assert to get the underlying tour.Progress
	if adapter, ok := progress.(*tourProgressAdapter); ok {
		return s.ProgressStore.Save(adapter.Progress)
	}
	// If it's not our adapter, we can't save it
	return nil
}
