package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCommand_UseAndShort(t *testing.T) {
	assert.Equal(t, "validate", validateCmd.Use)
	assert.Equal(t, "Validate configuration without applying", validateCmd.Short)
}

func TestValidateCommand_HasFlags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{"config_flag", "config", "preflight.yaml"},
		{"target_flag", "target", "default"},
		{"json_flag", "json", "false"},
		{"strict_flag", "strict", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := validateCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestValidateCommand_IsRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "validate" {
			found = true
			break
		}
	}
	assert.True(t, found, "validate command should be registered")
}

// captureStdout captures stdout during the execution of f
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestOutputValidationJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputValidationJSON(nil, assert.AnError)
	})

	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.False(t, result["valid"].(bool))
	assert.Equal(t, assert.AnError.Error(), result["error"].(string))
}

func TestOutputValidationJSON_WithValidResult(t *testing.T) {
	result := &app.ValidationResult{
		Info: []string{"Loaded config", "Target: default"},
	}

	output := captureStdout(t, func() {
		outputValidationJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.True(t, parsed["valid"].(bool))
	info := parsed["info"].([]interface{})
	assert.Len(t, info, 2)
}

func TestOutputValidationJSON_WithErrors(t *testing.T) {
	result := &app.ValidationResult{
		Errors: []string{"Compilation failed"},
		Info:   []string{"Loaded config"},
	}

	output := captureStdout(t, func() {
		outputValidationJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.False(t, parsed["valid"].(bool))
	errors := parsed["errors"].([]interface{})
	assert.Len(t, errors, 1)
	assert.Equal(t, "Compilation failed", errors[0].(string))
}

func TestOutputValidationText_Valid(t *testing.T) {
	result := &app.ValidationResult{
		Info: []string{"Loaded config from preflight.yaml", "Target: default"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "✓ Configuration is valid")
	assert.Contains(t, output, "Loaded config from preflight.yaml")
	assert.Contains(t, output, "Target: default")
}

func TestOutputValidationText_WithErrors(t *testing.T) {
	result := &app.ValidationResult{
		Errors: []string{"Compilation failed: missing provider"},
		Info:   []string{"Loaded config"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "✗ Validation errors")
	assert.Contains(t, output, "Compilation failed: missing provider")
}

func TestOutputValidationText_WithWarnings(t *testing.T) {
	result := &app.ValidationResult{
		Warnings: []string{"No steps generated"},
		Info:     []string{"Loaded config"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "⚠ Warnings")
	assert.Contains(t, output, "No steps generated")
}

func TestOutputValidationText_WithErrorsAndWarnings(t *testing.T) {
	result := &app.ValidationResult{
		Errors:   []string{"Missing dependency"},
		Warnings: []string{"Empty config section"},
		Info:     []string{"Loaded config"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "✗ Validation errors")
	assert.Contains(t, output, "Missing dependency")
	assert.Contains(t, output, "⚠ Warnings")
	assert.Contains(t, output, "Empty config section")
	assert.Contains(t, output, "ℹ Info")
	assert.Contains(t, output, "Loaded config")
}
