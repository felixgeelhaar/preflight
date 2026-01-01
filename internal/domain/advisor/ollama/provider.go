// Package ollama provides an AI provider implementation for Ollama (local LLMs).
package ollama

import (
	"context"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// Re-export common errors for backwards compatibility.
var (
	ErrNotConfigured = advisor.ErrNotConfigured
	ErrEmptyModel    = advisor.ErrEmptyModel
)

// Config holds the configuration for the Ollama provider.
type Config struct {
	Endpoint string
	Model    string
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Model == "" {
		return ErrEmptyModel
	}
	return nil
}

// Provider implements the AIProvider interface for Ollama.
type Provider struct {
	endpoint string
	model    string
}

// NewProvider creates a new Ollama provider.
func NewProvider(endpoint string) *Provider {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	return &Provider{
		endpoint: endpoint,
		model:    "llama3.2",
	}
}

// NewProviderWithConfig creates a new Ollama provider with custom configuration.
func NewProviderWithConfig(config Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	return &Provider{
		endpoint: endpoint,
		model:    config.Model,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Model returns the currently configured model.
func (p *Provider) Model() string {
	return p.model
}

// Endpoint returns the Ollama API endpoint.
func (p *Provider) Endpoint() string {
	return p.endpoint
}

// WithModel returns a new Provider with a different model.
func (p *Provider) WithModel(model string) *Provider {
	return &Provider{
		endpoint: p.endpoint,
		model:    model,
	}
}

// Available returns true if Ollama is configured.
// Note: This does not check if Ollama is actually running.
func (p *Provider) Available() bool {
	return p.endpoint != ""
}

// Complete sends a prompt to Ollama and returns the response.
// AI features require API integration which is planned for a future release.
func (p *Provider) Complete(_ context.Context, _ advisor.Prompt) (advisor.Response, error) {
	if !p.Available() {
		return advisor.Response{}, ErrNotConfigured
	}

	// AI completion requires Ollama API integration.
	// Use the noop provider or disable AI features with --no-ai flag.
	return advisor.Response{}, ErrNotConfigured
}
