//go:build e2e

// Package framework provides the E2E test infrastructure for preflight.
package framework

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// Environment represents an isolated test environment for E2E tests.
type Environment struct {
	t           *testing.T
	rootDir     string
	configDir   string
	binaryPath  string
	homeDir     string
	cleanupOnce sync.Once
}

var (
	buildOnce   sync.Once
	binaryPath  string
	buildErr    error
	projectRoot string
)

// findProjectRoot locates the project root directory.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// buildBinary builds the preflight binary once per test run.
func buildBinary(t *testing.T) (string, error) {
	buildOnce.Do(func() {
		projectRoot, buildErr = findProjectRoot()
		if buildErr != nil {
			return
		}

		tmpDir := os.TempDir()
		binaryPath = filepath.Join(tmpDir, "preflight-e2e-test")

		cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/preflight")
		cmd.Dir = projectRoot

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			buildErr = err
			t.Logf("Build stderr: %s", stderr.String())
			return
		}
	})

	return binaryPath, buildErr
}

// NewEnvironment creates a new isolated test environment.
func NewEnvironment(t *testing.T) *Environment {
	t.Helper()

	binary, err := buildBinary(t)
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	rootDir := t.TempDir()
	configDir := filepath.Join(rootDir, "config")
	homeDir := filepath.Join(rootDir, "home")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("Failed to create home directory: %v", err)
	}

	env := &Environment{
		t:          t,
		rootDir:    rootDir,
		configDir:  configDir,
		binaryPath: binary,
		homeDir:    homeDir,
	}

	t.Cleanup(func() {
		env.cleanup()
	})

	return env
}

// cleanup removes temporary files.
func (e *Environment) cleanup() {
	e.cleanupOnce.Do(func() {
		// TempDir is automatically cleaned up by testing package
	})
}

// ConfigDir returns the path to the config directory.
func (e *Environment) ConfigDir() string {
	return e.configDir
}

// HomeDir returns the path to the simulated home directory.
func (e *Environment) HomeDir() string {
	return e.homeDir
}

// RootDir returns the path to the test root directory.
func (e *Environment) RootDir() string {
	return e.rootDir
}

// BinaryPath returns the path to the built binary.
func (e *Environment) BinaryPath() string {
	return e.binaryPath
}

// WriteFile writes content to a file in the test environment.
func (e *Environment) WriteFile(path, content string) {
	e.t.Helper()

	fullPath := filepath.Join(e.rootDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		e.t.Fatalf("Failed to write file %s: %v", fullPath, err)
	}
}

// WriteConfig writes a preflight.yaml config file.
func (e *Environment) WriteConfig(content string) string {
	e.t.Helper()

	configPath := filepath.Join(e.configDir, "preflight.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		e.t.Fatalf("Failed to write config: %v", err)
	}
	return configPath
}

// WriteLayer writes a layer file to the layers directory.
func (e *Environment) WriteLayer(name, content string) string {
	e.t.Helper()

	layersDir := filepath.Join(e.configDir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		e.t.Fatalf("Failed to create layers directory: %v", err)
	}

	layerPath := filepath.Join(layersDir, name+".yaml")
	if err := os.WriteFile(layerPath, []byte(content), 0o644); err != nil {
		e.t.Fatalf("Failed to write layer: %v", err)
	}
	return layerPath
}

// FileExists checks if a file exists in the test environment.
func (e *Environment) FileExists(path string) bool {
	fullPath := filepath.Join(e.rootDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads a file from the test environment.
func (e *Environment) ReadFile(path string) string {
	e.t.Helper()

	fullPath := filepath.Join(e.rootDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		e.t.Fatalf("Failed to read file %s: %v", fullPath, err)
	}
	return string(content)
}
