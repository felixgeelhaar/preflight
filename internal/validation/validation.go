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
	ErrInvalidWingetID     = errors.New("invalid winget package ID")
	ErrInvalidWingetSource = errors.New("invalid winget source")
	ErrInvalidScoopBucket  = errors.New("invalid scoop bucket")
	ErrInvalidChocoPackage = errors.New("invalid chocolatey package name")
	ErrInvalidChocoSource  = errors.New("invalid chocolatey source")
	ErrInvalidURL          = errors.New("invalid URL")
	ErrInvalidNpmPackage   = errors.New("invalid npm package name")
	ErrInvalidGoTool       = errors.New("invalid Go tool path")
	ErrInvalidPipPackage   = errors.New("invalid pip package name")
	ErrInvalidGemName      = errors.New("invalid gem name")
	ErrInvalidCargoCrate   = errors.New("invalid cargo crate name")
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

	// wingetIDRegex matches valid winget package IDs: Publisher.PackageName format
	// Examples: "Microsoft.VisualStudioCode", "Git.Git", "7zip.7zip"
	wingetIDRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*\.[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

	// wingetSourceRegex matches valid winget source names
	// Examples: "winget", "msstore"
	wingetSourceRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	// scoopBucketRegex matches valid scoop bucket names
	// Can be simple names ("extras", "versions") or GitHub repos ("user/repo")
	// Examples: "extras", "versions", "ScoopInstaller/Main"
	scoopBucketRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*(/[a-zA-Z][a-zA-Z0-9_-]*)?$`)

	// chocoPackageRegex matches valid Chocolatey package names
	// Chocolatey uses lowercase names with dots and hyphens
	// Examples: "git", "nodejs", "vscode", "7zip.install", "python3"
	chocoPackageRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

	// chocoSourceRegex matches valid Chocolatey source names
	// Examples: "chocolatey", "internal", "my-feed"
	chocoSourceRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

	// urlRegex matches valid HTTP/HTTPS URLs for Chocolatey sources
	// Examples: "https://community.chocolatey.org/api/v2/", "https://nuget.internal.com/v3/"
	urlRegex = regexp.MustCompile(`^https?://[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

	// npmPackageRegex matches valid npm package names (scoped or unscoped with optional @version)
	// Examples: "lodash", "@types/node", "@anthropic-ai/claude-code@2.0.0", "pnpm@10.24.0"
	npmPackageRegex = regexp.MustCompile(`^(@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*(@[a-zA-Z0-9._-]+)?$`)

	// goToolRegex matches valid Go module paths with optional @version
	// Examples: "golang.org/x/tools/gopls@latest", "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.0"
	goToolRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*(\.[a-zA-Z0-9._-]+)*(/[a-zA-Z0-9._-]+)+(@[a-zA-Z0-9._-]+)?$`)

	// pipPackageRegex matches valid pip package names with optional version specifier
	// Examples: "requests", "black==23.1.0", "ruff>=0.1.0", "numpy~=1.24.0"
	pipPackageRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*([=<>!~]=?[a-zA-Z0-9._*-]+)?$`)

	// gemRegex matches valid gem names with optional @version
	// Examples: "rails", "bundler@2.4.0", "rake"
	gemRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*(@[a-zA-Z0-9._-]+)?$`)

	// crateRegex matches valid cargo crate names with optional @version
	// Examples: "ripgrep", "bat@0.22.1", "tokio"
	crateRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*(@[a-zA-Z0-9._-]+)?$`)

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

// ValidateCaskName validates a Homebrew cask name.
// Cask names follow similar rules to package names.
func ValidateCaskName(name string) error {
	return ValidatePackageName(name)
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

// ValidateWingetID validates a winget package ID (Publisher.PackageName format).
func ValidateWingetID(id string) error {
	if id == "" {
		return ErrEmptyInput
	}

	if len(id) > 256 {
		return fmt.Errorf("%w: package ID too long", ErrInvalidWingetID)
	}

	if !wingetIDRegex.MatchString(id) {
		return fmt.Errorf("%w: %q must be in 'Publisher.PackageName' format", ErrInvalidWingetID, id)
	}

	if containsShellMeta(id) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, id)
	}

	return nil
}

// ValidateWingetSource validates a winget source name.
func ValidateWingetSource(source string) error {
	if source == "" {
		return nil // Empty source is allowed (uses default)
	}

	if len(source) > 128 {
		return fmt.Errorf("%w: source name too long", ErrInvalidWingetSource)
	}

	if !wingetSourceRegex.MatchString(source) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidWingetSource, source)
	}

	if containsShellMeta(source) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, source)
	}

	return nil
}

// ValidateScoopBucket validates a scoop bucket name.
func ValidateScoopBucket(bucket string) error {
	if bucket == "" {
		return ErrEmptyInput
	}

	if len(bucket) > 256 {
		return fmt.Errorf("%w: bucket name too long", ErrInvalidScoopBucket)
	}

	if !scoopBucketRegex.MatchString(bucket) {
		return fmt.Errorf("%w: %q must be a valid bucket name or 'user/repo' format", ErrInvalidScoopBucket, bucket)
	}

	if containsShellMeta(bucket) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, bucket)
	}

	return nil
}

