// Package integration provides test utilities for integration testing.
package integration

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/app"
)

// TestHarness provides utilities for integration testing.
type TestHarness struct {
	T       *testing.T
	TempDir string
	HomeDir string
	Output  *bytes.Buffer

	preflight *app.Preflight
}

// NewHarness creates a new test harness.
func NewHarness(t *testing.T) *TestHarness {
	t.Helper()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("failed to create home directory: %v", err)
	}

	output := &bytes.Buffer{}

	return &TestHarness{
		T:         t,
		TempDir:   tempDir,
		HomeDir:   homeDir,
		Output:    output,
		preflight: app.New(output),
	}
}

// Preflight returns the preflight application instance.
func (h *TestHarness) Preflight() *app.Preflight {
	return h.preflight
}

// CreateConfig creates a preflight configuration in the temp directory.
func (h *TestHarness) CreateConfig(manifest, layer string) string {
	h.T.Helper()

	configDir := filepath.Join(h.TempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		h.T.Fatalf("failed to create config directory: %v", err)
	}

	layersDir := filepath.Join(configDir, "layers")
	if err := os.MkdirAll(layersDir, 0o755); err != nil {
		h.T.Fatalf("failed to create layers directory: %v", err)
	}

	manifestPath := filepath.Join(configDir, "preflight.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		h.T.Fatalf("failed to write manifest: %v", err)
	}

	layerPath := filepath.Join(layersDir, "base.yaml")
	if err := os.WriteFile(layerPath, []byte(layer), 0o644); err != nil {
		h.T.Fatalf("failed to write layer: %v", err)
	}

	return manifestPath
}

// CreateFile creates a file in the home directory.
func (h *TestHarness) CreateFile(relativePath, content string) string {
	h.T.Helper()

	path := filepath.Join(h.HomeDir, relativePath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.T.Fatalf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		h.T.Fatalf("failed to write file: %v", err)
	}

	return path
}

// FileExists checks if a file exists in the home directory.
func (h *TestHarness) FileExists(relativePath string) bool {
	path := filepath.Join(h.HomeDir, relativePath)
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads a file from the home directory.
func (h *TestHarness) ReadFile(relativePath string) string {
	h.T.Helper()

	path := filepath.Join(h.HomeDir, relativePath)
	content, err := os.ReadFile(path)
	if err != nil {
		h.T.Fatalf("failed to read file: %v", err)
	}
	return string(content)
}

// OutputContains checks if the output buffer contains a string.
func (h *TestHarness) OutputContains(s string) bool {
	return bytes.Contains(h.Output.Bytes(), []byte(s))
}

// Plan runs preflight plan and returns the plan.
func (h *TestHarness) Plan(configPath, target string) error {
	ctx := context.Background()
	_, err := h.preflight.Plan(ctx, configPath, target)
	return err
}

// Doctor runs preflight doctor and returns the report.
func (h *TestHarness) Doctor(configPath, target string) (*app.DoctorReport, error) {
	ctx := context.Background()
	opts := app.NewDoctorOptions(configPath, target)
	return h.preflight.Doctor(ctx, opts)
}

// Capture runs preflight capture and returns the findings.
func (h *TestHarness) Capture(providers ...string) (*app.CaptureFindings, error) {
	ctx := context.Background()
	opts := app.NewCaptureOptions()
	if len(providers) > 0 {
		opts = opts.WithProviders(providers...)
	}
	opts.HomeDir = h.HomeDir
	return h.preflight.Capture(ctx, opts)
}

// FixtureDir returns the path to the test fixtures directory.
func FixtureDir() string {
	// Find the fixtures directory relative to the test file
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(wd, "..", "fixtures")
}

// LoadFixture reads a fixture file.
func LoadFixture(name string) ([]byte, error) {
	path := filepath.Join(FixtureDir(), name)
	return os.ReadFile(path)
}

// MockWriter is a writer that captures output for testing.
type MockWriter struct {
	buf bytes.Buffer
}

// Write implements io.Writer.
func (w *MockWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

// String returns the captured output.
func (w *MockWriter) String() string {
	return w.buf.String()
}

// Reset clears the captured output.
func (w *MockWriter) Reset() {
	w.buf.Reset()
}

var _ io.Writer = (*MockWriter)(nil)
