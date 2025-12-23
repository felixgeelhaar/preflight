package advisor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestRecommendation(t *testing.T, presetID string, confidence ConfidenceLevel) Recommendation {
	t.Helper()
	rec, err := NewRecommendation(presetID, "Test rationale for "+presetID, confidence)
	require.NoError(t, err)
	return rec
}

func TestNewSuggestion_Valid(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	suggestion, err := NewSuggestion("openai", []Recommendation{rec})

	require.NoError(t, err)
	assert.Equal(t, "openai", suggestion.Provider())
	assert.Len(t, suggestion.Recommendations(), 1)
	assert.False(t, suggestion.GeneratedAt().IsZero())
}

func TestNewSuggestion_EmptyProvider(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	_, err := NewSuggestion("", []Recommendation{rec})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyProvider)
}

func TestNewSuggestion_EmptyRecommendations(t *testing.T) {
	t.Parallel()

	_, err := NewSuggestion("openai", []Recommendation{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRecommendations)
}

func TestNewSuggestion_NilRecommendations(t *testing.T) {
	t.Parallel()

	_, err := NewSuggestion("openai", nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRecommendations)
}

func TestSuggestion_PrimaryRecommendation(t *testing.T) {
	t.Parallel()

	rec1 := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	rec2 := createTestRecommendation(t, "nvim:minimal", ConfidenceMedium)

	suggestion, _ := NewSuggestion("openai", []Recommendation{rec1, rec2})

	primary := suggestion.PrimaryRecommendation()
	assert.Equal(t, "nvim:balanced", primary.PresetID())
}

func TestSuggestion_AlternativeRecommendations(t *testing.T) {
	t.Parallel()

	rec1 := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	rec2 := createTestRecommendation(t, "nvim:minimal", ConfidenceMedium)
	rec3 := createTestRecommendation(t, "nvim:pro", ConfidenceLow)

	suggestion, _ := NewSuggestion("openai", []Recommendation{rec1, rec2, rec3})

	alternatives := suggestion.AlternativeRecommendations()
	assert.Len(t, alternatives, 2)
}

func TestSuggestion_HasRecommendations(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	suggestion, _ := NewSuggestion("openai", []Recommendation{rec})

	assert.True(t, suggestion.HasRecommendations())
}

func TestSuggestion_Count(t *testing.T) {
	t.Parallel()

	rec1 := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	rec2 := createTestRecommendation(t, "nvim:minimal", ConfidenceMedium)

	suggestion, _ := NewSuggestion("openai", []Recommendation{rec1, rec2})

	assert.Equal(t, 2, suggestion.Count())
}

func TestSuggestion_WithContext(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	suggestion, _ := NewSuggestion("openai", []Recommendation{rec})

	updated := suggestion.WithContext("User is new to Neovim and prefers simplicity")

	assert.Empty(t, suggestion.Context())
	assert.Equal(t, "User is new to Neovim and prefers simplicity", updated.Context())
}

func TestSuggestion_WithTokensUsed(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	suggestion, _ := NewSuggestion("openai", []Recommendation{rec})

	updated := suggestion.WithTokensUsed(1500)

	assert.Equal(t, 0, suggestion.TokensUsed())
	assert.Equal(t, 1500, updated.TokensUsed())
}

func TestSuggestion_WithDuration(t *testing.T) {
	t.Parallel()

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	suggestion, _ := NewSuggestion("openai", []Recommendation{rec})

	duration := 500 * time.Millisecond
	updated := suggestion.WithDuration(duration)

	assert.Equal(t, time.Duration(0), suggestion.Duration())
	assert.Equal(t, duration, updated.Duration())
}

func TestSuggestion_IsZero(t *testing.T) {
	t.Parallel()

	var zero Suggestion
	assert.True(t, zero.IsZero())

	rec := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	nonZero, _ := NewSuggestion("openai", []Recommendation{rec})
	assert.False(t, nonZero.IsZero())
}

func TestSuggestion_String(t *testing.T) {
	t.Parallel()

	rec1 := createTestRecommendation(t, "nvim:balanced", ConfidenceHigh)
	rec2 := createTestRecommendation(t, "nvim:minimal", ConfidenceMedium)

	suggestion, _ := NewSuggestion("openai", []Recommendation{rec1, rec2})

	assert.Equal(t, "Suggestion from openai (2 recommendations)", suggestion.String())
}
