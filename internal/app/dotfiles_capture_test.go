package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotfilesCapturer_CaptureNvimConfig(t *testing.T) {
	t.Parallel()

	// Create temp directories
	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create nvim config structure
	nvimDir := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- init"), 0644))

	luaDir := filepath.Join(nvimDir, "lua")
	require.NoError(t, os.MkdirAll(luaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(luaDir, "config.lua"), []byte("-- config"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("nvim")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Positive(t, len(result.Dotfiles), "should capture nvim files")
	assert.Equal(t, filepath.Join(targetDir, "dotfiles"), result.TargetDir)

	// Verify files were copied
	copiedInit := filepath.Join(targetDir, "dotfiles", "nvim", "init.lua")
	assert.FileExists(t, copiedInit)

	copiedConfig := filepath.Join(targetDir, "dotfiles", "nvim", "lua", "config.lua")
	assert.FileExists(t, copiedConfig)
}

func TestDotfilesCapturer_ExcludesPluginCaches(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create nvim config with lazy plugin directory
	nvimDir := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- init"), 0644))

	// Create lazy directory (should be excluded)
	lazyDir := filepath.Join(nvimDir, "lazy")
	require.NoError(t, os.MkdirAll(lazyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(lazyDir, "plugin.lua"), []byte("-- plugin"), 0644))

	// Create pack directory (should be excluded)
	packDir := filepath.Join(nvimDir, "pack")
	require.NoError(t, os.MkdirAll(packDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(packDir, "native.lua"), []byte("-- native"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("nvim")
	require.NoError(t, err)

	// Check that lazy and pack directories were NOT captured
	for _, d := range result.Dotfiles {
		assert.NotContains(t, d.RelativePath, "lazy", "lazy directory should be excluded")
		assert.NotContains(t, d.RelativePath, "pack", "pack directory should be excluded")
	}

	// Verify excluded files don't exist in output
	assert.NoFileExists(t, filepath.Join(targetDir, "dotfiles", "nvim", "lazy", "plugin.lua"))
	assert.NoFileExists(t, filepath.Join(targetDir, "dotfiles", "nvim", "pack", "native.lua"))

	// But init.lua should exist
	assert.FileExists(t, filepath.Join(targetDir, "dotfiles", "nvim", "init.lua"))
}

func TestDotfilesCapturer_SkipsNonExistentPaths(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Don't create any nvim config - it shouldn't exist
	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("nvim")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have no dotfiles captured
	assert.Empty(t, result.Dotfiles)
	assert.Empty(t, result.Warnings)
}

func TestDotfilesCapturer_PerTargetDirectory(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create starship config
	starshipDir := filepath.Join(homeDir, ".config")
	require.NoError(t, os.MkdirAll(starshipDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(starshipDir, "starship.toml"), []byte("[character]"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir).WithTarget("work")

	result, err := capturer.CaptureProvider("starship")
	require.NoError(t, err)

	// Should use per-target directory
	assert.Equal(t, filepath.Join(targetDir, "dotfiles.work"), result.TargetDir)
	assert.Equal(t, "work", result.Target)

	// File should be in per-target directory
	copiedStarship := filepath.Join(targetDir, "dotfiles.work", "starship", "starship.toml")
	assert.FileExists(t, copiedStarship)
}

func TestDotfilesCapturer_CaptureShellConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create zsh config directory
	zshDir := filepath.Join(homeDir, ".zshrc.d")
	require.NoError(t, os.MkdirAll(zshDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(zshDir, "aliases.zsh"), []byte("alias ll='ls -la'"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(zshDir, "functions.zsh"), []byte("function foo() {}"), 0644))

	// Create files that should be excluded
	require.NoError(t, os.WriteFile(filepath.Join(zshDir, ".zcompdump"), []byte("cache"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("shell")
	require.NoError(t, err)

	// Check that aliases and functions were captured
	hasAliases := false
	hasFunctions := false
	for _, d := range result.Dotfiles {
		if d.RelativePath == "aliases.zsh" {
			hasAliases = true
		}
		if d.RelativePath == "functions.zsh" {
			hasFunctions = true
		}
		// .zcompdump should be excluded
		assert.NotContains(t, d.RelativePath, "zcompdump")
	}

	assert.True(t, hasAliases, "should capture aliases.zsh")
	assert.True(t, hasFunctions, "should capture functions.zsh")
}

func TestDotfilesCapturer_CaptureSSHConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create SSH config (should be captured)
	sshDir := filepath.Join(homeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host github.com"), 0644))

	// Create SSH keys (should NOT be captured)
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("PRIVATE KEY"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("PUBLIC KEY"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("github.com..."), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("ssh")
	require.NoError(t, err)

	// Should only capture config
	assert.Len(t, result.Dotfiles, 1)
	assert.Equal(t, "config", result.Dotfiles[0].RelativePath)

	// Verify private keys were NOT captured
	assert.NoFileExists(t, filepath.Join(targetDir, "dotfiles", "ssh", "id_ed25519"))
	assert.NoFileExists(t, filepath.Join(targetDir, "dotfiles", "ssh", "known_hosts"))
}

func TestDotfilesCapturer_CaptureAll(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create configs for multiple providers
	nvimDir := filepath.Join(homeDir, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	starshipPath := filepath.Join(homeDir, ".config", "starship.toml")
	require.NoError(t, os.WriteFile(starshipPath, []byte("[character]"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.Capture()
	require.NoError(t, err)

	// Should have dotfiles from multiple providers
	byProvider := result.DotfilesByProvider()
	assert.Contains(t, byProvider, "nvim")
	assert.Contains(t, byProvider, "starship")
}

func TestDotfilesCapturer_FileCount(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create nvim config with subdirectory
	nvimDir := filepath.Join(homeDir, ".config", "nvim")
	luaDir := filepath.Join(nvimDir, "lua")
	require.NoError(t, os.MkdirAll(luaDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(luaDir, "config.lua"), []byte("--"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("nvim")
	require.NoError(t, err)

	// FileCount should not include directories
	fileCount := result.FileCount()
	assert.Equal(t, 2, fileCount) // init.lua and config.lua

	// Total size should be sum of file sizes
	assert.Positive(t, result.TotalSize())
}

func TestDotfilesCapturer_ShouldExclude(t *testing.T) {
	t.Parallel()

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, "/home/user", "/tmp")

	tests := []struct {
		path     string
		excludes []string
		expected bool
	}{
		{"lazy", []string{"lazy"}, true},
		{"lazy/plugin.lua", []string{"lazy"}, true},
		{"some/path/lazy/file.lua", []string{"lazy"}, true},
		{"init.lua", []string{"lazy"}, false},
		{".zcompdump", []string{".zcompdump*"}, true},
		{".zcompdump-hostname-5.8", []string{".zcompdump*"}, true},
		{"aliases.zsh", []string{".zcompdump*"}, false},
		{"test.swp", []string{"*.swp"}, true},
		{"config.lua", []string{"*.swp"}, false},
	}

	for _, tt := range tests {
		result := capturer.shouldExclude(tt.path, tt.excludes)
		assert.Equal(t, tt.expected, result, "shouldExclude(%q, %v)", tt.path, tt.excludes)
	}
}

func TestDotfilesCapturer_ExpandPath(t *testing.T) {
	t.Parallel()

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, "/home/user", "/tmp")

	tests := []struct {
		input    string
		expected string
	}{
		{"~/.config/nvim", "/home/user/.config/nvim"},
		{"~/", "/home/user"}, // filepath.Join removes trailing slash
		{"~", "/home/user"},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := capturer.expandPath(tt.input)
		assert.Equal(t, tt.expected, result, "expandPath(%q)", tt.input)
	}
}

func TestDefaultCaptureConfigs(t *testing.T) {
	t.Parallel()

	configs := DefaultCaptureConfigs()

	// Should have configs for all expected providers
	providers := make(map[string]bool)
	for _, cfg := range configs {
		providers[cfg.Provider] = true
	}

	assert.Contains(t, providers, "nvim")
	assert.Contains(t, providers, "shell")
	assert.Contains(t, providers, "starship")
	assert.Contains(t, providers, "tmux")
	assert.Contains(t, providers, "vscode")
	assert.Contains(t, providers, "ssh")
	assert.Contains(t, providers, "git")

	// Verify nvim has correct excludes
	for _, cfg := range configs {
		if cfg.Provider == "nvim" {
			assert.Contains(t, cfg.ExcludePaths, "lazy")
			assert.Contains(t, cfg.ExcludePaths, "pack")
		}
		if cfg.Provider == "ssh" {
			assert.Contains(t, cfg.ExcludePaths, "id_*")
			assert.Contains(t, cfg.ExcludePaths, "known_hosts")
		}
	}
}
