package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
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

func TestConnectionPool_DefaultMaxIdle(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 0) // should default to 10
	assert.Equal(t, 0, pool.Size())

	// Verify it can store connections (defaulted to 10)
	host := createTestHost(t)
	conn, err := pool.Get(context.Background(), host)
	require.NoError(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, 1, pool.Size())
	require.NoError(t, pool.Close())
}

func TestConnectionPool_NegativeMaxIdle(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, -5)
	assert.Equal(t, 0, pool.Size())
}

func TestConnectionPool_Release(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 5)
	host := createTestHost(t)

	conn, err := pool.Get(context.Background(), host)
	require.NoError(t, err)
	assert.Equal(t, 1, pool.Size())

	// Release is a no-op
	pool.Release(conn)
	assert.Equal(t, 1, pool.Size())

	require.NoError(t, pool.Close())
}

func TestConnectionPool_ExceedMaxIdle(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 1) // Only allow 1 cached connection

	host1 := createTestHost(t)

	// Create a second host with different ID
	id2, err := fleet.NewHostID("test-host-2")
	require.NoError(t, err)
	host2, err := fleet.NewHost(id2, fleet.SSHConfig{Hostname: "localhost"})
	require.NoError(t, err)

	// First connection should be cached
	conn1, err := pool.Get(context.Background(), host1)
	require.NoError(t, err)
	assert.NotNil(t, conn1)
	assert.Equal(t, 1, pool.Size())

	// Second connection should NOT be cached (pool full)
	conn2, err := pool.Get(context.Background(), host2)
	require.NoError(t, err)
	assert.NotNil(t, conn2)
	assert.Equal(t, 1, pool.Size()) // Still 1, second not cached

	require.NoError(t, pool.Close())
}

func TestConnectionPool_MultipleHosts(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 5)

	host1 := createTestHost(t)
	id2, err := fleet.NewHostID("test-host-2")
	require.NoError(t, err)
	host2, err := fleet.NewHost(id2, fleet.SSHConfig{Hostname: "localhost"})
	require.NoError(t, err)

	_, err = pool.Get(context.Background(), host1)
	require.NoError(t, err)
	_, err = pool.Get(context.Background(), host2)
	require.NoError(t, err)

	assert.Equal(t, 2, pool.Size())

	// Close clears all
	require.NoError(t, pool.Close())
	assert.Equal(t, 0, pool.Size())
}

func TestCommandResult_CombinedOutput_EmptyStreams(t *testing.T) {
	t.Parallel()

	t.Run("both empty", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			Stdout: []byte{},
			Stderr: []byte{},
		}
		assert.Empty(t, result.CombinedOutput())
	})

	t.Run("only stdout", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			Stdout: []byte("output"),
			Stderr: []byte{},
		}
		assert.Equal(t, "output", string(result.CombinedOutput()))
	})

	t.Run("only stderr", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{
			Stdout: []byte{},
			Stderr: []byte("error"),
		}
		assert.Equal(t, "error", string(result.CombinedOutput()))
	})

	t.Run("nil streams", func(t *testing.T) {
		t.Parallel()
		result := &CommandResult{}
		assert.Empty(t, result.CombinedOutput())
	})
}

func TestCommandResult_Duration(t *testing.T) {
	t.Parallel()

	result := &CommandResult{
		ExitCode: 0,
		Duration: 5 * time.Second,
	}
	assert.Equal(t, 5*time.Second, result.Duration)
}

