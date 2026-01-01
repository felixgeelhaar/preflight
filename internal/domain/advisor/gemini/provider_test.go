package gemini

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

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-api-key")

	assert.NotNil(t, provider)
	assert.Equal(t, "gemini", provider.Name())
	assert.Equal(t, "gemini-2.0-flash", provider.Model())
	assert.True(t, provider.Available())
}

func TestNewProvider_EmptyAPIKey(t *testing.T) {
	provider := NewProvider("")

	assert.NotNil(t, provider)
	assert.False(t, provider.Available())
}

func TestNewProviderWithConfig(t *testing.T) {
	config := Config{
		APIKey: "test-key",
		Model:  "gemini-1.5-pro",
	}

	provider, err := NewProviderWithConfig(config)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "gemini-1.5-pro", provider.Model())
}

func TestNewProviderWithConfig_CustomEndpoint(t *testing.T) {
	config := Config{
		APIKey:   "test-key",
		Model:    "gemini-2.0-flash",
		Endpoint: "https://custom.endpoint.com",
	}

	provider, err := NewProviderWithConfig(config)

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProviderWithConfig_EmptyAPIKey(t *testing.T) {
	config := Config{
		Model: "gemini-2.0-flash",
	}

	_, err := NewProviderWithConfig(config)

	assert.ErrorIs(t, err, ErrEmptyAPIKey)
}

func TestNewProviderWithConfig_EmptyModel(t *testing.T) {
	config := Config{
		APIKey: "test-key",
	}

	_, err := NewProviderWithConfig(config)

	assert.ErrorIs(t, err, ErrEmptyModel)
}

func TestProvider_WithModel(t *testing.T) {
	provider := NewProvider("test-key")
	newProvider := provider.WithModel("gemini-1.5-pro")

	assert.Equal(t, "gemini-1.5-pro", newProvider.Model())
	assert.Equal(t, "gemini-2.0-flash", provider.Model()) // Original unchanged
}

func TestProvider_Complete_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/models/gemini-2.0-flash:generateContent")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock response
		resp := generateResponse{
			Candidates: []candidate{
				{
					Content: content{
						Parts: []part{{Text: "Test response"}},
						Role:  "model",
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
			},
			ModelVersion: "gemini-2.0-flash",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider := &Provider{
		apiKey:   "test-key",
		model:    "gemini-2.0-flash",
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("You are helpful", "Hello")
	response, err := provider.Complete(context.Background(), prompt)

	require.NoError(t, err)
	assert.Equal(t, "Test response", response.Content())
	assert.Equal(t, 15, response.TokensUsed())
	assert.Equal(t, "gemini-2.0-flash", response.Model())
}

func TestProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedBody generateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := generateResponse{
			Candidates: []candidate{
				{
					Content: content{
						Parts: []part{{Text: "OK"}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		apiKey:   "test-key",
		model:    "gemini-2.0-flash",
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("Be concise", "Hello")
	_, err := provider.Complete(context.Background(), prompt)

	require.NoError(t, err)
	require.NotNil(t, receivedBody.SystemInstruct)
	assert.Equal(t, "Be concise", receivedBody.SystemInstruct.Parts[0].Text)
}

func TestProvider_Complete_NotConfigured(t *testing.T) {
	provider := NewProvider("")

	prompt := advisor.NewPrompt("", "Hello")
	_, err := provider.Complete(context.Background(), prompt)

	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestProvider_Complete_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := errorResponse{}
		resp.Error.Message = "Invalid API key"
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		apiKey:   "invalid-key",
		model:    "gemini-2.0-flash",
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("", "Hello")
	_, err := provider.Complete(context.Background(), prompt)

	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		resp := errorResponse{}
		resp.Error.Message = "Rate limit exceeded"
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		apiKey:   "test-key",
		model:    "gemini-2.0-flash",
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("", "Hello")
	_, err := provider.Complete(context.Background(), prompt)

	assert.ErrorIs(t, err, ErrRateLimit)
}

func TestProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := errorResponse{}
		resp.Error.Message = "Invalid request"
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		apiKey:   "test-key",
		model:    "gemini-2.0-flash",
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("", "Hello")
	_, err := provider.Complete(context.Background(), prompt)

	assert.ErrorIs(t, err, ErrAPIError)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "test-key",
				Model:  "gemini-2.0-flash",
			},
			wantErr: nil,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "gemini-2.0-flash",
			},
			wantErr: ErrEmptyAPIKey,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "test-key",
			},
			wantErr: ErrEmptyModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
