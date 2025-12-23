package advisor

import (
	"errors"
	"fmt"
	"time"
)

// Suggestion errors.
var (
	ErrEmptyProvider     = errors.New("provider cannot be empty")
	ErrNoRecommendations = errors.New("at least one recommendation is required")
)

// Suggestion represents a set of AI-generated recommendations.
// It is an immutable value object.
type Suggestion struct {
	provider        string
	recommendations []Recommendation
	generatedAt     time.Time
	context         string
	tokensUsed      int
	duration        time.Duration
}

// NewSuggestion creates a new Suggestion.
func NewSuggestion(provider string, recommendations []Recommendation) (Suggestion, error) {
	if provider == "" {
		return Suggestion{}, ErrEmptyProvider
	}

	if len(recommendations) == 0 {
		return Suggestion{}, ErrNoRecommendations
	}

	// Defensive copy
	recs := make([]Recommendation, len(recommendations))
	copy(recs, recommendations)

	return Suggestion{
		provider:        provider,
		recommendations: recs,
		generatedAt:     time.Now(),
	}, nil
}

// Provider returns the AI provider that generated this suggestion.
func (s Suggestion) Provider() string {
	return s.provider
}

// Recommendations returns all recommendations.
func (s Suggestion) Recommendations() []Recommendation {
	result := make([]Recommendation, len(s.recommendations))
	copy(result, s.recommendations)
	return result
}

// GeneratedAt returns when this suggestion was generated.
func (s Suggestion) GeneratedAt() time.Time {
	return s.generatedAt
}

// Context returns the context used for generating this suggestion.
func (s Suggestion) Context() string {
	return s.context
}

// TokensUsed returns the number of tokens used for this suggestion.
func (s Suggestion) TokensUsed() int {
	return s.tokensUsed
}

// Duration returns how long it took to generate this suggestion.
func (s Suggestion) Duration() time.Duration {
	return s.duration
}

// PrimaryRecommendation returns the first (most confident) recommendation.
func (s Suggestion) PrimaryRecommendation() Recommendation {
	if len(s.recommendations) == 0 {
		return Recommendation{}
	}
	return s.recommendations[0]
}

// AlternativeRecommendations returns all recommendations except the primary one.
func (s Suggestion) AlternativeRecommendations() []Recommendation {
	if len(s.recommendations) <= 1 {
		return []Recommendation{}
	}
	result := make([]Recommendation, len(s.recommendations)-1)
	copy(result, s.recommendations[1:])
	return result
}

// HasRecommendations returns true if there are any recommendations.
func (s Suggestion) HasRecommendations() bool {
	return len(s.recommendations) > 0
}

// Count returns the number of recommendations.
func (s Suggestion) Count() int {
	return len(s.recommendations)
}

// WithContext returns a new Suggestion with context set.
func (s Suggestion) WithContext(context string) Suggestion {
	return Suggestion{
		provider:        s.provider,
		recommendations: s.recommendations,
		generatedAt:     s.generatedAt,
		context:         context,
		tokensUsed:      s.tokensUsed,
		duration:        s.duration,
	}
}

// WithTokensUsed returns a new Suggestion with tokens used set.
func (s Suggestion) WithTokensUsed(tokens int) Suggestion {
	return Suggestion{
		provider:        s.provider,
		recommendations: s.recommendations,
		generatedAt:     s.generatedAt,
		context:         s.context,
		tokensUsed:      tokens,
		duration:        s.duration,
	}
}

// WithDuration returns a new Suggestion with duration set.
func (s Suggestion) WithDuration(duration time.Duration) Suggestion {
	return Suggestion{
		provider:        s.provider,
		recommendations: s.recommendations,
		generatedAt:     s.generatedAt,
		context:         s.context,
		tokensUsed:      s.tokensUsed,
		duration:        duration,
	}
}

// IsZero returns true if this is a zero-value Suggestion.
func (s Suggestion) IsZero() bool {
	return s.provider == ""
}

// String returns a summary string.
func (s Suggestion) String() string {
	return fmt.Sprintf("Suggestion from %s (%d recommendations)", s.provider, len(s.recommendations))
}
