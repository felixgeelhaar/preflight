package snapshot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("creates snapshot with all fields", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		snap := NewSnapshot("/home/user/.zshrc", "abc123hash", 1024, now)

		assert.NotEmpty(t, snap.ID)
		assert.Equal(t, "/home/user/.zshrc", snap.Path)
		assert.Equal(t, "abc123hash", snap.Hash)
		assert.Equal(t, int64(1024), snap.Size)
		assert.Equal(t, now, snap.CreatedAt)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		snap1 := NewSnapshot("/path/a", "hash1", 100, now)
		snap2 := NewSnapshot("/path/b", "hash2", 200, now)

		assert.NotEqual(t, snap1.ID, snap2.ID)
	})
}

func TestSnapshot_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("returns false for recent snapshot", func(t *testing.T) {
		t.Parallel()

		snap := NewSnapshot("/path", "hash", 100, time.Now())
		assert.False(t, snap.IsExpired(24*time.Hour))
	})

	t.Run("returns true for old snapshot", func(t *testing.T) {
		t.Parallel()

		oldTime := time.Now().Add(-48 * time.Hour)
		snap := NewSnapshot("/path", "hash", 100, oldTime)
		assert.True(t, snap.IsExpired(24*time.Hour))
	})
}

func TestNewSet(t *testing.T) {
	t.Parallel()

	t.Run("creates set with reason and snapshots", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		snapshots := []Snapshot{
			NewSnapshot("/path/a", "hash1", 100, now),
			NewSnapshot("/path/b", "hash2", 200, now),
		}

		set := NewSet("apply", snapshots, now)

		assert.NotEmpty(t, set.ID)
		assert.Equal(t, "apply", set.Reason)
		assert.Len(t, set.Snapshots, 2)
		assert.Equal(t, now, set.CreatedAt)
	})

	t.Run("handles empty snapshots", func(t *testing.T) {
		t.Parallel()

		set := NewSet("fix", nil, time.Now())

		assert.NotEmpty(t, set.ID)
		assert.Equal(t, "fix", set.Reason)
		assert.Empty(t, set.Snapshots)
	})
}

func TestSet_GetSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("returns snapshot by path", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		snap1 := NewSnapshot("/path/a", "hash1", 100, now)
		snap2 := NewSnapshot("/path/b", "hash2", 200, now)
		set := NewSet("apply", []Snapshot{snap1, snap2}, now)

		found, ok := set.GetSnapshot("/path/b")

		require.True(t, ok)
		assert.Equal(t, "hash2", found.Hash)
	})

	t.Run("returns false for unknown path", func(t *testing.T) {
		t.Parallel()

		set := NewSet("apply", []Snapshot{
			NewSnapshot("/path/a", "hash1", 100, time.Now()),
		}, time.Now())

		_, ok := set.GetSnapshot("/path/unknown")
		assert.False(t, ok)
	})
}

func TestSet_Paths(t *testing.T) {
	t.Parallel()

	t.Run("returns all paths in set", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		set := NewSet("apply", []Snapshot{
			NewSnapshot("/path/a", "hash1", 100, now),
			NewSnapshot("/path/b", "hash2", 200, now),
			NewSnapshot("/path/c", "hash3", 300, now),
		}, now)

		paths := set.Paths()

		assert.Len(t, paths, 3)
		assert.Contains(t, paths, "/path/a")
		assert.Contains(t, paths, "/path/b")
		assert.Contains(t, paths, "/path/c")
	})

	t.Run("returns empty for empty set", func(t *testing.T) {
		t.Parallel()

		set := NewSet("apply", nil, time.Now())
		assert.Empty(t, set.Paths())
	})
}

func TestReason_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason Reason
		valid  bool
	}{
		{ReasonApply, true},
		{ReasonFix, true},
		{ReasonRollback, true},
		{Reason("invalid"), false},
		{Reason(""), false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.reason), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.valid, tc.reason.IsValid())
		})
	}
}
