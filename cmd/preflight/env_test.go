package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Note: TestEnvCmd_Exists and TestEnvCmd_HasSubcommands are in helpers_test.go
// Note: TestExtractEnvVars, TestWriteEnvFile, etc. are in helpers_test.go

func TestEnvVarStruct_Fields(t *testing.T) {
	t.Parallel()

	v := EnvVar{
		Name:   "TEST_VAR",
		Value:  "test_value",
		Layer:  "base",
		Secret: true,
	}

	assert.Equal(t, "TEST_VAR", v.Name)
	assert.Equal(t, "test_value", v.Value)
	assert.Equal(t, "base", v.Layer)
	assert.True(t, v.Secret)
}

func TestEnvCmd_IsSubcommandOfRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "env" {
			found = true
			break
		}
	}
	assert.True(t, found, "env should be a subcommand of root")
}

func TestEnvShowCmd_HasJSONFlag(t *testing.T) {
	t.Parallel()

	// Find the show subcommand
	var showCmd *cobra.Command
	for _, cmd := range envCmd.Commands() {
		if cmd.Name() == "show" {
			showCmd = cmd
			break
		}
	}

	if showCmd != nil {
		f := showCmd.Flags().Lookup("json")
		assert.NotNil(t, f, "show command should have json flag")
	}
}
