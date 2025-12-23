// Package noop provides a no-op AI provider for when AI features are disabled.
package noop

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// Provider is a no-op AI provider that returns empty responses.
type Provider struct{}

// NewProvider creates a new noop Provider.
func NewProvider() *Provider {
	return &Provider{}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "noop"
}

// Complete returns an empty response.
func (p *Provider) Complete(_ context.Context, _ advisor.Prompt) (advisor.Response, error) {
	return advisor.NewResponse("", 0, "noop"), nil
}

// Available always returns true for the noop provider.
func (p *Provider) Available() bool {
	return true
}

// Advisor is an Advisor implementation that works without AI.
// It provides default recommendations based on the catalog.
type Advisor struct{}

// NewAdvisor creates a new noop Advisor.
func NewAdvisor() *Advisor {
	return &Advisor{}
}

// Suggest returns default recommendations without AI.
func (a *Advisor) Suggest(_ context.Context, suggestCtx advisor.SuggestContext) (advisor.Suggestion, error) {
	// Return a sensible default based on category and experience
	category := suggestCtx.Category()
	experience := suggestCtx.UserProfile().Experience()

	var presetID string
	var rationale string
	var confidence advisor.ConfidenceLevel

	switch category {
	case "nvim":
		switch experience {
		case advisor.ExperienceBeginner:
			presetID = "nvim:minimal"
			rationale = "Minimal preset recommended for beginners - easy to learn"
		case advisor.ExperienceIntermediate:
			presetID = "nvim:balanced"
			rationale = "Balanced preset provides good features without complexity"
		case advisor.ExperienceAdvanced:
			presetID = "nvim:pro"
			rationale = "Pro preset for advanced users with full IDE features"
		}
	case "shell":
		presetID = "shell:starship"
		rationale = "Starship prompt works across shells and is easy to customize"
	default:
		presetID = fmt.Sprintf("%s:default", category)
		rationale = fmt.Sprintf("Default preset for %s", category)
	}
	confidence = advisor.ConfidenceMedium

	rec, err := advisor.NewRecommendation(presetID, rationale, confidence)
	if err != nil {
		return advisor.Suggestion{}, err
	}

	return advisor.NewSuggestion("noop", []advisor.Recommendation{rec})
}

// Explain returns a basic explanation without AI.
func (a *Advisor) Explain(_ context.Context, req advisor.ExplainRequest) (advisor.DetailedExplanation, error) {
	summary := fmt.Sprintf("This is the %s preset. AI explanations are disabled.", req.PresetID())

	return advisor.NewDetailedExplanation(req.PresetID(), summary)
}
