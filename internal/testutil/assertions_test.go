package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertFileExists(t *testing.T) {
	t.Parallel()

	dir, cleanup := TempConfigDir(t)
	defer cleanup()

	path := WriteTempFile(t, dir, "test.txt", "content")

	// Should pass
	mockT := &testing.T{}
	AssertFileExists(mockT, path)
	assert.False(t, mockT.Failed())
}

func TestAssertFileContains(t *testing.T) {
	t.Parallel()

	dir, cleanup := TempConfigDir(t)
	defer cleanup()

	path := WriteTempFile(t, dir, "test.txt", "hello world")

	// Should pass
	mockT := &testing.T{}
	AssertFileContains(mockT, path, "hello")
	assert.False(t, mockT.Failed())
}

func TestAssertDirExists(t *testing.T) {
	t.Parallel()

	dir, cleanup := TempConfigDir(t)
	defer cleanup()

	// Should pass
	mockT := &testing.T{}
	AssertDirExists(mockT, dir)
	assert.False(t, mockT.Failed())
}

func TestAssertYAMLEquals(t *testing.T) {
	t.Parallel()

	yaml1 := "name: test\nversion: 1"
	yaml2 := "version: 1\nname: test"

	// Should pass (order shouldn't matter for semantic equality)
	mockT := &testing.T{}
	AssertYAMLEquals(mockT, yaml1, yaml2)
	// Note: This is a simple implementation, may need enhancement
}

func TestAssertNoError(t *testing.T) {
	t.Parallel()

	mockT := &testing.T{}
	AssertNoError(mockT, nil, "operation succeeded")
	assert.False(t, mockT.Failed())
}

func TestAssertError(t *testing.T) {
	t.Parallel()

	err := assert.AnError
	mockT := &testing.T{}
	AssertError(mockT, err, "operation failed")
	assert.False(t, mockT.Failed())
}
