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

	t.Run("uses default target when empty", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items:      []CapturedItem{},
			CapturedAt: time.Now(),
		}

		// Pass empty target - should use "default"
		err := generator.GenerateFromCapture(findings, "")
		require.NoError(t, err)

		// Read manifest and check target is "default"
		manifestPath := filepath.Join(tmpDir, "preflight.yaml")
		content, err := os.ReadFile(manifestPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "default:")
	})

	t.Run("ignores unhandled providers", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// nvim and ssh providers are not handled by capture config generator
				{Provider: "nvim", Name: "preset", Value: "lazyvim", CapturedAt: time.Now()},
				{Provider: "ssh", Name: "host.github", Value: "github.com", CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim", "ssh"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Should create files without error even if providers are not handled
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Layer should just have the name since no sections are added
		assert.Contains(t, string(content), "name: captured")
	})

	t.Run("generates git init defaultBranch from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "git", Name: "init.defaultBranch", Value: "main", CapturedAt: time.Now()},
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

		// Verify git init config
		assert.Contains(t, string(content), "init:")
		assert.Contains(t, string(content), "defaultBranch: main")
	})

	t.Run("generates bash default when only bash shell files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "shell", Name: ".bashrc", Value: "~/.bashrc", CapturedAt: time.Now()},
				{Provider: "shell", Name: ".bash_profile", Value: "~/.bash_profile", CapturedAt: time.Now()},
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

		// Verify shell defaults to bash
		assert.Contains(t, string(content), "default: bash")
	})

	t.Run("returns nil shell when no shell files found", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Shell provider with unrecognized file
				{Provider: "shell", Name: ".config", Value: "~/.config", CapturedAt: time.Now()},
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

		// Shell section should not be present
		assert.NotContains(t, string(content), "shell:")
	})

	t.Run("returns nil runtime when no tools captured", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Runtime provider but with non-string value
				{Provider: "runtime", Name: "invalid", Value: 123, CapturedAt: time.Now()},
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

		// Runtime section should have the tool
		assert.Contains(t, string(content), "runtime:")
		assert.Contains(t, string(content), "name: invalid")
	})
}
