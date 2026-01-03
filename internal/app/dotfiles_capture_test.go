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
	// TargetDir now returns repo root (home-mirrored structure)
	assert.Equal(t, targetDir, result.TargetDir)

	// Verify files were copied to home-mirrored paths
	copiedInit := filepath.Join(targetDir, ".config", "nvim", "init.lua")
	assert.FileExists(t, copiedInit)

	copiedConfig := filepath.Join(targetDir, ".config", "nvim", "lua", "config.lua")
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
		assert.NotContains(t, d.HomeRelPath, "lazy", "lazy directory should be excluded")
		assert.NotContains(t, d.HomeRelPath, "pack", "pack directory should be excluded")
	}

	// Verify excluded files don't exist in output (home-mirrored paths)
	assert.NoFileExists(t, filepath.Join(targetDir, ".config", "nvim", "lazy", "plugin.lua"))
	assert.NoFileExists(t, filepath.Join(targetDir, ".config", "nvim", "pack", "native.lua"))

	// But init.lua should exist
	assert.FileExists(t, filepath.Join(targetDir, ".config", "nvim", "init.lua"))
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

	// TargetDir now returns repo root (home-mirrored structure)
	assert.Equal(t, targetDir, result.TargetDir)
	assert.Equal(t, "work", result.Target)

	// File should be in per-target suffixed path (.config.work/starship.toml)
	copiedStarship := filepath.Join(targetDir, ".config.work", "starship.toml")
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
		if filepath.Base(d.HomeRelPath) == "aliases.zsh" {
			hasAliases = true
		}
		if filepath.Base(d.HomeRelPath) == "functions.zsh" {
			hasFunctions = true
		}
		// .zcompdump should be excluded
		assert.NotContains(t, d.HomeRelPath, "zcompdump")
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
	assert.Equal(t, ".ssh/config", result.Dotfiles[0].HomeRelPath)

	// Verify private keys were NOT captured
	assert.NoFileExists(t, filepath.Join(targetDir, ".ssh", "id_ed25519"))
	assert.NoFileExists(t, filepath.Join(targetDir, ".ssh", "known_hosts"))
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

func TestDotfilesCapturer_BrokenSymlinks_SkipsAndReports(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a directory with a broken symlink
	zshDir := filepath.Join(homeDir, ".config", "zsh")
	require.NoError(t, os.MkdirAll(zshDir, 0755))

	// Create a valid file
	require.NoError(t, os.WriteFile(filepath.Join(zshDir, "aliases.zsh"), []byte("# aliases"), 0644))

	// Create a broken symlink (points to non-existent target)
	brokenLink := filepath.Join(zshDir, "broken.zsh")
	require.NoError(t, os.Symlink("/nonexistent/path/to/file.zsh", brokenLink))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	// Use a custom config that captures the zsh directory
	capturer = capturer.WithConfigs([]DotfilesCaptureConfig{
		{
			Provider:    "shell",
			SourcePaths: []string{"~/.config/zsh"},
		},
	})

	result, err := capturer.Capture()
	require.NoError(t, err, "capture should not fail on broken symlinks")
	require.NotNil(t, result)

	// Should have captured the valid file
	assert.Positive(t, len(result.Dotfiles), "should capture valid files")

	// Should report the broken symlink
	assert.Len(t, result.BrokenSymlinks, 1, "should report broken symlink")
	assert.Equal(t, brokenLink, result.BrokenSymlinks[0].Path)
	assert.Equal(t, "/nonexistent/path/to/file.zsh", result.BrokenSymlinks[0].Target)

	// Verify valid file was copied (home-mirrored structure)
	assert.FileExists(t, filepath.Join(targetDir, ".config", "zsh", "aliases.zsh"))

	// Verify broken symlink was NOT copied
	assert.NoFileExists(t, filepath.Join(targetDir, ".config", "zsh", "broken.zsh"))
}

