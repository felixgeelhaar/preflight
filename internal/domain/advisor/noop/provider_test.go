package noop

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider()

	assert.Equal(t, "noop", p.Name())
}

func TestProvider_Available(t *testing.T) {
	t.Parallel()

	p := NewProvider()

	// Noop provider is always available
	assert.True(t, p.Available())
}

func TestProvider_Complete(t *testing.T) {
	t.Parallel()

	p := NewProvider()
	prompt := advisor.NewPrompt("system", "user")

	resp, err := p.Complete(context.Background(), prompt)

	require.NoError(t, err)
	assert.Empty(t, resp.Content())
	assert.Equal(t, 0, resp.TokensUsed())
	assert.Equal(t, "noop", resp.Model())
}

func TestProvider_ImplementsAIProvider(t *testing.T) {
	t.Parallel()

	var _ advisor.AIProvider = (*Provider)(nil)
}

func TestNewAdvisor(t *testing.T) {
	t.Parallel()

	adv := NewAdvisor()

	assert.NotNil(t, adv)
}

func TestAdvisor_Suggest(t *testing.T) {
	t.Parallel()

	adv := NewAdvisor()
	profile, _ := advisor.NewUserProfile(advisor.ExperienceIntermediate)
	ctx, _ := advisor.NewSuggestContext("nvim", profile)

	suggestion, err := adv.Suggest(context.Background(), ctx)

	require.NoError(t, err)
	assert.Equal(t, "noop", suggestion.Provider())
	// Noop advisor returns a default recommendation
	assert.True(t, suggestion.HasRecommendations())
}

func TestAdvisor_Suggest_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		category     string
		experience   advisor.ExperienceLevel
		wantPresetID string
		wantHasRecs  bool
	}{
		{
			name:         "nvim beginner",
			category:     "nvim",
			experience:   advisor.ExperienceBeginner,
			wantPresetID: "nvim:minimal",
			wantHasRecs:  true,
		},
		{
			name:         "nvim intermediate",
			category:     "nvim",
			experience:   advisor.ExperienceIntermediate,
			wantPresetID: "nvim:balanced",
			wantHasRecs:  true,
		},
		{
			name:         "nvim advanced",
			category:     "nvim",
			experience:   advisor.ExperienceAdvanced,
			wantPresetID: "nvim:pro",
			wantHasRecs:  true,
		},
		{
			name:         "shell category",
			category:     "shell",
			experience:   advisor.ExperienceIntermediate,
			wantPresetID: "shell:starship",
			wantHasRecs:  true,
		},
		{
			name:         "unknown category defaults",
			category:     "unknown",
			experience:   advisor.ExperienceIntermediate,
			wantPresetID: "unknown:default",
			wantHasRecs:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adv := NewAdvisor()
			profile, err := advisor.NewUserProfile(tt.experience)
			require.NoError(t, err)

			ctx, err := advisor.NewSuggestContext(tt.category, profile)
			require.NoError(t, err)

			suggestion, err := adv.Suggest(context.Background(), ctx)

			require.NoError(t, err)
			assert.Equal(t, "noop", suggestion.Provider())
			assert.Equal(t, tt.wantHasRecs, suggestion.HasRecommendations())

			if tt.wantHasRecs {
				recs := suggestion.Recommendations()
				assert.NotEmpty(t, recs)
				assert.Equal(t, tt.wantPresetID, recs[0].PresetID())
			}
		})
	}
}

func TestAdvisor_Explain(t *testing.T) {
	t.Parallel()

	adv := NewAdvisor()
	req, _ := advisor.NewExplainRequest("nvim:balanced")

	explanation, err := adv.Explain(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", explanation.PresetID())
}

func TestAdvisor_ImplementsAdvisor(t *testing.T) {
	t.Parallel()

	var _ advisor.Advisor = (*Advisor)(nil)
}
