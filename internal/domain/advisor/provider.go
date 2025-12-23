package advisor

import (
	"context"
)

// Prompt represents a prompt to send to an AI provider.
type Prompt struct {
	systemPrompt string
	userPrompt   string
	maxTokens    int
	temperature  float64
}

// NewPrompt creates a new Prompt.
func NewPrompt(system, user string) Prompt {
	return Prompt{
		systemPrompt: system,
		userPrompt:   user,
		maxTokens:    1024,
		temperature:  0.7,
	}
}

// SystemPrompt returns the system prompt.
func (p Prompt) SystemPrompt() string {
	return p.systemPrompt
}

// UserPrompt returns the user prompt.
func (p Prompt) UserPrompt() string {
	return p.userPrompt
}

// MaxTokens returns the maximum tokens for the response.
func (p Prompt) MaxTokens() int {
	return p.maxTokens
}

// Temperature returns the temperature setting.
func (p Prompt) Temperature() float64 {
	return p.temperature
}

// WithMaxTokens returns a new Prompt with max tokens set.
func (p Prompt) WithMaxTokens(tokens int) Prompt {
	return Prompt{
		systemPrompt: p.systemPrompt,
		userPrompt:   p.userPrompt,
		maxTokens:    tokens,
		temperature:  p.temperature,
	}
}

// WithTemperature returns a new Prompt with temperature set.
func (p Prompt) WithTemperature(temp float64) Prompt {
	return Prompt{
		systemPrompt: p.systemPrompt,
		userPrompt:   p.userPrompt,
		maxTokens:    p.maxTokens,
		temperature:  temp,
	}
}

// Response represents a response from an AI provider.
type Response struct {
	content    string
	tokensUsed int
	model      string
}

// NewResponse creates a new Response.
func NewResponse(content string, tokensUsed int, model string) Response {
	return Response{
		content:    content,
		tokensUsed: tokensUsed,
		model:      model,
	}
}

// Content returns the response content.
func (r Response) Content() string {
	return r.content
}

// TokensUsed returns the number of tokens used.
func (r Response) TokensUsed() int {
	return r.tokensUsed
}

// Model returns the model used.
func (r Response) Model() string {
	return r.model
}

// AIProvider defines the interface for AI backends.
type AIProvider interface {
	// Name returns the provider name (e.g., "openai", "anthropic", "ollama").
	Name() string

	// Complete sends a prompt and returns the response.
	Complete(ctx context.Context, prompt Prompt) (Response, error)

	// Available returns true if this provider is configured and ready.
	Available() bool
}
