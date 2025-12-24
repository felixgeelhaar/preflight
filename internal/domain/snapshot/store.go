package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrSnapshotNotFound is returned when a snapshot cannot be found.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// Store provides snapshot persistence operations.
type Store interface {
	Save(ctx context.Context, path string, content []byte) (*Snapshot, error)
	Get(ctx context.Context, id string) ([]byte, error)
	List(ctx context.Context, path string) ([]Snapshot, error)
	Delete(ctx context.Context, id string) error
	Cleanup(ctx context.Context, maxAge time.Duration) (int, error)
}

// snapshotIndex stores metadata for all snapshots.
type snapshotIndex struct {
	Snapshots map[string]indexEntry `json:"snapshots"`
}

// indexEntry stores metadata for a single snapshot.
type indexEntry struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Filename  string    `json:"filename"`
}

// FileStore implements Store using the local filesystem.
type FileStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStore creates a new FileStore at the given base path.
func NewFileStore(basePath string) *FileStore {
	return &FileStore{
		basePath: basePath,
	}
}

// Save stores file content and returns a snapshot.
func (s *FileStore) Save(ctx context.Context, path string, content []byte) (*Snapshot, error) {
	_ = ctx // Reserved for future cancellation support
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure base directory exists
	if err := os.MkdirAll(s.basePath, 0o755); err != nil {
		return nil, err
	}

	// Generate snapshot metadata
	id := uuid.New().String()
	hash := sha256Hash(content)
	filename := id + ".snapshot"
	now := time.Now()

	// Write snapshot file
	snapshotPath := filepath.Join(s.basePath, filename)
	if err := os.WriteFile(snapshotPath, content, 0o644); err != nil {
		return nil, err
	}

	// Update index
	index, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	index.Snapshots[id] = indexEntry{
		ID:        id,
		Path:      path,
		Hash:      hash,
		Size:      int64(len(content)),
		CreatedAt: now,
		Filename:  filename,
	}

	if err := s.saveIndex(index); err != nil {
		// Clean up the snapshot file on failure
		_ = os.Remove(snapshotPath)
		return nil, err
	}

	snap := Snapshot{
		ID:        id,
		Path:      path,
		Hash:      hash,
		Size:      int64(len(content)),
		CreatedAt: now,
	}

	return &snap, nil
}

// Get retrieves snapshot content by ID.
func (s *FileStore) Get(ctx context.Context, id string) ([]byte, error) {
	_ = ctx // Reserved for future cancellation support
	s.mu.RLock()
	defer s.mu.RUnlock()

	index, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	entry, ok := index.Snapshots[id]
	if !ok {
		return nil, ErrSnapshotNotFound
	}

	snapshotPath := filepath.Join(s.basePath, entry.Filename)
	return os.ReadFile(snapshotPath)
}

// List returns all snapshots for a given path.
func (s *FileStore) List(ctx context.Context, path string) ([]Snapshot, error) {
	_ = ctx // Reserved for future cancellation support
	s.mu.RLock()
	defer s.mu.RUnlock()

	index, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	result := make([]Snapshot, 0)
	for _, entry := range index.Snapshots {
		if entry.Path == path {
			result = append(result, Snapshot{
				ID:        entry.ID,
				Path:      entry.Path,
				Hash:      entry.Hash,
				Size:      entry.Size,
				CreatedAt: entry.CreatedAt,
			})
		}
	}

	return result, nil
}

// Delete removes a snapshot by ID.
func (s *FileStore) Delete(ctx context.Context, id string) error {
	_ = ctx // Reserved for future cancellation support
	s.mu.Lock()
	defer s.mu.Unlock()

	index, err := s.loadIndex()
	if err != nil {
		return err
	}

	entry, ok := index.Snapshots[id]
	if !ok {
		return ErrSnapshotNotFound
	}

	// Remove snapshot file
	snapshotPath := filepath.Join(s.basePath, entry.Filename)
	if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Update index
	delete(index.Snapshots, id)
	return s.saveIndex(index)
}

// Cleanup removes snapshots older than maxAge.
func (s *FileStore) Cleanup(ctx context.Context, maxAge time.Duration) (int, error) {
	_ = ctx // Reserved for future cancellation support
	s.mu.Lock()
	defer s.mu.Unlock()

	index, err := s.loadIndex()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	count := 0
	toDelete := make([]string, 0)

	for id, entry := range index.Snapshots {
		if now.Sub(entry.CreatedAt) > maxAge {
			// Remove snapshot file
			snapshotPath := filepath.Join(s.basePath, entry.Filename)
			_ = os.Remove(snapshotPath)
			toDelete = append(toDelete, id)
			count++
		}
	}

	for _, id := range toDelete {
		delete(index.Snapshots, id)
	}

	if count > 0 {
		if err := s.saveIndex(index); err != nil {
			return count, err
		}
	}

	return count, nil
}

// loadIndex loads the snapshot index from disk.
func (s *FileStore) loadIndex() (*snapshotIndex, error) {
	indexPath := filepath.Join(s.basePath, "index.json")

	data, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) {
		return &snapshotIndex{
			Snapshots: make(map[string]indexEntry),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var index snapshotIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	if index.Snapshots == nil {
		index.Snapshots = make(map[string]indexEntry)
	}

	return &index, nil
}

// saveIndex saves the snapshot index to disk.
func (s *FileStore) saveIndex(index *snapshotIndex) error {
	indexPath := filepath.Join(s.basePath, "index.json")

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0o644)
}

// sha256Hash returns the SHA256 hash of content.
func sha256Hash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// Ensure FileStore implements Store.
var _ Store = (*FileStore)(nil)
