// Package testutil provides test helpers and utilities for preflight tests.
package testutil

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed fixtures/*
var fixturesFS embed.FS

// TempConfigDir creates a temporary directory for test configuration files.
// Returns the directory path and a cleanup function.
func TempConfigDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "preflight-test-*")
	require.NoError(t, err, "failed to create temp directory")

	cleanup := func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Logf("warning: failed to clean up temp directory: %v", err)
		}
	}

	return dir, cleanup
}

// WriteTempFile writes content to a file in the specified directory.
func WriteTempFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err, "failed to write temp file: %s", filename)

	return path
}

// WriteTempDir creates a subdirectory in the temp directory.
func WriteTempDir(t *testing.T, dir, dirname string) string {
	t.Helper()

	path := filepath.Join(dir, dirname)
	err := os.MkdirAll(path, 0o755)
	require.NoError(t, err, "failed to create temp subdirectory: %s", dirname)

	return path
}

// LoadFixture loads a fixture file from the embedded fixtures directory.
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()

	content, err := fixturesFS.ReadFile(filepath.Join("fixtures", name))
	require.NoError(t, err, "failed to load fixture: %s", name)

	return content
}

// LoadFixtureOrEmpty loads a fixture file or returns empty bytes if not found.
func LoadFixtureOrEmpty(name string) []byte {
	content, err := fixturesFS.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		return []byte{}
	}
	return content
}

// WriteFixtureToDir writes a fixture file to a directory.
func WriteFixtureToDir(t *testing.T, dir, fixtureName, destName string) string {
	t.Helper()

	content := LoadFixture(t, fixtureName)
	return WriteTempFile(t, dir, destName, string(content))
}

// SetEnv sets an environment variable for the duration of the test.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()

	original := os.Getenv(key)
	err := os.Setenv(key, value)
	require.NoError(t, err)

	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, original)
		}
	})
}

// UnsetEnv unsets an environment variable for the duration of the test.
func UnsetEnv(t *testing.T, key string) {
	t.Helper()

	original := os.Getenv(key)
	err := os.Unsetenv(key)
	require.NoError(t, err)

	t.Cleanup(func() {
		if original != "" {
			_ = os.Setenv(key, original)
		}
	})
}

// ChangeDir changes to a directory for the duration of the test.
func ChangeDir(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Chdir(original)
	})
}
