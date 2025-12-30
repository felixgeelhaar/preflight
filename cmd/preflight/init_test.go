package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestInitCmd_Exists and TestInitCmd_HasFlags are in helpers_test.go
// Note: TestDetectAIProvider_* tests are in root_test.go

func TestInitCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"provider default", "provider", ""},
		{"preset default", "preset", ""},
		{"skip-welcome default", "skip-welcome", "false"},
		{"yes default", "yes", "false"},
		{"no-ai default", "no-ai", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := initCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestInitCmd_YesShorthand(t *testing.T) {
	t.Parallel()

	f := initCmd.Flags().Lookup("yes")
	assert.NotNil(t, f)
	assert.Equal(t, "y", f.Shorthand)
}
