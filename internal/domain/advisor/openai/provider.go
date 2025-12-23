// Package openai provides an AI provider implementation for OpenAI.
package openai

import (
	"context"
	"errors"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// Provider errors.
var (
	ErrNotConfigured = errors.New("openai provider is not configured")
	ErrEmptyAPIKey   = errors.New("API key is required")
	ErrEmptyModel    = errors.New("model is required")
)

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
}

// NewProvider creates a new OpenAI provider.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:   apiKey,
		model:    "gpt-4o",
		endpoint: "https://api.openai.com/v1",
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
	}
}

// Available returns true if the provider is configured.
func (p *Provider) Available() bool {
	return p.apiKey != ""
}

// Complete sends a prompt to OpenAI and returns the response.
// Note: This is a skeleton implementation. Actual HTTP calls would be made
// in production, but tests use mocks.
func (p *Provider) Complete(ctx context.Context, prompt advisor.Prompt) (advisor.Response, error) {
	if !p.Available() {
		return advisor.Response{}, ErrNotConfigured
	}

	// In a full implementation, this would:
	// 1. Build the OpenAI API request
	// 2. Make an HTTP POST to the chat completions endpoint
	// 3. Parse the response and return it

	// For now, return a placeholder that would be replaced by actual API call
	// Production code would use the openai-go SDK or make HTTP requests directly

	// This allows the interface to be tested without making real API calls
	return advisor.Response{}, errors.New("not implemented: requires API call")
}
