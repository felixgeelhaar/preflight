package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: TestSyncCmd_Exists and TestSyncCmd_HasFlags are in helpers_test.go

func TestSyncCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"remote default", "remote", "origin"},
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := syncCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestFindRepoRoot(t *testing.T) {
	// Skip if not in a git repo
	if _, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err != nil {
		t.Skip("not in a git repository")
	}

	root, err := findRepoRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.DirExists(t, root)
	assert.DirExists(t, filepath.Join(root, ".git"))
}

func TestGetCurrentBranch(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Test getCurrentBranch
	branch, err := getCurrentBranch(tmpDir)
	require.NoError(t, err)
	// Could be "main" or "master" depending on git config
	assert.True(t, branch == "main" || branch == "master", "expected main or master, got %s", branch)
}

func TestHasUncommittedChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Run("no changes", func(t *testing.T) {
		hasChanges, err := hasUncommittedChanges(tmpDir)
		require.NoError(t, err)
		assert.False(t, hasChanges)
	})

	t.Run("with uncommitted changes", func(t *testing.T) {
		// Modify the file
		require.NoError(t, os.WriteFile(testFile, []byte("modified"), 0o644))

		hasChanges, err := hasUncommittedChanges(tmpDir)
		require.NoError(t, err)
		assert.True(t, hasChanges)
	})
}

func TestRunSync_NotInGitRepo(t *testing.T) {
	// Save and restore original working directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Change to temp dir without git
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	err = runSync(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}
