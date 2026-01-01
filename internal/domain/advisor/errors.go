// Package advisor provides shared error definitions for AI providers.
package advisor

import "errors"

// Provider configuration errors.
var (
	// ErrNotConfigured is returned when a provider is not properly configured.
	ErrNotConfigured = errors.New("provider is not configured")

	// ErrEmptyAPIKey is returned when an API key is required but not provided.
	ErrEmptyAPIKey = errors.New("API key is required")

	// ErrEmptyModel is returned when a model is required but not provided.
	ErrEmptyModel = errors.New("model is required")
)

// API communication errors.
var (
	// ErrAPIError is returned when the API returns an error response.
	ErrAPIError = errors.New("API error")

	// ErrRateLimit is returned when the API rate limit is exceeded.
	ErrRateLimit = errors.New("rate limit exceeded")

	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized - check API key")

	// ErrEmptyResponse is returned when the API returns an empty response.
	ErrEmptyResponse = errors.New("empty response from API")
)
