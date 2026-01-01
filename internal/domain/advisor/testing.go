package advisor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// MockServerOption configures a mock HTTP server for testing.
type MockServerOption func(*mockServerConfig)

type mockServerConfig struct {
	statusCode int
	response   interface{}
	headers    map[string]string
}

// WithStatusCode sets the HTTP status code for the mock response.
func WithStatusCode(code int) MockServerOption {
	return func(c *mockServerConfig) {
		c.statusCode = code
	}
}

// WithResponse sets the response body (will be JSON encoded).
func WithResponse(resp interface{}) MockServerOption {
	return func(c *mockServerConfig) {
		c.response = resp
	}
}

// WithHeader adds a header to the mock response.
func WithHeader(key, value string) MockServerOption {
	return func(c *mockServerConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	}
}

// NewMockServer creates a test HTTP server with the given options.
// Remember to call server.Close() when done.
func NewMockServer(opts ...MockServerOption) *httptest.Server {
	cfg := &mockServerConfig{
		statusCode: http.StatusOK,
		headers:    map[string]string{"Content-Type": "application/json"},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for k, v := range cfg.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(cfg.statusCode)
		if cfg.response != nil {
			_ = json.NewEncoder(w).Encode(cfg.response)
		}
	}))
}

// TestPrompt creates a simple prompt for testing.
func TestPrompt() Prompt {
	return NewPrompt("You are a test assistant", "Hello")
}

// TestPromptWithContent creates a prompt with custom content for testing.
func TestPromptWithContent(system, user string) Prompt {
	return NewPrompt(system, user)
}
