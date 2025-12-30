package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: TestTourCmd_Exists and TestTourCmd_HasListFlag are in helpers_test.go

func TestTourCmd_AcceptsTopicArg(t *testing.T) {
	t.Parallel()

	// Verify command accepts an optional topic argument
	assert.Equal(t, "tour [topic]", tourCmd.Use)
}

func TestTourCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "tour" {
			found = true
			break
		}
	}
	assert.True(t, found, "tour should be a subcommand of root")
}
