package drift

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateStore(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	assert.NotNil(t, store)
	assert.Equal(t, statePath, store.path)
}

func TestStateStore_Load_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	state, err := store.Load(ctx)
	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Empty(t, state.Files)
}

func TestStateStore_Save_And_Load(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	// Create state with files
	state := NewAppliedState()
	now := time.Now().Truncate(time.Second) // Truncate for JSON serialization
	state.SetFile("/path/to/file1", "hash1", "base", now)
	state.SetFile("/path/to/file2", "hash2", "work", now)

	// Save
	err := store.Save(ctx, state)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, statePath)

	// Load and verify
	loaded, err := store.Load(ctx)
	require.NoError(t, err)

	file1, exists := loaded.GetFile("/path/to/file1")
	assert.True(t, exists)
	assert.Equal(t, "hash1", file1.AppliedHash)
	assert.Equal(t, "base", file1.SourceLayer)

	file2, exists := loaded.GetFile("/path/to/file2")
	assert.True(t, exists)
	assert.Equal(t, "hash2", file2.AppliedHash)
	assert.Equal(t, "work", file2.SourceLayer)
}

func TestStateStore_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "state.json")

	store := NewStateStore(nestedPath)
	ctx := context.Background()

	state := NewAppliedState()
	state.SetFile("/path/to/file", "hash", "base", time.Now())

	err := store.Save(ctx, state)
	require.NoError(t, err)
	assert.FileExists(t, nestedPath)
}

func TestStateStore_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	err := os.WriteFile(statePath, []byte("not valid json"), 0o644)
	require.NoError(t, err)

	store := NewStateStore(statePath)
	ctx := context.Background()

	_, err = store.Load(ctx)
	assert.Error(t, err)
}

func TestStateStore_UpdateFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Update file (creates if doesn't exist)
	err := store.UpdateFile(ctx, "/path/to/file", "hash1", "base", now)
	require.NoError(t, err)

	// Verify
	state, err := store.Load(ctx)
	require.NoError(t, err)

	file, exists := state.GetFile("/path/to/file")
	assert.True(t, exists)
	assert.Equal(t, "hash1", file.AppliedHash)

	// Update again
	later := now.Add(time.Hour)
	err = store.UpdateFile(ctx, "/path/to/file", "hash2", "work", later)
	require.NoError(t, err)

	// Verify update
	state, err = store.Load(ctx)
	require.NoError(t, err)

	file, exists = state.GetFile("/path/to/file")
	assert.True(t, exists)
	assert.Equal(t, "hash2", file.AppliedHash)
	assert.Equal(t, "work", file.SourceLayer)
}

func TestStateStore_RemoveFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	// Add a file first
	now := time.Now()
	err := store.UpdateFile(ctx, "/path/to/file", "hash", "base", now)
	require.NoError(t, err)

	// Remove it
	err = store.RemoveFile(ctx, "/path/to/file")
	require.NoError(t, err)

	// Verify removed
	state, err := store.Load(ctx)
	require.NoError(t, err)

	_, exists := state.GetFile("/path/to/file")
	assert.False(t, exists)
}

func TestStateStore_GetFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	err := store.UpdateFile(ctx, "/path/to/file", "hash", "base", now)
	require.NoError(t, err)

	// Get existing file
	file, exists, err := store.GetFile(ctx, "/path/to/file")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "hash", file.AppliedHash)

	// Get non-existent file
	_, exists, err = store.GetFile(ctx, "/nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestStateStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	store := NewStateStore(statePath)
	ctx := context.Background()

	// Add files
	now := time.Now()
	err := store.UpdateFile(ctx, "/path/a", "hash1", "base", now)
	require.NoError(t, err)
	err = store.UpdateFile(ctx, "/path/b", "hash2", "work", now)
	require.NoError(t, err)

	// Clear
	err = store.Clear(ctx)
	require.NoError(t, err)

	// Verify empty
	state, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, state.Files)
}
