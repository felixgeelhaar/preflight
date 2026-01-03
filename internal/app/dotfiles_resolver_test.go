package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotfilesResolver_Resolve_SharedOnly(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create shared dotfiles in home-mirrored structure
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve to shared dotfiles
	resolved := resolver.Resolve(".config/nvim")
	assert.Equal(t, filepath.Join(configRoot, ".config", "nvim"), resolved)
}

func TestDotfilesResolver_Resolve_TargetOverride(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create both shared and target-specific dotfiles
	sharedNvim := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(sharedNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedNvim, "init.lua"), []byte("-- shared"), 0644))

	workNvim := filepath.Join(configRoot, ".config.work", "nvim")
	require.NoError(t, os.MkdirAll(workNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workNvim, "init.lua"), []byte("-- work"), 0644))

	resolver := NewDotfilesResolver(configRoot, "work")

	// Should resolve to target-specific dotfiles
	resolved := resolver.Resolve(".config/nvim")
	assert.Equal(t, filepath.Join(configRoot, ".config.work", "nvim"), resolved)
}

func TestDotfilesResolver_Resolve_TargetFallsBackToShared(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create only shared dotfiles (no target-specific)
	sharedNvim := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(sharedNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedNvim, "init.lua"), []byte("-- shared"), 0644))

	resolver := NewDotfilesResolver(configRoot, "work")

	// Should fall back to shared dotfiles
	resolved := resolver.Resolve(".config/nvim")
	assert.Equal(t, filepath.Join(configRoot, ".config", "nvim"), resolved)
}

