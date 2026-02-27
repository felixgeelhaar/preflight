package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestOutputEventsTable(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	now := time.Date(2026, 2, 24, 14, 30, 0, 0, time.UTC)

	events := []audit.Event{
		{
			ID:        "evt-1",
			Timestamp: now,
			Type:      audit.EventCatalogInstalled,
			Severity:  audit.SeverityInfo,
			Catalog:   "my-catalog",
			Success:   true,
		},
		{
			ID:        "evt-2",
			Timestamp: now.Add(time.Minute),
			Type:      audit.EventPluginExecuted,
			Severity:  audit.SeverityWarning,
			Plugin:    "my-plugin",
			Success:   false,
		},
		{
			ID:        "evt-3",
			Timestamp: now.Add(2 * time.Minute),
			Type:      audit.EventCapabilityDenied,
			Severity:  audit.SeverityError,
			Success:   true,
		},
	}

	output := captureStdout(t, func() {
		_ = outputEventsTable(events)
	})

	// Verify headers
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "EVENT")
	assert.Contains(t, output, "SEVERITY")
	assert.Contains(t, output, "SUBJECT")
	assert.Contains(t, output, "STATUS")

	// Verify catalog subject
	assert.Contains(t, output, "my-catalog")
	// Verify plugin subject
	assert.Contains(t, output, "my-plugin")
	// Verify bare subject falls back to "-"
	assert.Contains(t, output, "-")

	// Verify success/failure markers
	assert.Contains(t, output, string(rune(0x2713))) // checkmark
	assert.Contains(t, output, string(rune(0x2717))) // X mark

	// Verify event types appear
	assert.Contains(t, output, string(audit.EventCatalogInstalled))
	assert.Contains(t, output, string(audit.EventPluginExecuted))

	// Verify footer
	assert.Contains(t, output, "Showing 3 events")
}

func TestOutputSecurityEventsTable(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	now := time.Date(2026, 2, 24, 14, 30, 0, 0, time.UTC)

	events := []audit.Event{
		{
			ID:        "sec-1",
			Timestamp: now,
			Type:      audit.EventSandboxViolation,
			Severity:  audit.SeverityCritical,
			Plugin:    "bad-plugin",
			Error:     "tried to access /etc/passwd",
			Success:   false,
		},
		{
			ID:        "sec-2",
			Timestamp: now.Add(time.Minute),
			Type:      audit.EventCapabilityDenied,
			Severity:  audit.SeverityWarning,
			Catalog:   "untrusted-catalog",
			CapabilitiesDenied: []string{"network", "filesystem"},
			Success:   false,
		},
		{
			ID:        "sec-3",
			Timestamp: now.Add(2 * time.Minute),
			Type:      audit.EventSecurityAudit,
			Severity:  audit.SeverityError,
			Plugin:    "audit-target",
			Details:   map[string]interface{}{"violation": "unsigned binary detected"},
			Success:   false,
		},
	}

	output := captureStdout(t, func() {
		_ = outputSecurityEventsTable(events)
	})

	// Verify headers
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "EVENT")
	assert.Contains(t, output, "SEVERITY")
	assert.Contains(t, output, "SUBJECT")
	assert.Contains(t, output, "DETAILS")

	// Verify severity icons appear
	assert.Contains(t, output, "critical")
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "error")

	// Verify error detail
	assert.Contains(t, output, "tried to access /etc/passwd")

	// Verify denied capabilities detail
	assert.Contains(t, output, "denied: network, filesystem")

	// Verify details["violation"] content
	assert.Contains(t, output, "unsigned binary detected")

	// Verify footer
	assert.Contains(t, output, "Showing 3 security events")
}

func TestOutputEventsJSON(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	now := time.Date(2026, 2, 24, 14, 30, 0, 0, time.UTC)

	events := []audit.Event{
		{
			ID:        "json-1",
			Timestamp: now,
			Type:      audit.EventCatalogInstalled,
			Severity:  audit.SeverityInfo,
			Catalog:   "test-catalog",
			Success:   true,
		},
		{
			ID:        "json-2",
			Timestamp: now.Add(time.Minute),
			Type:      audit.EventPluginExecuted,
			Severity:  audit.SeverityError,
			Plugin:    "test-plugin",
			Success:   false,
			Error:     "execution failed",
		},
	}

	output := captureStdout(t, func() {
		err := outputEventsJSON(events)
		require.NoError(t, err)
	})

	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	// Verify first event fields
	assert.Equal(t, "json-1", parsed[0]["id"])
	assert.Equal(t, string(audit.EventCatalogInstalled), parsed[0]["event"])
	assert.Equal(t, string(audit.SeverityInfo), parsed[0]["severity"])
	assert.Equal(t, "test-catalog", parsed[0]["catalog"])
	assert.Equal(t, true, parsed[0]["success"])

	// Verify second event fields
	assert.Equal(t, "json-2", parsed[1]["id"])
	assert.Equal(t, string(audit.EventPluginExecuted), parsed[1]["event"])
	assert.Equal(t, "test-plugin", parsed[1]["plugin"])
	assert.Equal(t, false, parsed[1]["success"])
	assert.Equal(t, "execution failed", parsed[1]["error"])
}

func TestOutputJSON(t *testing.T) {
	// Do not use t.Parallel() - this test captures stdout.
	data := map[string]interface{}{
		"key":   "value",
		"count": float64(42),
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	output := captureStdout(t, func() {
		err := outputJSON(data)
		require.NoError(t, err)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "value", parsed["key"])
	assert.Equal(t, float64(42), parsed["count"])
	nested := parsed["nested"].(map[string]interface{})
	assert.Equal(t, "data", nested["inner"])
}
