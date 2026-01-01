// Package openai provides an AI provider implementation for OpenAI.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// Default configuration values.
const (
	DefaultTimeout  = 60 * time.Second
	MaxResponseSize = 10 * 1024 * 1024 // 10MB max response to prevent DoS
)

// defaultTransport creates an HTTP transport optimized for connection reuse.
func defaultTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
}

// Re-export common errors for backwards compatibility.
var (
	ErrNotConfigured = advisor.ErrNotConfigured
	ErrEmptyAPIKey   = advisor.ErrEmptyAPIKey
	ErrEmptyModel    = advisor.ErrEmptyModel
	ErrAPIError      = advisor.ErrAPIError
	ErrRateLimit     = advisor.ErrRateLimit
	ErrUnauthorized  = advisor.ErrUnauthorized
)

// API request types.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// API response types.
type chatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []choice  `json:"choices"`
	Usage   chatUsage `json:"usage"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Config holds the configuration for the OpenAI provider.
type Config struct {
	APIKey   string
	Model    string
	Endpoint string // Optional custom endpoint
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return ErrEmptyAPIKey
	}
	if c.Model == "" {
		return ErrEmptyModel
	}
	return nil
}

// Provider implements the AIProvider interface for OpenAI.
type Provider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewProvider creates a new OpenAI provider.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:   apiKey,
		model:    "gpt-4o",
		endpoint: "https://api.openai.com/v1",
		client: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: defaultTransport(),
		},
	}
}

// NewProviderWithConfig creates a new OpenAI provider with custom configuration.
func NewProviderWithConfig(config Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	return &Provider{
		apiKey:   config.APIKey,
		model:    config.Model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: defaultTransport(),
		},
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai"
}

// Model returns the currently configured model.
func (p *Provider) Model() string {
	return p.model
}

// WithModel returns a new Provider with a different model.
func (p *Provider) WithModel(model string) *Provider {
	return &Provider{
		apiKey:   p.apiKey,
		model:    model,
		endpoint: p.endpoint,
		client:   p.client,
	}
}

// Available returns true if the provider is configured.
func (p *Provider) Available() bool {
	return p.apiKey != ""
}

// Complete sends a prompt to OpenAI and returns the response.
func (p *Provider) Complete(ctx context.Context, prompt advisor.Prompt) (advisor.Response, error) {
	if !p.Available() {
		return advisor.Response{}, ErrNotConfigured
	}

	// Build messages array
	messages := []chatMessage{}
	if prompt.SystemPrompt() != "" {
		messages = append(messages, chatMessage{
			Role:    "system",
			Content: prompt.SystemPrompt(),
		})
	}
	messages = append(messages, chatMessage{
		Role:    "user",
		Content: prompt.UserPrompt(),
	})

	// Build request
	reqBody := chatRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   prompt.MaxTokens(),
		Temperature: prompt.Temperature(),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", p.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return advisor.Response{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Best effort close after reading body

	// Limit response size to prevent memory exhaustion DoS
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				return advisor.Response{}, ErrUnauthorized
			case http.StatusTooManyRequests:
				return advisor.Response{}, ErrRateLimit
			default:
				return advisor.Response{}, fmt.Errorf("%w: %s", ErrAPIError, errResp.Error.Message)
			}
		}
		return advisor.Response{}, fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
	}

	// Parse successful response
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return advisor.Response{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content from first choice
	var content string
	if len(chatResp.Choices) > 0 {
		content = chatResp.Choices[0].Message.Content
	}

	return advisor.NewResponse(content, chatResp.Usage.TotalTokens, chatResp.Model), nil
}
