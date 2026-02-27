package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: TestInitCmd_Exists and TestInitCmd_HasFlags are in helpers_test.go
// Note: TestDetectAIProvider_* tests are in root_test.go

func TestInitCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"provider default", "provider", ""},
		{"preset default", "preset", ""},
		{"skip-welcome default", "skip-welcome", "false"},
		{"yes default", "yes", "false"},
		{"no-ai default", "no-ai", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := initCmd.Flags().Lookup(tt.flag)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expected, f.DefValue)
		})
	}
}

func TestInitCmd_YesShorthand(t *testing.T) {
	t.Parallel()

	f := initCmd.Flags().Lookup("yes")
	assert.NotNil(t, f)
	assert.Equal(t, "y", f.Shorthand)
}

// --- Batch 3: Generation and non-interactive init tests ---

func TestGenerateManifestForPreset(t *testing.T) {
	tests := []struct {
		name   string
		preset string
	}{
		{"balanced preset", "balanced"},
		{"nvim minimal", "nvim:minimal"},
		{"custom preset", "my-custom-preset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			manifest := generateManifestForPreset(tt.preset)

			// All manifests should contain the preset name in the comment
			assert.Contains(t, manifest, tt.preset)

			// All manifests should have defaults section with mode
			assert.Contains(t, manifest, "defaults:")
			assert.Contains(t, manifest, "mode: intent")

			// All manifests should define targets
			assert.Contains(t, manifest, "targets:")
			assert.Contains(t, manifest, "default:")
			assert.Contains(t, manifest, "- base")
		})
	}
}

func TestGenerateLayerForPreset_AllPresets(t *testing.T) {
	tests := []struct {
		name             string
		preset           string
		expectedContains []string
	}{
		{
			name:   "nvim minimal",
			preset: "nvim:minimal",
			expectedContains: []string{
				"name: base",
				"nvim:",
				"preset: minimal",
				"ensure_install: true",
			},
		},
		{
			name:   "balanced",
			preset: "balanced",
			expectedContains: []string{
				"name: base",
				"nvim:",
				"preset: kickstart",
				"shell:",
				"default: zsh",
				"git:",
				"editor: nvim",
			},
		},
		{
			name:   "nvim balanced",
			preset: "nvim:balanced",
			expectedContains: []string{
				"name: base",
				"nvim:",
				"preset: kickstart",
				"shell:",
				"git:",
			},
		},
		{
			name:   "maximal",
			preset: "maximal",
			expectedContains: []string{
				"name: base",
				"nvim:",
				"preset: astronvim",
				"shell:",
				"starship:",
				"git:",
			},
		},
		{
			name:   "nvim maximal",
			preset: "nvim:maximal",
			expectedContains: []string{
				"name: base",
				"nvim:",
				"preset: astronvim",
				"starship:",
			},
		},
		{
			name:   "shell minimal",
			preset: "shell:minimal",
			expectedContains: []string{
				"name: base",
				"shell:",
				"default: zsh",
			},
		},
		{
			name:   "shell balanced",
			preset: "shell:balanced",
			expectedContains: []string{
				"name: base",
				"shell:",
				"default: zsh",
				"oh-my-zsh",
				"plugins:",
			},
		},
		{
			name:   "git minimal",
			preset: "git:minimal",
			expectedContains: []string{
				"name: base",
				"git:",
				"editor: vim",
			},
		},
		{
			name:   "brew minimal",
			preset: "brew:minimal",
			expectedContains: []string{
				"name: base",
				"packages:",
				"brew:",
				"formulae:",
				"ripgrep",
				"fzf",
			},
		},
		{
			name:   "default unknown preset",
			preset: "unknown-preset",
			expectedContains: []string{
				"name: base",
				"# Add your configuration here",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			layer := generateLayerForPreset(tt.preset)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, layer, expected,
					"layer for preset %q should contain %q", tt.preset, expected)
			}
		})
	}
}

func TestRunInitNonInteractive(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore global vars
	prevPreset := initPreset
	prevOutputDir := initOutputDir
	defer func() {
		initPreset = prevPreset
		initOutputDir = prevOutputDir
	}()

	initPreset = "balanced"
	initOutputDir = tmpDir

	configPath := filepath.Join(tmpDir, "preflight.yaml")

	output := captureStdout(t, func() {
		err := runInitNonInteractive(configPath)
		require.NoError(t, err)
	})

	// Verify preflight.yaml was created
	_, err := os.Stat(configPath)
	require.NoError(t, err, "preflight.yaml should exist")

	// Verify layers/base.yaml was created
	layerPath := filepath.Join(tmpDir, "layers", "base.yaml")
	_, err = os.Stat(layerPath)
	require.NoError(t, err, "layers/base.yaml should exist")

	// Verify preflight.yaml content
	manifestData, err := os.ReadFile(configPath)
	require.NoError(t, err)
	manifest := string(manifestData)
	assert.Contains(t, manifest, "balanced")
	assert.Contains(t, manifest, "defaults:")
	assert.Contains(t, manifest, "mode: intent")

	// Verify layer content
	layerData, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	layer := string(layerData)
	assert.Contains(t, layer, "name: base")
	assert.Contains(t, layer, "nvim:")
	assert.Contains(t, layer, "preset: kickstart")

	// Verify stdout output
	assert.Contains(t, output, "Configuration created:")
	assert.Contains(t, output, "preflight plan")
	assert.Contains(t, output, "preflight apply")
}

func TestRunInitNonInteractive_NoPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore global vars
	prevPreset := initPreset
	prevOutputDir := initOutputDir
	defer func() {
		initPreset = prevPreset
		initOutputDir = prevOutputDir
	}()

	initPreset = ""
	initOutputDir = tmpDir

	configPath := filepath.Join(tmpDir, "preflight.yaml")
	err := runInitNonInteractive(configPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--preset is required")

	// Verify no files were created
	_, statErr := os.Stat(configPath)
	assert.True(t, os.IsNotExist(statErr), "preflight.yaml should not exist")
}

func TestRunInitNonInteractive_CreatesNestedDirs(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "deep", "nested", "dir")

	// Save and restore global vars
	prevPreset := initPreset
	prevOutputDir := initOutputDir
	defer func() {
		initPreset = prevPreset
		initOutputDir = prevOutputDir
	}()

	initPreset = "nvim:minimal"
	initOutputDir = nestedDir

	configPath := filepath.Join(nestedDir, "preflight.yaml")

	output := captureStdout(t, func() {
		err := runInitNonInteractive(configPath)
		require.NoError(t, err)
	})

	// Verify the nested directories and files were created
	_, err := os.Stat(configPath)
	require.NoError(t, err, "preflight.yaml should exist in nested dir")

	layerPath := filepath.Join(nestedDir, "layers", "base.yaml")
	_, err = os.Stat(layerPath)
	require.NoError(t, err, "layers/base.yaml should exist in nested dir")

	// Verify layer content matches nvim:minimal preset
	layerData, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	layer := string(layerData)
	assert.Contains(t, layer, "preset: minimal")
	assert.Contains(t, layer, "nvim:")

	_ = strings.Contains(output, "Configuration created:")
}
