// Package lockfile provides adapters for lockfile persistence.
package lockfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/lock"
	"gopkg.in/yaml.v3"
)

// YAMLRepository implements lock.Repository using YAML files.
type YAMLRepository struct{}

// NewYAMLRepository creates a new YAML-based lockfile repository.
func NewYAMLRepository() *YAMLRepository {
	return &YAMLRepository{}
}

// Load reads a lockfile from the given path.
func (r *YAMLRepository) Load(_ context.Context, path string) (*lock.Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lock.ErrLockfileNotFound
		}
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var dto lock.LockfileDTO
	if err := yaml.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("%w: %w", lock.ErrLockfileCorrupt, err)
	}

	lockfile, err := lock.LockfileFromDTO(dto)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", lock.ErrLockfileCorrupt, err)
	}

	return lockfile, nil
}

// Save writes a lockfile to the given path.
func (r *YAMLRepository) Save(_ context.Context, path string, lockfile *lock.Lockfile) error {
	dto := lock.LockfileToDTO(lockfile)

	data, err := yaml.Marshal(&dto)
	if err != nil {
		return fmt.Errorf("%w: %w", lock.ErrSaveFailed, err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%w: failed to create directory: %w", lock.ErrSaveFailed, err)
	}

	// Write atomically by writing to temp file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("%w: %w", lock.ErrSaveFailed, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("%w: %w", lock.ErrSaveFailed, err)
	}

	return nil
}

// Exists returns true if a lockfile exists at the given path.
func (r *YAMLRepository) Exists(_ context.Context, path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Ensure YAMLRepository implements lock.Repository.
var _ lock.Repository = (*YAMLRepository)(nil)
