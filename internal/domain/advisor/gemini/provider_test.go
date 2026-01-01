package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-api-key")

	assert.NotNil(t, provider)
	assert.Equal(t, "gemini", provider.Name())
	assert.Equal(t, DefaultModel, provider.Model())
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
	assert.Equal(t, DefaultModel, provider.Model()) // Original unchanged
}

func TestProvider_Complete_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/models/")
		assert.Contains(t, r.URL.Path, ":generateContent")
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
			ModelVersion: DefaultModel,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider := &Provider{
		apiKey:   "test-key",
		model:    DefaultModel,
		endpoint: server.URL,
		client:   server.Client(),
	}

	prompt := advisor.NewPrompt("You are helpful", "Hello")
	response, err := provider.Complete(context.Background(), prompt)

	require.NoError(t, err)
	assert.Equal(t, "Test response", response.Content())
	assert.Equal(t, 15, response.TokensUsed())
	assert.Equal(t, DefaultModel, response.Model())
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
		{
			name: "invalid model",
			config: Config{
				APIKey: "test-key",
				Model:  "not-a-real-model",
			},
			wantErr: ErrInvalidModel,
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

func TestIsKnownModel(t *testing.T) {
	tests := []struct {
		model string
		known bool
	}{
		{"gemini-3-pro-preview", true},
		{"gemini-3-flash-preview", true},
		{"gemini-2.5-flash", true},
		{"gemini-2.5-pro", true},
		{"gemini-2.0-flash", true},
		{"gemini-1.5-pro", true},
		{"gemini-pro", true},
		{"not-a-model", false},
		{"gpt-4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			assert.Equal(t, tt.known, IsKnownModel(tt.model))
		})
	}
}

func TestNewProviderWithConfig_CustomTimeout(t *testing.T) {
	config := Config{
		APIKey:  "test-key",
		Model:   "gemini-2.0-flash",
		Timeout: 30 * time.Second,
	}

	provider, err := NewProviderWithConfig(config)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Timeout is set on the internal client
}

func TestNewProviderWithConfig_DefaultTimeout(t *testing.T) {
	config := Config{
		APIKey: "test-key",
		Model:  "gemini-2.0-flash",
		// No timeout specified
	}

	provider, err := NewProviderWithConfig(config)

	require.NoError(t, err)
	assert.NotNil(t, provider)
	// Should use DefaultTimeout
}

func TestProvider_Complete_EmptyResponse(t *testing.T) {
	tests := []struct {
		name     string
		response generateResponse
		errMsg   string
	}{
		{
			name: "no candidates",
			response: generateResponse{
				Candidates: []candidate{},
			},
			errMsg: "no candidates returned",
		},
		{
			name: "no content parts",
			response: generateResponse{
				Candidates: []candidate{
					{
						Content: content{
							Parts: []part{},
						},
					},
				},
			},
			errMsg: "no content parts",
		},
		{
			name: "empty text content",
			response: generateResponse{
				Candidates: []candidate{
					{
						Content: content{
							Parts: []part{{Text: ""}},
						},
					},
				},
			},
			errMsg: "empty text content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			provider := &Provider{
				apiKey:   "test-key",
				model:    DefaultModel,
				endpoint: server.URL,
				client:   server.Client(),
			}

			prompt := advisor.NewPrompt("", "Hello")
			_, err := provider.Complete(context.Background(), prompt)

			assert.ErrorIs(t, err, ErrEmptyResponse)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid HTTPS endpoint",
			endpoint: "https://api.example.com",
			wantErr:  false,
		},
		{
			name:     "HTTP localhost allowed",
			endpoint: "http://localhost:8080",
			wantErr:  false,
		},
		{
			name:     "HTTP 127.0.0.1 allowed",
			endpoint: "http://127.0.0.1:8080",
			wantErr:  false,
		},
		{
			name:     "HTTP not allowed for remote",
			endpoint: "http://api.example.com",
			wantErr:  true,
			errMsg:   "HTTPS required",
		},
		{
			name:     "AWS metadata blocked",
			endpoint: "https://169.254.169.254",
			wantErr:  true,
			errMsg:   "blocked host",
		},
		{
			name:     "GCP metadata blocked",
			endpoint: "https://metadata.google.internal",
			wantErr:  true,
			errMsg:   "blocked host",
		},
		{
			name:     "private IP 10.x blocked",
			endpoint: "https://10.0.0.1",
			wantErr:  true,
			errMsg:   "private IP",
		},
		{
			name:     "private IP 192.168.x blocked",
			endpoint: "https://192.168.1.1",
			wantErr:  true,
			errMsg:   "private IP",
		},
		{
			name:     "private IP 172.16.x blocked",
			endpoint: "https://172.16.0.1",
			wantErr:  true,
			errMsg:   "private IP",
		},
		{
			name:     "private IP 172.31.x blocked",
			endpoint: "https://172.31.255.255",
			wantErr:  true,
			errMsg:   "private IP",
		},
		{
			name:     "172.9.x NOT private (regression test)",
			endpoint: "https://172.9.0.1",
			wantErr:  false, // 172.9.x.x is NOT in the private range 172.16-31
		},
		{
			name:     "172.32.x NOT private",
			endpoint: "https://172.32.0.1",
			wantErr:  false, // 172.32.x.x is outside the private range
		},
		{
			name:     "invalid URL",
			endpoint: "://invalid",
			wantErr:  true,
			errMsg:   "invalid endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEndpoint(tt.endpoint)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate_Endpoint(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with custom HTTPS endpoint",
			config: Config{
				APIKey:   "test-key",
				Model:    "gemini-2.0-flash",
				Endpoint: "https://custom.api.com",
			},
			wantErr: false,
		},
		{
			name: "invalid config with HTTP endpoint",
			config: Config{
				APIKey:   "test-key",
				Model:    "gemini-2.0-flash",
				Endpoint: "http://custom.api.com",
			},
			wantErr: true,
		},
		{
			name: "valid config with localhost HTTP",
			config: Config{
				APIKey:   "test-key",
				Model:    "gemini-2.0-flash",
				Endpoint: "http://localhost:8080",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
