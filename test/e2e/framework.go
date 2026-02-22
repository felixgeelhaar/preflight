// Package e2e provides end-to-end testing utilities for preflight CLI.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Harness provides utilities for end-to-end CLI testing.
type Harness struct {
	T            *testing.T
	BinaryPath   string
	TempDir      string
	HomeDir      string
	ConfigDir    string
	EnvVars      map[string]string
	Timeout      time.Duration
	LastOutput   string
	LastError    string
	LastExitCode int
}

// NewHarness creates a new end-to-end test harness.
// It builds the preflight binary if needed.
func NewHarness(t *testing.T) *Harness {
	t.Helper()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	configDir := filepath.Join(tempDir, "config")

	// Create directories
	for _, dir := range []string{homeDir, configDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Get or build binary
	binaryPath := getBinary(t)

	return &Harness{
		T:          t,
		BinaryPath: binaryPath,
		TempDir:    tempDir,
		HomeDir:    homeDir,
		ConfigDir:  configDir,
		EnvVars:    make(map[string]string),
		Timeout:    30 * time.Second,
	}
}

// getBinary returns the path to the preflight binary.
// It builds the binary if it doesn't exist or if E2E_BUILD is set.
func getBinary(t *testing.T) string {
	t.Helper()

	// Check if binary path is specified via environment
	if path := os.Getenv("PREFLIGHT_BINARY"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Build binary in temp directory for tests
	binaryPath := filepath.Join(t.TempDir(), "preflight-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/preflight")
	cmd.Dir = findProjectRoot(t)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build preflight binary: %v\n%s", err, stderr.String())
	}

	return binaryPath
}

// findProjectRoot finds the project root directory.
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from current directory and go up until we find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

// WithEnv sets an environment variable for commands.
func (h *Harness) WithEnv(key, value string) *Harness {
	h.EnvVars[key] = value
	return h
}

// WithTimeout sets the timeout for commands.
func (h *Harness) WithTimeout(d time.Duration) *Harness {
	h.Timeout = d
	return h
}

// Run executes a preflight command and returns the exit code.
func (h *Harness) Run(args ...string) int {
	h.T.Helper()

	cmd := exec.Command(h.BinaryPath, args...)
	cmd.Dir = h.ConfigDir

	// Set up environment
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", h.HomeDir))
	for k, v := range h.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		h.LastOutput = stdout.String()
		h.LastError = stderr.String()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				h.LastExitCode = exitErr.ExitCode()
			} else {
				h.LastExitCode = -1
			}
		} else {
			h.LastExitCode = 0
		}
	case <-time.After(h.Timeout):
		_ = cmd.Process.Kill()
		h.T.Fatalf("command timed out after %v: %v", h.Timeout, args)
	}

	return h.LastExitCode
}

// RunSuccess executes a command and expects it to succeed.
func (h *Harness) RunSuccess(args ...string) string {
	h.T.Helper()

	exitCode := h.Run(args...)
	if exitCode != 0 {
		h.T.Fatalf("command failed with exit code %d: %v\nOutput: %s\nStderr: %s",
			exitCode, args, h.LastOutput, h.LastError)
	}

	return h.LastOutput
}

// RunFail executes a command and expects it to fail.
func (h *Harness) RunFail(args ...string) string {
	h.T.Helper()

	exitCode := h.Run(args...)
	if exitCode == 0 {
		h.T.Fatalf("command succeeded but expected failure: %v\nOutput: %s",
			args, h.LastOutput)
	}

	return h.LastOutput + h.LastError
}

// Init runs preflight init with given options.
func (h *Harness) Init(preset string, extraArgs ...string) string {
	h.T.Helper()

	args := make([]string, 0, 6+len(extraArgs))
	args = append(args, "init", "--non-interactive", "--preset", preset, "--output", h.ConfigDir)
	args = append(args, extraArgs...)

	return h.RunSuccess(args...)
}

