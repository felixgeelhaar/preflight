package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stdoutMu serializes stdout capture operations. All code that redirects
// file descriptor 1 must hold this lock.
var stdoutMu sync.Mutex

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

// captureStdout captures stdout during the execution of f.
//
// It redirects file descriptor 1 at the OS level using syscall.Dup2 instead
// of swapping the os.Stdout Go variable. This avoids a data race with any
// parallel goroutine that reads os.Stdout (e.g. exec.Cmd, app.New(os.Stdout)).
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	// Create a pipe to capture output.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	// Save the original fd 1 so we can restore it.
	origFd, err := syscall.Dup(int(os.Stdout.Fd()))
	require.NoError(t, err)
	defer syscall.Close(origFd)

	// Redirect fd 1 to the pipe write end. This does NOT modify
	// the os.Stdout Go variable, so no data race with readers.
	require.NoError(t, syscall.Dup2(int(w.Fd()), int(os.Stdout.Fd())))

	// Drain the pipe concurrently to avoid deadlock when output exceeds
	// the OS pipe buffer (~64 KB on macOS). Without this, f() blocks on
	// write and we never reach the io.Copy below.
	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&buf, r)
		done <- copyErr
	}()

	f()

	// Restore fd 1 to the original destination.
	require.NoError(t, syscall.Dup2(origFd, int(os.Stdout.Fd())))

	// Close the pipe write end so the goroutine's io.Copy sees EOF.
	w.Close()

	require.NoError(t, <-done)
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

func TestOutputValidationText_WithPolicyViolations(t *testing.T) {
	result := &app.ValidationResult{
		PolicyViolations: []string{"Security policy violation: secrets exposed"},
		Info:             []string{"Loaded config"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "⛔ Policy violations")
	assert.Contains(t, output, "Security policy violation: secrets exposed")
	assert.Contains(t, output, "ℹ Info")
}

func TestOutputValidationText_AllTypes(t *testing.T) {
	result := &app.ValidationResult{
		Errors:           []string{"Error 1"},
		PolicyViolations: []string{"Policy violation 1"},
		Warnings:         []string{"Warning 1"},
		Info:             []string{"Info 1"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})

	assert.Contains(t, output, "✗ Validation errors")
	assert.Contains(t, output, "⛔ Policy violations")
	assert.Contains(t, output, "⚠ Warnings")
	assert.Contains(t, output, "ℹ Info")
}

func TestRunValidate_SuccessJSON(t *testing.T) {
	t.Parallel()

	prev := newValidatePreflight
	fake := &fakeValidateClient{
		result: &app.ValidationResult{
			Info: []string{"Loaded config from preflight.yaml"},
		},
	}
	newValidatePreflight = func(_ io.Writer) validatePreflightClient {
		return fake
	}
	defer func() { newValidatePreflight = prev }()

	prevJSON := validateJSON
	validateJSON = true
	defer func() { validateJSON = prevJSON }()

	output := captureStdout(t, func() {
		err := runValidate(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"valid": true`)
	assert.True(t, fake.called)
	assert.Equal(t, "preflight.yaml", fake.configPath)
	assert.Equal(t, "default", fake.target)
}

type fakeValidateClient struct {
	result     *app.ValidationResult
	err        error
	called     bool
	configPath string
	target     string
	opts       app.ValidateOptions
	modeSet    bool
}

func (f *fakeValidateClient) ValidateWithOptions(_ context.Context, configPath, target string, opts app.ValidateOptions) (*app.ValidationResult, error) {
	f.called = true
	f.configPath = configPath
	f.target = target
	f.opts = opts
	return f.result, f.err
}

func (f *fakeValidateClient) WithMode(_ config.ReproducibilityMode) validatePreflightClient {
	f.modeSet = true
	return f
}