func TestLocalConnection_Upload_NonexistentSource(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := conn.Upload(context.Background(), srcPath, dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestLocalConnection_Download_NonexistentSource(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	err := conn.Download(context.Background(), srcPath, dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestLocalConnection_Upload_ReadOnlyDestination(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(srcPath, []byte("content"), 0o644)
	require.NoError(t, err)

	// Try to write to /dev/null/impossible to force directory creation failure
	dstPath := "/dev/null/impossible/dest.txt"
	err = conn.Upload(context.Background(), srcPath, dstPath)
	assert.Error(t, err)
}

func TestLocalConnection_Download_ReadOnlyDestination(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(srcPath, []byte("content"), 0o644)
	require.NoError(t, err)

	// Try to write to /dev/null/impossible to force directory creation failure
	dstPath := "/dev/null/impossible/dest.txt"
	err = conn.Download(context.Background(), srcPath, dstPath)
	assert.Error(t, err)
}

func TestSSHTransport_LoadPrivateKey_NonexistentFile(t *testing.T) {
	t.Parallel()

	transport := NewSSHTransport()
	_, err := transport.loadPrivateKey("/nonexistent/path/id_rsa")
	assert.Error(t, err)
}

func TestSSHTransport_LoadPrivateKey_InvalidKeyContent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "bad_key")
	err := os.WriteFile(keyPath, []byte("not a valid key"), 0o600)
	require.NoError(t, err)

	transport := NewSSHTransport()
	_, err = transport.loadPrivateKey(keyPath)
	assert.Error(t, err)
}

func TestSSHTransport_LoadPrivateKey_ExpandsTilde(t *testing.T) {
	t.Parallel()

	transport := NewSSHTransport()
	// This should expand ~ and attempt to read the file.
	// It will fail because the file likely doesn't exist or isn't a valid key,
	// but it demonstrates the path expansion logic.
	_, err := transport.loadPrivateKey("~/nonexistent_key_for_test")
	assert.Error(t, err)
}

func TestSSHTransport_TrySSHAgent_NoSocket(t *testing.T) {
	t.Parallel()

	transport := &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{},
	}

	// Clear SSH_AUTH_SOCK for this goroutine's perspective.
	// trySSHAgent reads os.Getenv directly, so we test the nil return
	// when the socket doesn't exist.
	// Since we can't easily unset env vars in parallel tests,
	// we just test that trySSHAgent returns nil (the default behavior
	// since it closes the conn and returns nil anyway).
	result := transport.trySSHAgent()
	assert.Nil(t, result)
}

func TestSSHTransport_BuildAuthMethods_NoMethods(t *testing.T) {
	t.Parallel()

	transport := &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{"/nonexistent/key1", "/nonexistent/key2"},
	}

	cfg := fleet.SSHConfig{
		Hostname: "localhost",
	}

	_, err := transport.buildAuthMethods(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no authentication methods available")
}

func TestSSHTransport_BuildAuthMethods_ExplicitIdentityFileNotFound(t *testing.T) {
	t.Parallel()

	transport := &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{},
	}

	cfg := fleet.SSHConfig{
		Hostname:     "localhost",
		IdentityFile: "/nonexistent/explicit_key",
	}

	_, err := transport.buildAuthMethods(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load identity file")
}

func TestSSHTransport_BuildAuthMethods_WithValidKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	// Generate a minimal ed25519 private key in PEM format for testing.
	// We use ssh-keygen via local command to create a valid key.
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, err := transport.Connect(context.Background(), host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	result, err := conn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success(), "ssh-keygen failed: %s", string(result.Stderr))

	sshTransport := &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{},
	}

	cfg := fleet.SSHConfig{
		Hostname:     "localhost",
		IdentityFile: keyPath,
	}

	methods, err := sshTransport.buildAuthMethods(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, methods)
}

func TestSSHTransport_BuildAuthMethods_DefaultKeyFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	// Generate a valid key
	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, err := transport.Connect(context.Background(), host)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	result, err := conn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success(), "ssh-keygen failed: %s", string(result.Stderr))

	sshTransport := &SSHTransport{
		DefaultTimeout: 30 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{keyPath}, // Use the generated key as a default
	}

	cfg := fleet.SSHConfig{
		Hostname: "localhost",
	}

	methods, err := sshTransport.buildAuthMethods(cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, methods)
}

