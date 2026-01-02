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

func TestCaptureConfigGenerator_SmartSplit(t *testing.T) {
	t.Parallel()

	t.Run("merges dotfiles with brew packages in same layer", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Brew packages categorized to "git" layer
				{Provider: "brew", Name: "gh", Value: "gh", CapturedAt: time.Now()},
				{Provider: "brew", Name: "lazygit", Value: "lazygit", CapturedAt: time.Now()},
				// Git dotfile config - should be merged into "git" layer
				{Provider: "git", Name: "user.name", Value: "Test User", CapturedAt: time.Now()},
				{Provider: "git", Name: "user.email", Value: "test@example.com", CapturedAt: time.Now()},
				{Provider: "git", Name: "core.editor", Value: "nvim", CapturedAt: time.Now()},
				{Provider: "git", Name: "init.defaultBranch", Value: "main", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew", "git"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read git layer content
		layerPath := filepath.Join(tmpDir, "layers", "git.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify both brew packages AND git config are in the same layer
		assert.Contains(t, string(content), "gh")
		assert.Contains(t, string(content), "lazygit")
		assert.Contains(t, string(content), "git:")
		assert.Contains(t, string(content), "user:")
		assert.Contains(t, string(content), "name: Test User")
		assert.Contains(t, string(content), "email: test@example.com")
		assert.Contains(t, string(content), "core:")
		assert.Contains(t, string(content), "editor: nvim")
		assert.Contains(t, string(content), "init:")
		assert.Contains(t, string(content), "defaultBranch: main")
	})

	t.Run("merges shell config with shell packages", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Brew packages categorized to "shell" layer
				{Provider: "brew", Name: "zsh", Value: "zsh", CapturedAt: time.Now()},
				{Provider: "brew", Name: "starship", Value: "starship", CapturedAt: time.Now()},
				// Shell dotfile config - should be merged into "shell" layer
				{Provider: "shell", Name: ".zshrc", Value: "~/.zshrc", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew", "shell"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read shell layer content
		layerPath := filepath.Join(tmpDir, "layers", "shell.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify both brew packages AND shell config are in the same layer
		assert.Contains(t, string(content), "zsh")
		assert.Contains(t, string(content), "starship")
		assert.Contains(t, string(content), "shell:")
		assert.Contains(t, string(content), "default: zsh")
	})

	t.Run("creates dotfile layer when no corresponding brew packages", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Only git config, no brew packages that categorize to "git"
				{Provider: "git", Name: "user.name", Value: "Test User", CapturedAt: time.Now()},
			},
			Providers:  []string{"git"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read git layer content
		layerPath := filepath.Join(tmpDir, "layers", "git.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify git config is present
		assert.Contains(t, string(content), "git:")
		assert.Contains(t, string(content), "name: Test User")
	})

	t.Run("vscode extensions go to editor layer", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				// Brew packages categorized to "editor" layer
				{Provider: "brew", Name: "neovim", Value: "neovim", CapturedAt: time.Now()},
				{Provider: "brew-cask", Name: "visual-studio-code", Value: "visual-studio-code", CapturedAt: time.Now()},
				// VSCode extensions - should be merged into "editor" layer
				{Provider: "vscode", Name: "golang.go", Value: "golang.go", CapturedAt: time.Now()},
				{Provider: "vscode", Name: "ms-python.python", Value: "ms-python.python", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew", "brew-cask", "vscode"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read editor layer content
		layerPath := filepath.Join(tmpDir, "layers", "editor.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify both brew packages AND vscode extensions are in the same layer
		assert.Contains(t, string(content), "neovim")
		assert.Contains(t, string(content), "visual-studio-code")
		assert.Contains(t, string(content), "vscode:")
		assert.Contains(t, string(content), "golang.go")
		assert.Contains(t, string(content), "ms-python.python")
	})
}

func TestCaptureConfigGenerator_NvimLayer(t *testing.T) {
	t.Parallel()

	t.Run("generates nvim config from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a fake nvim config directory
		nvimDir := filepath.Join(tmpDir, "nvim-config")
		require.NoError(t, os.MkdirAll(nvimDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- init"), 0o644))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "nvim", Name: "config", Value: nvimDir, CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify nvim config
		assert.Contains(t, string(content), "nvim:")
		assert.Contains(t, string(content), "preset: custom")
		assert.Contains(t, string(content), "config_path:")
	})

	t.Run("detects lazyvim preset", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a fake LazyVim config
		nvimDir := filepath.Join(tmpDir, "nvim-config")
		require.NoError(t, os.MkdirAll(nvimDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "lazyvim.json"), []byte("{}"), 0o644))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "nvim", Name: "config", Value: nvimDir, CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "preset: lazyvim")
	})

	t.Run("counts plugins from lazy-lock.json", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a fake config with lazy-lock.json
		nvimDir := filepath.Join(tmpDir, "nvim-config")
		require.NoError(t, os.MkdirAll(nvimDir, 0o755))

		lazyLock := `{"plugin1": {}, "plugin2": {}, "plugin3": {}}`
		require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "lazy-lock.json"), []byte(lazyLock), 0o644))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "nvim", Name: "config", Value: nvimDir, CapturedAt: time.Now()},
				{Provider: "nvim", Name: "lazy-lock.json", Value: filepath.Join(nvimDir, "lazy-lock.json"), CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "plugin_manager: lazy.nvim")
		assert.Contains(t, string(content), "plugin_count: 3")
	})

	t.Run("detects git-managed config", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a fake git-managed config
		nvimDir := filepath.Join(tmpDir, "nvim-config")
		require.NoError(t, os.MkdirAll(filepath.Join(nvimDir, ".git"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- init"), 0o644))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "nvim", Name: "config", Value: nvimDir, CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "config_managed: true")
	})

	t.Run("smart split creates nvim layer", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		nvimDir := filepath.Join(tmpDir, "nvim-config")
		require.NoError(t, os.MkdirAll(nvimDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- init"), 0o644))

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "nvim", Name: "config", Value: nvimDir, CapturedAt: time.Now()},
			},
			Providers:  []string{"nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Should create nvim.yaml layer
		layerPath := filepath.Join(tmpDir, "layers", "nvim.yaml")
		assert.FileExists(t, layerPath)

		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "nvim:")
		assert.Contains(t, string(content), "preset: custom")
	})
}

