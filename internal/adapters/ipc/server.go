package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
)

// AgentProvider provides access to agent operations.
type AgentProvider interface {
	Status() agent.Status
	Stop(ctx context.Context) error
	Approve(requestID string) error
}

// Server handles IPC communication via Unix socket.
type Server struct {
	socketPath string
	lockPath   string
	provider   AgentProvider
	version    string

	listener net.Listener
	mu       sync.RWMutex
	closed   bool
	wg       sync.WaitGroup
}

// ServerConfig contains configuration for the IPC server.
type ServerConfig struct {
	SocketPath string
	LockPath   string
	Version    string
}

// DefaultSocketPath returns the default socket path.
func DefaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".preflight", "agent.sock")
}

// DefaultLockPath returns the default lock file path.
func DefaultLockPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".preflight", "agent.lock")
}

// NewServer creates a new IPC server.
func NewServer(cfg ServerConfig, provider AgentProvider) *Server {
	if cfg.SocketPath == "" {
		cfg.SocketPath = DefaultSocketPath()
	}
	if cfg.LockPath == "" {
		cfg.LockPath = DefaultLockPath()
	}

	return &Server{
		socketPath: cfg.SocketPath,
		lockPath:   cfg.LockPath,
		provider:   provider,
		version:    cfg.Version,
	}
}

// Start begins listening for connections.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("server is closed")
	}

	// Ensure directory exists
	dir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove stale socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove stale socket: %w", err)
	}

	// Create lock file
	if err := s.createLockFile(); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	// Start listening
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		s.removeLockFile()
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = listener.Close() // Cleanup on error
		s.removeLockFile()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true

	if s.listener != nil {
		_ = s.listener.Close() // Best effort close
	}
	s.mu.Unlock()

	// Wait for connections to finish (outside of lock to avoid deadlock)
	s.wg.Wait()

	// Clean up socket and lock files
	_ = os.RemoveAll(s.socketPath) // Best effort cleanup
	s.removeLockFile()

	return nil
}

// SocketPath returns the socket path.
func (s *Server) SocketPath() string {
	return s.socketPath
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.RLock()
			closed := s.closed
			s.mu.RUnlock()

			if closed {
				return
			}
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() { _ = conn.Close() }()

	// Set read deadline
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read message
	decoder := json.NewDecoder(conn)
	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		if err != io.EOF {
			s.sendError(conn, msg.RequestID, "failed to decode message")
		}
		return
	}

	// Handle message
	s.handleMessage(conn, &msg)
}

// handleMessage routes the message to the appropriate handler.
func (s *Server) handleMessage(conn net.Conn, msg *Message) {
	switch msg.Type {
	case MessageTypeStatusRequest:
		s.handleStatusRequest(conn, msg)
	case MessageTypeStopRequest:
		s.handleStopRequest(conn, msg)
	case MessageTypeApproveRequest:
		s.handleApproveRequest(conn, msg)
	default:
		s.sendError(conn, msg.RequestID, "unknown message type")
	}
}

// handleStatusRequest handles a status request.
func (s *Server) handleStatusRequest(conn net.Conn, msg *Message) {
	status := s.provider.Status()

	response := StatusResponse{
		Status:  status,
		Version: s.version,
		PID:     os.Getpid(),
	}

	s.sendResponse(conn, msg.RequestID, MessageTypeStatusResponse, response)
}

// handleStopRequest handles a stop request.
func (s *Server) handleStopRequest(conn net.Conn, msg *Message) {
	var req StopRequest
	if msg.Payload != nil {
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			s.sendError(conn, msg.RequestID, "invalid stop request payload")
			return
		}
	}

	timeout := 30 * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := s.provider.Stop(ctx)
	if err != nil {
		s.sendResponse(conn, msg.RequestID, MessageTypeStopResponse, StopResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	s.sendResponse(conn, msg.RequestID, MessageTypeStopResponse, StopResponse{
		Success: true,
		Message: "agent stopped successfully",
	})
}

// handleApproveRequest handles an approval request.
func (s *Server) handleApproveRequest(conn net.Conn, msg *Message) {
	var req ApproveRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		s.sendError(conn, msg.RequestID, "invalid approve request payload")
		return
	}

	if req.RequestID == "" {
		s.sendError(conn, msg.RequestID, "request_id is required")
		return
	}

	err := s.provider.Approve(req.RequestID)
	if err != nil {
		s.sendResponse(conn, msg.RequestID, MessageTypeApproveResponse, ApproveResponse{
			Success:   false,
			RequestID: req.RequestID,
			Message:   err.Error(),
		})
		return
	}

	s.sendResponse(conn, msg.RequestID, MessageTypeApproveResponse, ApproveResponse{
		Success:   true,
		RequestID: req.RequestID,
		Message:   "request approved",
	})
}

// sendResponse sends a response message.
func (s *Server) sendResponse(conn net.Conn, requestID string, msgType MessageType, payload interface{}) {
	msg, err := NewMessage(msgType, requestID, payload)
	if err != nil {
		return
	}

	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = json.NewEncoder(conn).Encode(msg) // Best effort, connection may be closed
}

// sendError sends an error response.
func (s *Server) sendError(conn net.Conn, requestID, message string) {
	s.sendResponse(conn, requestID, MessageTypeErrorResponse, ErrorResponse{
		Code:    ErrorCodeInvalidRequest,
		Message: message,
	})
}

// createLockFile creates the lock file with the current PID.
func (s *Server) createLockFile() error {
	dir := filepath.Dir(s.lockPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data := fmt.Sprintf("%d\n", os.Getpid())
	return os.WriteFile(s.lockPath, []byte(data), 0o600)
}

// removeLockFile removes the lock file.
func (s *Server) removeLockFile() {
	_ = os.RemoveAll(s.lockPath) // Best effort cleanup
}
