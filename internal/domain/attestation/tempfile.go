package attestation

import (
	"os"
	"path/filepath"
)

// createTempDir creates a temporary directory with the given pattern.
func createTempDir(pattern string) (string, error) {
	return os.MkdirTemp("", pattern)
}

// cleanupTempDir removes a temporary directory and all its contents.
func cleanupTempDir(dir string) {
	_ = os.RemoveAll(dir)
}

// writeTempFile writes data to a named file in the given directory.
// Returns the full path to the created file.
func writeTempFile(dir, name string, data []byte) (string, error) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", err
	}
	return path, nil
}
