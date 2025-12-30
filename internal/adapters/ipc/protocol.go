// Package ipc provides IPC communication for the agent via Unix sockets.
package ipc

import (
	"encoding/json"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/agent"
)

// MessageType identifies the type of IPC message.
type MessageType string

const (
	// MessageTypeStatusRequest requests agent status.
	MessageTypeStatusRequest MessageType = "status_request"
	// MessageTypeStopRequest requests agent stop.
	MessageTypeStopRequest MessageType = "stop_request"
	// MessageTypeApproveRequest requests approval for a pending operation.
	MessageTypeApproveRequest MessageType = "approve_request"

	// MessageTypeStatusResponse contains agent status.
	MessageTypeStatusResponse MessageType = "status_response"
	// MessageTypeStopResponse contains stop result.
	MessageTypeStopResponse MessageType = "stop_response"
	// MessageTypeApproveResponse contains approval result.
	MessageTypeApproveResponse MessageType = "approve_response"
	// MessageTypeErrorResponse contains error details.
	MessageTypeErrorResponse MessageType = "error_response"
)

// Message is the envelope for all IPC messages.
type Message struct {
	Type      MessageType     `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType MessageType, requestID string, payload interface{}) (*Message, error) {
	var payloadBytes json.RawMessage
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Type:      msgType,
		RequestID: requestID,
		Timestamp: time.Now(),
		Payload:   payloadBytes,
	}, nil
}

// StatusRequest is the payload for a status request.
type StatusRequest struct {
	// Watch enables continuous status updates (not implemented yet)
	Watch bool `json:"watch,omitempty"`
}

// StatusResponse is the payload for a status response.
type StatusResponse struct {
	Status  agent.Status `json:"status"`
	Version string       `json:"version,omitempty"`
	PID     int          `json:"pid"`
}

// StopRequest is the payload for a stop request.
type StopRequest struct {
	// Force indicates whether to force stop without waiting
	Force bool `json:"force,omitempty"`
	// TimeoutSeconds is the max time to wait for graceful shutdown
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// StopResponse is the payload for a stop response.
type StopResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ApproveRequest is the payload for an approval request.
type ApproveRequest struct {
	// RequestID is the ID of the approval request to approve
	RequestID string `json:"request_id"`
}

// ApproveResponse is the payload for an approval response.
type ApproveResponse struct {
	Success   bool   `json:"success"`
	RequestID string `json:"request_id"`
	Message   string `json:"message,omitempty"`
}

// ErrorResponse is the payload for an error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common error codes
const (
	ErrorCodeInvalidRequest = "invalid_request"
	ErrorCodeNotRunning     = "not_running"
	ErrorCodeNotFound       = "not_found"
	ErrorCodeInternalError  = "internal_error"
	ErrorCodeTimeout        = "timeout"
)
