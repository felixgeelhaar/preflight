package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_BeforeApply(t *testing.T) {
	t.Parallel()

	t.Run("creates snapshot set for existing files", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0o644))

		ctx := context.Background()
		set, err := manager.BeforeApply(ctx, []string{file1, file2})

		require.NoError(t, err)
		assert.NotEmpty(t, set.ID)
		assert.Equal(t, string(ReasonApply), set.Reason)
		assert.Len(t, set.Snapshots, 2)
	})

	t.Run("skips non-existent files", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create only one file
		existingFile := filepath.Join(tmpDir, "exists.txt")
		require.NoError(t, os.WriteFile(existingFile, []byte("content"), 0o644))
		nonExistentFile := filepath.Join(tmpDir, "does-not-exist.txt")

		ctx := context.Background()
		set, err := manager.BeforeApply(ctx, []string{existingFile, nonExistentFile})

		require.NoError(t, err)
		assert.Len(t, set.Snapshots, 1)
		assert.Equal(t, existingFile, set.Snapshots[0].Path)
	})

	t.Run("handles empty paths slice", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		manager := NewManager(store)

		ctx := context.Background()
		set, err := manager.BeforeApply(ctx, []string{})

		require.NoError(t, err)
		assert.Empty(t, set.Snapshots)
	})

	t.Run("preserves file content correctly", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		originalContent := []byte("original content to preserve")
		require.NoError(t, os.WriteFile(testFile, originalContent, 0o644))

		ctx := context.Background()
		set, err := manager.BeforeApply(ctx, []string{testFile})
		require.NoError(t, err)

		// Verify content can be retrieved
		snap := set.Snapshots[0]
		content, err := store.Get(ctx, snap.ID)
		require.NoError(t, err)
		assert.Equal(t, originalContent, content)
	})
}

func TestManager_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores files from snapshot set", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test file with original content
		testFile := filepath.Join(tmpDir, "test.txt")
		originalContent := []byte("original content")
		require.NoError(t, os.WriteFile(testFile, originalContent, 0o644))

		ctx := context.Background()

		// Create snapshot
		set, err := manager.BeforeApply(ctx, []string{testFile})
		require.NoError(t, err)

		// Modify the file
		require.NoError(t, os.WriteFile(testFile, []byte("modified content"), 0o644))

		// Restore from snapshot
		err = manager.Restore(ctx, set.ID)
		require.NoError(t, err)

		// Verify content was restored
		restoredContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, originalContent, restoredContent)
	})

	t.Run("returns error for unknown snapshot set", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		manager := NewManager(store)

		ctx := context.Background()
		err := manager.Restore(ctx, "nonexistent-set-id")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSetNotFound)
	})

	t.Run("restores multiple files", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		content1 := []byte("content1")
		content2 := []byte("content2")
		require.NoError(t, os.WriteFile(file1, content1, 0o644))
		require.NoError(t, os.WriteFile(file2, content2, 0o644))

		ctx := context.Background()

		// Create snapshot
		set, err := manager.BeforeApply(ctx, []string{file1, file2})
		require.NoError(t, err)

		// Modify both files
		require.NoError(t, os.WriteFile(file1, []byte("modified1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("modified2"), 0o644))

		// Restore
		err = manager.Restore(ctx, set.ID)
		require.NoError(t, err)

		// Verify
		restored1, _ := os.ReadFile(file1)
		restored2, _ := os.ReadFile(file2)
		assert.Equal(t, content1, restored1)
		assert.Equal(t, content2, restored2)
	})
}

func TestManager_GetSet(t *testing.T) {
	t.Parallel()

	t.Run("returns snapshot set by ID", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

		ctx := context.Background()
		created, err := manager.BeforeApply(ctx, []string{testFile})
		require.NoError(t, err)

		// Retrieve
		retrieved, err := manager.GetSet(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Reason, retrieved.Reason)
		assert.Len(t, retrieved.Snapshots, 1)
	})

	t.Run("returns error for unknown ID", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		manager := NewManager(store)

		ctx := context.Background()
		_, err := manager.GetSet(ctx, "unknown-id")

		assert.ErrorIs(t, err, ErrSetNotFound)
	})
}

func TestManager_ListSets(t *testing.T) {
	t.Parallel()

	t.Run("returns all snapshot sets", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotDir)
		manager := NewManager(store)

		// Create test files
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0o644))

		ctx := context.Background()

		// Create multiple snapshot sets
		_, err := manager.BeforeApply(ctx, []string{file1})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
		_, err = manager.BeforeApply(ctx, []string{file2})
		require.NoError(t, err)

		sets, err := manager.ListSets(ctx)
		require.NoError(t, err)
		assert.Len(t, sets, 2)
	})

	t.Run("returns empty for no snapshots", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		manager := NewManager(store)

		ctx := context.Background()
		sets, err := manager.ListSets(ctx)

		require.NoError(t, err)
		assert.Empty(t, sets)
	})
}
