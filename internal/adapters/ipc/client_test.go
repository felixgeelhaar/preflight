package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClientWithServer(t *testing.T) (*Client, func()) {
	t.Helper()

	// Use /tmp directly to avoid long paths (macOS has ~104 byte limit on Unix socket paths)
	socketPath := fmt.Sprintf("/tmp/pf-client-%d.sock", os.Getpid())
	lockPath := fmt.Sprintf("/tmp/pf-client-%d.lock", os.Getpid())

	provider := &mockAgentProvider{
		status: agent.Status{
			State:          agent.StateRunning,
			ReconcileCount: 10,
			Health: agent.HealthStatus{
				Status: agent.HealthHealthy,
			},
		},
	}

	serverCfg := ServerConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
		Version:    "1.0.0-test",
	}

	server := NewServer(serverCfg, provider)
	err := server.Start()
	require.NoError(t, err)

	clientCfg := ClientConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
		Timeout:    5 * time.Second,
	}

	client := NewClient(clientCfg)

	cleanup := func() {
		_ = server.Stop()
		_ = os.Remove(socketPath)
		_ = os.Remove(lockPath)
	}

	return client, cleanup
}

func TestNewClient(t *testing.T) {
	cfg := ClientConfig{
		SocketPath: "/tmp/test.sock",
		LockPath:   "/tmp/test.lock",
		Timeout:    10 * time.Second,
	}

	client := NewClient(cfg)

	assert.Equal(t, "/tmp/test.sock", client.socketPath)
	assert.Equal(t, "/tmp/test.lock", client.lockPath)
	assert.Equal(t, 10*time.Second, client.timeout)
}

func TestNewClient_Defaults(t *testing.T) {
	cfg := ClientConfig{} // Empty config

	client := NewClient(cfg)

	assert.Contains(t, client.socketPath, ".preflight")
	assert.Contains(t, client.lockPath, ".preflight")
	assert.Equal(t, 30*time.Second, client.timeout)
}

func TestClient_IsAgentRunning(t *testing.T) {
	t.Run("returns true when agent is running", func(t *testing.T) {
		client, cleanup := newTestClientWithServer(t)
		defer cleanup()

		assert.True(t, client.IsAgentRunning())
	})

	t.Run("returns false when no lock file", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewClient(ClientConfig{
			SocketPath: filepath.Join(tmpDir, "agent.sock"),
			LockPath:   filepath.Join(tmpDir, "agent.lock"),
		})

		assert.False(t, client.IsAgentRunning())
	})

	t.Run("returns false when lock file has invalid PID", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "agent.lock")
		socketPath := filepath.Join(tmpDir, "agent.sock")

		// Create lock file with invalid PID
		_ = os.WriteFile(lockPath, []byte("invalid\n"), 0o600)

		client := NewClient(ClientConfig{
			SocketPath: socketPath,
			LockPath:   lockPath,
		})

		assert.False(t, client.IsAgentRunning())
	})
}

func TestClient_GetAgentPID(t *testing.T) {
	t.Run("returns PID when agent is running", func(t *testing.T) {
		client, cleanup := newTestClientWithServer(t)
		defer cleanup()

		pid := client.GetAgentPID()
		assert.Positive(t, pid)
	})

	t.Run("returns 0 when no lock file", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewClient(ClientConfig{
			SocketPath: filepath.Join(tmpDir, "agent.sock"),
			LockPath:   filepath.Join(tmpDir, "agent.lock"),
		})

		pid := client.GetAgentPID()
		assert.Zero(t, pid)
	})
}

func TestClient_Status(t *testing.T) {
	t.Run("returns status when agent is running", func(t *testing.T) {
		client, cleanup := newTestClientWithServer(t)
		defer cleanup()

		status, err := client.Status()
		require.NoError(t, err)

		assert.Equal(t, agent.StateRunning, status.Status.State)
		assert.Equal(t, 10, status.Status.ReconcileCount)
		assert.Equal(t, "1.0.0-test", status.Version)
		assert.Positive(t, status.PID)
	})

	t.Run("returns error when agent is not running", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewClient(ClientConfig{
			SocketPath: filepath.Join(tmpDir, "agent.sock"),
			LockPath:   filepath.Join(tmpDir, "agent.lock"),
		})

		_, err := client.Status()
		assert.ErrorIs(t, err, ErrAgentNotRunning)
	})
}

func TestClient_Stop(t *testing.T) {
	t.Run("stops agent successfully", func(t *testing.T) {
		client, cleanup := newTestClientWithServer(t)
		defer cleanup()

		resp, err := client.Stop(false, 10*time.Second)
		require.NoError(t, err)

		assert.True(t, resp.Success)
	})

	t.Run("returns error when agent is not running", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewClient(ClientConfig{
			SocketPath: filepath.Join(tmpDir, "agent.sock"),
			LockPath:   filepath.Join(tmpDir, "agent.lock"),
		})

		_, err := client.Stop(false, 10*time.Second)
		assert.ErrorIs(t, err, ErrAgentNotRunning)
	})
}

func TestClient_Approve(t *testing.T) {
	t.Run("approves request successfully", func(t *testing.T) {
		client, cleanup := newTestClientWithServer(t)
		defer cleanup()

		resp, err := client.Approve("test-request-id")
		require.NoError(t, err)

		assert.True(t, resp.Success)
		assert.Equal(t, "test-request-id", resp.RequestID)
	})

	t.Run("returns error when agent is not running", func(t *testing.T) {
		tmpDir := t.TempDir()
		client := NewClient(ClientConfig{
			SocketPath: filepath.Join(tmpDir, "agent.sock"),
			LockPath:   filepath.Join(tmpDir, "agent.lock"),
		})

		_, err := client.Approve("test-request-id")
		assert.ErrorIs(t, err, ErrAgentNotRunning)
	})
}

func TestClient_ConnectionTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "agent.sock")
	lockPath := filepath.Join(tmpDir, "agent.lock")

	// Create lock file but no server
	_ = os.WriteFile(lockPath, []byte("99999\n"), 0o600)

	client := NewClient(ClientConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
		Timeout:    100 * time.Millisecond,
	})

	// This should return ErrAgentNotRunning because the process doesn't exist
	_, err := client.Status()
	assert.ErrorIs(t, err, ErrAgentNotRunning)
}
