//go:build e2e

package framework

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Assertions provides common assertion helpers for E2E tests.

// AssertSuccess asserts that the command succeeded.
func AssertSuccess(t *testing.T, r *Result) {
	t.Helper()
	if !r.Success() {
		t.Errorf("Expected command to succeed, got exit code %d\nStdout: %s\nStderr: %s",
			r.ExitCode, r.Stdout, r.Stderr)
	}
}

// AssertFailed asserts that the command failed.
func AssertFailed(t *testing.T, r *Result) {
	t.Helper()
	if r.Success() {
		t.Errorf("Expected command to fail, but it succeeded\nStdout: %s", r.Stdout)
	}
}

// AssertExitCode asserts the expected exit code.
func AssertExitCode(t *testing.T, r *Result, expected int) {
	t.Helper()
	if r.ExitCode != expected {
		t.Errorf("Expected exit code %d, got %d\nStdout: %s\nStderr: %s",
			expected, r.ExitCode, r.Stdout, r.Stderr)
	}
}

// AssertStdoutContains asserts that stdout contains the expected substring.
func AssertStdoutContains(t *testing.T, r *Result, expected string) {
	t.Helper()
	if !strings.Contains(r.Stdout, expected) {
		t.Errorf("Expected stdout to contain %q, but got:\n%s", expected, r.Stdout)
	}
}

// AssertStdoutNotContains asserts that stdout does not contain the unexpected substring.
func AssertStdoutNotContains(t *testing.T, r *Result, unexpected string) {
	t.Helper()
	if strings.Contains(r.Stdout, unexpected) {
		t.Errorf("Expected stdout to NOT contain %q, but got:\n%s", unexpected, r.Stdout)
	}
}

// AssertStderrContains asserts that stderr contains the expected substring.
func AssertStderrContains(t *testing.T, r *Result, expected string) {
	t.Helper()
	if !strings.Contains(r.Stderr, expected) {
		t.Errorf("Expected stderr to contain %q, but got:\n%s", expected, r.Stderr)
	}
}

// AssertStderrEmpty asserts that stderr is empty.
func AssertStderrEmpty(t *testing.T, r *Result) {
	t.Helper()
	if r.Stderr != "" {
		t.Errorf("Expected stderr to be empty, but got:\n%s", r.Stderr)
	}
}

// AssertFileExists asserts that a file exists in the environment.
func AssertFileExists(t *testing.T, env *Environment, path string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", path)
	}
}

// AssertFileNotExists asserts that a file does not exist in the environment.
func AssertFileNotExists(t *testing.T, env *Environment, path string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), path)
	if _, err := os.Stat(fullPath); err == nil {
		t.Errorf("Expected file %s to NOT exist", path)
	}
}

// AssertFileContains asserts that a file contains the expected content.
func AssertFileContains(t *testing.T, env *Environment, path, expected string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	if !strings.Contains(string(content), expected) {
		t.Errorf("Expected file %s to contain %q, but got:\n%s", path, expected, string(content))
	}
}

// AssertFileEquals asserts that a file has exactly the expected content.
func AssertFileEquals(t *testing.T, env *Environment, path, expected string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	if string(content) != expected {
		t.Errorf("Expected file %s to equal %q, but got:\n%s", path, expected, string(content))
	}
}

// AssertSymlink asserts that a symlink exists and points to the expected target.
func AssertSymlink(t *testing.T, env *Environment, linkPath, expectedTarget string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), linkPath)
	target, err := os.Readlink(fullPath)
	if err != nil {
		t.Fatalf("Expected %s to be a symlink: %v", linkPath, err)
	}
	if target != expectedTarget {
		t.Errorf("Expected symlink %s to point to %s, but points to %s", linkPath, expectedTarget, target)
	}
}

// AssertDirExists asserts that a directory exists.
func AssertDirExists(t *testing.T, env *Environment, path string) {
	t.Helper()
	fullPath := filepath.Join(env.RootDir(), path)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist", path)
		return
	}
	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", path)
	}
}

// AssertOutputMatches checks if output matches a pattern using simple string matching.
func AssertOutputMatches(t *testing.T, r *Result, patterns ...string) {
	t.Helper()
	for _, pattern := range patterns {
		if !strings.Contains(r.Stdout, pattern) {
			t.Errorf("Expected output to contain pattern %q\nGot:\n%s", pattern, r.Stdout)
		}
	}
}
