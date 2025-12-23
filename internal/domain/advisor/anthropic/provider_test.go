package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider_NoAPIKey(t *testing.T) {
	t.Parallel()

	p := NewProvider("")

	assert.NotNil(t, p)
	assert.False(t, p.Available())
}

func TestNewProvider_WithAPIKey(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")

	assert.NotNil(t, p)
	assert.True(t, p.Available())
}

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")

	assert.Equal(t, "anthropic", p.Name())
}

func TestProvider_Model(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")

	assert.Equal(t, "claude-3-5-sonnet-20241022", p.Model())
}

func TestProvider_WithModel(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key").WithModel("claude-3-opus-20240229")

	assert.Equal(t, "claude-3-opus-20240229", p.Model())
}

func TestProvider_Complete_NotAvailable(t *testing.T) {
	t.Parallel()

	p := NewProvider("")
	prompt := advisor.NewPrompt("system", "user")

	_, err := p.Complete(context.Background(), prompt)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestProvider_ImplementsAIProvider(t *testing.T) {
	t.Parallel()

	var _ advisor.AIProvider = (*Provider)(nil)
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid",
			config: Config{
				APIKey: "sk-ant-test-key",
				Model:  "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "claude-3-5-sonnet-20241022",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "sk-ant-test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewProviderWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		wantErrType error
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "sk-ant-test-key",
				Model:  "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom endpoint",
			config: Config{
				APIKey:   "sk-ant-test-key",
				Model:    "claude-3-5-sonnet-20241022",
				Endpoint: "https://custom.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "claude-3-5-sonnet-20241022",
			},
			wantErr:     true,
			wantErrType: ErrEmptyAPIKey,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "sk-ant-test-key",
			},
			wantErr:     true,
			wantErrType: ErrEmptyModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := NewProviderWithConfig(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p)
				assert.Equal(t, tt.config.APIKey, p.apiKey)
				assert.Equal(t, tt.config.Model, p.model)
			}
		})
	}
}

func TestProvider_Complete_Success(t *testing.T) {
	t.Parallel()

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "sk-ant-test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// Return mock response
		resp := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{
					Type: "text",
					Text: "Hello! How can I help you?",
				},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage: usage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp) //nolint:errcheck // Test code
	}))
	defer server.Close()

	// Create provider with mock endpoint
	p, err := NewProviderWithConfig(Config{
		APIKey:   "sk-ant-test-key",
		Model:    "claude-3-5-sonnet-20241022",
		Endpoint: server.URL,
	})
	require.NoError(t, err)

	prompt := advisor.NewPrompt("You are a helpful assistant.", "Say hello")
	response, err := p.Complete(context.Background(), prompt)

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you?", response.Content())
	assert.Equal(t, 30, response.TokensUsed())
	assert.Equal(t, "claude-3-5-sonnet-20241022", response.Model())
}

func TestProvider_Complete_Unauthorized(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(errorResponse{ //nolint:errcheck // Test code
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		})
	}))
	defer server.Close()

	p, err := NewProviderWithConfig(Config{
		APIKey:   "invalid-key",
		Model:    "claude-3-5-sonnet-20241022",
		Endpoint: server.URL,
	})
	require.NoError(t, err)

	prompt := advisor.NewPrompt("system", "user")
	_, err = p.Complete(context.Background(), prompt)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestProvider_Complete_RateLimit(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns 429
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(errorResponse{ //nolint:errcheck // Test code
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "rate_limit_error",
				Message: "Rate limit exceeded",
			},
		})
	}))
	defer server.Close()

	p, err := NewProviderWithConfig(Config{
		APIKey:   "sk-ant-test-key",
		Model:    "claude-3-5-sonnet-20241022",
		Endpoint: server.URL,
	})
	require.NoError(t, err)

	prompt := advisor.NewPrompt("system", "user")
	_, err = p.Complete(context.Background(), prompt)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRateLimit)
}