// Plan runs preflight plan.
func (h *Harness) Plan(extraArgs ...string) string {
	h.T.Helper()

	args := make([]string, 0, 3+len(extraArgs))
	args = append(args, "plan", "--config", filepath.Join(h.ConfigDir, "preflight.yaml"))
	args = append(args, extraArgs...)

	return h.RunSuccess(args...)
}

// Apply runs preflight apply.
func (h *Harness) Apply(dryRun bool, extraArgs ...string) string {
	h.T.Helper()

	args := []string{"apply", "--config", filepath.Join(h.ConfigDir, "preflight.yaml")}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, extraArgs...)

	return h.RunSuccess(args...)
}

// Doctor runs preflight doctor with --quiet for non-interactive output.
func (h *Harness) Doctor(extraArgs ...string) string {
	h.T.Helper()

	args := make([]string, 0, 4+len(extraArgs))
	args = append(args, "doctor", "--quiet", "--config", filepath.Join(h.ConfigDir, "preflight.yaml"))
	args = append(args, extraArgs...)

	return h.RunSuccess(args...)
}

// Validate runs preflight validate.
func (h *Harness) Validate(extraArgs ...string) string {
	h.T.Helper()

	args := make([]string, 0, 3+len(extraArgs))
	args = append(args, "validate", "--config", filepath.Join(h.ConfigDir, "preflight.yaml"))
	args = append(args, extraArgs...)

	return h.RunSuccess(args...)
}

// CreateFile creates a file in the home directory.
func (h *Harness) CreateFile(relativePath, content string) string {
	h.T.Helper()

	path := filepath.Join(h.HomeDir, relativePath)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.T.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.T.Fatalf("failed to write file %s: %v", path, err)
	}

	return path
}

// CreateConfigFile creates a file in the config directory.
func (h *Harness) CreateConfigFile(relativePath, content string) string {
	h.T.Helper()

	path := filepath.Join(h.ConfigDir, relativePath)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.T.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.T.Fatalf("failed to write file %s: %v", path, err)
	}

	return path
}

// FileExists checks if a file exists in the home directory.
func (h *Harness) FileExists(relativePath string) bool {
	path := filepath.Join(h.HomeDir, relativePath)
	_, err := os.Stat(path)
	return err == nil
}

// ConfigFileExists checks if a file exists in the config directory.
func (h *Harness) ConfigFileExists(relativePath string) bool {
	path := filepath.Join(h.ConfigDir, relativePath)
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file from the home directory.
func (h *Harness) ReadFile(relativePath string) string {
	h.T.Helper()

	path := filepath.Join(h.HomeDir, relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		h.T.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}

// ReadConfigFile reads a file from the config directory.
func (h *Harness) ReadConfigFile(relativePath string) string {
	h.T.Helper()

	path := filepath.Join(h.ConfigDir, relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		h.T.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(content)
}

// OutputContains checks if the last output contains a string.
func (h *Harness) OutputContains(s string) bool {
	return strings.Contains(h.LastOutput, s) || strings.Contains(h.LastError, s)
}

// AssertOutputContains asserts the last output contains a string.
func (h *Harness) AssertOutputContains(s string) {
	h.T.Helper()

	if !h.OutputContains(s) {
		h.T.Errorf("expected output to contain %q, got:\n%s", s, h.LastOutput+h.LastError)
	}
}

// AssertFileExists asserts a file exists in the home directory.
func (h *Harness) AssertFileExists(relativePath string) {
	h.T.Helper()

	if !h.FileExists(relativePath) {
		h.T.Errorf("expected file to exist: %s", relativePath)
	}
}

// AssertConfigFileExists asserts a file exists in the config directory.
func (h *Harness) AssertConfigFileExists(relativePath string) {
	h.T.Helper()

	if !h.ConfigFileExists(relativePath) {
		h.T.Errorf("expected config file to exist: %s", relativePath)
	}
}

// AssertFileContains asserts a file in home directory contains a string.
func (h *Harness) AssertFileContains(relativePath, expected string) {
	h.T.Helper()

	content := h.ReadFile(relativePath)
	if !strings.Contains(content, expected) {
		h.T.Errorf("expected %s to contain %q, got:\n%s", relativePath, expected, content)
	}
}
