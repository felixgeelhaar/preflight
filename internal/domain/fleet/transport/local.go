package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
)

// LocalTransport implements Transport for local execution.
// This is useful for testing and for single-machine scenarios.
type LocalTransport struct{}

// NewLocalTransport creates a new local transport.
func NewLocalTransport() *LocalTransport {
	return &LocalTransport{}
}

// Name returns "local".
func (t *LocalTransport) Name() string {
	return "local"
}

// Connect returns a local connection (no actual connection needed).
func (t *LocalTransport) Connect(_ context.Context, host *fleet.Host) (Connection, error) {
	host.MarkOnline()
	return &LocalConnection{host: host}, nil
}

// Ping always succeeds for local.
func (t *LocalTransport) Ping(_ context.Context, host *fleet.Host) error {
	host.MarkOnline()
	return nil
}

// LocalConnection implements Connection for local execution.
type LocalConnection struct {
	host *fleet.Host
}

// Host returns the host.
func (c *LocalConnection) Host() *fleet.Host {
	return c.host
}

// Run executes a command locally.
func (c *LocalConnection) Run(ctx context.Context, cmdStr string) (*CommandResult, error) {
	return c.RunWithInput(ctx, cmdStr, nil)
}

// RunWithInput executes a command with stdin.
func (c *LocalConnection) RunWithInput(ctx context.Context, cmdStr string, stdin io.Reader) (*CommandResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdin != nil {
		cmd.Stdin = stdin
	}

	err := cmd.Run()
	duration := time.Since(start)

	result := &CommandResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		Duration: duration,
	}

	if err != nil {
		// Check context cancellation first
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	return result, nil
}

// Upload copies a file locally (for testing).
func (c *LocalConnection) Upload(_ context.Context, localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(remotePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(remotePath, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// Download copies a file locally (for testing).
func (c *LocalConnection) Download(_ context.Context, remotePath, localPath string) error {
	data, err := os.ReadFile(remotePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// Close is a no-op for local connections.
func (c *LocalConnection) Close() error {
	return nil
}
