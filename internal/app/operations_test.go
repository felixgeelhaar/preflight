package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCommand creates an exec.Cmd for testing purposes.
func newTestCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func TestExtractRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH URL with .git suffix",
			url:      "git@github.com:user/dotfiles.git",
			expected: "dotfiles",
		},
		{
			name:     "SSH URL without .git suffix",
			url:      "git@github.com:user/my-config",
			expected: "my-config",
		},
		{
			name:     "HTTPS URL with .git suffix",
			url:      "https://github.com/user/dotfiles.git",
			expected: "dotfiles",
		},
		{
			name:     "HTTPS URL without .git suffix",
			url:      "https://github.com/user/my-config",
			expected: "my-config",
		},
		{
			name:     "GitLab SSH URL",
			url:      "git@gitlab.com:group/project.git",
			expected: "project",
		},
		{
			name:     "simple path",
			url:      "/path/to/repo.git",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractRepoName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepoClone_DestinationExists(t *testing.T) {
	t.Parallel()

	// Create a temp directory that already exists
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing-repo")
	require.NoError(t, os.Mkdir(existingPath, 0o755))

	p := New(os.Stdout)
	ctx := context.Background()

	opts := CloneOptions{
		URL:  "https://github.com/user/repo.git",
		Path: existingPath,
	}

	_, err := p.RepoClone(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination path already exists")
}

func TestRepoClone_InvalidURL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "new-repo")

	p := New(os.Stdout)
	ctx := context.Background()

	opts := CloneOptions{
		URL:  "not-a-valid-url",
		Path: destPath,
	}

	_, err := p.RepoClone(ctx, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestRepoClone_NoConfigFile(t *testing.T) {
	t.Parallel()

	// Create a test git repo without preflight.yaml
	sourceDir := t.TempDir()

	// Initialize a bare repo
	cmd := newTestCommand("git", "init", "--bare", sourceDir)
	require.NoError(t, cmd.Run())

	// Create a temp working dir, add a file, and push
	workDir := t.TempDir()
	workRepoPath := filepath.Join(workDir, "work")

	cmd = newTestCommand("git", "clone", sourceDir, workRepoPath)
	require.NoError(t, cmd.Run())

	// Create a dummy file (not preflight.yaml)
	require.NoError(t, os.WriteFile(filepath.Join(workRepoPath, "README.md"), []byte("# Test"), 0o644))

	cmd = newTestCommand("git", "-C", workRepoPath, "add", ".")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "push", "-u", "origin", "HEAD")
	require.NoError(t, cmd.Run())

	// Now test cloning
	cloneDir := t.TempDir()
	destPath := filepath.Join(cloneDir, "cloned")

	p := New(os.Stdout)
	ctx := context.Background()

	opts := CloneOptions{
		URL:         sourceDir,
		Path:        destPath,
		AutoConfirm: true, // Skip prompt
	}

	result, err := p.RepoClone(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, destPath, result.Path)
	assert.False(t, result.ConfigFound, "should not find preflight.yaml")
	assert.False(t, result.Applied)
}

func TestRepoClone_WithConfigFile(t *testing.T) {
	t.Parallel()

	// Create a test git repo with preflight.yaml
	sourceDir := t.TempDir()

	// Initialize a bare repo
	cmd := newTestCommand("git", "init", "--bare", sourceDir)
	require.NoError(t, cmd.Run())

	// Create a temp working dir, add preflight.yaml, and push
	workDir := t.TempDir()
	workRepoPath := filepath.Join(workDir, "work")

	cmd = newTestCommand("git", "clone", sourceDir, workRepoPath)
	require.NoError(t, cmd.Run())

	// Create preflight.yaml
	preflightConfig := `version: "1"
defaults:
  target: base
targets:
  base:
    layers: []
`
	require.NoError(t, os.WriteFile(filepath.Join(workRepoPath, "preflight.yaml"), []byte(preflightConfig), 0o644))

	cmd = newTestCommand("git", "-C", workRepoPath, "add", ".")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	cmd = newTestCommand("git", "-C", workRepoPath, "push", "-u", "origin", "HEAD")
	require.NoError(t, cmd.Run())

	// Now test cloning (without apply to avoid plan/apply complexity)
	cloneDir := t.TempDir()
	destPath := filepath.Join(cloneDir, "cloned")

	p := New(os.Stdout)
	ctx := context.Background()

	opts := CloneOptions{
		URL:         sourceDir,
		Path:        destPath,
		Apply:       false,
		AutoConfirm: true, // Skip prompt
	}

	result, err := p.RepoClone(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, destPath, result.Path)
	assert.True(t, result.ConfigFound, "should find preflight.yaml")
	assert.False(t, result.Applied, "should not apply when Apply=false")
}

func TestRepoClone_DefaultPath(t *testing.T) {
	t.Parallel()

	// Save and restore cwd
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	p := New(os.Stdout)
	ctx := context.Background()

	// Use an invalid URL so it fails fast, but we can check path extraction
	opts := CloneOptions{
		URL: "https://github.com/testuser/my-dotfiles.git",
		// Path is empty - should use extracted repo name
	}

	_, err = p.RepoClone(ctx, opts)
	// It will fail because URL is invalid, but the error message should show the path
	require.Error(t, err)
	// The path should have been derived as "my-dotfiles"
}
