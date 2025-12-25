package plugin

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for programmatic error handling.
var (
	// ErrNilPlugin indicates a nil plugin was provided.
	ErrNilPlugin = errors.New("plugin cannot be nil")
	// ErrEmptyPluginName indicates a plugin name was empty.
	ErrEmptyPluginName = errors.New("plugin name cannot be empty")
	// ErrManifestNotFound indicates plugin.yaml was not found.
	ErrManifestNotFound = errors.New("plugin.yaml not found")
)

// PluginExistsError indicates a plugin is already registered.
//
//nolint:revive // Name kept for backward compatibility
type PluginExistsError struct {
	Name string
}

func (e *PluginExistsError) Error() string {
	return fmt.Sprintf("plugin %q already registered", e.Name)
}

// ValidationError collects multiple validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(e.Errors, "; "))
}

// Add adds an error message to the collection.
func (e *ValidationError) Add(msg string) {
	e.Errors = append(e.Errors, msg)
}

// Addf adds a formatted error message to the collection.
func (e *ValidationError) Addf(format string, args ...any) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

// HasErrors returns true if there are validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// DiscoveryError represents an error loading a specific plugin.
type DiscoveryError struct {
	Path string
	Err  error
}

func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("loading plugin at %s: %v", e.Path, e.Err)
}

func (e *DiscoveryError) Unwrap() error {
	return e.Err
}

// DiscoveryResult captures both successful loads and errors.
type DiscoveryResult struct {
	Plugins []*Plugin
	Errors  []DiscoveryError
}

// HasErrors returns true if there were errors during discovery.
func (r *DiscoveryResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// PathTraversalError indicates a path traversal attempt was detected.
type PathTraversalError struct {
	Path string
}

func (e *PathTraversalError) Error() string {
	return fmt.Sprintf("path traversal detected in: %s", e.Path)
}

// InvalidURLError indicates a URL is malformed.
type InvalidURLError struct {
	URL    string
	Reason string
}

func (e *InvalidURLError) Error() string {
	return fmt.Sprintf("invalid URL %q: %s", e.URL, e.Reason)
}

// IsPluginExists returns true if the error indicates a plugin already exists.
func IsPluginExists(err error) bool {
	var existsErr *PluginExistsError
	return errors.As(err, &existsErr)
}

// IsValidationError returns true if the error is a validation error.
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsPathTraversal returns true if the error indicates path traversal.
func IsPathTraversal(err error) bool {
	var traversalErr *PathTraversalError
	return errors.As(err, &traversalErr)
}

// ChecksumError indicates a checksum verification failure.
type ChecksumError struct {
	Expected string
	Actual   string
}

func (e *ChecksumError) Error() string {
	return fmt.Sprintf("checksum mismatch: expected %s, got %s", e.Expected, e.Actual)
}

// IsChecksumError returns true if the error is a checksum verification failure.
func IsChecksumError(err error) bool {
	var checksumErr *ChecksumError
	return errors.As(err, &checksumErr)
}

// SignatureError indicates a signature verification failure.
type SignatureError struct {
	Reason string
}

func (e *SignatureError) Error() string {
	return fmt.Sprintf("signature verification failed: %s", e.Reason)
}

// IsSignatureError returns true if the error is a signature verification failure.
func IsSignatureError(err error) bool {
	var sigErr *SignatureError
	return errors.As(err, &sigErr)
}

// CapabilityError indicates an unauthorized capability request.
type CapabilityError struct {
	Capability string
	Reason     string
}

func (e *CapabilityError) Error() string {
	return fmt.Sprintf("capability %q not allowed: %s", e.Capability, e.Reason)
}

// IsCapabilityError returns true if the error is a capability validation failure.
func IsCapabilityError(err error) bool {
	var capErr *CapabilityError
	return errors.As(err, &capErr)
}

// TrustError indicates a plugin does not meet trust requirements.
type TrustError struct {
	PluginName string
	Level      TrustLevel
	Required   TrustLevel
	Reason     string
}

func (e *TrustError) Error() string {
	return fmt.Sprintf("plugin %q trust level %q does not meet requirement %q: %s",
		e.PluginName, e.Level, e.Required, e.Reason)
}

// IsTrustError returns true if the error is a trust level violation.
func IsTrustError(err error) bool {
	var trustErr *TrustError
	return errors.As(err, &trustErr)
}

// ManifestSizeError indicates a manifest exceeds the size limit.
type ManifestSizeError struct {
	Size  int64
	Limit int64
}

func (e *ManifestSizeError) Error() string {
	return fmt.Sprintf("manifest size %d bytes exceeds limit of %d bytes", e.Size, e.Limit)
}

// IsManifestSizeError returns true if the error is a manifest size violation.
func IsManifestSizeError(err error) bool {
	var sizeErr *ManifestSizeError
	return errors.As(err, &sizeErr)
}

// GitNotFoundError indicates git is not installed or not in PATH.
type GitNotFoundError struct{}

func (e *GitNotFoundError) Error() string {
	return "git not found: please install git and ensure it is in your PATH"
}

// IsGitNotFound returns true if the error indicates git is not available.
func IsGitNotFound(err error) bool {
	var gitErr *GitNotFoundError
	return errors.As(err, &gitErr)
}

// GitCloneError indicates a git clone operation failed.
type GitCloneError struct {
	URL    string
	Reason string
}

func (e *GitCloneError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("git clone failed for %s: %s", e.URL, e.Reason)
	}
	return fmt.Sprintf("git clone failed for %s", e.URL)
}

// IsGitCloneError returns true if the error is a git clone failure.
func IsGitCloneError(err error) bool {
	var cloneErr *GitCloneError
	return errors.As(err, &cloneErr)
}
