package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestCleanCmd_Exists and TestCleanCmd_HasFlags are in helpers_test.go

func TestCleanCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
		{"apply default", "apply", "false"},
		{"providers default", "providers", ""},
		{"ignore default", "ignore", ""},
		{"json default", "json", "false"},
		{"force default", "force", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := cleanCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f, "flag %s should exist", tt.flag)
			if f != nil {
				assert.Equal(t, tt.expected, f.DefValue)
			}
		})
	}
}

func TestCleanCmd_ConfigShorthand(t *testing.T) {
	t.Parallel()

	f := cleanCmd.Flags().Lookup("config")
	assert.NotNil(t, f)
	assert.Equal(t, "c", f.Shorthand)
}

func TestCleanCmd_TargetShorthand(t *testing.T) {
	t.Parallel()

	f := cleanCmd.Flags().Lookup("target")
	assert.NotNil(t, f)
	assert.Equal(t, "t", f.Shorthand)
}

func TestCleanCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "clean" {
			found = true
			break
		}
	}
	assert.True(t, found, "clean should be a subcommand of root")
}

// Note: TestOrphanedItemFields is in helpers_test.go
