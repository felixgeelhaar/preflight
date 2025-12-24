package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSnapshotService(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	assert.NotNil(t, service)
	assert.Equal(t, tmpDir, service.baseDir)
	assert.NotNil(t, service.manager)
}

func TestDefaultSnapshotService(t *testing.T) {
	t.Parallel()

	service, err := DefaultSnapshotService()

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Should use ~/.preflight as base
	home, _ := os.UserHomeDir()
	expectedBase := filepath.Join(home, ".preflight")
	assert.Equal(t, expectedBase, service.baseDir)
}

func TestSnapshotService_BeforeApply(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	// Create a test file to snapshot
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	snapshotSet, err := service.BeforeApply(ctx, []string{testFile})

	require.NoError(t, err)
	assert.NotNil(t, snapshotSet)
	assert.NotEmpty(t, snapshotSet.ID)
	assert.Len(t, snapshotSet.Snapshots, 1)
}

func TestSnapshotService_BeforeApply_NonexistentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	ctx := context.Background()
	snapshotSet, err := service.BeforeApply(ctx, []string{"/nonexistent/path"})

	// Should not error for nonexistent files, just skip them
	require.NoError(t, err)
	assert.NotNil(t, snapshotSet)
	assert.Empty(t, snapshotSet.Snapshots)
}

func TestSnapshotService_BeforeApply_EmptyPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	ctx := context.Background()
	snapshotSet, err := service.BeforeApply(ctx, []string{})

	require.NoError(t, err)
	assert.NotNil(t, snapshotSet)
	assert.Empty(t, snapshotSet.Snapshots)
}

func TestSnapshotService_GetSnapshotSet(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	// Create a test file and snapshot it
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	createdSet, err := service.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// Retrieve the snapshot set
	retrievedSet, err := service.GetSnapshotSet(ctx, createdSet.ID)

	require.NoError(t, err)
	assert.Equal(t, createdSet.ID, retrievedSet.ID)
	assert.Len(t, retrievedSet.Snapshots, 1)
}

func TestSnapshotService_GetSnapshotSet_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	ctx := context.Background()
	_, err := service.GetSnapshotSet(ctx, "nonexistent-id")

	assert.Error(t, err)
}

func TestSnapshotService_ListSnapshotSets(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Create multiple snapshot sets
	_, err = service.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	_, err = service.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// List all sets
	sets, err := service.ListSnapshotSets(ctx)

	require.NoError(t, err)
	assert.Len(t, sets, 2)
}

func TestSnapshotService_ListSnapshotSets_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	ctx := context.Background()
	sets, err := service.ListSnapshotSets(ctx)

	require.NoError(t, err)
	assert.Empty(t, sets)
}

func TestSnapshotService_Restore(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	originalContent := []byte("original content")
	err := os.WriteFile(testFile, originalContent, 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Take snapshot
	snapshotSet, err := service.BeforeApply(ctx, []string{testFile})
	require.NoError(t, err)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Restore from snapshot
	err = service.Restore(ctx, snapshotSet.ID)
	require.NoError(t, err)

	// Verify content is restored
	restoredContent, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, restoredContent)
}

func TestSnapshotService_Restore_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewSnapshotService(tmpDir)

	ctx := context.Background()
	err := service.Restore(ctx, "nonexistent-id")

	assert.Error(t, err)
}
