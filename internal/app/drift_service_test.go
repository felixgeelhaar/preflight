package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDriftService(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	assert.NotNil(t, service)
	assert.Equal(t, tmpDir, service.baseDir)
	assert.NotNil(t, service.store)
}

func TestDefaultDriftService(t *testing.T) {
	t.Parallel()

	service, err := DefaultDriftService()

	require.NoError(t, err)
	assert.NotNil(t, service)

	// Should use ~/.preflight as base
	home, _ := os.UserHomeDir()
	expectedBase := filepath.Join(home, ".preflight")
	assert.Equal(t, expectedBase, service.baseDir)
}

func TestDriftService_RecordApplied(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = service.RecordApplied(ctx, testFile, "base-layer")

	require.NoError(t, err)
}

func TestDriftService_CheckDrift(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Record the file
	err = service.RecordApplied(ctx, testFile, "base-layer")
	require.NoError(t, err)

	// Check for drift - should have none
	drift, err := service.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.False(t, drift.HasDrift())
}

func TestDriftService_CheckDrift_WithDrift(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Record the file
	err = service.RecordApplied(ctx, testFile, "base-layer")
	require.NoError(t, err)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Check for drift - should detect modification
	drift, err := service.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.True(t, drift.HasDrift())
}

func TestDriftService_CheckDrift_FileNotTracked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create a test file but don't record it
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	drift, err := service.CheckDrift(ctx, testFile)

	require.NoError(t, err)
	assert.False(t, drift.HasDrift()) // Not tracked = no drift
}

func TestDriftService_CheckAll(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Record both files
	err = service.RecordApplied(ctx, file1, "base")
	require.NoError(t, err)
	err = service.RecordApplied(ctx, file2, "base")
	require.NoError(t, err)

	// Modify file2
	err = os.WriteFile(file2, []byte("modified"), 0644)
	require.NoError(t, err)

	// Check all
	drifts, err := service.CheckAll(ctx)
	require.NoError(t, err)
	assert.Len(t, drifts, 1)
	assert.Equal(t, file2, drifts[0].Path)
}

func TestDriftService_CheckAll_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	ctx := context.Background()
	drifts, err := service.CheckAll(ctx)

	require.NoError(t, err)
	assert.Empty(t, drifts)
}

func TestDriftService_CheckPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file3, []byte("content3"), 0644)
	require.NoError(t, err)

	ctx := context.Background()

	// Record all files
	err = service.RecordApplied(ctx, file1, "base")
	require.NoError(t, err)
	err = service.RecordApplied(ctx, file2, "base")
	require.NoError(t, err)
	err = service.RecordApplied(ctx, file3, "base")
	require.NoError(t, err)

	// Modify files 1 and 2
	err = os.WriteFile(file1, []byte("modified1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("modified2"), 0644)
	require.NoError(t, err)

	// Check only file1 and file3
	drifts, err := service.CheckPaths(ctx, []string{file1, file3})
	require.NoError(t, err)

	// Should only find drift in file1 (file3 unchanged, file2 not checked)
	assert.Len(t, drifts, 1)
	assert.Equal(t, file1, drifts[0].Path)
}

func TestDriftService_RemoveTracking(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create and record a file
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = service.RecordApplied(ctx, testFile, "base")
	require.NoError(t, err)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified"), 0644)
	require.NoError(t, err)

	// File should show drift
	drift, err := service.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.True(t, drift.HasDrift())

	// Remove tracking
	err = service.RemoveTracking(ctx, testFile)
	require.NoError(t, err)

	// File should no longer show drift (not tracked)
	drift, err = service.CheckDrift(ctx, testFile)
	require.NoError(t, err)
	assert.False(t, drift.HasDrift())
}

func TestDriftService_ListTrackedFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	// Create and record files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	err = service.RecordApplied(ctx, file1, "base")
	require.NoError(t, err)
	err = service.RecordApplied(ctx, file2, "work")
	require.NoError(t, err)

	// List tracked files
	files, err := service.ListTrackedFiles(ctx)
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestDriftService_ListTrackedFiles_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	service := NewDriftService(tmpDir)

	ctx := context.Background()
	files, err := service.ListTrackedFiles(ctx)

	require.NoError(t, err)
	assert.Empty(t, files)
}