func TestDotfilesCapturer_BrokenSymlinks_SourcePath(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create a broken symlink as the source path itself
	brokenSourceDir := filepath.Join(homeDir, ".config", "broken-app")
	require.NoError(t, os.MkdirAll(filepath.Dir(brokenSourceDir), 0755))
	require.NoError(t, os.Symlink("/nonexistent/config/dir", brokenSourceDir))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	// Use a custom config that points to the broken symlink
	capturer = capturer.WithConfigs([]DotfilesCaptureConfig{
		{
			Provider:    "broken",
			SourcePaths: []string{"~/.config/broken-app"},
		},
	})

	result, err := capturer.Capture()
	require.NoError(t, err, "capture should not fail when source path is broken symlink")
	require.NotNil(t, result)

	// Should report the broken symlink
	assert.Len(t, result.BrokenSymlinks, 1, "should report broken source symlink")
	assert.Equal(t, brokenSourceDir, result.BrokenSymlinks[0].Path)
}

func TestDotfilesCapturer_ValidSymlinks_Followed(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create actual content in a different location
	actualDir := filepath.Join(homeDir, "actual-configs")
	require.NoError(t, os.MkdirAll(actualDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(actualDir, "config.lua"), []byte("-- config"), 0644))

	// Create a symlink that points to the actual content
	configDir := filepath.Join(homeDir, ".config")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	symlinkDir := filepath.Join(configDir, "app")
	require.NoError(t, os.Symlink(actualDir, symlinkDir))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	capturer = capturer.WithConfigs([]DotfilesCaptureConfig{
		{
			Provider:    "app",
			SourcePaths: []string{"~/.config/app"},
		},
	})

	result, err := capturer.Capture()
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should capture the content through the valid symlink
	assert.Positive(t, len(result.Dotfiles), "should capture files through valid symlinks")
	assert.Empty(t, result.BrokenSymlinks, "should not report any broken symlinks")

	// Verify content was captured (home-mirrored structure)
	assert.FileExists(t, filepath.Join(targetDir, ".config", "app", "config.lua"))
}

func TestDotfilesCapturer_CaptureGitConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create .gitconfig file
	gitconfigContent := `[user]
	name = Test User
	email = test@example.com
[core]
	editor = nvim
`
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte(gitconfigContent), 0644))

	// Create .config/git/ignore file
	gitDir := filepath.Join(homeDir, ".config", "git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "ignore"), []byte("*.log\n"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("git")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should capture both .gitconfig and .config/git/ignore
	assert.GreaterOrEqual(t, len(result.Dotfiles), 2, "should capture both .gitconfig and git/ignore")

	// Verify .gitconfig was captured (home-mirrored structure)
	capturedGitconfig := filepath.Join(targetDir, ".gitconfig")
	assert.FileExists(t, capturedGitconfig, ".gitconfig should be captured")

	// Verify content is correct
	content, err := os.ReadFile(capturedGitconfig)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Test User")

	// Verify .config/git/ignore was captured (home-mirrored structure)
	capturedIgnore := filepath.Join(targetDir, ".config", "git", "ignore")
	assert.FileExists(t, capturedIgnore, "git/ignore should be captured")
}

func TestDotfilesCapturer_ExcludesGitconfigLocal(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	targetDir := t.TempDir()

	// Create .gitconfig file (should be captured)
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("[user]\n\tname = Test\n"), 0644))

	// Create .gitconfig.local file (should be excluded - may contain secrets)
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig.local"), []byte("[user]\n\tsigningkey = SECRET\n"), 0644))

	fs := filesystem.NewRealFileSystem()
	capturer := NewDotfilesCapturer(fs, homeDir, targetDir)

	result, err := capturer.CaptureProvider("git")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify .gitconfig was captured (home-mirrored structure)
	assert.FileExists(t, filepath.Join(targetDir, ".gitconfig"))

	// Verify .gitconfig.local was NOT captured
	assert.NoFileExists(t, filepath.Join(targetDir, ".gitconfig.local"))
}