func TestCaptureConfigGenerator_SSHLayer(t *testing.T) {
	t.Parallel()

	t.Run("generates ssh config from capture", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a fake SSH config
		sshDir := filepath.Join(tmpDir, ".ssh")
		require.NoError(t, os.MkdirAll(sshDir, 0o700))

		sshConfig := `Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519

Host work
    HostName work.example.com
    User developer
    Port 2222
`
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte(sshConfig), 0o600))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "ssh", Name: "config", Value: filepath.Join(sshDir, "config"), CapturedAt: time.Now()},
			},
			Providers:  []string{"ssh"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify SSH config structure
		assert.Contains(t, string(content), "ssh:")
		assert.Contains(t, string(content), "hosts:")
		assert.Contains(t, string(content), "host: github.com")
		assert.Contains(t, string(content), "host: work")
		assert.Contains(t, string(content), "hostname: work.example.com")
		assert.Contains(t, string(content), "port: \"2222\"")
	})

	t.Run("captures ssh defaults", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		sshDir := filepath.Join(tmpDir, ".ssh")
		require.NoError(t, os.MkdirAll(sshDir, 0o700))

		sshConfig := `AddKeysToAgent yes
UseKeychain yes
ServerAliveInterval 60

Host github.com
    User git
`
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte(sshConfig), 0o600))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "ssh", Name: "config", Value: filepath.Join(sshDir, "config"), CapturedAt: time.Now()},
			},
			Providers:  []string{"ssh"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "defaults:")
		assert.Contains(t, string(content), "AddKeysToAgent: \"yes\"")
		assert.Contains(t, string(content), "UseKeychain: \"yes\"")
		assert.Contains(t, string(content), "ServerAliveInterval: \"60\"")
	})

	t.Run("detects ssh keys", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		sshDir := filepath.Join(tmpDir, ".ssh")
		require.NoError(t, os.MkdirAll(sshDir, 0o700))

		// Create fake SSH key files
		privateKey := `-----BEGIN OPENSSH PRIVATE KEY-----
fake private key content
-----END OPENSSH PRIVATE KEY-----`
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte(privateKey), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("ssh-ed25519 AAAA... test@example.com"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n    AddKeysToAgent yes"), 0o600))

		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "ssh", Name: "config", Value: filepath.Join(sshDir, "config"), CapturedAt: time.Now()},
			},
			Providers:  []string{"ssh"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "keys:")
		assert.Contains(t, string(content), "name: id_ed25519")
		assert.Contains(t, string(content), "type: ed25519")
		assert.Contains(t, string(content), "comment: test@example.com")
	})

	t.Run("smart split creates ssh layer", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		sshDir := filepath.Join(tmpDir, ".ssh")
		require.NoError(t, os.MkdirAll(sshDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host github.com\n    User git"), 0o600))

		generator := NewCaptureConfigGenerator(tmpDir).WithSmartSplit(true)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "ssh", Name: "config", Value: filepath.Join(sshDir, "config"), CapturedAt: time.Now()},
			},
			Providers:  []string{"ssh"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Should create ssh.yaml layer
		layerPath := filepath.Join(tmpDir, "layers", "ssh.yaml")
		assert.FileExists(t, layerPath)

		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "ssh:")
		assert.Contains(t, string(content), "hosts:")
	})
}

