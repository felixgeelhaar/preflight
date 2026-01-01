package main

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
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
