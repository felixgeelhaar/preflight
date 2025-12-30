package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrAgentNotRunning indicates the agent is not running.
var ErrAgentNotRunning = errors.New("agent is not running")

// Client communicates with the agent via IPC.
type Client struct {
	socketPath string
	lockPath   string
	timeout    time.Duration
}

// ClientConfig contains configuration for the IPC client.
type ClientConfig struct {
	SocketPath string
	LockPath   string
	Timeout    time.Duration
}

// NewClient creates a new IPC client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath()
	}
	if cfg.LockPath == "" {
		cfg.LockPath = DefaultLockPath()
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		socketPath: cfg.SocketPath,
		lockPath:   cfg.LockPath,
		timeout:    cfg.Timeout,
	}
}

// IsAgentRunning checks if the agent is currently running.
func (c *Client) IsAgentRunning() bool {
	// Check lock file exists
	if _, err := os.Stat(c.lockPath); err != nil {
		return false
	}

	// Check socket exists
	if _, err := os.Stat(c.socketPath); err != nil {
		return false
	}

	// Try to connect to verify the server is actually listening
	conn, err := net.DialTimeout("unix", c.socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close() // Best effort cleanup

	return true
}

// GetAgentPID returns the PID of the running agent, or 0 if not running.
func (c *Client) GetAgentPID() int {
	data, err := os.ReadFile(c.lockPath)
	if err != nil {
		return 0
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0
	}

	return pid
}

// Status requests the agent status.
func (c *Client) Status() (*StatusResponse, error) {
	if !c.IsAgentRunning() {
		return nil, ErrAgentNotRunning
	}

	req := StatusRequest{}
	resp, err := c.sendRequest(MessageTypeStatusRequest, req)
	if err != nil {
		return nil, err
	}

	if resp.Type == MessageTypeErrorResponse {
		var errResp ErrorResponse
		if err := json.Unmarshal(resp.Payload, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(resp.Payload, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	return &statusResp, nil
}

// Stop requests the agent to stop.
func (c *Client) Stop(force bool, timeout time.Duration) (*StopResponse, error) {
	if !c.IsAgentRunning() {
		return nil, ErrAgentNotRunning
	}

	req := StopRequest{
		Force:          force,
		TimeoutSeconds: int(timeout.Seconds()),
	}

	resp, err := c.sendRequest(MessageTypeStopRequest, req)
	if err != nil {
		return nil, err
	}

	if resp.Type == MessageTypeErrorResponse {
		var errResp ErrorResponse
		if err := json.Unmarshal(resp.Payload, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
	}

	var stopResp StopResponse
	if err := json.Unmarshal(resp.Payload, &stopResp); err != nil {
		return nil, fmt.Errorf("failed to parse stop response: %w", err)
	}

	return &stopResp, nil
}

// Approve sends an approval for a pending request.
func (c *Client) Approve(requestID string) (*ApproveResponse, error) {
	if !c.IsAgentRunning() {
		return nil, ErrAgentNotRunning
	}

	req := ApproveRequest{
		RequestID: requestID,
	}

	resp, err := c.sendRequest(MessageTypeApproveRequest, req)
	if err != nil {
		return nil, err
	}

	if resp.Type == MessageTypeErrorResponse {
		var errResp ErrorResponse
		if err := json.Unmarshal(resp.Payload, &errResp); err != nil {
			return nil, fmt.Errorf("failed to parse error response: %w", err)
		}
		return nil, fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
	}

	var approveResp ApproveResponse
	if err := json.Unmarshal(resp.Payload, &approveResp); err != nil {
		return nil, fmt.Errorf("failed to parse approve response: %w", err)
	}

	return &approveResp, nil
}

// sendRequest sends a request and waits for a response.
func (c *Client) sendRequest(msgType MessageType, payload interface{}) (*Message, error) {
	// Connect to socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadlines
	_ = conn.SetDeadline(time.Now().Add(c.timeout))

	// Create request message
	requestID := uuid.New().String()
	msg, err := NewMessage(msgType, requestID, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Send request
	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var resp Message
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}
