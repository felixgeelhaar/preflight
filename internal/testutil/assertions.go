package testutil

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// AssertFileExists asserts that a file exists at the given path.
func AssertFileExists(t testing.TB, path string, msgAndArgs ...interface{}) {
	t.Helper()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		assert.Fail(t, "file does not exist", "expected file to exist: %s", path)
		return
	}
	require.NoError(t, err)
	assert.False(t, info.IsDir(), "expected file but got directory: %s", path)
}

// AssertFileNotExists asserts that no file exists at the given path.
func AssertFileNotExists(t testing.TB, path string, msgAndArgs ...interface{}) {
	t.Helper()

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "expected file to not exist: %s", path)
}

// AssertDirExists asserts that a directory exists at the given path.
func AssertDirExists(t testing.TB, path string, msgAndArgs ...interface{}) {
	t.Helper()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		assert.Fail(t, "directory does not exist", "expected directory to exist: %s", path)
		return
	}
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "expected directory but got file: %s", path)
}

// AssertFileContains asserts that a file contains the expected substring.
func AssertFileContains(t testing.TB, path, expected string, msgAndArgs ...interface{}) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file: %s", path)

	assert.Contains(t, string(content), expected, msgAndArgs...)
}

// AssertFileNotContains asserts that a file does not contain the substring.
func AssertFileNotContains(t testing.TB, path, unexpected string, msgAndArgs ...interface{}) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file: %s", path)

	assert.NotContains(t, string(content), unexpected, msgAndArgs...)
}

// AssertFileEquals asserts that a file contains exactly the expected content.
func AssertFileEquals(t testing.TB, path, expected string, msgAndArgs ...interface{}) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read file: %s", path)

	// Normalize line endings
	actual := strings.ReplaceAll(string(content), "\r\n", "\n")
	expected = strings.ReplaceAll(expected, "\r\n", "\n")

	assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertYAMLEquals asserts that two YAML strings are semantically equal.
func AssertYAMLEquals(t testing.TB, expected, actual string, msgAndArgs ...interface{}) {
	t.Helper()

	var expectedMap, actualMap interface{}

	err := yaml.Unmarshal([]byte(expected), &expectedMap)
	require.NoError(t, err, "failed to parse expected YAML")

	err = yaml.Unmarshal([]byte(actual), &actualMap)
	require.NoError(t, err, "failed to parse actual YAML")

	assert.Equal(t, expectedMap, actualMap, msgAndArgs...)
}

// AssertNoError asserts that err is nil.
func AssertNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NoError(t, err, msgAndArgs...)
}

// AssertError asserts that err is not nil.
func AssertError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Error(t, err, msgAndArgs...)
}

// AssertErrorContains asserts that err contains the expected message.
func AssertErrorContains(t testing.TB, err error, expected string, msgAndArgs ...interface{}) {
	t.Helper()

	require.Error(t, err)
	assert.Contains(t, err.Error(), expected, msgAndArgs...)
}

// RequireNoError requires that err is nil, failing immediately if not.
func RequireNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// AssertEventually asserts that a condition becomes true within a timeout.
// This is useful for testing asynchronous operations.
// waitFor and tick are in milliseconds.
func AssertEventually(t testing.TB, condition func() bool, waitForMs, tickMs int, msgAndArgs ...interface{}) {
	t.Helper()

	waitFor := time.Duration(waitForMs) * time.Millisecond
	tick := time.Duration(tickMs) * time.Millisecond

	assert.Eventually(t, condition, waitFor, tick, msgAndArgs...)
}
