package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestPlanCmd_Exists and TestPlanCmd_HasFlags are in helpers_test.go

func TestPlanCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"config default", "config", "preflight.yaml"},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := planCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestPlanCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "plan" {
			found = true
			break
		}
	}
	assert.True(t, found, "plan should be a subcommand of root")
}
