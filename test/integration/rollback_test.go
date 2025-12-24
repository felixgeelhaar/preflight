package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
)

func TestSnapshot_Manager_BeforeApply_CreatesSnapshots(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("original content"), 0o644))

	// Create snapshot set
	set, err := manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)
	require.NotNil(t, set)

	assert.NotEmpty(t, set.ID)
	assert.Equal(t, "apply", set.Reason)
	assert.Len(t, set.Snapshots, 1)
	assert.Equal(t, testFile, set.Snapshots[0].Path)
}

func TestSnapshot_Manager_Restore_RecoversOriginalContent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "original content"
	require.NoError(t, os.WriteFile(testFile, []byte(originalContent), 0o644))

	// Create snapshot before modification
	set, err := manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// Modify the file
	modifiedContent := "modified content"
	require.NoError(t, os.WriteFile(testFile, []byte(modifiedContent), 0o644))

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, string(content))

	// Restore from snapshot set
	err = manager.Restore(ctx, set.ID)
	require.NoError(t, err)

	// Verify file was restored
	content, err = os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(content))
}

func TestSnapshot_Manager_ListSets_ReturnsAllSets(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

	// Create multiple snapshot sets
	_, err := manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	// Modify file for second snapshot
	require.NoError(t, os.WriteFile(testFile, []byte("modified"), 0o644))
	_, err = manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// List snapshot sets
	sets, err := manager.ListSets(ctx)
	require.NoError(t, err)
	assert.Len(t, sets, 2)
}

func TestSnapshot_Manager_GetSet_ReturnsSpecificSet(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

	// Create snapshot set
	created, err := manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// Retrieve by ID
	retrieved, err := manager.GetSet(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Reason, retrieved.Reason)
}

func TestSnapshot_Manager_GetSet_ReturnsErrorForNonexistent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	_, err := manager.GetSet(ctx, "nonexistent-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, snapshot.ErrSetNotFound)
}

func TestSnapshot_Manager_Restore_HandlesMultipleFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	require.NoError(t, os.WriteFile(file1, []byte("content 1"), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte("content 2"), 0o644))

	// Create snapshot set
	set, err := manager.BeforeApply(ctx, []string{file1, file2})
	require.NoError(t, err)
	assert.Len(t, set.Snapshots, 2)

	// Modify files
	require.NoError(t, os.WriteFile(file1, []byte("modified 1"), 0o644))
	require.NoError(t, os.WriteFile(file2, []byte("modified 2"), 0o644))

	// Restore
	err = manager.Restore(ctx, set.ID)
	require.NoError(t, err)

	// Verify both restored
	content1, _ := os.ReadFile(file1)
	content2, _ := os.ReadFile(file2)
	assert.Equal(t, "content 1", string(content1))
	assert.Equal(t, "content 2", string(content2))
}

func TestSnapshot_Manager_Restore_CreatesDirectoryIfNeeded(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create test file in subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	testFile := filepath.Join(subDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

	// Create snapshot set
	set, err := manager.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// Remove the entire subdirectory
	require.NoError(t, os.RemoveAll(subDir))

	// Restore should recreate directory
	err = manager.Restore(ctx, set.ID)
	require.NoError(t, err)

	// Verify file exists
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "content", string(content))
}

func TestSnapshot_FileStore_Cleanup_RemovesOldSnapshots(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	ctx := context.Background()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))
	content, _ := os.ReadFile(testFile)

	// Create several snapshots
	for i := 0; i < 5; i++ {
		_, err := store.Save(ctx, testFile, content)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// List snapshots before cleanup
	beforeCleanup, err := store.List(ctx, testFile)
	require.NoError(t, err)
	assert.Len(t, beforeCleanup, 5)

	// Cleanup with very short maxAge (all should be deleted since they're older than 1ns)
	deleted, err := store.Cleanup(ctx, 1*time.Nanosecond)
	require.NoError(t, err)
	assert.Equal(t, 5, deleted)

	// Verify all removed
	afterCleanup, err := store.List(ctx, testFile)
	require.NoError(t, err)
	assert.Len(t, afterCleanup, 0)
}

func TestSnapshot_Manager_BeforeApply_SkipsNonexistentFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	store := snapshot.NewFileStore(snapshotDir)
	manager := snapshot.NewManager(store)
	ctx := context.Background()

	// Create one test file
	existingFile := filepath.Join(tempDir, "existing.txt")
	require.NoError(t, os.WriteFile(existingFile, []byte("content"), 0o644))

	// Reference a non-existent file
	nonexistentFile := filepath.Join(tempDir, "nonexistent.txt")

	// Create snapshot set with both files
	set, err := manager.BeforeApply(ctx, []string{existingFile, nonexistentFile})
	require.NoError(t, err)
	require.NotNil(t, set)

	// Should only have snapshot for existing file
	assert.Len(t, set.Snapshots, 1)
	assert.Equal(t, existingFile, set.Snapshots[0].Path)
}

func TestSnapshot_Set_GetSnapshot(t *testing.T) {
	t.Parallel()

	set := snapshot.NewSet("test", []snapshot.Snapshot{
		{ID: "1", Path: "/path/to/file1.txt"},
		{ID: "2", Path: "/path/to/file2.txt"},
	}, time.Now())

	// Get existing snapshot
	snap, found := set.GetSnapshot("/path/to/file1.txt")
	assert.True(t, found)
	assert.Equal(t, "1", snap.ID)

	// Get non-existent snapshot
	_, found = set.GetSnapshot("/nonexistent")
	assert.False(t, found)
}

func TestSnapshot_Set_Paths(t *testing.T) {
	t.Parallel()

	set := snapshot.NewSet("test", []snapshot.Snapshot{
		{ID: "1", Path: "/path/to/file1.txt"},
		{ID: "2", Path: "/path/to/file2.txt"},
	}, time.Now())

	paths := set.Paths()
	assert.Len(t, paths, 2)
	assert.Contains(t, paths, "/path/to/file1.txt")
	assert.Contains(t, paths, "/path/to/file2.txt")
}

func TestSnapshot_Reason_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason snapshot.Reason
		valid  bool
	}{
		{snapshot.ReasonApply, true},
		{snapshot.ReasonFix, true},
		{snapshot.ReasonRollback, true},
		{snapshot.Reason("unknown"), false},
		{snapshot.Reason(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.reason.IsValid())
		})
	}
}

func TestSnapshot_IsExpired(t *testing.T) {
	t.Parallel()

	// Create snapshot from 1 hour ago
	oldSnapshot := snapshot.Snapshot{
		ID:        "old",
		Path:      "/test",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	// Check expiration with different durations
	assert.True(t, oldSnapshot.IsExpired(30*time.Minute))
	assert.False(t, oldSnapshot.IsExpired(2*time.Hour))
}
