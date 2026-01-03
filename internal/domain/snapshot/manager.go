package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrSetNotFound is returned when a snapshot set cannot be found.
var ErrSetNotFound = errors.New("snapshot set not found")

// Manager orchestrates snapshot operations.
type Manager struct {
	store Store
	mu    sync.RWMutex
	index *snapshotSetIndex
}

// snapshotSetIndex stores metadata for all snapshot sets.
type snapshotSetIndex struct {
	Sets map[string]snapshotSetEntry `json:"sets"`
}

// snapshotSetEntry stores metadata for a single snapshot set.
type snapshotSetEntry struct {
	ID          string    `json:"id"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
	SnapshotIDs []string  `json:"snapshot_ids"`
}

// NewManager creates a new snapshot manager.
func NewManager(store Store) *Manager {
	return &Manager{
		store: store,
	}
}

// BeforeApply creates snapshots for all existing files before applying changes.
func (m *Manager) BeforeApply(ctx context.Context, paths []string) (*Set, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshots := make([]Snapshot, 0, len(paths))
	snapshotIDs := make([]string, 0, len(paths))

	for _, path := range paths {
		// Skip non-existent files
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Save snapshot
		snap, err := m.store.Save(ctx, path, content)
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, *snap)
		snapshotIDs = append(snapshotIDs, snap.ID)
	}

	// Create snapshot set
	now := time.Now()
	set := Set{
		ID:        uuid.New().String(),
		Snapshots: snapshots,
		CreatedAt: now,
		Reason:    string(ReasonApply),
	}

	// Persist snapshot set index
	if err := m.saveSet(ctx, &set, snapshotIDs); err != nil {
		return nil, err
	}

	return &set, nil
}

// Restore restores files from a snapshot set.
func (m *Manager) Restore(ctx context.Context, snapshotSetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	set, snapshotIDs, err := m.loadSet(ctx, snapshotSetID)
	if err != nil {
		return err
	}

	for _, snapID := range snapshotIDs {
		// Get snapshot content
		content, err := m.store.Get(ctx, snapID)
		if err != nil {
			if errors.Is(err, ErrSnapshotNotFound) {
				continue
			}
			return err
		}

		// Find the path for this snapshot
		var snapPath string
		for _, snap := range set.Snapshots {
			if snap.ID == snapID {
				snapPath = snap.Path
				break
			}
		}

		if snapPath == "" {
			continue
		}

		// Ensure parent directory exists (0700 for privacy)
		dir := filepath.Dir(snapPath)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}

		// Restore file (0600 for privacy - may contain sensitive content)
		if err := os.WriteFile(snapPath, content, 0o600); err != nil {
			return err
		}
	}

	return nil
}

// GetSet retrieves a snapshot set by ID.
func (m *Manager) GetSet(ctx context.Context, id string) (*Set, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	set, _, err := m.loadSet(ctx, id)
	return set, err
}

// ListSets returns all snapshot sets.
func (m *Manager) ListSets(ctx context.Context) ([]Set, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	index, err := m.loadIndex(ctx)
	if err != nil {
		return nil, err
	}

	sets := make([]Set, 0, len(index.Sets))
	for _, entry := range index.Sets {
		set := Set{
			ID:        entry.ID,
			Reason:    entry.Reason,
			CreatedAt: entry.CreatedAt,
			Snapshots: make([]Snapshot, 0, len(entry.SnapshotIDs)),
		}

		// Load snapshot metadata
		for _, snapID := range entry.SnapshotIDs {
			// We just need minimal info, actual content isn't needed for listing
			set.Snapshots = append(set.Snapshots, Snapshot{ID: snapID})
		}

		sets = append(sets, set)
	}

	return sets, nil
}

// saveSet persists a snapshot set to the index.
func (m *Manager) saveSet(ctx context.Context, set *Set, snapshotIDs []string) error {
	index, err := m.loadIndex(ctx)
	if err != nil {
		return err
	}

	index.Sets[set.ID] = snapshotSetEntry{
		ID:          set.ID,
		Reason:      set.Reason,
		CreatedAt:   set.CreatedAt,
		SnapshotIDs: snapshotIDs,
	}

	return m.persistIndex(ctx, index)
}

// loadSet loads a snapshot set from the index.
func (m *Manager) loadSet(ctx context.Context, id string) (*Set, []string, error) {
	index, err := m.loadIndex(ctx)
	if err != nil {
		return nil, nil, err
	}

	entry, ok := index.Sets[id]
	if !ok {
		return nil, nil, ErrSetNotFound
	}

	snapshots := make([]Snapshot, 0, len(entry.SnapshotIDs))
	for _, snapID := range entry.SnapshotIDs {
		// Load full snapshot info
		snaps, err := m.listAllSnapshots(ctx)
		if err != nil {
			return nil, nil, err
		}

		for _, snap := range snaps {
			if snap.ID == snapID {
				snapshots = append(snapshots, snap)
				break
			}
		}
	}

	set := &Set{
		ID:        entry.ID,
		Reason:    entry.Reason,
		CreatedAt: entry.CreatedAt,
		Snapshots: snapshots,
	}

	return set, entry.SnapshotIDs, nil
}

// listAllSnapshots lists all snapshots from the store.
func (m *Manager) listAllSnapshots(ctx context.Context) ([]Snapshot, error) {
	_ = ctx // Reserved for future cancellation support
	// This is a workaround since we need to get all snapshots
	// In a real implementation, the store would have a ListAll method
	fileStore, ok := m.store.(*FileStore)
	if !ok {
		return nil, nil
	}

	index, err := fileStore.loadIndex()
	if err != nil {
		return nil, err
	}

	snapshots := make([]Snapshot, 0, len(index.Snapshots))
	for _, entry := range index.Snapshots {
		snapshots = append(snapshots, Snapshot{
			ID:        entry.ID,
			Path:      entry.Path,
			Hash:      entry.Hash,
			Size:      entry.Size,
			CreatedAt: entry.CreatedAt,
		})
	}

	return snapshots, nil
}

// loadIndex loads the snapshot set index.
func (m *Manager) loadIndex(ctx context.Context) (*snapshotSetIndex, error) {
	_ = ctx // Reserved for future cancellation support
	if m.index != nil {
		return m.index, nil
	}

	// Get base path from store
	fileStore, ok := m.store.(*FileStore)
	if !ok {
		return &snapshotSetIndex{Sets: make(map[string]snapshotSetEntry)}, nil
	}

	indexPath := filepath.Join(fileStore.basePath, "sets.json")

	data, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) {
		m.index = &snapshotSetIndex{Sets: make(map[string]snapshotSetEntry)}
		return m.index, nil
	}
	if err != nil {
		return nil, err
	}

	var index snapshotSetIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	if index.Sets == nil {
		index.Sets = make(map[string]snapshotSetEntry)
	}

	m.index = &index
	return m.index, nil
}

// persistIndex saves the snapshot set index.
func (m *Manager) persistIndex(ctx context.Context, index *snapshotSetIndex) error {
	_ = ctx // Reserved for future cancellation support
	// Get base path from store
	fileStore, ok := m.store.(*FileStore)
	if !ok {
		return nil
	}

	// Ensure directory exists (0700 for privacy - stores user file backups)
	if err := os.MkdirAll(fileStore.basePath, 0o700); err != nil {
		return err
	}

	indexPath := filepath.Join(fileStore.basePath, "sets.json")

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	m.index = index
	return os.WriteFile(indexPath, data, 0o600)
}
