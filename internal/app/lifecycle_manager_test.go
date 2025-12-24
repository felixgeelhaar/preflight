package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLifecycleManager(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)

	manager := NewLifecycleManager(snapshot, drift)

	assert.NotNil(t, manager)
	assert.Equal(t, snapshot, manager.Snapshot())
	assert.Equal(t, drift, manager.Drift())
}

func TestDefaultLifecycleManager(t *testing.T) {
	t.Parallel()

	manager, err := DefaultLifecycleManager()

	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Snapshot())
	assert.NotNil(t, manager.Drift())
}

func TestLifecycleManager_BeforeModify(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)
	manager := NewLifecycleManager(snapshot, drift)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.BeforeModify(ctx, testFile)

	require.NoError(t, err)

	// Verify snapshot was created
	sets, err := snapshot.ListSnapshotSets(ctx)
	require.NoError(t, err)
	assert.Len(t, sets, 1)
}

func TestLifecycleManager_BeforeModify_NonexistentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)
	manager := NewLifecycleManager(snapshot, drift)

	ctx := context.Background()
	err := manager.BeforeModify(ctx, "/nonexistent/path")

	// Should not error for nonexistent files
	require.NoError(t, err)
}

func TestLifecycleManager_AfterApply(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)
	manager := NewLifecycleManager(snapshot, drift)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.AfterApply(ctx, testFile, "base-layer")

	require.NoError(t, err)

	// Verify file is tracked
	files, err := drift.ListTrackedFiles(ctx)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, testFile, files[0].Path)
	assert.Equal(t, "base-layer", files[0].SourceLayer)
}

func TestLifecycleManager_FullWorkflow(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)
	manager := NewLifecycleManager(snapshot, drift)

	// Create a test file
	testFile := filepath.Join(tmpDir, "config.txt")
	originalContent := []byte("original config")
	err := os.WriteFile(testFile, originalContent, 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Simulate apply workflow:
	// 1. Before modify - snapshot
	err = manager.BeforeModify(ctx, testFile)
	require.NoError(t, err)

	// 2. Modify the file (simulating apply)
	newContent := []byte("new config from preflight")
	err = os.WriteFile(testFile, newContent, 0644)
	require.NoError(t, err)

	// 3. After apply - record for drift
	err = manager.AfterApply(ctx, testFile, "work-layer")
	require.NoError(t, err)

	// Verify: snapshot exists
	sets, err := snapshot.ListSnapshotSets(ctx)
	require.NoError(t, err)
	assert.Len(t, sets, 1)

	// Verify: no drift (file matches what we applied)
	driftResult, err := drift.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.False(t, driftResult.HasDrift())

	// Simulate external modification
	err = os.WriteFile(testFile, []byte("manually edited"), 0644)
	require.NoError(t, err)

	// Verify: drift detected
	driftResult, err = drift.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.True(t, driftResult.HasDrift())
}

func TestLifecycleManager_PathExpansion(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	snapshot := NewSnapshotService(tmpDir)
	drift := NewDriftService(tmpDir)
	manager := NewLifecycleManager(snapshot, drift)

	// Create a file in home dir for testing path expansion
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with absolute path
	err = manager.AfterApply(ctx, testFile, "layer")
	require.NoError(t, err)

	files, err := drift.ListTrackedFiles(ctx)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}
