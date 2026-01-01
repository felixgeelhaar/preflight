package advisor

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockServer_Default(t *testing.T) {
	server := NewMockServer()
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup //nolint:errcheck // test cleanup

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

func TestNewMockServer_WithStatusCode(t *testing.T) {
	server := NewMockServer(WithStatusCode(http.StatusUnauthorized))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNewMockServer_WithResponse(t *testing.T) {
	type testResponse struct {
		Message string `json:"message"`
	}

	server := NewMockServer(WithResponse(testResponse{Message: "hello"}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result testResponse
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Message)
}

func TestNewMockServer_WithHeader(t *testing.T) {
	server := NewMockServer(WithHeader("X-Custom-Header", "test-value"))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	assert.Equal(t, "test-value", resp.Header.Get("X-Custom-Header"))
}

func TestNewMockServer_MultipleOptions(t *testing.T) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	server := NewMockServer(
		WithStatusCode(http.StatusBadRequest),
		WithResponse(errorResponse{Error: "bad request"}),
		WithHeader("X-Error-Code", "400"),
	)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "400", resp.Header.Get("X-Error-Code"))
}

func TestTestPrompt(t *testing.T) {
	prompt := TestPrompt()

	assert.Equal(t, "You are a test assistant", prompt.SystemPrompt())
	assert.Equal(t, "Hello", prompt.UserPrompt())
}

func TestTestPromptWithContent(t *testing.T) {
	prompt := TestPromptWithContent("custom system", "custom user")

	assert.Equal(t, "custom system", prompt.SystemPrompt())
	assert.Equal(t, "custom user", prompt.UserPrompt())
}