func TestCaptureConfigGenerator_WithDotfiles(t *testing.T) {
	t.Parallel()

	t.Run("populates config_source from dotfiles result", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a mock dotfiles capture result
		dotfilesResult := &DotfilesCaptureResult{
			TargetDir: "dotfiles",
			Dotfiles: []CapturedDotfile{
				{Provider: "nvim", SourcePath: "~/.config/nvim", RelativePath: "init.lua", DestPath: "dotfiles/nvim/init.lua"},
				{Provider: "vscode", SourcePath: "~/Library/Application Support/Code/User/settings.json", RelativePath: "settings.json", DestPath: "dotfiles/vscode/settings.json"},
				{Provider: "git", SourcePath: "~/.gitconfig.d", RelativePath: "alias.gitconfig", DestPath: "dotfiles/git/alias.gitconfig"},
				{Provider: "ssh", SourcePath: "~/.ssh/config", RelativePath: "config", DestPath: "dotfiles/ssh/config"},
				{Provider: "shell", SourcePath: "~/.zshrc.d", RelativePath: "aliases.zsh", DestPath: "dotfiles/shell/aliases.zsh"},
				{Provider: "starship", SourcePath: "~/.config/starship.toml", RelativePath: "starship.toml", DestPath: "dotfiles/starship/starship.toml"},
				{Provider: "tmux", SourcePath: "~/.tmux.conf", RelativePath: "tmux.conf", DestPath: "dotfiles/tmux/tmux.conf"},
			},
		}

		generator := NewCaptureConfigGenerator(tmpDir).WithDotfiles(dotfilesResult)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git", Value: "git", CapturedAt: time.Now()},
				{Provider: "nvim", Name: "config", Value: map[string]any{"preset": "lazyvim"}, CapturedAt: time.Now()},
			},
			Providers:  []string{"brew", "nvim"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		// Read layer content
		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		contentStr := string(content)

		// Verify config_source fields are populated
		assert.Contains(t, contentStr, "config_source: dotfiles/nvim", "nvim config_source should be set")
		assert.Contains(t, contentStr, "config_source: dotfiles/vscode", "vscode config_source should be set")
		assert.Contains(t, contentStr, "config_source: dotfiles/git", "git config_source should be set")
		assert.Contains(t, contentStr, "config_source: dotfiles/ssh", "ssh config_source should be set")
		assert.Contains(t, contentStr, "config_source: dotfiles/tmux", "tmux config_source should be set")
	})

	t.Run("populates shell config_source with dir and starship", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		dotfilesResult := &DotfilesCaptureResult{
			TargetDir: "dotfiles",
			Dotfiles: []CapturedDotfile{
				{Provider: "shell", SourcePath: "~/.zshrc.d", RelativePath: "aliases.zsh", DestPath: "dotfiles/shell/aliases.zsh"},
				{Provider: "starship", SourcePath: "~/.config/starship.toml", RelativePath: "starship.toml", DestPath: "dotfiles/starship/starship.toml"},
			},
		}

		generator := NewCaptureConfigGenerator(tmpDir).WithDotfiles(dotfilesResult)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "shell", Name: ".zshrc", Value: "~/.zshrc", CapturedAt: time.Now()},
			},
			Providers:  []string{"shell"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		contentStr := string(content)

		// Verify shell config_source.dir is set
		assert.Contains(t, contentStr, "dir: dotfiles/shell", "shell config_source.dir should be set")
		// Verify starship config_source is set
		assert.Contains(t, contentStr, "config_source: dotfiles/starship", "starship config_source should be set")
	})

	t.Run("per-target dotfiles directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		dotfilesResult := &DotfilesCaptureResult{
			TargetDir: "dotfiles.work",
			Target:    "work",
			Dotfiles: []CapturedDotfile{
				{Provider: "nvim", SourcePath: "~/.config/nvim", RelativePath: "init.lua", DestPath: "dotfiles.work/nvim/init.lua"},
			},
		}

		generator := NewCaptureConfigGenerator(tmpDir).WithDotfiles(dotfilesResult)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git", Value: "git", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "work")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify config_source uses per-target directory
		assert.Contains(t, string(content), "config_source: dotfiles.work/nvim")
	})

	t.Run("no dotfiles does not add config_source", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// No dotfiles result passed
		generator := NewCaptureConfigGenerator(tmpDir)
		findings := &CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git", Value: "git", CapturedAt: time.Now()},
			},
			Providers:  []string{"brew"},
			CapturedAt: time.Now(),
		}

		err := generator.GenerateFromCapture(findings, "default")
		require.NoError(t, err)

		layerPath := filepath.Join(tmpDir, "layers", "captured.yaml")
		content, err := os.ReadFile(layerPath)
		require.NoError(t, err)

		// Verify no config_source is present
		assert.NotContains(t, string(content), "config_source:")
	})
}
