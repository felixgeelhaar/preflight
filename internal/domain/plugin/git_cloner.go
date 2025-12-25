package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// GitCloner handles cloning plugins from Git repositories.
type GitCloner struct {
	// MaxCloneDepth sets shallow clone depth (1 = shallow, 0 = full).
	MaxCloneDepth int
	// Timeout is the maximum time for clone operations in seconds.
	Timeout int
	// GitPath is the path to the git binary (defaults to "git").
	GitPath string
}

// NewGitCloner creates a new GitCloner with sensible defaults.
func NewGitCloner() *GitCloner {
	return &GitCloner{
		MaxCloneDepth: 1,  // Shallow clone by default
		Timeout:       60, // 60 second timeout
		GitPath:       "git",
	}
}

// Clone clones a Git repository to the target path.
// If ref is empty, clones the default branch.
// If ref is specified, clones that specific tag or branch.
func (g *GitCloner) Clone(ctx context.Context, repoURL, ref, targetPath string) error {
	// Check if git is available
	if err := g.checkGitAvailable(ctx); err != nil {
		return err
	}

	// Create timeout context
	timeout := time.Duration(g.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build clone command
	args := []string{"clone", "--single-branch"}

	// Add depth for shallow clone
	if g.MaxCloneDepth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", g.MaxCloneDepth))
	}

	// Add specific ref if provided
	if ref != "" {
		args = append(args, "--branch", ref)
	}

	args = append(args, repoURL, targetPath)

	// Execute clone with security environment
	cmd := exec.CommandContext(ctx, g.gitPath(), args...)
	cmd.Env = g.safeEnv()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Clean up partial clone on failure
		_ = os.RemoveAll(targetPath)

		if ctx.Err() == context.DeadlineExceeded {
			return &GitCloneError{
				URL:    repoURL,
				Reason: "clone timed out",
			}
		}
		return &GitCloneError{
			URL:    repoURL,
			Reason: strings.TrimSpace(stderr.String()),
		}
	}

	return nil
}

// FetchTags retrieves available tags from a remote repository.
// Returns tags sorted by semantic version (newest first).
func (g *GitCloner) FetchTags(ctx context.Context, repoURL string) ([]string, error) {
	// Check if git is available
	if err := g.checkGitAvailable(ctx); err != nil {
		return nil, err
	}

	// Create timeout context
	timeout := time.Duration(g.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use ls-remote to fetch tags without cloning
	cmd := exec.CommandContext(ctx, g.gitPath(), "ls-remote", "--tags", "--refs", repoURL)
	cmd.Env = g.safeEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &GitCloneError{
				URL:    repoURL,
				Reason: "fetching tags timed out",
			}
		}
		return nil, &GitCloneError{
			URL:    repoURL,
			Reason: strings.TrimSpace(stderr.String()),
		}
	}

	return parseTags(stdout.String()), nil
}

// LatestVersion returns the latest semantic version tag from a repository.
// Returns empty string if no valid semver tags are found.
func (g *GitCloner) LatestVersion(ctx context.Context, repoURL string) (string, error) {
	tags, err := g.FetchTags(ctx, repoURL)
	if err != nil {
		return "", err
	}

	// Filter for valid semver tags
	var versions []string
	for _, tag := range tags {
		// Normalize to semver format
		v := tag
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}
		if semver.IsValid(v) {
			versions = append(versions, tag)
		}
	}

	if len(versions) == 0 {
		return "", nil
	}

	// Sort by semver (descending)
	sort.Slice(versions, func(i, j int) bool {
		vi, vj := versions[i], versions[j]
		if !strings.HasPrefix(vi, "v") {
			vi = "v" + vi
		}
		if !strings.HasPrefix(vj, "v") {
			vj = "v" + vj
		}
		return semver.Compare(vi, vj) > 0
	})

	return versions[0], nil
}

// checkGitAvailable verifies git is installed and accessible.
func (g *GitCloner) checkGitAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, g.gitPath(), "--version")
	if err := cmd.Run(); err != nil {
		return &GitNotFoundError{}
	}
	return nil
}

// gitPath returns the configured git binary path.
func (g *GitCloner) gitPath() string {
	if g.GitPath != "" {
		return g.GitPath
	}
	return "git"
}

// safeEnv returns environment variables for secure git operations.
func (g *GitCloner) safeEnv() []string {
	// Start with minimal environment
	env := []string{
		// Prevent git from prompting for credentials
		"GIT_TERMINAL_PROMPT=0",
		// Disable credential helpers that might prompt
		"GIT_ASKPASS=",
		// Ensure consistent behavior
		"LC_ALL=C",
	}

	// Preserve essential environment variables
	for _, key := range []string{"HOME", "PATH", "USER", "LANG", "SSH_AUTH_SOCK"} {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	return env
}

// parseTags extracts tag names from git ls-remote output.
func parseTags(output string) []string {
	var tags []string
	// Match refs/tags/<tagname> format
	tagPattern := regexp.MustCompile(`refs/tags/([^\s^]+)$`)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := tagPattern.FindStringSubmatch(line)
		if len(matches) >= 2 {
			tags = append(tags, matches[1])
		}
	}

	return tags
}

// Update pulls the latest changes for an already installed plugin.
func (g *GitCloner) Update(ctx context.Context, pluginPath string) error {
	// Check if git is available
	if err := g.checkGitAvailable(ctx); err != nil {
		return err
	}

	// Verify it's a git repository
	gitDir := filepath.Join(pluginPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &GitCloneError{
			URL:    pluginPath,
			Reason: "not a git repository",
		}
	}

	// Create timeout context
	timeout := time.Duration(g.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Fetch and reset to origin
	cmd := exec.CommandContext(ctx, g.gitPath(), "-C", pluginPath, "fetch", "--depth=1", "origin")
	cmd.Env = g.safeEnv()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &GitCloneError{
			URL:    pluginPath,
			Reason: "fetch failed: " + strings.TrimSpace(stderr.String()),
		}
	}

	// Reset to origin/HEAD
	cmd = exec.CommandContext(ctx, g.gitPath(), "-C", pluginPath, "reset", "--hard", "origin/HEAD")
	cmd.Env = g.safeEnv()
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &GitCloneError{
			URL:    pluginPath,
			Reason: "reset failed: " + strings.TrimSpace(stderr.String()),
		}
	}

	return nil
}
