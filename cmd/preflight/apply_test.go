package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestApplyCmd_Exists and TestApplyCmd_HasFlags are in helpers_test.go

func TestApplyCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
		{"dry-run default", "dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := applyCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestApplyCmd_ConfigShorthand(t *testing.T) {
	t.Parallel()

	f := applyCmd.Flags().Lookup("config")
	assert.NotNil(t, f)
	assert.Equal(t, "c", f.Shorthand)
}

func TestApplyCmd_TargetShorthand(t *testing.T) {
	t.Parallel()

	f := applyCmd.Flags().Lookup("target")
	assert.NotNil(t, f)
	assert.Equal(t, "t", f.Shorthand)
}

func TestApplyCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "apply" {
			found = true
			break
		}
	}
	assert.True(t, found, "apply should be a subcommand of root")
}
