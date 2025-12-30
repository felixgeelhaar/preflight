package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestDoctorCmd_Exists and TestDoctorCmd_HasFlags are in helpers_test.go

func TestDoctorCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"fix default", "fix", "false"},
		{"verbose default", "verbose", "false"},
		{"update-config default", "update-config", "false"},
		{"dry-run default", "dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := doctorCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestDoctorCmd_VerboseShorthand(t *testing.T) {
	t.Parallel()

	f := doctorCmd.Flags().Lookup("verbose")
	assert.NotNil(t, f)
	assert.Equal(t, "v", f.Shorthand)
}

func TestDoctorCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "doctor" {
			found = true
			break
		}
	}
	assert.True(t, found, "doctor should be a subcommand of root")
}
