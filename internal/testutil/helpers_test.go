package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTempConfigDir(t *testing.T) {
	t.Parallel()

	dir, cleanup := TempConfigDir(t)
	defer cleanup()

	// Directory should exist
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Should be able to create files in it
	testFile := filepath.Join(dir, "test.yaml")
	err = os.WriteFile(testFile, []byte("test: true"), 0644)
	require.NoError(t, err)
}

func TestTempConfigDir_Cleanup(t *testing.T) {
	t.Parallel()

	var savedDir string

	func() {
		dir, cleanup := TempConfigDir(t)
		savedDir = dir
		cleanup()
	}()

	// Directory should be removed after cleanup
	_, err := os.Stat(savedDir)
	assert.True(t, os.IsNotExist(err))
}

func TestWriteTempFile(t *testing.T) {
	t.Parallel()

	dir, cleanup := TempConfigDir(t)
	defer cleanup()

	path := WriteTempFile(t, dir, "config.yaml", "version: 1")

	// File should exist
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "version: 1", string(content))
}

func TestLoadFixture(t *testing.T) {
	t.Parallel()

	content := LoadFixture(t, "minimal.yaml")

	assert.Contains(t, string(content), "version: 1")
}

func TestLoadFixture_NotFound(t *testing.T) {
	t.Parallel()

	// This should return empty for non-existent fixtures
	content := LoadFixtureOrEmpty("nonexistent.yaml")
	assert.Empty(t, content)
}
