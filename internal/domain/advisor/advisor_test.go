package advisor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetailedExplanation_Valid(t *testing.T) {
	t.Parallel()

	exp, err := NewDetailedExplanation(
		"nvim:balanced",
		"This preset provides a balanced Neovim configuration",
	)

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", exp.PresetID())
	assert.Equal(t, "This preset provides a balanced Neovim configuration", exp.Summary())
}

func TestDetailedExplanation_EmptyPresetID(t *testing.T) {
	t.Parallel()

	_, err := NewDetailedExplanation("", "Some summary")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPresetID)
}

func TestDetailedExplanation_EmptySummary(t *testing.T) {
	t.Parallel()

	_, err := NewDetailedExplanation("nvim:balanced", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptySummary)
}

func TestDetailedExplanation_WithSections(t *testing.T) {
	t.Parallel()

	exp, _ := NewDetailedExplanation("nvim:balanced", "A balanced config")
	sections := map[string]string{
		"Features":   "LSP, Treesitter, Telescope",
		"Tradeoffs":  "More startup time for more features",
		"Comparison": "Better than minimal, less than pro",
	}

	updated := exp.WithSections(sections)

	assert.Empty(t, exp.Sections())
	assert.Equal(t, sections, updated.Sections())
}

func TestDetailedExplanation_WithBulletPoints(t *testing.T) {
	t.Parallel()

	exp, _ := NewDetailedExplanation("nvim:balanced", "A balanced config")
	bullets := []string{
		"Includes LSP support out of the box",
		"Telescope for fuzzy finding",
		"Treesitter for syntax highlighting",
	}

	updated := exp.WithBulletPoints(bullets)

	assert.Empty(t, exp.BulletPoints())
	assert.Equal(t, bullets, updated.BulletPoints())
}

func TestDetailedExplanation_WithCodeExamples(t *testing.T) {
	t.Parallel()

	exp, _ := NewDetailedExplanation("nvim:balanced", "A balanced config")
	examples := map[string]string{
		"keybindings": "<leader>ff to find files",
	}

	updated := exp.WithCodeExamples(examples)

	assert.Empty(t, exp.CodeExamples())
	assert.Equal(t, examples, updated.CodeExamples())
}

func TestDetailedExplanation_IsZero(t *testing.T) {
	t.Parallel()

	var zero DetailedExplanation
	assert.True(t, zero.IsZero())

	nonZero, _ := NewDetailedExplanation("nvim:balanced", "Summary")
	assert.False(t, nonZero.IsZero())
}

// MockAdvisor is a test implementation of the Advisor interface.
type MockAdvisor struct {
	suggestFunc func(ctx context.Context, suggestCtx SuggestContext) (Suggestion, error)
	explainFunc func(ctx context.Context, req ExplainRequest) (DetailedExplanation, error)
}

func (m *MockAdvisor) Suggest(ctx context.Context, suggestCtx SuggestContext) (Suggestion, error) {
	if m.suggestFunc != nil {
		return m.suggestFunc(ctx, suggestCtx)
	}
	return Suggestion{}, nil
}

func (m *MockAdvisor) Explain(ctx context.Context, req ExplainRequest) (DetailedExplanation, error) {
	if m.explainFunc != nil {
		return m.explainFunc(ctx, req)
	}
	return DetailedExplanation{}, nil
}

func TestAdvisorInterface_Suggest(t *testing.T) {
	t.Parallel()

	rec, _ := NewRecommendation("nvim:balanced", "Good choice", ConfidenceHigh)
	expectedSuggestion, _ := NewSuggestion("mock", []Recommendation{rec})

	advisor := &MockAdvisor{
		suggestFunc: func(ctx context.Context, suggestCtx SuggestContext) (Suggestion, error) {
			return expectedSuggestion, nil
		},
	}

	profile, _ := NewUserProfile(ExperienceIntermediate)
	suggestCtx, _ := NewSuggestContext("nvim", profile)

	suggestion, err := advisor.Suggest(context.Background(), suggestCtx)

	require.NoError(t, err)
	assert.Equal(t, "mock", suggestion.Provider())
	assert.Len(t, suggestion.Recommendations(), 1)
}

func TestAdvisorInterface_Explain(t *testing.T) {
	t.Parallel()

	expectedExplanation, _ := NewDetailedExplanation("nvim:balanced", "A balanced preset")

	advisor := &MockAdvisor{
		explainFunc: func(ctx context.Context, req ExplainRequest) (DetailedExplanation, error) {
			return expectedExplanation, nil
		},
	}

	req, _ := NewExplainRequest("nvim:balanced")

	explanation, err := advisor.Explain(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", explanation.PresetID())
}
