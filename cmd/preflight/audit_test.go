package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
)

func TestTruncateStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string truncated",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateStr(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverityIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity audit.Severity
		expected string
	}{
		{
			name:     "critical severity",
			severity: audit.SeverityCritical,
			expected: "⛔ critical",
		},
		{
			name:     "error severity",
			severity: audit.SeverityError,
			expected: "❌ error",
		},
		{
			name:     "warning severity",
			severity: audit.SeverityWarning,
			expected: "⚠️  warning",
		},
		{
			name:     "info severity",
			severity: audit.SeverityInfo,
			expected: "ℹ️  info",
		},
		{
			name:     "unknown severity",
			severity: audit.Severity("unknown"),
			expected: "ℹ️  info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := severityIcon(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFilter_Default(t *testing.T) {
	// Test with default flag values (package-level variables)
	// Reset flags before test
	oldLimit := auditLimit
	oldDays := auditDays
	oldEventType := auditEventType
	oldSeverity := auditSeverity
	oldCatalog := auditCatalog
	oldPlugin := auditPlugin
	oldUser := auditUser
	oldFailures := auditFailures
	oldSuccesses := auditSuccesses
	defer func() {
		auditLimit = oldLimit
		auditDays = oldDays
		auditEventType = oldEventType
		auditSeverity = oldSeverity
		auditCatalog = oldCatalog
		auditPlugin = oldPlugin
		auditUser = oldUser
		auditFailures = oldFailures
		auditSuccesses = oldSuccesses
	}()

	// Test default (empty) filter
	auditLimit = 0
	auditDays = 0
	auditEventType = ""
	auditSeverity = ""
	auditCatalog = ""
	auditPlugin = ""
	auditUser = ""
	auditFailures = false
	auditSuccesses = false

	filter := buildFilter()
	assert.Empty(t, filter.EventTypes)
	assert.Equal(t, 0, filter.Limit)
}

func TestBuildFilter_WithValues(t *testing.T) {
	// Reset flags before test
	oldLimit := auditLimit
	oldDays := auditDays
	oldEventType := auditEventType
	oldSeverity := auditSeverity
	oldCatalog := auditCatalog
	oldPlugin := auditPlugin
	oldUser := auditUser
	oldFailures := auditFailures
	oldSuccesses := auditSuccesses
	defer func() {
		auditLimit = oldLimit
		auditDays = oldDays
		auditEventType = oldEventType
		auditSeverity = oldSeverity
		auditCatalog = oldCatalog
		auditPlugin = oldPlugin
		auditUser = oldUser
		auditFailures = oldFailures
		auditSuccesses = oldSuccesses
	}()

	// Test with specific values
	auditLimit = 10
	auditDays = 7
	auditEventType = "catalog_installed"
	auditSeverity = "warning"
	auditCatalog = "test-catalog"
	auditPlugin = "test-plugin"
	auditUser = "test-user"
	auditFailures = true
	auditSuccesses = false

	filter := buildFilter()
	assert.Equal(t, 10, filter.Limit)
	assert.False(t, filter.Since.IsZero())
	assert.Contains(t, filter.EventTypes, audit.EventCatalogInstalled)
	assert.Contains(t, filter.Severities, audit.SeverityWarning)
	assert.Equal(t, "test-catalog", filter.Catalog)
	assert.Equal(t, "test-plugin", filter.Plugin)
	assert.Equal(t, "test-user", filter.User)
	assert.True(t, filter.FailuresOnly)
}

func TestBuildFilter_SuccessesOnly(t *testing.T) {
	// Reset flags before test
	oldSuccesses := auditSuccesses
	oldFailures := auditFailures
	defer func() {
		auditSuccesses = oldSuccesses
		auditFailures = oldFailures
	}()

	auditSuccesses = true
	auditFailures = false

	filter := buildFilter()
	assert.True(t, filter.SuccessOnly)
}
