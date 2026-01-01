package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
)

func TestOutdatedCommand_Flags(t *testing.T) {
	t.Parallel()

	// Verify command exists and has expected flags
	assert.NotNil(t, outdatedCmd)
	assert.Equal(t, "outdated [packages...]", outdatedCmd.Use)

	// Check flags exist
	flags := outdatedCmd.Flags()
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("fail-on"))
	assert.NotNil(t, flags.Lookup("ignore"))
	assert.NotNil(t, flags.Lookup("json"))
	assert.NotNil(t, flags.Lookup("quiet"))
}

func TestParseUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected security.UpdateType
	}{
		{"major", security.UpdateMajor},
		{"MAJOR", security.UpdateMajor},
		{"Minor", security.UpdateMinor},
		{"minor", security.UpdateMinor},
		{"patch", security.UpdatePatch},
		{"PATCH", security.UpdatePatch},
		{"invalid", security.UpdateMinor}, // Default to minor
		{"", security.UpdateMinor},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := parseUpdateType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldFailOutdated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		packages   security.OutdatedPackages
		failOn     security.UpdateType
		shouldFail bool
	}{
		{
			name:       "no packages",
			packages:   security.OutdatedPackages{},
			failOn:     security.UpdateMajor,
			shouldFail: false,
		},
		{
			name: "major when fail-on major",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMajor},
			},
			failOn:     security.UpdateMajor,
			shouldFail: true,
		},
		{
			name: "minor when fail-on major",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMinor},
			},
			failOn:     security.UpdateMajor,
			shouldFail: false,
		},
		{
			name: "minor when fail-on minor",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdateMinor},
			},
			failOn:     security.UpdateMinor,
			shouldFail: true,
		},
		{
			name: "patch when fail-on minor",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdatePatch},
			},
			failOn:     security.UpdateMinor,
			shouldFail: false,
		},
		{
			name: "patch when fail-on patch",
			packages: security.OutdatedPackages{
				{Name: "pkg", UpdateType: security.UpdatePatch},
			},
			failOn:     security.UpdatePatch,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &security.OutdatedResult{
				Packages: tt.packages,
			}

			got := shouldFailOutdated(result, tt.failOn)
			assert.Equal(t, tt.shouldFail, got)
		})
	}
}

func TestToOutdatedPackagesJSON(t *testing.T) {
	t.Parallel()

	packages := security.OutdatedPackages{
		{
			Name:           "go",
			CurrentVersion: "1.21.0",
			LatestVersion:  "1.22.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
			Pinned:         false,
		},
		{
			Name:           "python",
			CurrentVersion: "3.11.0",
			LatestVersion:  "3.12.0",
			UpdateType:     security.UpdateMinor,
			Provider:       "brew",
			Pinned:         true,
		},
	}

	result := toOutdatedPackagesJSON(packages)

	assert.Len(t, result, 2)

	assert.Equal(t, "go", result[0].Name)
	assert.Equal(t, "1.21.0", result[0].CurrentVersion)
	assert.Equal(t, "1.22.0", result[0].LatestVersion)
	assert.Equal(t, "minor", result[0].UpdateType)
	assert.Equal(t, "brew", result[0].Provider)
	assert.False(t, result[0].Pinned)

	assert.Equal(t, "python", result[1].Name)
	assert.True(t, result[1].Pinned)
}

func TestFormatUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		updateType security.UpdateType
		contains   string
	}{
		{security.UpdateMajor, "MAJOR"},
		{security.UpdateMinor, "MINOR"},
		{security.UpdatePatch, "PATCH"},
		{security.UpdateUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.updateType), func(t *testing.T) {
			t.Parallel()

			result := formatUpdateType(tt.updateType)
			assert.Contains(t, result, tt.contains)
		})
	}
}
