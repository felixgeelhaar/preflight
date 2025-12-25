package sandbox

import (
	"context"
	"errors"
	"io"

	"github.com/felixgeelhaar/preflight/internal/domain/capability"
)

// HostFunction represents a function callable by plugins.
type HostFunction struct {
	// Name is the function name exported to WASM
	Name string

	// Module is the WASM module name (e.g., "preflight")
	Module string

	// RequiredCapability needed to call this function
	RequiredCapability capability.Capability

	// Description for documentation
	Description string
}

// HostFunctions defines all functions available to plugins.
var HostFunctions = []HostFunction{
	// File operations
	{
		Name:               "read_file",
		Module:             "preflight",
		RequiredCapability: capability.CapFilesRead,
		Description:        "Read a file from the filesystem",
	},
	{
		Name:               "write_file",
		Module:             "preflight",
		RequiredCapability: capability.CapFilesWrite,
		Description:        "Write a file to the filesystem",
	},
	{
		Name:               "file_exists",
		Module:             "preflight",
		RequiredCapability: capability.CapFilesRead,
		Description:        "Check if a file exists",
	},

	// Package operations
	{
		Name:               "brew_install",
		Module:             "preflight",
		RequiredCapability: capability.CapPackagesBrew,
		Description:        "Install a Homebrew package",
	},
	{
		Name:               "brew_list",
		Module:             "preflight",
		RequiredCapability: capability.CapPackagesBrew,
		Description:        "List installed Homebrew packages",
	},
	{
		Name:               "apt_install",
		Module:             "preflight",
		RequiredCapability: capability.CapPackagesApt,
		Description:        "Install an APT package",
	},

	// Shell operations
	{
		Name:               "shell_exec",
		Module:             "preflight",
		RequiredCapability: capability.CapShellExecute,
		Description:        "Execute a shell command",
	},

	// Network operations
	{
		Name:               "http_get",
		Module:             "preflight",
		RequiredCapability: capability.CapNetworkFetch,
		Description:        "Perform HTTP GET request",
	},
	{
		Name:               "http_post",
		Module:             "preflight",
		RequiredCapability: capability.CapNetworkFetch,
		Description:        "Perform HTTP POST request",
	},

	// Logging (always allowed)
	{
		Name:        "log_info",
		Module:      "preflight",
		Description: "Log an info message",
	},
	{
		Name:        "log_warn",
		Module:      "preflight",
		Description: "Log a warning message",
	},
	{
		Name:        "log_error",
		Module:      "preflight",
		Description: "Log an error message",
	},
}

// HostServices provides implementations for host functions.
type HostServices struct {
	// FileSystem for file operations
	FileSystem FileSystem

	// PackageManager for package operations
	PackageManager PackageManager

	// Shell for command execution
	Shell Shell

	// HTTP for network operations
	HTTP HTTPClient

	// Logger for plugin output
	Logger Logger

	// Policy for capability checks
	Policy *capability.Policy
}

// FileSystem interface for file operations.
type FileSystem interface {
	// ReadFile reads a file
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes a file
	WriteFile(ctx context.Context, path string, data []byte) error

	// Exists checks if a path exists
	Exists(ctx context.Context, path string) (bool, error)

	// Remove removes a file or directory
	Remove(ctx context.Context, path string) error
}

// PackageManager interface for package operations.
type PackageManager interface {
	// Install installs a package
	Install(ctx context.Context, pkg string) error

	// List lists installed packages
	List(ctx context.Context) ([]string, error)

	// IsInstalled checks if a package is installed
	IsInstalled(ctx context.Context, pkg string) (bool, error)
}

// Shell interface for command execution.
type Shell interface {
	// Exec executes a command
	Exec(ctx context.Context, cmd string, args ...string) ([]byte, error)

	// ExecWithInput executes with stdin
	ExecWithInput(ctx context.Context, input io.Reader, cmd string, args ...string) ([]byte, error)
}

// HTTPClient interface for network operations.
type HTTPClient interface {
	// Get performs HTTP GET
	Get(ctx context.Context, url string) ([]byte, int, error)

	// Post performs HTTP POST
	Post(ctx context.Context, url string, contentType string, body []byte) ([]byte, int, error)
}

// Logger interface for plugin logging.
type Logger interface {
	// Info logs an info message
	Info(msg string)

	// Warn logs a warning message
	Warn(msg string)

	// Error logs an error message
	Error(msg string)
}

// CheckCapability verifies a capability is allowed.
func (h *HostServices) CheckCapability(c capability.Capability) error {
	if h.Policy == nil {
		return nil
	}
	return h.Policy.Check(c)
}

// NullFileSystem is a no-op filesystem for full isolation.
type NullFileSystem struct{}

// ReadFile always returns an error.
func (NullFileSystem) ReadFile(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("filesystem access denied")
}

// WriteFile always returns an error.
func (NullFileSystem) WriteFile(_ context.Context, _ string, _ []byte) error {
	return errors.New("filesystem access denied")
}

// Exists always returns false.
func (NullFileSystem) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// Remove always returns an error.
func (NullFileSystem) Remove(_ context.Context, _ string) error {
	return errors.New("filesystem access denied")
}

// NullPackageManager is a no-op package manager.
type NullPackageManager struct{}

// Install always returns an error.
func (NullPackageManager) Install(_ context.Context, _ string) error {
	return errors.New("package installation denied")
}

// List returns an empty list.
func (NullPackageManager) List(_ context.Context) ([]string, error) {
	return nil, nil
}

// IsInstalled always returns false.
func (NullPackageManager) IsInstalled(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// NullShell is a no-op shell.
type NullShell struct{}

// Exec always returns an error.
func (NullShell) Exec(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return nil, errors.New("shell execution denied")
}

// ExecWithInput always returns an error.
func (NullShell) ExecWithInput(_ context.Context, _ io.Reader, _ string, _ ...string) ([]byte, error) {
	return nil, errors.New("shell execution denied")
}

// NullHTTPClient is a no-op HTTP client.
type NullHTTPClient struct{}

// Get always returns an error.
func (NullHTTPClient) Get(_ context.Context, _ string) ([]byte, int, error) {
	return nil, 0, errors.New("network access denied")
}

// Post always returns an error.
func (NullHTTPClient) Post(_ context.Context, _ string, _ string, _ []byte) ([]byte, int, error) {
	return nil, 0, errors.New("network access denied")
}

// NullLogger discards all logs.
type NullLogger struct{}

// Info does nothing.
func (NullLogger) Info(_ string) {}

// Warn does nothing.
func (NullLogger) Warn(_ string) {}

// Error does nothing.
func (NullLogger) Error(_ string) {}

// NewIsolatedServices creates services for full isolation mode.
func NewIsolatedServices(policy *capability.Policy) *HostServices {
	return &HostServices{
		FileSystem:     NullFileSystem{},
		PackageManager: NullPackageManager{},
		Shell:          NullShell{},
		HTTP:           NullHTTPClient{},
		Logger:         NullLogger{},
		Policy:         policy,
	}
}