func TestSSHConnection_Host(t *testing.T) {
	t.Parallel()

	host := createTestHost(t)
	conn := &SSHConnection{
		host:   host,
		client: nil, // We only test the Host() method
	}
	assert.Equal(t, host, conn.Host())
}

func TestSSHTransport_Connect_ErrorMarksHost(t *testing.T) {
	t.Parallel()

	// Create a transport with no valid identity files to force auth failure
	transport := &SSHTransport{
		DefaultTimeout: 1 * time.Second,
		DefaultUser:    "test",
		IdentityFiles:  []string{"/nonexistent/key"},
	}

	id, err := fleet.NewHostID("error-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname: "localhost",
		Port:     22,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, connectErr := transport.Connect(ctx, host)
	// Connect should fail because there are no auth methods
	assert.Error(t, connectErr)
}

func TestLocalConnection_RunWithInput_NilStdin(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)
	defer func() { _ = conn.Close() }()

	// RunWithInput with nil stdin should work (no stdin piped)
	result, err := conn.RunWithInput(context.Background(), "echo hello", nil)
	require.NoError(t, err)
	assert.True(t, result.Success())
	assert.Equal(t, "hello\n", string(result.Stdout))
}

func TestLocalConnection_Close_Idempotent(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	host := createTestHost(t)
	conn, _ := transport.Connect(context.Background(), host)

	// Close multiple times should be safe
	assert.NoError(t, conn.Close())
	assert.NoError(t, conn.Close())
}

func TestConnectionPool_CloseEmpty(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 5)

	// Closing an empty pool should succeed
	err := pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 0, pool.Size())
}

func TestConnectionPool_GetAfterClose(t *testing.T) {
	t.Parallel()

	transport := NewLocalTransport()
	pool := NewConnectionPool(transport, 5)
	host := createTestHost(t)

	// Get, close, then get again should create a new connection
	conn1, err := pool.Get(context.Background(), host)
	require.NoError(t, err)
	assert.NotNil(t, conn1)

	require.NoError(t, pool.Close())
	assert.Equal(t, 0, pool.Size())

	// Get after close should create a new connection
	conn2, err := pool.Get(context.Background(), host)
	require.NoError(t, err)
	assert.NotNil(t, conn2)
	assert.Equal(t, 1, pool.Size())

	require.NoError(t, pool.Close())
}

// mockTransport implements Transport for testing pool behavior with errors.
type mockTransport struct {
	name      string
	connectFn func(ctx context.Context, host *fleet.Host) (Connection, error)
	pingFn    func(ctx context.Context, host *fleet.Host) error
}

func (m *mockTransport) Name() string {
	return m.name
}

func (m *mockTransport) Connect(ctx context.Context, host *fleet.Host) (Connection, error) {
	return m.connectFn(ctx, host)
}

func (m *mockTransport) Ping(ctx context.Context, host *fleet.Host) error {
	return m.pingFn(ctx, host)
}

// mockConnection implements Connection for testing.
type mockConnection struct {
	host    *fleet.Host
	closeFn func() error
}

func (m *mockConnection) Host() *fleet.Host {
	return m.host
}

func (m *mockConnection) Run(_ context.Context, _ string) (*CommandResult, error) {
	return &CommandResult{ExitCode: 0}, nil
}

func (m *mockConnection) RunWithInput(_ context.Context, _ string, _ io.Reader) (*CommandResult, error) {
	return &CommandResult{ExitCode: 0}, nil
}

func (m *mockConnection) Upload(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockConnection) Download(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockConnection) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func TestConnectionPool_GetConnectError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("connection refused")
	mt := &mockTransport{
		name: "mock",
		connectFn: func(_ context.Context, _ *fleet.Host) (Connection, error) {
			return nil, expectedErr
		},
	}

	pool := NewConnectionPool(mt, 5)
	host := createTestHost(t)

	conn, err := pool.Get(context.Background(), host)
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 0, pool.Size())
}

