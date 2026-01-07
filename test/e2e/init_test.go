//go:build e2e
// +build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_Init_NonInteractive_Balanced(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Run init with balanced preset
	output := h.Init("balanced")

	// Verify output
	assert.Contains(t, output, "Configuration created")

	// Verify files were created
	h.AssertConfigFileExists("preflight.yaml")
	h.AssertConfigFileExists("layers/base.yaml")

	// Verify manifest content
	manifest := h.ReadConfigFile("preflight.yaml")
	assert.Contains(t, manifest, "targets:")
	assert.Contains(t, manifest, "default:")
	assert.Contains(t, manifest, "- base")

	// Verify layer content
	layer := h.ReadConfigFile("layers/base.yaml")
	assert.Contains(t, layer, "name: base")
}

func TestE2E_Init_NonInteractive_GitMinimal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Run init with git:minimal preset
	output := h.Init("git:minimal")

	// Verify output
	assert.Contains(t, output, "Configuration created")

	// Verify layer contains git config
	layer := h.ReadConfigFile("layers/base.yaml")
	assert.Contains(t, layer, "git:")
	assert.Contains(t, layer, "editor:")
}

func TestE2E_Init_NonInteractive_ShellBalanced(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Run init with shell:balanced preset
	output := h.Init("shell:balanced")

	// Verify output
	assert.Contains(t, output, "Configuration created")

	// Verify layer contains shell config
	layer := h.ReadConfigFile("layers/base.yaml")
	assert.Contains(t, layer, "shell:")
	assert.Contains(t, layer, "zsh")
}

func TestE2E_Init_NonInteractive_RequiresPreset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Run init without preset - should fail
	exitCode := h.Run("init", "--non-interactive", "--output", h.ConfigDir)
	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, h.LastError, "--preset is required")
}

func TestE2E_Init_AlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	// Create existing config
	h.CreateConfigFile("preflight.yaml", "targets:\n  default:\n    - base\n")

	// Run init - should report already exists
	output := h.RunSuccess("init", "--non-interactive", "--preset", "balanced", "--output", h.ConfigDir)
	assert.Contains(t, output, "already exists")
}

func TestE2E_Init_AllPresets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	presets := []string{
		"balanced",
		"nvim:minimal",
		"nvim:balanced",
		"nvim:maximal",
		"shell:minimal",
		"shell:balanced",
		"git:minimal",
		"brew:minimal",
	}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()

			h := NewHarness(t)

			// Run init
			output := h.Init(preset)
			assert.Contains(t, output, "Configuration created")

			// Verify config was created
			h.AssertConfigFileExists("preflight.yaml")
			h.AssertConfigFileExists("layers/base.yaml")

			// Verify we can parse the config
			manifest := h.ReadConfigFile("preflight.yaml")
			require.Contains(t, manifest, "targets:")
		})
	}
}

func TestE2E_Version(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	output := h.RunSuccess("version")
	// Version output should contain version info
	assert.True(t, strings.Contains(output, "preflight") || strings.Contains(output, "version") || strings.Contains(output, "dev"))
}

func TestE2E_Help(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	t.Parallel()

	h := NewHarness(t)

	output := h.RunSuccess("--help")
	assert.Contains(t, output, "init")
	assert.Contains(t, output, "plan")
	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "doctor")
}
