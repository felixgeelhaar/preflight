// Package validation provides input validation utilities.
package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// Git input validation patterns.
var (
	// gitBranchPattern allows alphanumeric, hyphens, underscores, slashes, and dots.
	gitBranchPattern = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

	// gitRemoteURLPatterns for valid git remote URLs and local paths.
	gitRemoteURLPatterns = []*regexp.Regexp{
		// HTTPS URLs: https://github.com/user/repo.git or https://github.com/user/repo
		regexp.MustCompile(`^https://[a-zA-Z0-9.-]+/[a-zA-Z0-9_./-]+(?:\.git)?$`),
		// SSH URLs: git@github.com:user/repo.git
		regexp.MustCompile(`^git@[a-zA-Z0-9.-]+:[a-zA-Z0-9_./-]+(?:\.git)?$`),
		// SSH protocol: ssh://git@github.com/user/repo.git
		regexp.MustCompile(`^ssh://[a-zA-Z0-9@.-]+/[a-zA-Z0-9_./-]+(?:\.git)?$`),
		// file:// URLs: file:///path/to/repo
		regexp.MustCompile(`^file:///[a-zA-Z0-9_./-]+$`),
		// Unix absolute paths: /path/to/repo
		regexp.MustCompile(`^/[a-zA-Z0-9_./-]+$`),
		// Windows paths: C:\path\to\repo or C:/path/to/repo
		regexp.MustCompile(`^[a-zA-Z]:[/\\][a-zA-Z0-9_./\\-]+$`),
	}

	// Dangerous characters that should never appear in git inputs.
	// Note: null byte (\x00) is checked separately for a more specific error message
	dangerousChars = []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "!", "\n", "\r"}
)

// ValidateGitBranch validates a git branch name.
func ValidateGitBranch(branch string) error {
	if branch == "" {
		return nil // Empty is allowed, will use default
	}

	if len(branch) > 255 {
		return fmt.Errorf("branch name too long (max 255 characters)")
	}

	// Check for null bytes first (specific error message)
	if strings.ContainsRune(branch, '\x00') {
		return fmt.Errorf("branch name contains null byte")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.Contains(branch, char) {
			return fmt.Errorf("branch name contains invalid character: %q", char)
		}
	}

	if !gitBranchPattern.MatchString(branch) {
		return fmt.Errorf("invalid branch name format: must contain only alphanumeric characters, hyphens, underscores, slashes, and dots")
	}

	// Prevent path traversal in branch names
	if strings.Contains(branch, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}

	return nil
}

// ValidateGitRemoteURL validates a git remote URL.
func ValidateGitRemoteURL(url string) error {
	if url == "" {
		return nil // Empty is allowed
	}

	if len(url) > 2048 {
		return fmt.Errorf("remote URL too long (max 2048 characters)")
	}

	// Check for null bytes first (specific error message)
	if strings.ContainsRune(url, '\x00') {
		return fmt.Errorf("remote URL contains null byte")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.Contains(url, char) {
			return fmt.Errorf("remote URL contains invalid character: %q", char)
		}
	}

	// Must match one of the valid patterns
	for _, pattern := range gitRemoteURLPatterns {
		if pattern.MatchString(url) {
			return nil
		}
	}

	return fmt.Errorf("invalid git remote URL format: must be HTTPS, SSH URL, or local path")
}

// ValidateGitPath validates a path for git operations.
func ValidateGitPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if len(path) > 4096 {
		return fmt.Errorf("path too long (max 4096 characters)")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains invalid character: %q", char)
		}
	}

	// Check for null bytes
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("path contains null byte")
	}

	return nil
}

// ValidateGitRepoName validates a repository name.
func ValidateGitRepoName(name string) error {
	if name == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("repository name too long (max 100 characters)")
	}

	// Repository names should be alphanumeric with hyphens and underscores
	pattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	if !pattern.MatchString(name) {
		return fmt.Errorf("invalid repository name: must start with alphanumeric and contain only alphanumeric, hyphens, underscores, and dots")
	}

	return nil
}

// ValidateGitRemoteName validates a git remote name.
func ValidateGitRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	if len(name) > 255 {
		return fmt.Errorf("remote name too long (max 255 characters)")
	}

	// Check for null bytes first (specific error message)
	if strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("remote name contains null byte")
	}

	// Check for dangerous characters
	for _, char := range dangerousChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("remote name contains invalid character: %q", char)
		}
	}

	if !gitBranchPattern.MatchString(name) {
		return fmt.Errorf("invalid remote name format")
	}

	// Prevent path traversal in remote names
	if strings.Contains(name, "..") {
		return fmt.Errorf("remote name cannot contain '..'")
	}

	return nil
}
