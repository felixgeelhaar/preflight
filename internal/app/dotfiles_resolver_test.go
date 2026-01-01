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

	// Create shared dotfiles
	nvimDir := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve to shared dotfiles
	resolved := resolver.Resolve("nvim")
	assert.Equal(t, filepath.Join(configRoot, "dotfiles", "nvim"), resolved)
}

func TestDotfilesResolver_Resolve_TargetOverride(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create both shared and target-specific dotfiles
	sharedNvim := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(sharedNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedNvim, "init.lua"), []byte("-- shared"), 0644))

	workNvim := filepath.Join(configRoot, "dotfiles.work", "nvim")
	require.NoError(t, os.MkdirAll(workNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workNvim, "init.lua"), []byte("-- work"), 0644))

	resolver := NewDotfilesResolver(configRoot, "work")

	// Should resolve to target-specific dotfiles
	resolved := resolver.Resolve("nvim")
	assert.Equal(t, filepath.Join(configRoot, "dotfiles.work", "nvim"), resolved)
}

func TestDotfilesResolver_Resolve_TargetFallsBackToShared(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create only shared dotfiles (no target-specific)
	sharedNvim := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(sharedNvim, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedNvim, "init.lua"), []byte("-- shared"), 0644))

	resolver := NewDotfilesResolver(configRoot, "work")

	// Should fall back to shared dotfiles
	resolved := resolver.Resolve("nvim")
	assert.Equal(t, filepath.Join(configRoot, "dotfiles", "nvim"), resolved)
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

func TestDotfilesResolver_ResolveWithFallback_DotfilesPrefix(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create dotfiles
	nvimDir := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Path with dotfiles/ prefix should resolve correctly
	resolved := resolver.ResolveWithFallback("dotfiles/nvim")
	assert.Equal(t, filepath.Join(configRoot, "dotfiles", "nvim"), resolved)
}

func TestDotfilesResolver_ResolveDirectory(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a directory
	nvimDir := filepath.Join(configRoot, "dotfiles", "nvim")
	require.NoError(t, os.MkdirAll(nvimDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nvimDir, "init.lua"), []byte("--"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve directory
	resolved, exists := resolver.ResolveDirectory("nvim")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(configRoot, "dotfiles", "nvim"), resolved)

	// File should not resolve as directory
	resolved, exists = resolver.ResolveDirectory("nvim/init.lua")
	assert.False(t, exists)
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_ResolveFile(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	// Create a file
	starshipDir := filepath.Join(configRoot, "dotfiles", "starship")
	require.NoError(t, os.MkdirAll(starshipDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(starshipDir, "starship.toml"), []byte("[character]"), 0644))

	resolver := NewDotfilesResolver(configRoot, "")

	// Should resolve file
	resolved, exists := resolver.ResolveFile("starship/starship.toml")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(configRoot, "dotfiles", "starship", "starship.toml"), resolved)

	// Directory should not resolve as file
	resolved, exists = resolver.ResolveFile("starship")
	assert.False(t, exists)
	assert.Equal(t, "", resolved)
}

func TestDotfilesResolver_GetTargetDir(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"

	// With target
	resolver := NewDotfilesResolver(configRoot, "work")
	assert.Equal(t, "/home/user/preflight/dotfiles.work", resolver.GetTargetDir())

	// Without target
	resolver = NewDotfilesResolver(configRoot, "")
	assert.Equal(t, "/home/user/preflight/dotfiles", resolver.GetTargetDir())
}

func TestDotfilesResolver_GetSharedDir(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"
	resolver := NewDotfilesResolver(configRoot, "work")

	assert.Equal(t, "/home/user/preflight/dotfiles", resolver.GetSharedDir())
}

func TestDotfilesResolver_Accessors(t *testing.T) {
	t.Parallel()

	configRoot := "/home/user/preflight"
	target := "work"

	resolver := NewDotfilesResolver(configRoot, target)

	assert.Equal(t, configRoot, resolver.ConfigRoot())
	assert.Equal(t, target, resolver.Target())
}
