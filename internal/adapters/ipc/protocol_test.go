package ipc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	t.Run("creates message with payload", func(t *testing.T) {
		payload := StatusRequest{Watch: true}
		msg, err := NewMessage(MessageTypeStatusRequest, "req-123", payload)

		require.NoError(t, err)
		assert.Equal(t, MessageTypeStatusRequest, msg.Type)
		assert.Equal(t, "req-123", msg.RequestID)
		assert.False(t, msg.Timestamp.IsZero())
		assert.NotEmpty(t, msg.Payload)

		// Verify payload can be unmarshaled
		var decoded StatusRequest
		err = json.Unmarshal(msg.Payload, &decoded)
		require.NoError(t, err)
		assert.True(t, decoded.Watch)
	})

	t.Run("creates message without payload", func(t *testing.T) {
		msg, err := NewMessage(MessageTypeStatusRequest, "req-456", nil)

		require.NoError(t, err)
		assert.Equal(t, MessageTypeStatusRequest, msg.Type)
		assert.Nil(t, msg.Payload)
	})
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	original := &Message{
		Type:      MessageTypeStopRequest,
		RequestID: "test-id",
	}

	// Add payload
	payload, _ := json.Marshal(StopRequest{Force: true})
	original.Payload = payload

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.RequestID, decoded.RequestID)

	// Verify payload
	var stopReq StopRequest
	err = json.Unmarshal(decoded.Payload, &stopReq)
	require.NoError(t, err)
	assert.True(t, stopReq.Force)
}

func TestStatusResponse_JSON(t *testing.T) {
	resp := StatusResponse{
		Version: "1.0.0",
		PID:     12345,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded StatusResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", decoded.Version)
	assert.Equal(t, 12345, decoded.PID)
}

func TestErrorResponse_JSON(t *testing.T) {
	resp := ErrorResponse{
		Code:    ErrorCodeInvalidRequest,
		Message: "test error",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ErrorResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ErrorCodeInvalidRequest, decoded.Code)
	assert.Equal(t, "test error", decoded.Message)
}
