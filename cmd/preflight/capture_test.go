package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestCaptureCmd_Exists and TestCaptureCmd_HasFlags are in helpers_test.go

func TestCaptureCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"all default", "all", "false"},
		{"provider default", "provider", ""},
		{"output default", "output", "."},
		{"target default", "target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := captureCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestCaptureCmd_OutputShorthand(t *testing.T) {
	t.Parallel()

	f := captureCmd.Flags().Lookup("output")
	assert.NotNil(t, f)
	assert.Equal(t, "o", f.Shorthand)
}

func TestCaptureCmd_TargetShorthand(t *testing.T) {
	t.Parallel()

	f := captureCmd.Flags().Lookup("target")
	assert.NotNil(t, f)
	assert.Equal(t, "t", f.Shorthand)
}

func TestCaptureCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "capture" {
			found = true
			break
		}
	}
	assert.True(t, found, "capture should be a subcommand of root")
}
