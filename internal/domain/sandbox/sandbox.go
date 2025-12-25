// Package sandbox provides WASM-based plugin isolation.
package sandbox

import (
	"context"
	"errors"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

// Sandbox errors.
var (
	ErrPluginNotFound     = errors.New("plugin not found")
	ErrPluginInvalid      = errors.New("invalid plugin")
	ErrSandboxTimeout     = errors.New("sandbox execution timeout")
	ErrResourceExhausted  = errors.New("resource limit exceeded")
	ErrCapabilityDenied   = errors.New("capability denied by sandbox")
	ErrSandboxUnavailable = errors.New("sandbox runtime unavailable")
)

// Mode represents the sandbox isolation level.
type Mode string

// Sandbox modes.
const (
	// ModeFull provides complete isolation with no side effects.
	// Used for previewing/auditing unknown plugins.
	ModeFull Mode = "full"

	// ModeRestricted limits plugin to declared capabilities.
	// Normal operation mode for installed plugins.
	ModeRestricted Mode = "restricted"

	// ModeTrusted provides full access like builtin plugins.
	// Only for verified publishers.
	ModeTrusted Mode = "trusted"
)

// Config holds sandbox configuration.
type Config struct {
	// Mode determines the isolation level
	Mode Mode

	// Timeout for plugin execution
	Timeout time.Duration

	// Limits for resource consumption
	Limits ResourceLimits

	// Policy for capability enforcement
	Policy *capability.Policy

	// AllowNetwork enables network access (requires capability)
	AllowNetwork bool

	// AllowFileSystem enables filesystem access (requires capability)
	AllowFileSystem bool
}

// ResourceLimits defines resource constraints.
type ResourceLimits struct {
	// MaxMemoryBytes limits memory allocation
	MaxMemoryBytes uint64

	// MaxCPUTime limits CPU execution time
	MaxCPUTime time.Duration

	// MaxFileDescriptors limits open file handles
	MaxFileDescriptors int

	// MaxOutputBytes limits stdout/stderr size
	MaxOutputBytes int64
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Mode:    ModeRestricted,
		Timeout: 30 * time.Second,
		Limits:  DefaultLimits(),
		Policy:  capability.DefaultPolicy(),
	}
}

// DefaultLimits returns default resource limits.
func DefaultLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryBytes:     64 * 1024 * 1024, // 64 MB
		MaxCPUTime:         10 * time.Second,
		MaxFileDescriptors: 16,
		MaxOutputBytes:     1024 * 1024, // 1 MB
	}
}

// FullIsolationConfig returns config for complete isolation.
func FullIsolationConfig() Config {
	return Config{
		Mode:            ModeFull,
		Timeout:         10 * time.Second,
		Limits:          RestrictedLimits(),
		Policy:          capability.RestrictedPolicy(),
		AllowNetwork:    false,
		AllowFileSystem: false,
	}
}

// RestrictedLimits returns tighter resource limits.
func RestrictedLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryBytes:     16 * 1024 * 1024, // 16 MB
		MaxCPUTime:         5 * time.Second,
		MaxFileDescriptors: 4,
		MaxOutputBytes:     256 * 1024, // 256 KB
	}
}

// TrustedConfig returns config for trusted plugins.
func TrustedConfig() Config {
	return Config{
		Mode:            ModeTrusted,
		Timeout:         5 * time.Minute,
		Limits:          TrustedLimits(),
		Policy:          capability.FullAccessPolicy(),
		AllowNetwork:    true,
		AllowFileSystem: true,
	}
}

// TrustedLimits returns relaxed limits for trusted plugins.
func TrustedLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryBytes:     256 * 1024 * 1024, // 256 MB
		MaxCPUTime:         60 * time.Second,
		MaxFileDescriptors: 64,
		MaxOutputBytes:     10 * 1024 * 1024, // 10 MB
	}
}

// Plugin represents a sandboxed plugin.
type Plugin struct {
	// ID is the unique plugin identifier
	ID string

	// Name is the human-readable name
	Name string

	// Version is the plugin version
	Version string

	// Module is the compiled WASM module bytes
	Module []byte

	// Capabilities required by the plugin
	Capabilities *capability.Requirements

	// Checksum is the SHA256 hash of the module
	Checksum string
}

// Validate checks if the plugin is valid.
func (p *Plugin) Validate() error {
	if p.ID == "" {
		return errors.New("plugin ID is required")
	}
	if p.Name == "" {
		return errors.New("plugin name is required")
	}
	if len(p.Module) == 0 {
		return errors.New("plugin module is required")
	}
	return nil
}

// ExecutionResult holds the result of plugin execution.
type ExecutionResult struct {
	// Success indicates if execution completed without errors
	Success bool

	// Output from the plugin (stdout)
	Output []byte

	// Errors from the plugin (stderr)
	Errors []byte

	// Duration of execution
	Duration time.Duration

	// ResourceUsage during execution
	ResourceUsage ResourceUsage

	// Error if execution failed
	Error error
}

// ResourceUsage tracks resource consumption.
type ResourceUsage struct {
	// MemoryBytes used during execution
	MemoryBytes uint64

	// CPUTime consumed
	CPUTime time.Duration

	// FileDescriptors opened
	FileDescriptors int

	// OutputBytes written
	OutputBytes int64
}

// Sandbox provides isolated plugin execution.
type Sandbox interface {
	// Execute runs a plugin with the given input
	Execute(ctx context.Context, plugin *Plugin, input []byte) (*ExecutionResult, error)

	// Validate checks if a plugin can be loaded
	Validate(ctx context.Context, plugin *Plugin) error

	// Close releases sandbox resources
	Close() error
}

// Runtime manages sandbox instances.
type Runtime interface {
	// NewSandbox creates a new sandbox with the given config
	NewSandbox(config Config) (Sandbox, error)

	// IsAvailable checks if the runtime is available
	IsAvailable() bool

	// Version returns the runtime version
	Version() string

	// Close releases runtime resources
	Close() error
}
