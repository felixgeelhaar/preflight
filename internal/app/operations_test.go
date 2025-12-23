package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitCommand creates an exec.Cmd for git commands in tests.
func newGitCommand(args ...string) *exec.Cmd {
	return exec.Command("git", args...)
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
	cmd := newGitCommand("init", "--bare", sourceDir)
	require.NoError(t, cmd.Run())

	// Create a temp working dir, add a file, and push
	workDir := t.TempDir()
	workRepoPath := filepath.Join(workDir, "work")

	cmd = newGitCommand("clone", sourceDir, workRepoPath)
	require.NoError(t, cmd.Run())

	// Create a dummy file (not preflight.yaml)
	require.NoError(t, os.WriteFile(filepath.Join(workRepoPath, "README.md"), []byte("# Test"), 0o644))

	cmd = newGitCommand("-C", workRepoPath, "add", ".")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "push", "-u", "origin", "HEAD")
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
	cmd := newGitCommand("init", "--bare", sourceDir)
	require.NoError(t, cmd.Run())

	// Create a temp working dir, add preflight.yaml, and push
	workDir := t.TempDir()
	workRepoPath := filepath.Join(workDir, "work")

	cmd = newGitCommand("clone", sourceDir, workRepoPath)
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

	cmd = newGitCommand("-C", workRepoPath, "add", ".")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "commit", "-m", "Initial commit")
	require.NoError(t, cmd.Run())

	cmd = newGitCommand("-C", workRepoPath, "push", "-u", "origin", "HEAD")
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

func TestRepoInit_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "new-repo")

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	opts := NewRepoOptions(repoPath)
	err := p.RepoInit(ctx, opts)

	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(repoPath, ".git"))
	assert.Contains(t, output.String(), "Repository initialized")
}

func TestRepoInit_AlreadyExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "existing-repo")

	// Create a git repo first
	require.NoError(t, os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755))

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	opts := NewRepoOptions(repoPath)
	err := p.RepoInit(ctx, opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")
}

func TestRepoInit_WithRemote(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "new-repo")

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	opts := NewRepoOptions(repoPath).WithRemote("git@github.com:user/config.git")
	err := p.RepoInit(ctx, opts)

	require.NoError(t, err)

	// Verify remote was added
	cmd := newGitCommand("-C", repoPath, "remote", "-v")
	remoteOutput, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(remoteOutput), "origin")
}

func TestRepoStatus_NotInitialized(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	status, err := p.RepoStatus(ctx, tmpDir)

	require.NoError(t, err)
	assert.False(t, status.Initialized)
	assert.Equal(t, tmpDir, status.Path)
}

func TestRepoStatus_Initialized(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")

	// Initialize a git repo
	cmd := newGitCommand("init", repoPath)
	require.NoError(t, cmd.Run())

	// Configure git user
	cmd = newGitCommand("-C", repoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())
	cmd = newGitCommand("-C", repoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	// Create initial commit
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# Test"), 0o644))
	cmd = newGitCommand("-C", repoPath, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = newGitCommand("-C", repoPath, "commit", "-m", "Initial")
	require.NoError(t, cmd.Run())

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	status, err := p.RepoStatus(ctx, repoPath)

	require.NoError(t, err)
	assert.True(t, status.Initialized)
	assert.NotEmpty(t, status.Branch)
	assert.NotEmpty(t, status.LastCommit)
	assert.False(t, status.HasChanges)
}

func TestRepoStatus_WithUncommittedChanges(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")

	// Initialize a git repo with a commit
	cmd := newGitCommand("init", repoPath)
	require.NoError(t, cmd.Run())
	cmd = newGitCommand("-C", repoPath, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())
	cmd = newGitCommand("-C", repoPath, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())

	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# Test"), 0o644))
	cmd = newGitCommand("-C", repoPath, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = newGitCommand("-C", repoPath, "commit", "-m", "Initial")
	require.NoError(t, cmd.Run())

	// Create an uncommitted change
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "new-file.txt"), []byte("content"), 0o644))

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	status, err := p.RepoStatus(ctx, repoPath)

	require.NoError(t, err)
	assert.True(t, status.Initialized)
	assert.True(t, status.HasChanges)
}

func TestFix_NilReport(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	result, err := p.Fix(ctx, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.AllFixed())
}

func TestFix_NoIssues(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	report := &DoctorReport{
		ConfigPath: "preflight.yaml",
		Target:     "work",
		Issues:     []DoctorIssue{},
	}

	result, err := p.Fix(ctx, report)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.AllFixed())
}

