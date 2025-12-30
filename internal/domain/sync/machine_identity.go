// Package sync provides multi-machine synchronization primitives for Preflight v4.
// It implements version vectors for causal ordering, conflict detection, and
// resolution strategies for lock files across multiple machines.
package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// MachineID is a stable, persistent identifier for a machine.
// It is stored outside the git repository (typically ~/.preflight/machine-id)
// to ensure uniqueness per physical machine regardless of repo clones.
type MachineID struct {
	id string
}

var (
	// ErrInvalidMachineID indicates the machine ID format is invalid.
	ErrInvalidMachineID = errors.New("invalid machine ID format")

	// ErrMachineIDNotFound indicates no machine ID file exists.
	ErrMachineIDNotFound = errors.New("machine ID not found")

	// uuidRegex matches a valid UUID v4 format.
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// NewMachineID creates a new random machine identifier using UUID v4.
func NewMachineID() MachineID {
	return MachineID{id: uuid.New().String()}
}

// ParseMachineID parses an existing machine ID string.
// Returns ErrInvalidMachineID if the format is invalid.
func ParseMachineID(s string) (MachineID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return MachineID{}, ErrInvalidMachineID
	}

	// Validate UUID format
	if !uuidRegex.MatchString(strings.ToLower(s)) {
		return MachineID{}, fmt.Errorf("%w: %s", ErrInvalidMachineID, s)
	}

	return MachineID{id: strings.ToLower(s)}, nil
}

// String returns the string representation of the machine ID.
func (m MachineID) String() string {
	return m.id
}

// IsZero returns true if the machine ID is empty/uninitialized.
func (m MachineID) IsZero() bool {
	return m.id == ""
}

// Equal returns true if two machine IDs are identical.
func (m MachineID) Equal(other MachineID) bool {
	return m.id == other.id
}

// ShortID returns a truncated version for display (first 8 characters).
func (m MachineID) ShortID() string {
	if len(m.id) < 8 {
		return m.id
	}
	return m.id[:8]
}

// MachineIdentityRepository provides persistence for machine identity.
type MachineIdentityRepository interface {
	// Load reads the machine ID from persistent storage.
	Load() (MachineID, error)

	// Save writes the machine ID to persistent storage.
	Save(id MachineID) error

	// Exists returns true if a machine ID is already stored.
	Exists() bool

	// Path returns the file path where the machine ID is stored.
	Path() string
}

// FileMachineIdentityRepository stores machine ID in a local file.
type FileMachineIdentityRepository struct {
	path string
}

// NewFileMachineIdentityRepository creates a repository that stores the machine ID
// at the specified path. The default path is ~/.preflight/machine-id.
func NewFileMachineIdentityRepository(path string) *FileMachineIdentityRepository {
	return &FileMachineIdentityRepository{path: path}
}

// DefaultMachineIDPath returns the default path for storing machine identity.
func DefaultMachineIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".preflight/machine-id"
	}
	return filepath.Join(home, ".preflight", "machine-id")
}

// Load reads the machine ID from the file.
func (r *FileMachineIdentityRepository) Load() (MachineID, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return MachineID{}, ErrMachineIDNotFound
		}
		return MachineID{}, fmt.Errorf("failed to read machine ID: %w", err)
	}

	return ParseMachineID(string(data))
}

// Save writes the machine ID to the file, creating parent directories if needed.
func (r *FileMachineIdentityRepository) Save(id MachineID) error {
	if id.IsZero() {
		return ErrInvalidMachineID
	}

	// Ensure parent directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(r.path, []byte(id.String()+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write machine ID: %w", err)
	}

	return nil
}

// Exists returns true if the machine ID file exists.
func (r *FileMachineIdentityRepository) Exists() bool {
	_, err := os.Stat(r.path)
	return err == nil
}

// Path returns the file path where the machine ID is stored.
func (r *FileMachineIdentityRepository) Path() string {
	return r.path
}

// LoadOrCreate loads an existing machine ID or creates a new one.
// This is the primary entry point for obtaining a machine's identity.
func LoadOrCreate(repo MachineIdentityRepository) (MachineID, error) {
	if repo.Exists() {
		return repo.Load()
	}

	// Create new machine ID
	id := NewMachineID()
	if err := repo.Save(id); err != nil {
		return MachineID{}, fmt.Errorf("failed to save new machine ID: %w", err)
	}

	return id, nil
}

// GetMachineID is a convenience function that uses the default path.
func GetMachineID() (MachineID, error) {
	repo := NewFileMachineIdentityRepository(DefaultMachineIDPath())
	return LoadOrCreate(repo)
}