// ValidateChocoPackage validates a Chocolatey package name.
func ValidateChocoPackage(name string) error {
	if name == "" {
		return ErrEmptyInput
	}

	if len(name) > 256 {
		return fmt.Errorf("%w: package name too long", ErrInvalidChocoPackage)
	}

	if !chocoPackageRegex.MatchString(name) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidChocoPackage, name)
	}

	if containsShellMeta(name) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, name)
	}

	return nil
}

// ValidateChocoSource validates a Chocolatey source name.
func ValidateChocoSource(source string) error {
	if source == "" {
		return ErrEmptyInput
	}

	if len(source) > 128 {
		return fmt.Errorf("%w: source name too long", ErrInvalidChocoSource)
	}

	if !chocoSourceRegex.MatchString(source) {
		return fmt.Errorf("%w: %q contains invalid characters", ErrInvalidChocoSource, source)
	}

	if containsShellMeta(source) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, source)
	}

	return nil
}

// ValidateURL validates a URL for Chocolatey sources.
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return ErrEmptyInput
	}

	if len(urlStr) > 2048 {
		return fmt.Errorf("%w: URL too long", ErrInvalidURL)
	}

	if !urlRegex.MatchString(urlStr) {
		return fmt.Errorf("%w: %q must be a valid HTTP/HTTPS URL", ErrInvalidURL, urlStr)
	}

	if containsShellMeta(urlStr) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, urlStr)
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

// ValidateNpmPackage validates an npm package name with optional version.
// Supports scoped packages (@org/pkg) and version suffixes (@version).
// Examples: "lodash", "@types/node", "@anthropic-ai/claude-code@2.0.0", "pnpm@10.24.0"
func ValidateNpmPackage(name string) error {
	if name == "" {
		return ErrEmptyInput
	}

	if len(name) > 256 {
		return fmt.Errorf("%w: package name too long", ErrInvalidNpmPackage)
	}

	// Convert to lowercase for validation (npm packages are case-insensitive)
	lower := strings.ToLower(name)
	if !npmPackageRegex.MatchString(lower) {
		return fmt.Errorf("%w: %q is not a valid npm package name", ErrInvalidNpmPackage, name)
	}

	if containsShellMeta(name) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, name)
	}

	return nil
}

// ValidateGoTool validates a Go tool module path with optional version.
// Examples: "golang.org/x/tools/gopls@latest", "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.0"
func ValidateGoTool(tool string) error {
	if tool == "" {
		return ErrEmptyInput
	}

	if len(tool) > 512 {
		return fmt.Errorf("%w: tool path too long", ErrInvalidGoTool)
	}

	if !goToolRegex.MatchString(tool) {
		return fmt.Errorf("%w: %q is not a valid Go module path", ErrInvalidGoTool, tool)
	}

	if containsShellMeta(tool) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, tool)
	}

	return nil
}

// ValidatePipPackage validates a pip package name with optional version specifier.
// Examples: "requests", "black==23.1.0", "ruff>=0.1.0", "numpy~=1.24.0"
func ValidatePipPackage(pkg string) error {
	if pkg == "" {
		return ErrEmptyInput
	}

	if len(pkg) > 256 {
		return fmt.Errorf("%w: package name too long", ErrInvalidPipPackage)
	}

	if !pipPackageRegex.MatchString(pkg) {
		return fmt.Errorf("%w: %q is not a valid pip package name", ErrInvalidPipPackage, pkg)
	}

	if containsShellMeta(pkg) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, pkg)
	}

	return nil
}

// ValidateGemName validates a Ruby gem name with optional version.
// Examples: "rails", "bundler@2.4.0", "rake"
func ValidateGemName(gem string) error {
	if gem == "" {
		return ErrEmptyInput
	}

	if len(gem) > 256 {
		return fmt.Errorf("%w: gem name too long", ErrInvalidGemName)
	}

	if !gemRegex.MatchString(gem) {
		return fmt.Errorf("%w: %q is not a valid gem name", ErrInvalidGemName, gem)
	}

	if containsShellMeta(gem) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, gem)
	}

	return nil
}

// ValidateCargoCrate validates a Cargo crate name with optional version.
// Examples: "ripgrep", "bat@0.22.1", "tokio"
func ValidateCargoCrate(crate string) error {
	if crate == "" {
		return ErrEmptyInput
	}

	if len(crate) > 256 {
		return fmt.Errorf("%w: crate name too long", ErrInvalidCargoCrate)
	}

	if !crateRegex.MatchString(crate) {
		return fmt.Errorf("%w: %q is not a valid crate name", ErrInvalidCargoCrate, crate)
	}

	if containsShellMeta(crate) {
		return fmt.Errorf("%w: %q contains shell metacharacters", ErrCommandInjection, crate)
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
