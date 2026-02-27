package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeprecatedCommand_Flags(t *testing.T) {
	t.Parallel()

	// Verify command exists and has expected flags
	assert.NotNil(t, deprecatedCmd)
	assert.Equal(t, "deprecated", deprecatedCmd.Use)

	// Check flags exist
	flags := deprecatedCmd.Flags()
	assert.NotNil(t, flags.Lookup("ignore"))
	assert.NotNil(t, flags.Lookup("json"))
	assert.NotNil(t, flags.Lookup("quiet"))
}

func TestToDeprecatedPackagesJSON(t *testing.T) {
	t.Parallel()

	date := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)

	packages := security.DeprecatedPackages{
		{
			Name:        "python@2",
			Version:     "2.7.18",
			Provider:    "brew",
			Reason:      security.ReasonDeprecated,
			Date:        &date,
			Alternative: "python@3",
			Message:     "uses Python 2",
		},
		{
			Name:     "old-tool",
			Version:  "1.0.0",
			Provider: "brew",
			Reason:   security.ReasonDisabled,
			Message:  "no longer maintained",
		},
	}

	result := toDeprecatedPackagesJSON(packages)

	assert.Len(t, result, 2)

	// Check deprecated package
	assert.Equal(t, "python@2", result[0].Name)
	assert.Equal(t, "2.7.18", result[0].Version)
	assert.Equal(t, "brew", result[0].Provider)
	assert.Equal(t, "deprecated", result[0].Reason)
	assert.Equal(t, "2023-06-15", result[0].Date)
	assert.Equal(t, "python@3", result[0].Alternative)
	assert.Equal(t, "uses Python 2", result[0].Message)

	// Check disabled package
	assert.Equal(t, "old-tool", result[1].Name)
	assert.Equal(t, "disabled", result[1].Reason)
	assert.Empty(t, result[1].Date)
}

func TestFormatDeprecationStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason   security.DeprecationReason
		contains string
	}{
		{security.ReasonDisabled, "DISABLED"},
		{security.ReasonDeprecated, "DEPRECATED"},
		{security.ReasonEOL, "EOL"},
		{security.ReasonUnmaintained, "UNMAINTAINED"},
		{security.DeprecationReason("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()

			result := formatDeprecationStatus(tt.reason)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestOutputDeprecatedJSON_WithResults(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{
				Name:        "python@2",
				Version:     "2.7.18",
				Provider:    "brew",
				Reason:      security.ReasonDeprecated,
				Date:        &date,
				Alternative: "python@3",
				Message:     "uses Python 2",
			},
			{
				Name:     "old-tool",
				Version:  "1.0.0",
				Provider: "brew",
				Reason:   security.ReasonDisabled,
				Message:  "no longer maintained",
			},
		},
	}

	output := captureStdout(t, func() {
		outputDeprecatedJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "brew", parsed["checker"])
	assert.Empty(t, parsed["error"])

	packages, ok := parsed["packages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, packages, 2)

	pkg0 := packages[0].(map[string]interface{})
	assert.Equal(t, "python@2", pkg0["name"])
	assert.Equal(t, "2.7.18", pkg0["version"])
	assert.Equal(t, "brew", pkg0["provider"])
	assert.Equal(t, "deprecated", pkg0["reason"])
	assert.Equal(t, "2024-01-15", pkg0["date"])
	assert.Equal(t, "python@3", pkg0["alternative"])
	assert.Equal(t, "uses Python 2", pkg0["message"])

	summary, ok := parsed["summary"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), summary["total"])
	assert.Equal(t, float64(1), summary["deprecated"])
	assert.Equal(t, float64(1), summary["disabled"])
}

func TestOutputDeprecatedJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputDeprecatedJSON(nil, assert.AnError)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, assert.AnError.Error(), parsed["error"])
	assert.Nil(t, parsed["checker"])
	assert.Nil(t, parsed["packages"])
	assert.Nil(t, parsed["summary"])
}

func TestOutputDeprecatedText_NoPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker:  "brew",
		Packages: security.DeprecatedPackages{},
	}

	output := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, output, "No deprecated packages found")
	assert.Contains(t, output, "brew")
}

func TestOutputDeprecatedText_WithPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{
				Name:     "python@2",
				Version:  "2.7.18",
				Provider: "brew",
				Reason:   security.ReasonDeprecated,
				Message:  "uses Python 2",
			},
			{
				Name:     "old-tool",
				Version:  "1.0.0",
				Provider: "brew",
				Reason:   security.ReasonDisabled,
				Message:  "no longer maintained",
			},
		},
	}

	output := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})

	assert.Contains(t, output, "Deprecated Packages Check (brew)")
	assert.Contains(t, output, "2 packages require attention")
	// Table headers
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "PACKAGE")
	assert.Contains(t, output, "VERSION")
	assert.Contains(t, output, "MESSAGE")
	// Package data
	assert.Contains(t, output, "python@2")
	assert.Contains(t, output, "2.7.18")
	assert.Contains(t, output, "old-tool")
	// Recommendations
	assert.Contains(t, output, "Recommendations")
	assert.Contains(t, output, "DISABLED")
	assert.Contains(t, output, "DEPRECATED")
}

func TestOutputDeprecatedText_Quiet(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{
				Name:     "old-tool",
				Version:  "1.0.0",
				Provider: "brew",
				Reason:   security.ReasonDeprecated,
				Message:  "deprecated package",
			},
		},
	}

	output := captureStdout(t, func() {
		outputDeprecatedText(result, true)
	})

	assert.Contains(t, output, "1 packages require attention")
	// In quiet mode, the table should not be printed
	assert.NotContains(t, output, "STATUS")
	assert.NotContains(t, output, "PACKAGE")
}

func TestPrintDeprecationSummaryBar(t *testing.T) {
	summary := security.DeprecatedSummary{
		Total:        4,
		Deprecated:   1,
		Disabled:     1,
		EOL:          1,
		Unmaintained: 1,
	}

	output := captureStdout(t, func() {
		printDeprecationSummaryBar(summary)
	})

	assert.Contains(t, output, "DISABLED: 1")
	assert.Contains(t, output, "DEPRECATED: 1")
	assert.Contains(t, output, "EOL: 1")
	assert.Contains(t, output, "UNMAINTAINED: 1")
}

func TestPrintDeprecatedTable(t *testing.T) {
	packages := security.DeprecatedPackages{
		{
			Name:     "python@2",
			Version:  "2.7.18",
			Provider: "brew",
			Reason:   security.ReasonDeprecated,
			Message:  "uses Python 2",
		},
		{
			Name:     "no-version",
			Provider: "brew",
			Reason:   security.ReasonDisabled,
		},
	}

	output := captureStdout(t, func() {
		printDeprecatedTable(packages)
	})

	// Verify column headers
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "PACKAGE")
	assert.Contains(t, output, "VERSION")
	assert.Contains(t, output, "MESSAGE")
	// Verify data rows
	assert.Contains(t, output, "python@2")
	assert.Contains(t, output, "2.7.18")
	assert.Contains(t, output, "uses Python 2")
	assert.Contains(t, output, "no-version")
	// Empty version/message should show "-"
	assert.Contains(t, output, "-")
}
