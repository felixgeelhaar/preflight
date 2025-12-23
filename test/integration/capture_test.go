package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapture_DetectsInstalledPackages(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Create some config files to capture
	h.CreateFile(".zshrc", "# zsh config\n")
	h.CreateFile(".gitconfig", "[user]\n    name = Test\n    email = test@example.com\n")
	h.CreateFile(".ssh/config", "Host github.com\n    HostName github.com\n")

	// Run capture for shell, git, and ssh providers
	findings, err := h.Capture("shell", "git", "ssh")
	require.NoError(t, err)

	// Verify items were captured
	assert.NotEmpty(t, findings.Items)

	// Check shell items
	byProvider := findings.ItemsByProvider()
	shellItems := byProvider["shell"]
	assert.NotEmpty(t, shellItems, "should have captured shell config")

	// Check ssh items
	sshItems := byProvider["ssh"]
	assert.NotEmpty(t, sshItems, "should have captured ssh config")
}

func TestCapture_WithSingleProvider(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Create shell config
	h.CreateFile(".zshrc", "# zsh config\n")

	// Run capture for shell only
	findings, err := h.Capture("shell")
	require.NoError(t, err)

	// Verify only shell provider was captured
	assert.Len(t, findings.Providers, 1)
	assert.Equal(t, "shell", findings.Providers[0])
}

func TestCapture_HandlesEmptyHome(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Run capture with no config files
	findings, err := h.Capture("shell", "git", "ssh")
	require.NoError(t, err)

	// Verify no items were captured
	assert.Empty(t, findings.Items)
}

func TestCapture_NvimConfig(t *testing.T) {
	t.Parallel()

	h := NewHarness(t)

	// Create nvim config directory with lazy-lock.json
	h.CreateFile(".config/nvim/init.lua", "-- nvim config\n")
	h.CreateFile(".config/nvim/lazy-lock.json", "{}")

	// Run capture for nvim
	findings, err := h.Capture("nvim")
	require.NoError(t, err)

	// Verify nvim items were captured
	byProvider := findings.ItemsByProvider()
	nvimItems := byProvider["nvim"]
	assert.NotEmpty(t, nvimItems, "should have captured nvim config")

	// Check for lazy-lock.json
	hasLazyLock := false
	for _, item := range nvimItems {
		if item.Name == "lazy-lock.json" {
			hasLazyLock = true
			break
		}
	}
	assert.True(t, hasLazyLock, "should have captured lazy-lock.json")
}