func TestDotfilesResolver_Resolve_RootLevelFile(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create root-level file like .gitconfig
	require.NoError(t, os.WriteFile(filepath.Join(configRoot, ".gitconfig"), []byte("[user]"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve root-level file
	resolved := resolver.Resolve(".gitconfig")
	assert.Equal(t, filepath.Join(configRoot, ".gitconfig"), resolved)
}

func TestDotfilesResolver_Resolve_RootLevelTargetOverride(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create both shared and target-specific root-level files
	require.NoError(t, os.WriteFile(filepath.Join(configRoot, ".gitconfig"), []byte("[user]\nname = Shared"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(configRoot, ".gitconfig.work"), []byte("[user]\nname = Work"), 0644))

	resolver := NewDotfilesResolver(configRoot, "work")

	// Should resolve to target-specific file
	resolved := resolver.Resolve(".gitconfig")
	assert.Equal(t, filepath.Join(configRoot, ".gitconfig.work"), resolved)
}

func TestDotfilesResolver_Resolve_NotFound(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "work")

	// Non-existent path should return empty
	resolved := resolver.Resolve("nonexistent")
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_Resolve_EmptyPath(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "work")

	// Empty path should return empty
	resolved := resolver.Resolve("")
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_ResolveWithFallback_AbsolutePath(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	absPath := "/absolute/path/to/config"
	resolved := resolver.ResolveWithFallback(absPath)
	assert.Equal(t, absPath, resolved)
}

func TestDotfilesResolver_ResolveWithFallback_LegacyDotfilesPrefix(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create home-mirrored nvim config
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Path with legacy dotfiles/ prefix should resolve to home-mirrored path
	resolved := resolver.ResolveWithFallback("dotfiles/nvim")
	assert.Equal(t, filepath.Join(configRoot, ".config", "nvim"), resolved)
}

func TestDotfilesResolver_ResolveWithFallback_LegacyGit(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create home-mirrored git config
	require.NoError(t, os.WriteFile(filepath.Join(configRoot, ".gitconfig"), []byte("[user]"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Legacy dotfiles/git should resolve to .gitconfig
	resolved := resolver.ResolveWithFallback("dotfiles/git")
	assert.Equal(t, filepath.Join(configRoot, ".gitconfig"), resolved)
}

func TestDotfilesResolver_ResolveDirectory(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a directory
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve directory
	resolved, exists := resolver.ResolveDirectory(".config/nvim")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(configRoot, ".config", "nvim"), resolved)

	// File should not resolve as directory
	resolved, exists = resolver.ResolveDirectory(".config/nvim/init.lua")
	assert.False(t, exists)
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_ResolveFile(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a file in home-mirrored structure
	starshipDir := filepath.Join(configRoot, ".config")
	require.NoError(t, os.MkdirAll(starshipDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(starshipDir, "starship.toml"), []byte("[character]"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve file
	resolved, exists := resolver.ResolveFile(".config/starship.toml")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(configRoot, ".config", "starship.toml"), resolved)

	// Directory should not resolve as file
	resolved, exists = resolver.ResolveFile(".config")
	assert.False(t, exists)
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_GetTargetDir(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"

	// With target - still returns config root (home-mirrored uses suffixes)
	resolver := NewDotfilesResolver(configRoot, "work")
	assert.Equal(t, "/home/user/preflight", resolver.GetTargetDir())

	// Without target - returns config root
	resolver = NewDotfilesResolver(configRoot, "")
	assert.Equal(t, "/home/user/preflight", resolver.GetTargetDir())
}

func TestDotfilesResolver_GetSharedDir(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"
	resolver := NewDotfilesResolver(configRoot, "work")

	// Returns config root (home-mirrored structure)
	assert.Equal(t, "/home/user/preflight", resolver.GetSharedDir())
}

func TestDotfilesResolver_Accessors(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"
	target := "work"

	resolver := NewDotfilesResolver(configRoot, target)

	assert.Equal(t, configRoot, resolver.ConfigRoot())
	assert.Equal(t, target, resolver.Target())
}

func TestDotfilesResolver_PathTraversalProtection(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a valid file inside configRoot
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Test path traversal attempts
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"basic traversal", "../../../etc/passwd", ""},
		{"hidden traversal", ".config/../../../etc/passwd", ""},
		{"double dot in middle", ".config/../../etc/passwd", ""},
		{"double dot only", "..", ""},
		{"valid path", ".config/nvim", filepath.Join(configRoot, ".config", "nvim")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := resolver.Resolve(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDotfilesResolver_ResolveWithFallback_PathTraversal(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	// All path traversal attempts should return empty string
	traversalPaths := []string{
		"../../../etc/passwd",
		".config/../../../etc/passwd",
		"dotfiles/../../../etc/passwd",
		"..\\..\\..\\etc\\passwd",
	}

	for _, path := range traversalPaths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			result := resolver.ResolveWithFallback(path)
			assert.Equal(t, "", result, "path traversal should be rejected: %s", path)
		})
	}
}

func TestIsPathWithinRoot(t *testing.T) {
	t.Parallel()

	root := "/home/user/config"

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid subpath", "/home/user/config/.config/nvim", true},
		{"valid file", "/home/user/config/.gitconfig", true},
		{"exact root", "/home/user/config", true},
		{"escapes root", "/home/user/other", false},
		{"parent traversal", "/home/user/config/../other", false},
		{"escapes via symlink-like", "/home/user/config/../../etc/passwd", false},
		{"absolute escape", "/etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isPathWithinRoot(root, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDotfilesResolver_ResolveWithFallback_DirectResolve(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a direct home-mirrored path
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve directly without fallback
	resolved := resolver.ResolveWithFallback(".config/nvim")
	assert.Equal(t, nvimDir, resolved)
}

func TestDotfilesResolver_ResolveWithFallback_FallbackToConfigRoot(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Don't create the path - should fall back to configRoot + path
	resolver := NewDotfilesResolver(configRoot, "")

	// Should return path under configRoot as last resort
	resolved := resolver.ResolveWithFallback("some/new/path")
	assert.Equal(t, filepath.Join(configRoot, "some", "new", "path"), resolved)
}

func TestDotfilesResolver_ResolveWithFallback_EmptyPath(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	// Empty path should return empty
	resolved := resolver.ResolveWithFallback("")
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_ResolveWithFallback_LegacyDotfilesStructure(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create BOTH legacy and home-mirrored structure
	// Legacy structure exists but home-mirrored takes priority if present
	legacyDir := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(legacyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "init.lua"), []byte("-- legacy"), 0644))

	// Create home-mirrored structure
	nvimDir := filepath.Join(configRoot, ".config", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("-- new"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Legacy dotfiles/ prefix should map to home-mirrored path
	resolved := resolver.ResolveWithFallback("dotfiles/nvim")
	assert.Equal(t, filepath.Join(configRoot, ".config", "nvim"), resolved)
}

func TestDotfilesResolver_LegacyToHomeRelPath_KnownProviders(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	// Test known provider mappings
	tests := []struct {
		legacy   string
		expected string
	}{
		{"nvim", ".config/nvim"},
		{"shell", ".zshrc"},
		{"starship", ".config/starship.toml"},
		{"tmux", ".tmux.conf"},
		{"ssh", ".ssh"},
		{"git", ".gitconfig"},
		{"terminal", ".config/wezterm"},
		{"unknown", "unknown"}, // Unknown should pass through
	}

	for _, tt := range tests {
		t.Run(tt.legacy, func(t *testing.T) {
			t.Parallel()
			result := resolver.legacyToHomeRelPath(tt.legacy)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDotfilesResolver_LegacyToHomeRelPath_WithSubpath(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	// Test legacy path with subpath
	result := resolver.legacyToHomeRelPath("nvim/lua/plugins")
	assert.Equal(t, ".config/nvim/lua/plugins", result)
}

func TestDotfilesResolver_IsDirectory_File(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a file (not directory)
	filePath := filepath.Join(configRoot, ".gitconfig")
	require.NoError(t, os.WriteFile(filePath, []byte("[user]"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// isDirectory should return false for files
	assert.False(t, resolver.isDirectory(filePath))
}

func TestDotfilesResolver_IsDirectory_NonExistent(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	resolver := NewDotfilesResolver(configRoot, "")

	// isDirectory should return false for non-existent paths
	assert.False(t, resolver.isDirectory(filepath.Join(configRoot, "nonexistent")))
}
