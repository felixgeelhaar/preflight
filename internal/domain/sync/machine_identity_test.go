package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMachineID(t *testing.T) {
	t.Parallel()

	id := NewMachineID()
	assert.False(t, id.IsZero())
	assert.Len(t, id.String(), 36) // UUID v4 length
}

func TestNewMachineID_Uniqueness(t *testing.T) {
	t.Parallel()

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewMachineID()
		assert.False(t, ids[id.String()], "generated duplicate ID")
		ids[id.String()] = true
	}
}

func TestParseMachineID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid UUID v4",
			input:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid UUID v4 uppercase",
			input:   "550E8400-E29B-41D4-A716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid with whitespace",
			input:   "  550e8400-e29b-41d4-a716-446655440000  \n",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "550e8400-e29b-41d4",
			wantErr: true,
		},
		{
			name:    "invalid version",
			input:   "550e8400-e29b-31d4-a716-446655440000", // version 3, not 4
			wantErr: true,
		},
		{
			name:    "invalid variant",
			input:   "550e8400-e29b-41d4-c716-446655440000", // variant c, not 8-b
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := ParseMachineID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, id.IsZero())
			} else {
				assert.NoError(t, err)
				assert.False(t, id.IsZero())
			}
		})
	}
}

func TestMachineID_String(t *testing.T) {
	t.Parallel()

	id, err := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
}

func TestMachineID_StringNormalizesToLowercase(t *testing.T) {
	t.Parallel()

	id, err := ParseMachineID("550E8400-E29B-41D4-A716-446655440000")
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id.String())
}

func TestMachineID_IsZero(t *testing.T) {
	t.Parallel()

	var zeroID MachineID
	assert.True(t, zeroID.IsZero())

	nonZeroID := NewMachineID()
	assert.False(t, nonZeroID.IsZero())
}

func TestMachineID_Equal(t *testing.T) {
	t.Parallel()

	id1, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	id2, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	id3, _ := ParseMachineID("660e8400-e29b-41d4-a716-446655440000")

	assert.True(t, id1.Equal(id2))
	assert.False(t, id1.Equal(id3))
}

func TestMachineID_ShortID(t *testing.T) {
	t.Parallel()

	id, _ := ParseMachineID("550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, "550e8400", id.ShortID())

	// Test with zero value
	var zeroID MachineID
	assert.Equal(t, "", zeroID.ShortID())
}

func TestFileMachineIdentityRepository_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".preflight", "machine-id")

	repo := NewFileMachineIdentityRepository(path)

	// Initially should not exist
	assert.False(t, repo.Exists())

	// Load should return not found error
	_, err := repo.Load()
	assert.ErrorIs(t, err, ErrMachineIDNotFound)

	// Save a new ID
	id := NewMachineID()
	err = repo.Save(id)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, repo.Exists())

	// Load should return the same ID
	loaded, err := repo.Load()
	require.NoError(t, err)
	assert.True(t, id.Equal(loaded))
}

func TestFileMachineIdentityRepository_SaveCreatesParentDirs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "deep", "nested", "dir", "machine-id")

	repo := NewFileMachineIdentityRepository(path)
	id := NewMachineID()

	err := repo.Save(id)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestFileMachineIdentityRepository_SaveZeroIDFails(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)
	var zeroID MachineID

	err := repo.Save(zeroID)
	assert.ErrorIs(t, err, ErrInvalidMachineID)
}

func TestFileMachineIdentityRepository_FilePermissions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)
	id := NewMachineID()

	err := repo.Save(id)
	require.NoError(t, err)

	// Check file permissions (should be 0600)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestFileMachineIdentityRepository_Path(t *testing.T) {
	t.Parallel()

	path := "/some/path/machine-id"
	repo := NewFileMachineIdentityRepository(path)
	assert.Equal(t, path, repo.Path())
}

func TestLoadOrCreate_CreatesNewID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)

	// Should create new ID
	id, err := LoadOrCreate(repo)
	require.NoError(t, err)
	assert.False(t, id.IsZero())

	// File should exist now
	assert.True(t, repo.Exists())
}

func TestLoadOrCreate_LoadsExistingID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)

	// Create and save an ID
	original := NewMachineID()
	err := repo.Save(original)
	require.NoError(t, err)

	// LoadOrCreate should return the existing ID
	loaded, err := LoadOrCreate(repo)
	require.NoError(t, err)
	assert.True(t, original.Equal(loaded))
}

func TestLoadOrCreate_Idempotent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)

	// Call LoadOrCreate multiple times
	id1, err := LoadOrCreate(repo)
	require.NoError(t, err)

	id2, err := LoadOrCreate(repo)
	require.NoError(t, err)

	id3, err := LoadOrCreate(repo)
	require.NoError(t, err)

	// All should be the same
	assert.True(t, id1.Equal(id2))
	assert.True(t, id2.Equal(id3))
}

func TestDefaultMachineIDPath(t *testing.T) {
	t.Parallel()

	path := DefaultMachineIDPath()
	assert.Contains(t, path, ".preflight")
	assert.Contains(t, path, "machine-id")
}

func TestFileMachineIdentityRepository_LoadCorruptedFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	// Write invalid content
	err := os.WriteFile(path, []byte("not-a-valid-uuid"), 0600)
	require.NoError(t, err)

	repo := NewFileMachineIdentityRepository(path)
	_, err = repo.Load()
	assert.Error(t, err)
}

func TestFileMachineIdentityRepository_Overwrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "machine-id")

	repo := NewFileMachineIdentityRepository(path)

	// Save first ID
	id1 := NewMachineID()
	err := repo.Save(id1)
	require.NoError(t, err)

	// Save second ID (overwrite)
	id2 := NewMachineID()
	err = repo.Save(id2)
	require.NoError(t, err)

	// Load should return second ID
	loaded, err := repo.Load()
	require.NoError(t, err)
	assert.True(t, id2.Equal(loaded))
	assert.False(t, id1.Equal(loaded))
}
