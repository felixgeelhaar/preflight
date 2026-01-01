package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "good status",
			status:   "good",
			expected: "✓",
		},
		{
			name:     "warning status",
			status:   "warning",
			expected: "⚠",
		},
		{
			name:     "needs attention status",
			status:   "needs_attention",
			expected: "⛔",
		},
		{
			name:     "unknown status",
			status:   "unknown",
			expected: "○",
		},
		{
			name:     "empty status",
			status:   "",
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
		priority string
		contains string
	}{
		{
			name:     "high priority - red",
			priority: "high",
			contains: "\033[91m", // Red ANSI code
		},
		{
			name:     "medium priority - yellow",
			priority: "medium",
			contains: "\033[93m", // Yellow ANSI code
		},
		{
			name:     "low priority - green",
			priority: "low",
			contains: "\033[32m", // Green ANSI code
		},
		{
			name:     "unknown priority - plain bullet",
			priority: "unknown",
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
