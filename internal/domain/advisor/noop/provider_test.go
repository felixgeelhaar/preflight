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

func TestNewNoopAdvisor(t *testing.T) {
	t.Parallel()

	adv := NewNoopAdvisor()

	assert.NotNil(t, adv)
}

func TestNoopAdvisor_Suggest(t *testing.T) {
	t.Parallel()

	adv := NewNoopAdvisor()
	profile, _ := advisor.NewUserProfile(advisor.ExperienceIntermediate)
	ctx, _ := advisor.NewSuggestContext("nvim", profile)

	suggestion, err := adv.Suggest(context.Background(), ctx)

	require.NoError(t, err)
	assert.Equal(t, "noop", suggestion.Provider())
	// Noop advisor returns a default recommendation
	assert.True(t, suggestion.HasRecommendations())
}

func TestNoopAdvisor_Explain(t *testing.T) {
	t.Parallel()

	adv := NewNoopAdvisor()
	req, _ := advisor.NewExplainRequest("nvim:balanced")

	explanation, err := adv.Explain(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", explanation.PresetID())
}

func TestNoopAdvisor_ImplementsAdvisor(t *testing.T) {
	t.Parallel()

	var _ advisor.Advisor = (*NoopAdvisor)(nil)
}