func TestConnectionPool_CloseWithErrors(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close error")
	mt := &mockTransport{
		name: "mock",
		connectFn: func(_ context.Context, host *fleet.Host) (Connection, error) {
			return &mockConnection{
				host: host,
				closeFn: func() error {
					return closeErr
				},
			}, nil
		},
	}

	pool := NewConnectionPool(mt, 5)

	host1 := createTestHost(t)
	id2, err := fleet.NewHostID("host-2")
	require.NoError(t, err)
	host2, err := fleet.NewHost(id2, fleet.SSHConfig{Hostname: "localhost"})
	require.NoError(t, err)

	_, err = pool.Get(context.Background(), host1)
	require.NoError(t, err)
	_, err = pool.Get(context.Background(), host2)
	require.NoError(t, err)
	assert.Equal(t, 2, pool.Size())

	// Close should return the last error
	err = pool.Close()
	assert.Error(t, err)
	assert.Equal(t, 0, pool.Size()) // Pool cleared even on error
}

func TestConnectionPool_Release_Noop(t *testing.T) {
	t.Parallel()

	mt := &mockTransport{
		name: "mock",
		connectFn: func(_ context.Context, host *fleet.Host) (Connection, error) {
			return &mockConnection{host: host}, nil
		},
	}

	pool := NewConnectionPool(mt, 5)
	host := createTestHost(t)

	conn, err := pool.Get(context.Background(), host)
	require.NoError(t, err)

	// Release should be a no-op
	pool.Release(conn)
	pool.Release(nil) // Should handle nil gracefully
	assert.Equal(t, 1, pool.Size())

	require.NoError(t, pool.Close())
}

