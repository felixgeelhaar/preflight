package advisor

import (
	"errors"
	"fmt"
	"strings"
)

// Recommendation errors.
var (
	ErrEmptyPresetID     = errors.New("preset ID cannot be empty")
	ErrEmptyRationale    = errors.New("rationale cannot be empty")
	ErrInvalidConfidence = errors.New("invalid confidence level")
)

// ConfidenceLevel indicates how confident the AI is in its recommendation.
type ConfidenceLevel string

// Confidence level constants.
const (
	ConfidenceLow    ConfidenceLevel = "low"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceHigh   ConfidenceLevel = "high"
)

// String returns the confidence level as a string.
func (c ConfidenceLevel) String() string {
	return string(c)
}

// IsValid returns true if this is a known confidence level.
func (c ConfidenceLevel) IsValid() bool {
	switch c {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		return true
	default:
		return false
	}
}

// ParseConfidenceLevel parses a string into a ConfidenceLevel.
func ParseConfidenceLevel(s string) (ConfidenceLevel, error) {
	level := ConfidenceLevel(strings.ToLower(s))
	if !level.IsValid() {
		return "", fmt.Errorf("%w: %s", ErrInvalidConfidence, s)
	}
	return level, nil
}

// Recommendation represents an AI-generated suggestion for a preset or configuration.
// It is an immutable value object.
type Recommendation struct {
	presetID     string
	rationale    string
	confidence   ConfidenceLevel
	tradeoffs    []string
	alternatives []string
	docLinks     map[string]string
}

// NewRecommendation creates a new Recommendation.
func NewRecommendation(presetID, rationale string, confidence ConfidenceLevel) (Recommendation, error) {
	if presetID == "" {
		return Recommendation{}, ErrEmptyPresetID
	}

	if rationale == "" {
		return Recommendation{}, ErrEmptyRationale
	}

	return Recommendation{
		presetID:     presetID,
		rationale:    rationale,
		confidence:   confidence,
		tradeoffs:    []string{},
		alternatives: []string{},
		docLinks:     map[string]string{},
	}, nil
}

// PresetID returns the recommended preset identifier.
func (r Recommendation) PresetID() string {
	return r.presetID
}

// Rationale returns the explanation for this recommendation.
func (r Recommendation) Rationale() string {
	return r.rationale
}

// Confidence returns the confidence level.
func (r Recommendation) Confidence() ConfidenceLevel {
	return r.confidence
}

// Tradeoffs returns the list of tradeoffs to consider.
func (r Recommendation) Tradeoffs() []string {
	result := make([]string, len(r.tradeoffs))
	copy(result, r.tradeoffs)
	return result
}

// Alternatives returns alternative preset IDs to consider.
func (r Recommendation) Alternatives() []string {
	result := make([]string, len(r.alternatives))
	copy(result, r.alternatives)
	return result
}

// DocLinks returns documentation links.
func (r Recommendation) DocLinks() map[string]string {
	result := make(map[string]string, len(r.docLinks))
	for k, v := range r.docLinks {
		result[k] = v
	}
	return result
}

// WithTradeoffs returns a new Recommendation with tradeoffs set.
func (r Recommendation) WithTradeoffs(tradeoffs []string) Recommendation {
	newTradeoffs := make([]string, len(tradeoffs))
	copy(newTradeoffs, tradeoffs)

	return Recommendation{
		presetID:     r.presetID,
		rationale:    r.rationale,
		confidence:   r.confidence,
		tradeoffs:    newTradeoffs,
		alternatives: r.alternatives,
		docLinks:     r.docLinks,
	}
}

// WithAlternatives returns a new Recommendation with alternatives set.
func (r Recommendation) WithAlternatives(alternatives []string) Recommendation {
	newAlternatives := make([]string, len(alternatives))
	copy(newAlternatives, alternatives)

	return Recommendation{
		presetID:     r.presetID,
		rationale:    r.rationale,
		confidence:   r.confidence,
		tradeoffs:    r.tradeoffs,
		alternatives: newAlternatives,
		docLinks:     r.docLinks,
	}
}

// WithDocLinks returns a new Recommendation with doc links set.
func (r Recommendation) WithDocLinks(links map[string]string) Recommendation {
	newLinks := make(map[string]string, len(links))
	for k, v := range links {
		newLinks[k] = v
	}

	return Recommendation{
		presetID:     r.presetID,
		rationale:    r.rationale,
		confidence:   r.confidence,
		tradeoffs:    r.tradeoffs,
		alternatives: r.alternatives,
		docLinks:     newLinks,
	}
}

// IsZero returns true if this is a zero-value Recommendation.
func (r Recommendation) IsZero() bool {
	return r.presetID == ""
}

// String returns a summary string.
func (r Recommendation) String() string {
	return fmt.Sprintf("%s (%s confidence)", r.presetID, r.confidence)
}
