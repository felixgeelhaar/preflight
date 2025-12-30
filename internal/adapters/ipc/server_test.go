package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAgentProvider implements AgentProvider for testing.
type mockAgentProvider struct {
	status       agent.Status
	stopError    error
	approveError error
	stopCalled   bool
	approveCalls []string
}

func (m *mockAgentProvider) Status() agent.Status {
	return m.status
}

func (m *mockAgentProvider) Stop(_ context.Context) error {
	m.stopCalled = true
	return m.stopError
}

func (m *mockAgentProvider) Approve(requestID string) error {
	m.approveCalls = append(m.approveCalls, requestID)
	return m.approveError
}

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	// Use /tmp directly to avoid long paths (macOS has ~104 byte limit on Unix socket paths)
	socketPath := fmt.Sprintf("/tmp/pf-test-%d.sock", os.Getpid())
	lockPath := fmt.Sprintf("/tmp/pf-test-%d.lock", os.Getpid())

	// Ensure cleanup
	t.Cleanup(func() {
		_ = os.Remove(socketPath)
		_ = os.Remove(lockPath)
	})

	provider := &mockAgentProvider{
		status: agent.Status{
			State:          agent.StateRunning,
			ReconcileCount: 5,
		},
	}

	cfg := ServerConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
		Version:    "1.0.0-test",
	}

	server := NewServer(cfg, provider)
	return server, socketPath
}

func TestNewServer(t *testing.T) {
	provider := &mockAgentProvider{}
	cfg := ServerConfig{
		SocketPath: "/tmp/test.sock",
		LockPath:   "/tmp/test.lock",
		Version:    "1.0.0",
	}

	server := NewServer(cfg, provider)

	assert.Equal(t, "/tmp/test.sock", server.socketPath)
	assert.Equal(t, "/tmp/test.lock", server.lockPath)
	assert.Equal(t, "1.0.0", server.version)
}

func TestNewServer_Defaults(t *testing.T) {
	provider := &mockAgentProvider{}
	cfg := ServerConfig{} // Empty config

	server := NewServer(cfg, provider)

	assert.Contains(t, server.socketPath, ".preflight")
	assert.Contains(t, server.socketPath, "agent.sock")
	assert.Contains(t, server.lockPath, ".preflight")
	assert.Contains(t, server.lockPath, "agent.lock")
}

func TestServer_StartStop(t *testing.T) {
	server, socketPath := newTestServer(t)

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Verify socket exists
	_, err = os.Stat(socketPath)
	require.NoError(t, err)

	// Stop server
	err = server.Stop()
	require.NoError(t, err)

	// Verify socket removed
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err))
}

func TestServer_StartCreatesLockFile(t *testing.T) {
	// Use /tmp for short paths
	socketPath := fmt.Sprintf("/tmp/pf-lock-%d.sock", os.Getpid())
	lockPath := fmt.Sprintf("/tmp/pf-lock-%d.lock", os.Getpid())
	t.Cleanup(func() {
		_ = os.Remove(socketPath)
		_ = os.Remove(lockPath)
	})

	provider := &mockAgentProvider{}
	cfg := ServerConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
	}

	server := NewServer(cfg, provider)
	err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Stop() }()

	// Verify lock file exists with PID
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\n")
}

func TestServer_HandleStatusRequest(t *testing.T) {
	server, socketPath := newTestServer(t)
	err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Stop() }()

	// Connect and send status request
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	msg, _ := NewMessage(MessageTypeStatusRequest, "req-1", StatusRequest{})
	err = json.NewEncoder(conn).Encode(msg)
	require.NoError(t, err)

	// Read response
	var resp Message
	err = json.NewDecoder(conn).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, MessageTypeStatusResponse, resp.Type)
	assert.Equal(t, "req-1", resp.RequestID)

	var statusResp StatusResponse
	err = json.Unmarshal(resp.Payload, &statusResp)
	require.NoError(t, err)

	assert.Equal(t, agent.StateRunning, statusResp.Status.State)
	assert.Equal(t, 5, statusResp.Status.ReconcileCount)
	assert.Equal(t, "1.0.0-test", statusResp.Version)
	assert.Positive(t, statusResp.PID)
}