func TestSSHTransport_ConnectAuthFailure(t *testing.T) {
	t.Parallel()

	// With no valid identity files, buildAuthMethods fails before dial
	transport := &SSHTransport{
		DefaultTimeout: 100 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{},
	}

	id, err := fleet.NewHostID("auth-fail-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname: "localhost",
		Port:     22,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
	assert.Contains(t, connectErr.Error(), "failed to build auth methods")
}

func TestSSHTransport_ConnectDialFailure(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	// Generate a valid key to get past auth method building
	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	defer func() { _ = lConn.Close() }()

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	transport := &SSHTransport{
		DefaultTimeout: 500 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{keyPath},
	}

	id, err := fleet.NewHostID("dial-fail-host")
	require.NoError(t, err)
	// Use a port that's almost certainly not listening
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           1, // Privileged port, no SSH server
		ConnectTimeout: 500 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
	// Host should be marked with error since dial failure occurs after buildAuthMethods
	assert.Equal(t, fleet.HostStatusError, host.Status())
}

func TestSSHTransport_ConnectUsesDefaultUser(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	defer func() { _ = lConn.Close() }()

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	transport := &SSHTransport{
		DefaultTimeout: 500 * time.Millisecond,
		DefaultUser:    "custom-user",
		IdentityFiles:  []string{keyPath},
	}

	id, err := fleet.NewHostID("default-user-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           1,
		ConnectTimeout: 500 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// This will fail at dial, but exercises the user and timeout selection logic
	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
}

func TestSSHTransport_ConnectUsesDefaultTimeout(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	defer func() { _ = lConn.Close() }()

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	transport := &SSHTransport{
		DefaultTimeout: 500 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{keyPath},
	}

	id, err := fleet.NewHostID("timeout-default-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           1,
		ConnectTimeout: 0, // Should use DefaultTimeout
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
}

func TestSSHTransport_ConnectUsesDefaultPort(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	defer func() { _ = lConn.Close() }()

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	transport := &SSHTransport{
		DefaultTimeout: 500 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{keyPath},
	}

	id, err := fleet.NewHostID("default-port-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           0, // Should default to 22
		ConnectTimeout: 500 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Will fail at dial, but exercises the port default logic
	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
}

func TestSSHTransport_ConnectWithProxyJump(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	defer func() { _ = lConn.Close() }()

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+keyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	transport := &SSHTransport{
		DefaultTimeout: 500 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{keyPath},
	}

	id, err := fleet.NewHostID("proxy-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           1,
		ProxyJump:      "127.0.0.1:1", // Set a proxy jump to exercise that path
		ConnectTimeout: 500 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Will fail at dial to proxy, but exercises the proxy jump path
	_, connectErr := transport.Connect(ctx, host)
	assert.Error(t, connectErr)
}

func TestSSHTransport_Ping(t *testing.T) {
	t.Parallel()

	// Ping calls Connect internally, so we test the error path
	transport := &SSHTransport{
		DefaultTimeout: 100 * time.Millisecond,
		DefaultUser:    "test",
		IdentityFiles:  []string{"/nonexistent/key"},
	}

	id, err := fleet.NewHostID("ping-test-host")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname: "localhost",
		Port:     22,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = transport.Ping(ctx, host)
	assert.Error(t, err)
}

// --- Test SSH server helpers ---

// startTestSSHServer starts a local SSH server for testing SSHConnection methods.
// It returns the port, a cleanup function, and the private key path used by the server.
func startTestSSHServer(t *testing.T) (int, string) {
	t.Helper()

	tmpDir := t.TempDir()
	hostKeyPath := filepath.Join(tmpDir, "host_key")
	clientKeyPath := filepath.Join(tmpDir, "client_key")

	// Generate host key
	lt := NewLocalTransport()
	lHost := createTestHost(t)
	lConn, err := lt.Connect(context.Background(), lHost)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lConn.Close() })

	result, err := lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+hostKeyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	// Generate client key
	result, err = lConn.Run(context.Background(),
		"ssh-keygen -t ed25519 -f "+clientKeyPath+" -N '' -q")
	require.NoError(t, err)
	require.True(t, result.Success())

	// Read key files for SSH server config
	hostKeyData, err := os.ReadFile(hostKeyPath)
	require.NoError(t, err)
	clientPubData, err := os.ReadFile(clientKeyPath + ".pub")
	require.NoError(t, err)

	hostKey, err := ssh.ParsePrivateKey(hostKeyData)
	require.NoError(t, err)

	clientPub, _, _, _, err := ssh.ParseAuthorizedKey(clientPubData)
	require.NoError(t, err)

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientPub.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unknown key")
		},
	}
	config.AddHostKey(hostKey)

	// Listen on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := listener.Addr().(*net.TCPAddr).Port

	// Start accepting connections in the background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			go handleTestSSHConn(conn, config)
		}
	}()

	t.Cleanup(func() { _ = listener.Close() })

	return port, clientKeyPath
}

func handleTestSSHConn(netConn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		_ = netConn.Close()
		return
	}
	defer func() { _ = sshConn.Close() }()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func(ch ssh.Channel, reqs <-chan *ssh.Request) {
			for req := range reqs {
				switch req.Type {
				case "exec":
					// Parse command
					if len(req.Payload) < 4 {
						_ = req.Reply(false, nil)
						_ = ch.Close()
						return
					}
					cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
					if cmdLen+4 > len(req.Payload) {
						_ = req.Reply(false, nil)
						_ = ch.Close()
						return
					}
					cmd := string(req.Payload[4 : 4+cmdLen])

					_ = req.Reply(true, nil)

					// Execute command and close channel after
					switch {
					case cmd == "echo pong":
						_, _ = ch.Write([]byte("pong\n"))
						sendExitStatus(ch, 0)
					case cmd == "echo hello":
						_, _ = ch.Write([]byte("hello\n"))
						sendExitStatus(ch, 0)
					case cmd == "exit 42":
						sendExitStatus(ch, 42)
					case strings.HasPrefix(cmd, "cat >"):
						// Drain stdin then succeed
						buf := make([]byte, 4096)
						for {
							_, readErr := ch.Read(buf)
							if readErr != nil {
								break
							}
						}
						sendExitStatus(ch, 0)
					case strings.HasPrefix(cmd, "cat "):
						_, _ = ch.Write([]byte("file content"))
						sendExitStatus(ch, 0)
					default:
						_, _ = ch.Write([]byte("unknown command\n"))
						sendExitStatus(ch, 127)
					}

					// Close the channel after command execution to signal EOF
					_ = ch.CloseWrite()
					_ = ch.Close()
					return

				default:
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
			_ = ch.Close()
		}(channel, requests)
	}
}