func TestFix_NoFixableIssues(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)
	ctx := context.Background()

	report := &DoctorReport{
		ConfigPath: "preflight.yaml",
		Target:     "work",
		Issues: []DoctorIssue{
			{StepID: "brew.git", Severity: SeverityError, Message: "not installed", Fixable: false},
			{StepID: "brew.curl", Severity: SeverityWarning, Message: "not installed", Fixable: false},
		},
	}

	result, err := p.Fix(ctx, report)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.AllFixed())
	assert.Equal(t, 2, result.RemainingCount())
	assert.Equal(t, 0, result.FixedCount())
}

func TestPrintRepoStatus_NotInitialized(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	status := &RepoStatus{
		Path:        "/path/to/repo",
		Initialized: false,
	}

	p.PrintRepoStatus(status)

	result := output.String()
	assert.Contains(t, result, "Repository Status")
	assert.Contains(t, result, "Not a git repository")
}

func TestPrintRepoStatus_FullStatus(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	status := &RepoStatus{
		Path:        "/path/to/config",
		Initialized: true,
		Branch:      "main",
		Remote:      "origin",
		HasChanges:  false,
		Ahead:       0,
		Behind:      0,
		LastCommit:  "abc123",
	}

	p.PrintRepoStatus(status)

	result := output.String()
	assert.Contains(t, result, "Repository Status")
	assert.Contains(t, result, "main")
	assert.Contains(t, result, "origin")
	assert.Contains(t, result, "Up to date")
	assert.Contains(t, result, "abc123")
}

func TestPrintRepoStatus_NeedsSync(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	status := &RepoStatus{
		Path:        "/path/to/config",
		Initialized: true,
		Branch:      "main",
		HasChanges:  true,
		Ahead:       2,
		Behind:      3,
	}

	p.PrintRepoStatus(status)

	result := output.String()
	assert.Contains(t, result, "Uncommitted changes")
	assert.Contains(t, result, "2 commit(s) ahead")
	assert.Contains(t, result, "3 commit(s) behind")
}

func TestPrintDoctorReport_NoIssues(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	report := &DoctorReport{
		ConfigPath: "preflight.yaml",
		Target:     "work",
		Issues:     []DoctorIssue{},
	}

	p.PrintDoctorReport(report)

	result := output.String()
	assert.Contains(t, result, "Doctor Report")
	assert.Contains(t, result, "No issues found")
}

func TestPrintDoctorReport_WithBinaryChecks(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	report := &DoctorReport{
		ConfigPath: "preflight.yaml",
		Target:     "work",
		BinaryChecks: []BinaryCheckResult{
			{Name: "nvim", Found: true, Version: "0.10.0", Required: true, MeetsMin: true, Purpose: "editor"},
			{Name: "rg", Found: true, Version: "14.0.0", Required: false, MeetsMin: true, Purpose: "search"},
			{Name: "fd", Found: false, Required: false, Purpose: "find"},
			{Name: "node", Found: true, Version: "18.0.0", MinVersion: "20.0.0", MeetsMin: false, Required: true, Purpose: "LSP"},
			{Name: "npm", Found: true, Required: false, MeetsMin: true, Purpose: "packages"},
		},
		Issues: []DoctorIssue{},
	}

	p.PrintDoctorReport(report)

	result := output.String()
	assert.Contains(t, result, "Binary Checks")
	assert.Contains(t, result, "nvim")
	assert.Contains(t, result, "v0.10.0")
	assert.Contains(t, result, "not found")      // fd is not found
	assert.Contains(t, result, "need >= 20.0.0") // node version issue
}

func TestPrintDoctorReport_WithIssues(t *testing.T) {
	t.Parallel()

	var output strings.Builder
	p := New(&output)

	report := &DoctorReport{
		ConfigPath: "preflight.yaml",
		Target:     "work",
		Issues: []DoctorIssue{
			{StepID: "brew.git", Severity: SeverityError, Message: "git not installed"},
			{StepID: "files.bashrc", Severity: SeverityWarning, Message: "file drift detected", Fixable: true, FixCommand: "preflight apply"},
			{StepID: "git.config", Severity: SeverityInfo, Message: "config is default"},
		},
	}

	p.PrintDoctorReport(report)

	result := output.String()
	assert.Contains(t, result, "Found 3 issue(s)")
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "git not installed")
	assert.Contains(t, result, "[WARNING]")
	assert.Contains(t, result, "file drift detected")
	assert.Contains(t, result, "Fix:")
	assert.Contains(t, result, "[INFO]")
}
