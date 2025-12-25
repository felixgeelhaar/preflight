package tui

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvailableEditors(t *testing.T) {
	t.Parallel()

	editors := AvailableEditors()

	// Should return non-empty list
	require.NotEmpty(t, editors, "should return at least some editors")

	// Should be sorted alphabetically by name
	names := make([]string, len(editors))
	for i, e := range editors {
		names[i] = e.Name
	}
	assert.True(t, sort.StringsAreSorted(names), "editors should be sorted alphabetically")

	// Should include common editors
	hasNeovim := false
	hasVSCode := false
	hasVim := false
	for _, e := range editors {
		switch e.ID {
		case "neovim":
			hasNeovim = true
			assert.Equal(t, "Neovim", e.Name)
			assert.Equal(t, "nvim", e.Command)
			assert.True(t, e.HasProvider)
		case "vscode":
			hasVSCode = true
			assert.Equal(t, "VS Code", e.Name)
			assert.Equal(t, "code", e.Command)
			assert.True(t, e.HasProvider)
		case "vim":
			hasVim = true
			assert.Equal(t, "Vim", e.Name)
			assert.Equal(t, "vim", e.Command)
			assert.False(t, e.HasProvider)
		}
	}
	assert.True(t, hasNeovim, "should include Neovim")
	assert.True(t, hasVSCode, "should include VS Code")
	assert.True(t, hasVim, "should include Vim")
}

func TestAvailableEditors_AllHaveRequiredFields(t *testing.T) {
	t.Parallel()

	editors := AvailableEditors()

	for _, e := range editors {
		e := e // capture range variable
		t.Run(e.ID, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, e.ID, "ID should not be empty")
			assert.NotEmpty(t, e.Name, "Name should not be empty")
			assert.NotEmpty(t, e.Description, "Description should not be empty")
			assert.NotEmpty(t, e.Command, "Command should not be empty")
			assert.NotEmpty(t, e.InstallHint, "InstallHint should not be empty")
		})
	}
}

func TestGetEditorByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id       string
		wantName string
		wantOK   bool
	}{
		{"neovim", "Neovim", true},
		{"vscode", "VS Code", true},
		{"vim", "Vim", true},
		{"cursor", "Cursor", true},
		{"helix", "Helix", true},
		{"zed", "Zed", true},
		{"nonexistent", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			editor, ok := GetEditorByID(tt.id)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantName, editor.Name)
			}
		})
	}
}

func TestInstalledEditors(t *testing.T) {
	t.Parallel()

	installed := InstalledEditors()

	// All returned editors should have IsInstalled = true
	for _, e := range installed {
		assert.True(t, e.IsInstalled, "editor %s should be marked as installed", e.ID)
	}
}

func TestEditorsWithProviders(t *testing.T) {
	t.Parallel()

	withProviders := EditorsWithProviders()

	// All returned editors should have HasProvider = true
	for _, e := range withProviders {
		assert.True(t, e.HasProvider, "editor %s should have provider support", e.ID)
		assert.NotEmpty(t, e.Provider, "editor %s should have provider ID", e.ID)
	}

	// Should include at least neovim and vscode
	hasNeovim := false
	hasVSCode := false
	for _, e := range withProviders {
		if e.ID == "neovim" {
			hasNeovim = true
		}
		if e.ID == "vscode" {
			hasVSCode = true
		}
	}
	assert.True(t, hasNeovim, "should include Neovim")
	assert.True(t, hasVSCode, "should include VS Code")
}

func TestEditorProviderMapping(t *testing.T) {
	t.Parallel()

	// Editors with provider support should have correct provider IDs
	tests := []struct {
		editorID   string
		providerID string
	}{
		{"neovim", "nvim"},
		{"vscode", "vscode"},
		{"cursor", "vscode"},
		{"vscodium", "vscode"},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.editorID, func(t *testing.T) {
			t.Parallel()
			editor, ok := GetEditorByID(tt.editorID)
			require.True(t, ok, "editor %s should exist", tt.editorID)
			assert.Equal(t, tt.providerID, editor.Provider)
		})
	}
}

func TestEditorAltCommands(t *testing.T) {
	t.Parallel()

	// Test editors with alternative commands
	editor, ok := GetEditorByID("helix")
	require.True(t, ok)
	assert.Contains(t, editor.AltCommands, "helix", "Helix should have 'helix' as alt command")

	editor, ok = GetEditorByID("emacs")
	require.True(t, ok)
	assert.Contains(t, editor.AltCommands, "emacsclient", "Emacs should have 'emacsclient' as alt command")
}

func TestProviderDisplayNameForEditors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id       string
		wantName string
	}{
		{"neovim", "Neovim"},
		{"vscode", "VS Code"},
		{"cursor", "Cursor"},
		{"helix", "Helix"},
		{"zed", "Zed"},
		{"vim", "Vim"},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			name := providerDisplayName(tt.id)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestProviderDescriptionForEditors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want string
	}{
		{"neovim", "Hyperextensible Vim-based editor"},
		{"vscode", "Popular code editor"},
		{"helix", "Post-modern modal editor"},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			desc := providerDescription(tt.id)
			assert.Equal(t, tt.want, desc)
		})
	}
}

func TestProviderInstalledStatus(t *testing.T) {
	t.Parallel()

	// Non-editor should return empty
	status := providerInstalledStatus("shell")
	assert.Empty(t, status, "non-editor should have empty status")

	status = providerInstalledStatus("git")
	assert.Empty(t, status, "non-editor should have empty status")
}

func TestFallbackCatalogServiceGetProviders(t *testing.T) {
	t.Parallel()

	svc := &fallbackCatalogService{}
	providers := svc.GetProviders()

	// Should include all editors
	editors := AvailableEditors()
	for _, e := range editors {
		assert.Contains(t, providers, e.ID, "should include editor %s", e.ID)
	}

	// Should include other core providers
	assert.Contains(t, providers, "brew")
	assert.Contains(t, providers, "git")
	assert.Contains(t, providers, "shell")
	assert.Contains(t, providers, "docker")
	assert.Contains(t, providers, "ssh")
}
