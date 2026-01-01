// Package security provides vulnerability scanning, outdated detection, and deprecation warnings.
package security

import (
	"fmt"
	"strings"
)

// Severity represents the severity level of a vulnerability.
type Severity string

const (
	// SeverityCritical represents critical severity (CVSS 9.0-10.0).
	SeverityCritical Severity = "critical"
	// SeverityHigh represents high severity (CVSS 7.0-8.9).
	SeverityHigh Severity = "high"
	// SeverityMedium represents medium severity (CVSS 4.0-6.9).
	SeverityMedium Severity = "medium"
	// SeverityLow represents low severity (CVSS 0.1-3.9).
	SeverityLow Severity = "low"
	// SeverityNegligible represents negligible/informational severity.
	SeverityNegligible Severity = "negligible"
	// SeverityUnknown represents unknown severity.
	SeverityUnknown Severity = "unknown"
)

// severityOrder defines the ordering of severities from most to least severe.
var severityOrder = map[Severity]int{
	SeverityCritical:   5,
	SeverityHigh:       4,
	SeverityMedium:     3,
	SeverityLow:        2,
	SeverityNegligible: 1,
	SeverityUnknown:    0,
}

// ParseSeverity converts a string to a Severity.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return SeverityCritical, nil
	case "high":
		return SeverityHigh, nil
	case "medium", "moderate":
		return SeverityMedium, nil
	case "low":
		return SeverityLow, nil
	case "negligible", "none", "informational":
		return SeverityNegligible, nil
	case "unknown", "":
		return SeverityUnknown, nil
	default:
		return SeverityUnknown, fmt.Errorf("unknown severity: %q", s)
	}
}

// String returns the string representation of the severity.
func (s Severity) String() string {
	return string(s)
}

// Order returns the numeric order of the severity (higher = more severe).
func (s Severity) Order() int {
	if order, ok := severityOrder[s]; ok {
		return order
	}
	return 0
}

// IsHigherThan returns true if this severity is higher than the other.
func (s Severity) IsHigherThan(other Severity) bool {
	return s.Order() > other.Order()
}

// IsAtLeast returns true if this severity is at least as severe as the threshold.
func (s Severity) IsAtLeast(threshold Severity) bool {
	return s.Order() >= threshold.Order()
}

// AllSeverities returns all severity levels in order from most to least severe.
func AllSeverities() []Severity {
	return []Severity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityNegligible,
	}
}

// ValidSeverityStrings returns valid severity strings for validation/help.
func ValidSeverityStrings() []string {
	return []string{"critical", "high", "medium", "low", "negligible"}
}
