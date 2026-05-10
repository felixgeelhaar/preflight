// Package anthropic provides an AI provider implementation for Anthropic Claude.
package anthropic

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

// DefaultModel is the current default Claude model used by the advisor.
// Refreshed 2026-05; previous default (claude-3-5-sonnet-20241022) was 14+
// months stale. Sonnet 4.6 is the cost/latency sweet spot for advisory work.
const DefaultModel = "claude-sonnet-4-6"

// API request types.
//
// system is encoded as a slice of cacheable blocks rather than a plain string
// so we can mark the static system prompt with cache_control: ephemeral and
// take advantage of Anthropic's prompt cache (~90% cost reduction on repeated
// runs of init/analyze that share the same system prompt).
type messagesRequest struct {
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	System      []systemBlock `json:"system,omitempty"`
	Messages    []message     `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

// systemBlock is a single segment of the system prompt. Setting CacheControl
// to a non-nil ephemeral marker tells the API to cache this prefix for ~5
// minutes; subsequent calls with an identical prefix avoid re-billing it.
type systemBlock struct {
	Type         string        `json:"type"` // always "text"
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"` // always "ephemeral"
}

// newCacheableSystemBlock builds a system block whose prefix the server may
// cache. Use for the static system prompt; do NOT use for per-request content.
func newCacheableSystemBlock(text string) systemBlock {
	return systemBlock{
		Type:         "text",
		Text:         text,
		CacheControl: &cacheControl{Type: "ephemeral"},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// API response types.
type messagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        usage          `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type errorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Config holds the configuration for the Anthropic provider.
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

// Provider implements the AIProvider interface for Anthropic.
type Provider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewProvider creates a new Anthropic provider with DefaultModel.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:   apiKey,
		model:    DefaultModel,
		endpoint: "https://api.anthropic.com",
		client: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: defaultTransport(),
		},
	}
}

// NewProviderWithConfig creates a new Anthropic provider with custom configuration.
func NewProviderWithConfig(config Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com"
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
	return "anthropic"
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

// Complete sends a prompt to Anthropic and returns the response.
func (p *Provider) Complete(ctx context.Context, prompt advisor.Prompt) (advisor.Response, error) {
	if !p.Available() {
		return advisor.Response{}, ErrNotConfigured
	}

	// Build request. The static system prompt is sent as a cacheable block so
	// repeated calls (init wizard, analyze) hit Anthropic's prompt cache.
	var systemBlocks []systemBlock
	if sp := prompt.SystemPrompt(); sp != "" {
		systemBlocks = []systemBlock{newCacheableSystemBlock(sp)}
	}

	reqBody := messagesRequest{
		Model:       p.model,
		MaxTokens:   prompt.MaxTokens(),
		System:      systemBlocks,
		Temperature: prompt.Temperature(),
		Messages: []message{
			{
				Role:    "user",
				Content: prompt.UserPrompt(),
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/messages", p.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return advisor.Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
	var msgResp messagesResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return advisor.Response{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text content
	var content string
	for _, block := range msgResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	tokensUsed := msgResp.Usage.InputTokens + msgResp.Usage.OutputTokens
	return advisor.NewResponse(content, tokensUsed, msgResp.Model), nil
}
