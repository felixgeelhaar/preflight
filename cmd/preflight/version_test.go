package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestVersionCmd_Exists is in helpers_test.go

func TestVersionCmd_HasShort(t *testing.T) {
	t.Parallel()

	assert.Contains(t, versionCmd.Short, "version")
}

func TestVersionVariables(t *testing.T) {
	t.Parallel()

	// Verify version variables exist and have default values
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, commit)
	assert.NotEmpty(t, buildDate)
}

func TestVersionCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	assert.True(t, found, "version should be a subcommand of root")
}
