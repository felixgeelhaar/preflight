package drift

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateStore handles persistence of AppliedState.
type StateStore struct {
	path string
	mu   sync.RWMutex
}

// NewStateStore creates a new StateStore.
func NewStateStore(path string) *StateStore {
	return &StateStore{
		path: path,
	}
}

// Load reads the applied state from disk.
func (s *StateStore) Load(ctx context.Context) (*AppliedState, error) {
	_ = ctx // Reserved for future cancellation support

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return NewAppliedState(), nil
	}
	if err != nil {
		return nil, err
	}

	var state AppliedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}

	return &state, nil
}

// Save writes the applied state to disk.
func (s *StateStore) Save(ctx context.Context, state *AppliedState) error {
	_ = ctx // Reserved for future cancellation support

	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists (0700 for privacy - contains file tracking data)
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}

// UpdateFile updates a single file in the state.
func (s *StateStore) UpdateFile(ctx context.Context, path, hash, sourceLayer string, appliedAt time.Time) error {
	_ = ctx // Reserved for future cancellation support

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	state.SetFile(path, hash, sourceLayer, appliedAt)

	return s.saveUnsafe(state)
}

// RemoveFile removes a file from the state.
func (s *StateStore) RemoveFile(ctx context.Context, path string) error {
	_ = ctx // Reserved for future cancellation support

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	state.RemoveFile(path)

	return s.saveUnsafe(state)
}

// GetFile retrieves a file's state.
func (s *StateStore) GetFile(ctx context.Context, path string) (FileState, bool, error) {
	_ = ctx // Reserved for future cancellation support

	s.mu.RLock()
	defer s.mu.RUnlock()

	state, err := s.loadUnsafe()
	if err != nil {
		return FileState{}, false, err
	}

	fileState, exists := state.GetFile(path)
	return fileState, exists, nil
}

// Clear removes all files from the state.
func (s *StateStore) Clear(ctx context.Context) error {
	_ = ctx // Reserved for future cancellation support

	s.mu.Lock()
	defer s.mu.Unlock()

	state := NewAppliedState()
	return s.saveUnsafe(state)
}

// loadUnsafe loads state without locking (caller must hold lock).
func (s *StateStore) loadUnsafe() (*AppliedState, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return NewAppliedState(), nil
	}
	if err != nil {
		return nil, err
	}

	var state AppliedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}

	return &state, nil
}

// saveUnsafe saves state without locking (caller must hold lock).
func (s *StateStore) saveUnsafe(state *AppliedState) error {
	// Ensure directory exists (0700 for privacy - contains file tracking data)
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o600)
}
