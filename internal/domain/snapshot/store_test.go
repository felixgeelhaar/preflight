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

func TestFileStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("saves content and returns snapshot", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		content := []byte("original file content")
		snap, err := store.Save(ctx, "/home/user/.zshrc", content)

		require.NoError(t, err)
		assert.NotEmpty(t, snap.ID)
		assert.Equal(t, "/home/user/.zshrc", snap.Path)
		assert.NotEmpty(t, snap.Hash)
		assert.Equal(t, int64(len(content)), snap.Size)
	})

	t.Run("creates nested directories for path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		_, err := store.Save(ctx, "/home/user/deep/nested/file.txt", []byte("content"))
		require.NoError(t, err)

		// Verify the snapshot file exists somewhere in the store
		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})

	t.Run("stores metadata in index", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		snap, err := store.Save(ctx, "/path/to/file", []byte("content"))
		require.NoError(t, err)

		// Should be able to retrieve it
		content, err := store.Get(ctx, snap.ID)
		require.NoError(t, err)
		assert.Equal(t, []byte("content"), content)
	})
}

func TestFileStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns saved content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		originalContent := []byte("my important config")
		snap, err := store.Save(ctx, "/etc/config", originalContent)
		require.NoError(t, err)

		retrieved, err := store.Get(ctx, snap.ID)
		require.NoError(t, err)
		assert.Equal(t, originalContent, retrieved)
	})

	t.Run("returns error for unknown ID", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		_, err := store.Get(ctx, "nonexistent-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrSnapshotNotFound)
	})
}

func TestFileStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns all snapshots for a path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		// Save multiple snapshots for the same path
		_, err := store.Save(ctx, "/home/user/.zshrc", []byte("version 1"))
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		_, err = store.Save(ctx, "/home/user/.zshrc", []byte("version 2"))
		require.NoError(t, err)

		snapshots, err := store.List(ctx, "/home/user/.zshrc")
		require.NoError(t, err)
		assert.Len(t, snapshots, 2)
	})

	t.Run("returns empty for path with no snapshots", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		snapshots, err := store.List(ctx, "/nonexistent/path")
		require.NoError(t, err)
		assert.Empty(t, snapshots)
	})

	t.Run("does not return snapshots for different paths", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		_, err := store.Save(ctx, "/path/a", []byte("content a"))
		require.NoError(t, err)
		_, err = store.Save(ctx, "/path/b", []byte("content b"))
		require.NoError(t, err)

		snapshots, err := store.List(ctx, "/path/a")
		require.NoError(t, err)
		assert.Len(t, snapshots, 1)
		assert.Equal(t, "/path/a", snapshots[0].Path)
	})
}

func TestFileStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes snapshot by ID", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		snap, err := store.Save(ctx, "/path/file", []byte("content"))
		require.NoError(t, err)

		err = store.Delete(ctx, snap.ID)
		require.NoError(t, err)

		// Should no longer be retrievable
		_, err = store.Get(ctx, snap.ID)
		assert.ErrorIs(t, err, ErrSnapshotNotFound)
	})

	t.Run("returns error for unknown ID", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		err := store.Delete(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, ErrSnapshotNotFound)
	})
}

func TestFileStore_Cleanup(t *testing.T) {
	t.Parallel()

	t.Run("removes expired snapshots", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		// Save a snapshot and manually make it appear old
		snap, err := store.Save(ctx, "/path/file", []byte("old content"))
		require.NoError(t, err)

		// Modify the index to make snapshot appear old
		// This tests the cleanup logic
		count, err := store.Cleanup(ctx, 1*time.Nanosecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)

		// Should be deleted
		_, err = store.Get(ctx, snap.ID)
		assert.ErrorIs(t, err, ErrSnapshotNotFound)
	})

	t.Run("keeps recent snapshots", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		store := NewFileStore(tmpDir)
		ctx := context.Background()

		snap, err := store.Save(ctx, "/path/file", []byte("recent content"))
		require.NoError(t, err)

		count, err := store.Cleanup(ctx, 24*time.Hour)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Should still exist
		content, err := store.Get(ctx, snap.ID)
		require.NoError(t, err)
		assert.Equal(t, []byte("recent content"), content)
	})
}

func TestFileStore_BasePath(t *testing.T) {
	t.Parallel()

	t.Run("stores files under base path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		snapshotsDir := filepath.Join(tmpDir, "snapshots")
		store := NewFileStore(snapshotsDir)
		ctx := context.Background()

		_, err := store.Save(ctx, "/path/file", []byte("content"))
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(snapshotsDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}
