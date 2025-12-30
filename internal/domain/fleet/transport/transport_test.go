package transport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestHost(t *testing.T) *fleet.Host {
	t.Helper()
	id, _ := fleet.NewHostID("test-host")
	host, _ := fleet.NewHost(id, fleet.SSHConfig{Hostname: "localhost"})
	return host
}

func TestCommandResult(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			ExitCode: 0,
			Stdout:   []byte("output"),
			Stderr:   []byte(""),
		}
		assert.True(t, result.Success())
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			ExitCode: 1,
			Stdout:   []byte(""),
			Stderr:   []byte("error"),
		}
		assert.False(t, result.Success())
	})

	t.Run("combined output", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			Stdout: []byte("out"),
			Stderr: []byte("err"),
		}
		combined := result.CombinedOutput()
		assert.Equal(t, "outerr", string(combined))
	})
}

func TestLocalTransport_Name(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	assert.Equal(t, "local", transport.Name())
}

func TestLocalTransport_Connect(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)

	conn, err := transport.Connect(context.Background(), host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	assert.Equal(t, host, conn.Host())
	assert.Equal(t, fleet.HostStatusOnline, host.Status())
}

func TestLocalTransport_Ping(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)

	err := transport.Ping(context.Background(), host)
	require.NoError(t, err)
	assert.Equal(t, fleet.HostStatusOnline, host.Status())
}

func TestLocalConnection_Run(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	t.Cleanup(func() { _ = conn.Close() })

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()
		result, err := conn.Run(context.Background(), "echo hello")
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Equal(t, "hello\n", string(result.Stdout))
		assert.Positive(t, result.Duration)
	})

	t.Run("failing command", func(t *testing.T) {
		t.Parallel()
		result, err := conn.Run(context.Background(), "exit 42")
		require.NoError(t, err)
		assert.False(t, result.Success())
		assert.Equal(t, 42, result.ExitCode)
	})

	t.Run("command with stderr", func(t *testing.T) {
		t.Parallel()
		result, err := conn.Run(context.Background(), "echo error >&2")
		require.NoError(t, err)
		assert.Equal(t, "error\n", string(result.Stderr))
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		// Start a long-running command
		done := make(chan struct{})
		var runErr error
		go func() {
			_, runErr = conn.Run(ctx, "sleep 10")
			close(done)
		}()

		// Cancel immediately
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait for command to finish
		<-done
		assert.Error(t, runErr)
	})
}

func TestLocalConnection_RunWithInput(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	result, err := conn.RunWithInput(context.Background(), "cat", strings.NewReader("hello"))
	require.NoError(t, err)
	assert.True(t, result.Success())
	assert.Equal(t, "hello", string(result.Stdout))
}

func TestLocalConnection_Upload(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")

	// Create source file
	err := os.WriteFile(srcPath, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Upload
	err = conn.Upload(context.Background(), srcPath, dstPath)
	require.NoError(t, err)

	// Verify
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

func TestLocalConnection_Download(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")

	// Create source file
	err := os.WriteFile(srcPath, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Download
	err = conn.Download(context.Background(), srcPath, dstPath)
	require.NoError(t, err)

	// Verify
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(data))
}

//nolint:tparallel // Subtests share state and must run sequentially
func TestConnectionPool(t *testing.T) {
	t.Parallel()
	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 5)

	host := createTestHost(t)

	t.Run("get creates connection", func(t *testing.T) {
		conn, err := pool.Get(context.Background(), host)
		require.NoError(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, 1, pool.Size())
	})

	t.Run("get returns cached connection", func(t *testing.T) {
		conn1, _ := pool.Get(context.Background(), host)
		conn2, _ := pool.Get(context.Background(), host)
		assert.Equal(t, conn1, conn2)
		assert.Equal(t, 1, pool.Size())
	})

	t.Run("close clears pool", func(t *testing.T) {
		err := pool.Close()
		require.NoError(t, err)
		assert.Equal(t, 0, pool.Size())
	})
}

func TestNewSSHTransport(t *testing.T) {
	t.Parallel()
	transport := NewSSHTransport()

	assert.Equal(t, "ssh", transport.Name())
	assert.NotZero(t, transport.DefaultTimeout)
	assert.NotEmpty(t, transport.IdentityFiles)
}
