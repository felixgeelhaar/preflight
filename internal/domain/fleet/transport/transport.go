// Package transport provides remote execution transports for fleet operations.
package transport

import (
	"context"
	"io"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
)

// CommandResult holds the result of a remote command execution.
type CommandResult struct {
	// ExitCode is the command's exit code.
	ExitCode int
	// Stdout is the standard output.
	Stdout []byte
	// Stderr is the standard error output.
	Stderr []byte
	// Duration is how long the command took.
	Duration time.Duration
}

// Success returns true if the command exited with code 0.
func (r *CommandResult) Success() bool {
	return r.ExitCode == 0
}

// CombinedOutput returns stdout and stderr combined.
func (r *CommandResult) CombinedOutput() []byte {
	result := make([]byte, 0, len(r.Stdout)+len(r.Stderr))
	result = append(result, r.Stdout...)
	result = append(result, r.Stderr...)
	return result
}

// Connection represents an active connection to a remote host.
type Connection interface {
	// Host returns the connected host.
	Host() *fleet.Host

	// Run executes a command and returns the result.
	Run(ctx context.Context, cmd string) (*CommandResult, error)

	// RunWithInput executes a command with stdin input.
	RunWithInput(ctx context.Context, cmd string, stdin io.Reader) (*CommandResult, error)

	// Upload transfers a file to the remote host.
	Upload(ctx context.Context, localPath, remotePath string) error

	// Download transfers a file from the remote host.
	Download(ctx context.Context, remotePath, localPath string) error

	// Close closes the connection.
	Close() error
}

// Transport defines the interface for remote execution transports.
type Transport interface {
	// Name returns the transport name (e.g., "ssh", "local").
	Name() string

	// Connect establishes a connection to a host.
	Connect(ctx context.Context, host *fleet.Host) (Connection, error)

	// Ping tests connectivity to a host without establishing a full connection.
	Ping(ctx context.Context, host *fleet.Host) error
}

// ConnectionPool manages a pool of reusable connections.
type ConnectionPool struct {
	transport   Transport
	connections map[fleet.HostID]Connection
	maxIdle     int
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(transport Transport, maxIdle int) *ConnectionPool {
	if maxIdle <= 0 {
		maxIdle = 10
	}
	return &ConnectionPool{
		transport:   transport,
		connections: make(map[fleet.HostID]Connection),
		maxIdle:     maxIdle,
	}
}

// Get returns a connection for the host, creating one if needed.
func (p *ConnectionPool) Get(ctx context.Context, host *fleet.Host) (Connection, error) {
	if conn, ok := p.connections[host.ID()]; ok {
		return conn, nil
	}

	conn, err := p.transport.Connect(ctx, host)
	if err != nil {
		return nil, err
	}

	if len(p.connections) < p.maxIdle {
		p.connections[host.ID()] = conn
	}

	return conn, nil
}

// Release returns a connection to the pool.
func (p *ConnectionPool) Release(_ Connection) {
	// Connection is already in the pool via Get
}

// Close closes all connections in the pool.
func (p *ConnectionPool) Close() error {
	var lastErr error
	for _, conn := range p.connections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	p.connections = make(map[fleet.HostID]Connection)
	return lastErr
}

// Size returns the number of connections in the pool.
func (p *ConnectionPool) Size() int {
	return len(p.connections)
}