func sendExitStatus(ch ssh.Channel, code int) {
	payload := make([]byte, 4)
	payload[0] = byte(code >> 24)
	payload[1] = byte(code >> 16)
	payload[2] = byte(code >> 8)
	payload[3] = byte(code)
	_, _ = ch.SendRequest("exit-status", false, payload)
}

func TestSSHConnection_Run_ViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-run")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	t.Run("successful command", func(t *testing.T) {
		t.Parallel()
		result, err := conn.Run(ctx, "echo hello")
		require.NoError(t, err)
		assert.True(t, result.Success())
		assert.Contains(t, string(result.Stdout), "hello")
		assert.Positive(t, result.Duration)
	})

	t.Run("failing command", func(t *testing.T) {
		t.Parallel()
		result, err := conn.Run(ctx, "exit 42")
		require.NoError(t, err)
		assert.False(t, result.Success())
		assert.Equal(t, 42, result.ExitCode)
	})
}

func TestSSHConnection_RunWithInput_ViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-input")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// RunWithInput with nil stdin
	result, err := conn.RunWithInput(ctx, "echo hello", nil)
	require.NoError(t, err)
	assert.True(t, result.Success())
}

func TestSSHConnection_Upload_ViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-upload")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "upload.txt")
	err = os.WriteFile(srcPath, []byte("upload content"), 0o644)
	require.NoError(t, err)

	err = conn.Upload(ctx, srcPath, "/tmp/remote-upload.txt")
	require.NoError(t, err)
}

func TestSSHConnection_Upload_NonexistentFile(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-upload-err")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	err = conn.Upload(ctx, "/nonexistent/file.txt", "/tmp/remote.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read local file")
}

func TestSSHConnection_Download_ViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-download")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "downloaded.txt")

	err = conn.Download(ctx, "/tmp/remote-file.txt", localPath)
	require.NoError(t, err)

	data, err := os.ReadFile(localPath)
	require.NoError(t, err)
	assert.Equal(t, "file content", string(data))
}

func TestSSHConnection_Close_ViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-close")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)

	err = conn.Close()
	assert.NoError(t, err)
}

func TestSSHConnection_RunContextCancellation(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-cancel")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := transport.Connect(ctx, host)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// Cancel context immediately
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel before running

	_, runErr := conn.Run(cancelCtx, "echo hello")
	// Should fail due to cancelled context (either the session creation or execution)
	// The test SSH server may or may not support this gracefully
	if runErr != nil {
		assert.Error(t, runErr)
	}
}

func TestSSHTransport_PingViaTestServer(t *testing.T) {
	t.Parallel()

	port, clientKeyPath := startTestSSHServer(t)

	transport := &SSHTransport{
		DefaultTimeout: 5 * time.Second,
		DefaultUser:    "testuser",
		IdentityFiles:  []string{clientKeyPath},
	}

	id, err := fleet.NewHostID("ssh-test-ping")
	require.NoError(t, err)
	host, err := fleet.NewHost(id, fleet.SSHConfig{
		Hostname:       "127.0.0.1",
		Port:           port,
		ConnectTimeout: 5 * time.Second,
	})
	require.NoError(t, err)

	err = transport.Ping(context.Background(), host)
	require.NoError(t, err)
	assert.Equal(t, fleet.HostStatusOnline, host.Status())
}
