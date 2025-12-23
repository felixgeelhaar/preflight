package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigGenerator_GenerateFromPreset(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	gen := NewConfigGenerator(tempDir)

	preset := PresetItem{
		ID:          "nvim:minimal",
		Title:       "Minimal Neovim",
		Description: "Essential plugins only",
		Difficulty:  "Beginner",
	}

	err := gen.GenerateFromPreset(preset)
	require.NoError(t, err)

	// Check preflight.yaml exists
	manifestPath := filepath.Join(tempDir, "preflight.yaml")
	assert.FileExists(t, manifestPath)

	// Verify manifest content
	manifestContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(manifestContent), "targets:")
	assert.Contains(t, string(manifestContent), "default:")
	assert.Contains(t, string(manifestContent), "base")

	// Check layers/base.yaml exists
	layerPath := filepath.Join(tempDir, "layers", "base.yaml")
	assert.FileExists(t, layerPath)

	// Verify layer content
	layerContent, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(layerContent), "name: base")
	assert.Contains(t, string(layerContent), "nvim:")
	assert.Contains(t, string(layerContent), "preset: minimal")
}

func TestConfigGenerator_GenerateFromPreset_ShellPreset(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	gen := NewConfigGenerator(tempDir)

	preset := PresetItem{
		ID:          "shell:oh-my-zsh",
		Title:       "Oh My Zsh",
		Description: "Popular Zsh framework",
		Difficulty:  "Beginner",
	}

	err := gen.GenerateFromPreset(preset)
	require.NoError(t, err)

	// Check layer content includes shell config
	layerPath := filepath.Join(tempDir, "layers", "base.yaml")
	layerContent, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(layerContent), "shell:")
}

func TestConfigGenerator_GenerateFromPreset_GitPreset(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	gen := NewConfigGenerator(tempDir)

	preset := PresetItem{
		ID:          "git:standard",
		Title:       "Standard Git",
		Description: "Common git configuration",
		Difficulty:  "Beginner",
	}

	err := gen.GenerateFromPreset(preset)
	require.NoError(t, err)

	// Check layer content includes git config
	layerPath := filepath.Join(tempDir, "layers", "base.yaml")
	layerContent, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(layerContent), "git:")
}

func TestConfigGenerator_GenerateFromPreset_CreatesDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "new-dir")
	gen := NewConfigGenerator(targetDir)

	preset := PresetItem{
		ID:          "nvim:minimal",
		Title:       "Minimal",
		Description: "Essential plugins",
		Difficulty:  "Beginner",
	}

	err := gen.GenerateFromPreset(preset)
	require.NoError(t, err)

	// Directory and files should exist
	assert.DirExists(t, targetDir)
	assert.FileExists(t, filepath.Join(targetDir, "preflight.yaml"))
}

func TestConfigGenerator_InvalidPresetID(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	gen := NewConfigGenerator(tempDir)

	preset := PresetItem{
		ID:          "invalid", // No colon separator
		Title:       "Invalid",
		Description: "Invalid preset",
	}

	err := gen.GenerateFromPreset(preset)
	assert.Error(t, err)
}
