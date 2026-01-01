package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected Severity
		wantErr  bool
	}{
		{"critical", "critical", SeverityCritical, false},
		{"Critical (capitalized)", "Critical", SeverityCritical, false},
		{"CRITICAL (uppercase)", "CRITICAL", SeverityCritical, false},
		{"high", "high", SeverityHigh, false},
		{"medium", "medium", SeverityMedium, false},
		{"moderate (alias)", "moderate", SeverityMedium, false},
		{"low", "low", SeverityLow, false},
		{"negligible", "negligible", SeverityNegligible, false},
		{"none (alias)", "none", SeverityNegligible, false},
		{"informational (alias)", "informational", SeverityNegligible, false},
		{"unknown", "unknown", SeverityUnknown, false},
		{"empty string", "", SeverityUnknown, false},
		{"invalid", "invalid", SeverityUnknown, true},
		{"whitespace", "  high  ", SeverityHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseSeverity(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_Order(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity Severity
		order    int
	}{
		{SeverityCritical, 5},
		{SeverityHigh, 4},
		{SeverityMedium, 3},
		{SeverityLow, 2},
		{SeverityNegligible, 1},
		{SeverityUnknown, 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.order, tt.severity.Order())
		})
	}
}

func TestSeverity_IsHigherThan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        Severity
		b        Severity
		expected bool
	}{
		{"critical > high", SeverityCritical, SeverityHigh, true},
		{"high > medium", SeverityHigh, SeverityMedium, true},
		{"medium > low", SeverityMedium, SeverityLow, true},
		{"low > negligible", SeverityLow, SeverityNegligible, true},
		{"high not > critical", SeverityHigh, SeverityCritical, false},
		{"same severity", SeverityHigh, SeverityHigh, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.a.IsHigherThan(tt.b))
		})
	}
}

func TestSeverity_IsAtLeast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		severity  Severity
		threshold Severity
		expected  bool
	}{
		{"critical at least high", SeverityCritical, SeverityHigh, true},
		{"high at least high", SeverityHigh, SeverityHigh, true},
		{"medium at least high", SeverityMedium, SeverityHigh, false},
		{"low at least medium", SeverityLow, SeverityMedium, false},
		{"critical at least critical", SeverityCritical, SeverityCritical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.severity.IsAtLeast(tt.threshold))
		})
	}
}

func TestAllSeverities(t *testing.T) {
	t.Parallel()

	severities := AllSeverities()
	assert.Len(t, severities, 5)
	assert.Equal(t, SeverityCritical, severities[0])
	assert.Equal(t, SeverityNegligible, severities[4])
}

func TestValidSeverityStrings(t *testing.T) {
	t.Parallel()

	strings := ValidSeverityStrings()
	assert.Contains(t, strings, "critical")
	assert.Contains(t, strings, "high")
	assert.Contains(t, strings, "medium")
	assert.Contains(t, strings, "low")
	assert.Contains(t, strings, "negligible")
}
