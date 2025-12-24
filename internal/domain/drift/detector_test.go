package drift

import (
	"context"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetector(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	detector := NewDetector(fs, state)
	assert.NotNil(t, detector)
}

func TestDetector_Detect_NoDrift(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Add file to filesystem with known content
	content := "file content"
	fs.AddFile("/path/to/file", content)

	// Calculate expected hash (SHA256 of content)
	hash, err := fs.FileHash("/path/to/file")
	require.NoError(t, err)

	// Record in state with matching hash
	now := time.Now()
	state.SetFile("/path/to/file", hash, "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/path/to/file")
	require.NoError(t, err)

	assert.False(t, drift.HasDrift())
	assert.Equal(t, TypeNone, drift.Type)
}

func TestDetector_Detect_HasDrift(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Add file to filesystem
	fs.AddFile("/path/to/file", "modified content")

	// Record in state with different hash
	now := time.Now()
	state.SetFile("/path/to/file", "original-hash", "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/path/to/file")
	require.NoError(t, err)

	assert.True(t, drift.HasDrift())
	assert.Equal(t, "original-hash", drift.ExpectedHash)
	assert.NotEqual(t, "original-hash", drift.CurrentHash)
}

func TestDetector_Detect_FileNotTracked(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Add file to filesystem but not to state
	fs.AddFile("/path/to/file", "content")

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/path/to/file")
	require.NoError(t, err)

	// File not tracked means no drift to report
	assert.False(t, drift.HasDrift())
	assert.Equal(t, TypeNone, drift.Type)
}

func TestDetector_Detect_FileDeleted(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Track file that doesn't exist in filesystem
	now := time.Now()
	state.SetFile("/path/to/deleted", "some-hash", "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/path/to/deleted")
	require.NoError(t, err)

	assert.True(t, drift.HasDrift())
	assert.Equal(t, TypeManual, drift.Type)
	assert.Empty(t, drift.CurrentHash)
}

func TestDetector_DetectAll(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Setup: one unchanged, one modified, one deleted
	fs.AddFile("/unchanged", "original")
	fs.AddFile("/modified", "changed content")

	unchangedHash, _ := fs.FileHash("/unchanged")

	now := time.Now()
	state.SetFile("/unchanged", unchangedHash, "base", now)
	state.SetFile("/modified", "original-hash", "base", now)
	state.SetFile("/deleted", "some-hash", "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drifts, err := detector.DetectAll(ctx)
	require.NoError(t, err)

	// Should detect 2 drifts: modified and deleted
	assert.Len(t, drifts, 2)

	// Find specific drifts
	var modifiedDrift, deletedDrift *Drift
	for i := range drifts {
		switch drifts[i].Path {
		case "/modified":
			modifiedDrift = &drifts[i]
		case "/deleted":
			deletedDrift = &drifts[i]
		}
	}

	assert.NotNil(t, modifiedDrift)
	assert.True(t, modifiedDrift.HasDrift())

	assert.NotNil(t, deletedDrift)
	assert.True(t, deletedDrift.HasDrift())
}

func TestDetector_DetectAll_Empty(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drifts, err := detector.DetectAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, drifts)
}

func TestDetector_DetectPaths(t *testing.T) {
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	// Setup files
	fs.AddFile("/a", "content a")
	fs.AddFile("/b", "content b")
	fs.AddFile("/c", "content c")

	hashA, _ := fs.FileHash("/a")
	hashC, _ := fs.FileHash("/c")

	now := time.Now()
	state.SetFile("/a", hashA, "base", now)      // unchanged
	state.SetFile("/b", "old-hash", "base", now) // modified
	state.SetFile("/c", hashC, "base", now)      // unchanged

	detector := NewDetector(fs, state)
	ctx := context.Background()

	// Only check /a and /b
	drifts, err := detector.DetectPaths(ctx, []string{"/a", "/b"})
	require.NoError(t, err)

	// Should only find drift in /b
	assert.Len(t, drifts, 1)
	assert.Equal(t, "/b", drifts[0].Path)
}

func TestDetector_ClassifyDrift_Manual(t *testing.T) {
	// When a file is deleted, classify as manual
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	now := time.Now()
	state.SetFile("/deleted", "hash", "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/deleted")
	require.NoError(t, err)

	assert.Equal(t, TypeManual, drift.Type)
}

func TestDetector_ClassifyDrift_Unknown(t *testing.T) {
	// When we can't determine source, classify as unknown
	fs := mocks.NewFileSystem()
	state := NewAppliedState()

	fs.AddFile("/modified", "new content")

	now := time.Now()
	state.SetFile("/modified", "original-hash", "base", now)

	detector := NewDetector(fs, state)
	ctx := context.Background()

	drift, err := detector.Detect(ctx, "/modified")
	require.NoError(t, err)

	// Default classification for modified files is unknown
	assert.Equal(t, TypeUnknown, drift.Type)
}
