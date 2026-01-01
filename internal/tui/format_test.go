package tui

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
)

func TestFormatStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   advisor.AnalysisStatus
		expected string
	}{
		{
			name:     "good status",
			status:   advisor.StatusGood,
			expected: "✓",
		},
		{
			name:     "warning status",
			status:   advisor.StatusWarning,
			expected: "⚠",
		},
		{
			name:     "needs attention status",
			status:   advisor.StatusNeedsAttention,
			expected: "⛔",
		},
		{
			name:     "unknown status",
			status:   advisor.AnalysisStatus("unknown"),
			expected: "○",
		},
		{
			name:     "empty status",
			status:   advisor.AnalysisStatus(""),
			expected: "○",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatStatusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPriorityPrefix(t *testing.T) {
	tests := []struct {
		name     string
		priority advisor.RecommendationPriority
		contains string
	}{
		{
			name:     "high priority - red",
			priority: advisor.PriorityHigh,
			contains: "\033[91m", // Red ANSI code
		},
		{
			name:     "medium priority - yellow",
			priority: advisor.PriorityMedium,
			contains: "\033[93m", // Yellow ANSI code
		},
		{
			name:     "low priority - green",
			priority: advisor.PriorityLow,
			contains: "\033[32m", // Green ANSI code
		},
		{
			name:     "unknown priority - plain bullet",
			priority: advisor.RecommendationPriority("unknown"),
			contains: "•",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPriorityPrefix(tt.priority)
			assert.Contains(t, result, tt.contains)
			assert.Contains(t, result, "•") // All should have bullet
		})
	}
}