func TestServer_HandleStopRequest(t *testing.T) {
	// Use /tmp for short paths
	socketPath := fmt.Sprintf("/tmp/pf-stop-%d.sock", os.Getpid())
	lockPath := fmt.Sprintf("/tmp/pf-stop-%d.lock", os.Getpid())
	t.Cleanup(func() {
		_ = os.Remove(socketPath)
		_ = os.Remove(lockPath)
	})

	provider := &mockAgentProvider{}
	cfg := ServerConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
	}

	server := NewServer(cfg, provider)
	err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Stop() }()

	// Connect and send stop request
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	msg, _ := NewMessage(MessageTypeStopRequest, "req-2", StopRequest{Force: true})
	err = json.NewEncoder(conn).Encode(msg)
	require.NoError(t, err)

	// Read response
	var resp Message
	err = json.NewDecoder(conn).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, MessageTypeStopResponse, resp.Type)

	var stopResp StopResponse
	err = json.Unmarshal(resp.Payload, &stopResp)
	require.NoError(t, err)

	assert.True(t, stopResp.Success)
	assert.True(t, provider.stopCalled)
}

func TestServer_HandleApproveRequest(t *testing.T) {
	// Use /tmp for short paths
	socketPath := fmt.Sprintf("/tmp/pf-approve-%d.sock", os.Getpid())
	lockPath := fmt.Sprintf("/tmp/pf-approve-%d.lock", os.Getpid())
	t.Cleanup(func() {
		_ = os.Remove(socketPath)
		_ = os.Remove(lockPath)
	})

	provider := &mockAgentProvider{}
	cfg := ServerConfig{
		SocketPath: socketPath,
		LockPath:   lockPath,
	}

	server := NewServer(cfg, provider)
	err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Stop() }()

	// Connect and send approve request
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	msg, _ := NewMessage(MessageTypeApproveRequest, "req-3", ApproveRequest{RequestID: "approval-1"})
	err = json.NewEncoder(conn).Encode(msg)
	require.NoError(t, err)

	// Read response
	var resp Message
	err = json.NewDecoder(conn).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, MessageTypeApproveResponse, resp.Type)

	var approveResp ApproveResponse
	err = json.Unmarshal(resp.Payload, &approveResp)
	require.NoError(t, err)

	assert.True(t, approveResp.Success)
	assert.Equal(t, "approval-1", approveResp.RequestID)
	assert.Contains(t, provider.approveCalls, "approval-1")
}

func TestServer_HandleUnknownMessageType(t *testing.T) {
	server, socketPath := newTestServer(t)
	err := server.Start()
	require.NoError(t, err)
	defer func() { _ = server.Stop() }()

	// Connect and send unknown message type
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	msg := &Message{
		Type:      MessageType("unknown_type"),
		RequestID: "req-4",
		Timestamp: time.Now(),
	}
	err = json.NewEncoder(conn).Encode(msg)
	require.NoError(t, err)

	// Read response
	var resp Message
	err = json.NewDecoder(conn).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, MessageTypeErrorResponse, resp.Type)

	var errResp ErrorResponse
	err = json.Unmarshal(resp.Payload, &errResp)
	require.NoError(t, err)

	assert.Equal(t, ErrorCodeInvalidRequest, errResp.Code)
	assert.Contains(t, errResp.Message, "unknown message type")
}

func TestServer_DoubleStop(t *testing.T) {
	server, _ := newTestServer(t)
	err := server.Start()
	require.NoError(t, err)

	// First stop
	err = server.Stop()
	require.NoError(t, err)

	// Second stop should not error
	err = server.Stop()
	require.NoError(t, err)
}

func TestServer_StartAfterClose(t *testing.T) {
	server, _ := newTestServer(t)

	err := server.Start()
	require.NoError(t, err)

	err = server.Stop()
	require.NoError(t, err)

	// Cannot start after close
	err = server.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestDefaultSocketPath(t *testing.T) {
	path := DefaultSocketPath()
	assert.Contains(t, path, ".preflight")
	assert.Contains(t, path, "agent.sock")
}

func TestDefaultLockPath(t *testing.T) {
	path := DefaultLockPath()
	assert.Contains(t, path, ".preflight")
	assert.Contains(t, path, "agent.lock")
}
