// Package advisor provides AI-powered recommendations and explanations.
// AI is advisory only - it never executes changes.
package advisor

import (
	"context"
	"errors"
)

// Advisor errors.
var (
	ErrEmptySummary = errors.New("summary cannot be empty")
)

// Advisor defines the interface for AI-powered recommendations and explanations.
// AI is advisory only - it never executes changes.
type Advisor interface {
	// Suggest generates recommendations based on the provided context.
	Suggest(ctx context.Context, suggestCtx SuggestContext) (Suggestion, error)

	// Explain provides a detailed explanation of a preset.
	Explain(ctx context.Context, req ExplainRequest) (DetailedExplanation, error)
}

// DetailedExplanation provides an in-depth explanation of a preset.
type DetailedExplanation struct {
	presetID     string
	summary      string
	sections     map[string]string
	bulletPoints []string
	codeExamples map[string]string
}

// NewDetailedExplanation creates a new DetailedExplanation.
func NewDetailedExplanation(presetID, summary string) (DetailedExplanation, error) {
	if presetID == "" {
		return DetailedExplanation{}, ErrEmptyPresetID
	}

	if summary == "" {
		return DetailedExplanation{}, ErrEmptySummary
	}

	return DetailedExplanation{
		presetID:     presetID,
		summary:      summary,
		sections:     map[string]string{},
		bulletPoints: []string{},
		codeExamples: map[string]string{},
	}, nil
}

// PresetID returns the preset being explained.
func (e DetailedExplanation) PresetID() string {
	return e.presetID
}

// Summary returns the main summary.
func (e DetailedExplanation) Summary() string {
	return e.summary
}

// Sections returns detailed sections (e.g., "Features", "Tradeoffs").
func (e DetailedExplanation) Sections() map[string]string {
	result := make(map[string]string, len(e.sections))
	for k, v := range e.sections {
		result[k] = v
	}
	return result
}

// BulletPoints returns key points as a list.
func (e DetailedExplanation) BulletPoints() []string {
	result := make([]string, len(e.bulletPoints))
	copy(result, e.bulletPoints)
	return result
}

// CodeExamples returns code examples (name -> code).
func (e DetailedExplanation) CodeExamples() map[string]string {
	result := make(map[string]string, len(e.codeExamples))
	for k, v := range e.codeExamples {
		result[k] = v
	}
	return result
}

// WithSections returns a new DetailedExplanation with sections set.
func (e DetailedExplanation) WithSections(sections map[string]string) DetailedExplanation {
	newSections := make(map[string]string, len(sections))
	for k, v := range sections {
		newSections[k] = v
	}

	return DetailedExplanation{
		presetID:     e.presetID,
		summary:      e.summary,
		sections:     newSections,
		bulletPoints: e.bulletPoints,
		codeExamples: e.codeExamples,
	}
}

// WithBulletPoints returns a new DetailedExplanation with bullet points set.
func (e DetailedExplanation) WithBulletPoints(bullets []string) DetailedExplanation {
	newBullets := make([]string, len(bullets))
	copy(newBullets, bullets)

	return DetailedExplanation{
		presetID:     e.presetID,
		summary:      e.summary,
		sections:     e.sections,
		bulletPoints: newBullets,
		codeExamples: e.codeExamples,
	}
}

// WithCodeExamples returns a new DetailedExplanation with code examples set.
func (e DetailedExplanation) WithCodeExamples(examples map[string]string) DetailedExplanation {
	newExamples := make(map[string]string, len(examples))
	for k, v := range examples {
		newExamples[k] = v
	}

	return DetailedExplanation{
		presetID:     e.presetID,
		summary:      e.summary,
		sections:     e.sections,
		bulletPoints: e.bulletPoints,
		codeExamples: newExamples,
	}
}

// IsZero returns true if this is a zero-value DetailedExplanation.
func (e DetailedExplanation) IsZero() bool {
	return e.presetID == ""
}
