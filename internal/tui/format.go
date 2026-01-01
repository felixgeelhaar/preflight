// Package tui provides formatting utilities for terminal output.
package tui

import "github.com/felixgeelhaar/preflight/internal/domain/advisor"

// FormatStatusIcon returns a display icon for the given analysis status.
// This is a presentation concern that belongs in the TUI layer.
func FormatStatusIcon(status advisor.AnalysisStatus) string {
	switch status {
	case advisor.StatusGood:
		return "✓"
	case advisor.StatusWarning:
		return "⚠"
	case advisor.StatusNeedsAttention:
		return "⛔"
	default:
		return "○"
	}
}

// FormatPriorityPrefix returns a colored prefix for the given priority.
// Uses ANSI escape codes for terminal coloring.
func FormatPriorityPrefix(priority advisor.RecommendationPriority) string {
	switch priority {
	case advisor.PriorityHigh:
		return "\033[91m•\033[0m" // Red
	case advisor.PriorityMedium:
		return "\033[93m•\033[0m" // Yellow
	case advisor.PriorityLow:
		return "\033[32m•\033[0m" // Green
	default:
		return "•"
	}
}
