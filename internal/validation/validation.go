// Package validation provides input validation utilities to prevent security vulnerabilities
// such as command injection, path traversal, and other input-based attacks.
package validation

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Common validation errors.
var (
	ErrEmptyInput          = errors.New("input cannot be empty")
	ErrInvalidPackageName  = errors.New("invalid package name")
	ErrInvalidTapName      = errors.New("invalid tap name")
	ErrInvalidPPA          = errors.New("invalid PPA format")
	ErrPathTraversal       = errors.New("path traversal detected")
	ErrInvalidPath         = errors.New("invalid path")
	ErrCommandInjection    = errors.New("potential command injection detected")
	ErrInvalidHostname     = errors.New("invalid hostname")
	ErrNewlineInjection    = errors.New("newline injection detected")
	ErrInvalidGitConfig    = errors.New("invalid git config value")
	ErrInvalidSSHParameter = errors.New("invalid SSH parameter")
	ErrInvalidBrewArg      = errors.New("invalid brew argument")
)

// Compiled regex patterns for validation (compiled once for performance).
var (
	// packageNameRegex matches valid package names: alphanumeric, hyphens, underscores, dots, plus
	// Examples: "git", "node-lts", "python3.11", "g++"
	packageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+-]*$`)

	// brewArgRegex matches valid Homebrew arguments: options and flags
	// Examples: "--HEAD", "--with-openssl", "--force"
	brewArgRegex = regexp.MustCompile(`^--?[a-zA-Z][a-zA-Z0-9_-]*$`)

	// tapNameRegex matches valid Homebrew tap names: "owner/repo" format
	// Examples: "homebrew/core", "github/gh"
	tapNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$`)

	// ppaRegex matches valid PPA format: "ppa:owner/name" or "owner/name"
	// Examples: "ppa:deadsnakes/ppa", "git-core/ppa"
	ppaRegex = regexp.MustCompile(`^(ppa:)?[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$`)

	// hostnameRegex matches valid hostnames (including wildcards for SSH)
	// Examples: "github.com", "*.example.com", "192.168.1.1"
	hostnameRegex = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9][a-zA-Z0-9._*-]*$`)

	// gitConfigSafeRegex matches safe git config values (no newlines, no control chars)
	gitConfigSafeRegex = regexp.MustCompile(`^[^\x00-\x1f\x7f]*$`)

	// shellMetaChars contains shell metacharacters that could enable injection
	shellMetaChars = []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "<", ">", "\n", "\r", "\\"}
)

// ValidatePackageName validates a package name for brew or apt.
// Returns an error if the name is empty or contains invalid characters.
func ValidatePackageName(name string) error {
	if name == "" {
		return ErrEmptyInput
	}

	// Check for maximum length (reasonable limit)
	if len(name) > 256 {
		return fmt.Errorf("%w: name too long (max 256 characters)", ErrInvalidPackageName)
	}

	// Check against valid pattern
	if !packageNameRegex.MatchString(name) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidPackageName, name)
	}

	// Check for shell metacharacters (defense in depth)
	if containsShellMeta(name) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, name)
	}

	return nil
}

// ValidateBrewArg validates a Homebrew install argument (e.g., --HEAD, --with-openssl).
func ValidateBrewArg(arg string) error {
	if arg == "" {
		return ErrEmptyInput
	}

	if len(arg) > 256 {
		return fmt.Errorf("%w: argument too long", ErrInvalidBrewArg)
	}

	// Check against valid pattern (--flag or -flag format)
	if !brewArgRegex.MatchString(arg) {
		return fmt.Errorf("%w: %q is not a valid brew argument", ErrInvalidBrewArg, arg)
	}

	return nil
}

// ValidateTapName validates a Homebrew tap name (owner/repo format).
func ValidateTapName(tap string) error {
	if tap == "" {
		return ErrEmptyInput
	}

	if len(tap) > 256 {
		return fmt.Errorf("%w: tap name too long", ErrInvalidTapName)
	}

	if !tapNameRegex.MatchString(tap) {
		return fmt.Errorf("%w: %q must be in 'owner/repo' format", ErrInvalidTapName, tap)
	}

	if containsShellMeta(tap) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, tap)
	}

	return nil
}

// ValidatePPA validates an APT PPA name.
func ValidatePPA(ppa string) error {
	if ppa == "" {
		return ErrEmptyInput
	}

	if len(ppa) > 256 {
		return fmt.Errorf("%w: PPA name too long", ErrInvalidPPA)
	}

	if !ppaRegex.MatchString(ppa) {
		return fmt.Errorf("%w: %q must be in 'ppa:owner/name' or 'owner/name' format", ErrInvalidPPA, ppa)
	}

	if containsShellMeta(ppa) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, ppa)
	}

	return nil
}

// ValidatePath validates a file path and prevents path traversal attacks.
// It ensures the path doesn't escape the intended base directory.
func ValidatePath(path string) error {
	if path == "" {
		return ErrEmptyInput
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("%w: path contains null byte", ErrInvalidPath)
	}

	// Check for path traversal sequences
	if containsPathTraversal(path) {
		return fmt.Errorf("%w: %q contains traversal sequence", ErrPathTraversal, path)
	}

	return nil
}

// ValidatePathWithBase validates a path is within the expected base directory.
// This is the recommended function for file operations.
func ValidatePathWithBase(path, basePath string) error {
	if err := ValidatePath(path); err != nil {
		return err
	}

	// Expand and clean both paths
	expandedPath := expandPath(path)
	cleanPath := filepath.Clean(expandedPath)
	cleanBase := filepath.Clean(basePath)

	// Ensure the path is within the base directory
	if !strings.HasPrefix(cleanPath, cleanBase) {
		return fmt.Errorf("%w: path %q escapes base directory %q", ErrPathTraversal, path, basePath)
	}

	return nil
}

// ValidateHostname validates a hostname for SSH configuration.
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return ErrEmptyInput
	}

	if len(hostname) > 253 {
		return fmt.Errorf("%w: hostname too long", ErrInvalidHostname)
	}

	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidHostname, hostname)
	}

	if containsShellMeta(hostname) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, hostname)
	}

	return nil
}

// ValidateGitConfigValue validates a git config value for injection attacks.
func ValidateGitConfigValue(value string) error {
	// Check for newlines which could inject additional config lines
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("%w: git config value contains newlines", ErrNewlineInjection)
	}

	// Check for control characters
	if !gitConfigSafeRegex.MatchString(value) {
		return fmt.Errorf("%w: contains control characters", ErrInvalidGitConfig)
	}

	return nil
}

// ValidateSSHProxyCommand validates an SSH ProxyCommand value.
// This is particularly security-sensitive as it's executed by SSH.
func ValidateSSHProxyCommand(cmd string) error {
	if cmd == "" {
		return nil // Empty is allowed
	}

	// Check for newlines
	if strings.ContainsAny(cmd, "\n\r") {
		return fmt.Errorf("%w: command contains newlines", ErrNewlineInjection)
	}

	// Only allow known-safe patterns for ProxyCommand
	// Valid examples: "ssh -W %h:%p jump-host", "nc -X 5 -x proxy:port %h %p"
	// The %h and %p are SSH placeholders, safe to use

	// Check for dangerous shell metacharacters (beyond what's needed for basic commands)
	dangerousChars := []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "<", ">", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(cmd, char) {
			return fmt.Errorf("%w: ProxyCommand contains dangerous character %q", ErrCommandInjection, char)
		}
	}

	return nil
}

// ValidateSSHParameter validates generic SSH config parameters.
func ValidateSSHParameter(value string) error {
	if value == "" {
		return nil
	}

	// Check for newlines (could inject additional config)
	if strings.ContainsAny(value, "\n\r") {
		return fmt.Errorf("%w: parameter contains newlines", ErrNewlineInjection)
	}

	// Check for control characters
	if !gitConfigSafeRegex.MatchString(value) {
		return fmt.Errorf("%w: contains control characters", ErrInvalidSSHParameter)
	}

	return nil
}

// ValidatePluginName validates a plugin name for shell frameworks.
func ValidatePluginName(name string) error {
	if name == "" {
		return ErrEmptyInput
	}

	if len(name) > 256 {
		return fmt.Errorf("%w: plugin name too long", ErrInvalidPackageName)
	}

	// Plugin names can be GitHub repos (owner/repo) or simple names
	if !packageNameRegex.MatchString(name) && !tapNameRegex.MatchString(name) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidPackageName, name)
	}

	if containsShellMeta(name) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, name)
	}

	return nil
}

// containsShellMeta checks if a string contains shell metacharacters.
func containsShellMeta(s string) bool {
	for _, char := range shellMetaChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

// containsPathTraversal checks for common path traversal patterns.
func containsPathTraversal(path string) bool {
	// Normalize the path to catch encoded traversal attempts
	normalized := filepath.Clean(path)

	// Check for ".." sequences in the normalized path
	segments := strings.Split(normalized, string(filepath.Separator))
	for _, seg := range segments {
		if seg == ".." {
			return true
		}
	}

	// Check for URL-encoded traversal
	if strings.Contains(path, "%2e%2e") || strings.Contains(path, "%2E%2E") {
		return true
	}

	return false
}

// expandPath expands ~ to the home directory.
// Note: This is a simplified version - ports.ExpandPath should be used in production.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		// In actual implementation, this would use os.UserHomeDir()
		// For validation purposes, we just clean the path
		return path
	}
	return path
}
