package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureConfigGenerator_GenerateFromCapture(t *testing.T) {
	t.Parallel()

	t.Run("generates basic manifest and layer", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git", Value: "git", CapturedAt: time.Now()},
				{Provider: "brew", Name: "neovim", Value: "neovim", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Check manifest was created
		manifestPath := filepath.Join(tmpDir, "preflight.yaml")
		assert.FileExists(t, manifestPath)

		// Check layer was created
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		assert.FileExists(t, layerPath)

		// Read and verify layer content
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "git")
		assert.Contains(t, string(content), "neovim")
	})

	t.Run("generates git config from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "git", Name: "user.name", Value: "Test User", CapturedAt: time.Now()},
				{Provider: "git", Name: "user.email", Value: "test@example.com", CapturedAt: time.Now()},
				{Provider: "git", Name: "core.editor", Value: "nvim", CapturedAt: time.Now()},
			},
			Providers:  []string{"git"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify git config
		assert.Contains(t, string(content), "user:")
		assert.Contains(t, string(content), "name: Test User")
		assert.Contains(t, string(content), "email: test@example.com")
		assert.Contains(t, string(content), "editor: nvim")
	})

	t.Run("generates shell config from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "shell", Name: ".zshrc", Value: "~/.zshrc", CapturedAt: time.Now()},
				{Provider: "shell", Name: ".bashrc", Value: "~/.bashrc", CapturedAt: time.Now()},
			},
			Providers:  []string{"shell"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify shell config
		assert.Contains(t, string(content), "shell:")
		assert.Contains(t, string(content), "default: zsh")
	})

	t.Run("generates vscode extensions from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "vscode", Name: "golang.go", Value: "golang.go", CapturedAt: time.Now()},
				{Provider: "vscode", Name: "ms-python.python", Value: "ms-python.python", CapturedAt: time.Now()},
			},
			Providers:  []string{"vscode"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify vscode config
		assert.Contains(t, string(content), "vscode:")
		assert.Contains(t, string(content), "extensions:")
		assert.Contains(t, string(content), "golang.go")
		assert.Contains(t, string(content), "ms-python.python")
	})

	t.Run("generates runtime tools from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "runtime", Name: "go", Value: "1.22.0", CapturedAt: time.Now()},
				{Provider: "runtime", Name: "node", Value: "20.10.0", CapturedAt: time.Now()},
			},
			Providers:  []string{"runtime"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify runtime config
		assert.Contains(t, string(content), "runtime:")
		assert.Contains(t, string(content), "tools:")
		assert.Contains(t, string(content), "name: go")
		assert.Contains(t, string(content), "version: 1.22.0")
		assert.Contains(t, string(content), "name: node")
	})

	t.Run("handles empty findings", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items:      []CapturedItem{},
			Providers:  []string{},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Check files were created
		assert.FileExists(t, filepath.Join(tmpDir, "preflight.yaml"))
		assert.FileExists(t, filepath.Join(tmpDir, "layers", "captured.yaml"))
	})

	t.Run("uses custom target name", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items:      []CapturedItem{},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "work")
		require.NoError(t, err)

		// Read manifest and check target
		manifestPath := filepath.Join(tmpDir, "preflight.yaml")
		content, err := os.ReadFile(manifestPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "work:")
	})

	t.Run("handles multiple providers", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git", Value: "git", CapturedAt: time.Now()},
				{Provider: "git", Name: "user.name", Value: "Test", CapturedAt: time.Now()},
				{Provider: "shell", Name: ".zshrc", Value: "~/.zshrc", CapturedAt: time.Now()},
				{Provider: "vscode", Name: "golang.go", Value: "golang.go", CapturedAt: time.Now()},
				{Provider: "runtime", Name: "go", Value: "1.22.0", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew", "git", "shell", "vscode", "runtime"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify all sections present
		assert.Contains(t, string(content), "packages:")
		assert.Contains(t, string(content), "git:")
		assert.Contains(t, string(content), "shell:")
		assert.Contains(t, string(content), "vscode:")
		assert.Contains(t, string(content), "runtime:")
	})
}
