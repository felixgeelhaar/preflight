// Package gemini provides an AI provider implementation for Google Gemini.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// Provider errors.
var (
	ErrNotConfigured = errors.New("gemini provider is not configured")
	ErrEmptyAPIKey   = errors.New("API key is required")
	ErrEmptyModel    = errors.New("model is required")
	ErrAPIError      = errors.New("gemini API error")
	ErrRateLimit     = errors.New("rate limit exceeded")
	ErrUnauthorized  = errors.New("unauthorized - check API key")
)

// API request types.
type generateRequest struct {
	Contents         []content         `json:"contents"`
	SystemInstruct   *content          `json:"systemInstruction,omitempty"`
	GenerationConfig *generationConfig `json:"generationConfig,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// API response types.
type generateResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
	ModelVersion  string        `json:"modelVersion"`
}

type candidate struct {
	Content      content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}

type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Config holds the configuration for the Gemini provider.
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

// Provider implements the AIProvider interface for Google Gemini.
type Provider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewProvider creates a new Gemini provider.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:   apiKey,
		model:    "gemini-2.0-flash",
		endpoint: "https://generativelanguage.googleapis.com/v1beta",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// NewProviderWithConfig creates a new Gemini provider with custom configuration.
func NewProviderWithConfig(config Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &Provider{
		apiKey:   config.APIKey,
		model:    config.Model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "gemini"
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

// Complete sends a prompt to Gemini and returns the response.
func (p *Provider) Complete(ctx context.Context, prompt advisor.Prompt) (advisor.Response, error) {
	if !p.Available() {
		return advisor.Response{}, ErrNotConfigured
	}

	// Build request
	reqBody := generateRequest{
		Contents: []content{
			{
				Role: "user",
				Parts: []part{
					{Text: prompt.UserPrompt()},
				},
			},
		},
		GenerationConfig: &generationConfig{
			Temperature:     prompt.Temperature(),
			MaxOutputTokens: prompt.MaxTokens(),
		},
	}

	// Add system instruction if present
	if prompt.SystemPrompt() != "" {
		reqBody.SystemInstruct = &content{
			Parts: []part{
				{Text: prompt.SystemPrompt()},
			},
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	// Gemini API uses the model in the URL path
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.endpoint, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := p.client.Do(req)
	if err != nil {
		return advisor.Response{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // Best effort close after reading body

	body, err := io.ReadAll(resp.Body)
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
	var genResp generateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return advisor.Response{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text content from first candidate
	var content string
	if len(genResp.Candidates) > 0 && len(genResp.Candidates[0].Content.Parts) > 0 {
		content = genResp.Candidates[0].Content.Parts[0].Text
	}

	return advisor.NewResponse(content, genResp.UsageMetadata.TotalTokenCount, genResp.ModelVersion), nil
}
