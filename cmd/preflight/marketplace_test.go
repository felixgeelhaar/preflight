package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatInstallAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 2 * 24 * time.Hour, "2d ago"},
		{"weeks", 2 * 7 * 24 * time.Hour, "2w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatInstallAge(time.Now().Add(-tt.duration))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFormatInstallAge_OldDate(t *testing.T) {
	t.Parallel()

	oldDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	result := formatInstallAge(oldDate)
	assert.Equal(t, "2024-01-15", result)
}

func TestMarketplaceCmd_Exists(t *testing.T) {
	t.Parallel()

	// Verify marketplace command exists
	assert.NotNil(t, marketplaceCmd)
	assert.Equal(t, "marketplace", marketplaceCmd.Use)
	assert.Contains(t, marketplaceCmd.Aliases, "mp")
	assert.Contains(t, marketplaceCmd.Aliases, "market")

	// Verify subcommands exist
	subcommands := marketplaceCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "search")
	assert.Contains(t, names, "install")
	assert.Contains(t, names, "uninstall")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "info")
}
