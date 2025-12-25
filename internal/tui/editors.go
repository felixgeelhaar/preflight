package tui

import (
	"os/exec"
	"sort"
)

// Editor represents an editor with its detection and display info.
type Editor struct {
	ID          string
	Name        string
	Description string
	Command     string   // Primary command to check
	AltCommands []string // Alternative commands to check
	Provider    string   // Provider ID if supported (nvim, vscode, etc.)
	HasProvider bool     // Whether preflight has a provider for this editor
	IsInstalled bool     // Detected at runtime
	InstallHint string   // How to install if not present
}

// AvailableEditors returns all supported editors with installation status.
// Editors are returned in alphabetical order to avoid bias.
func AvailableEditors() []Editor {
	editors := []Editor{
		{
			ID:          "cursor",
			Name:        "Cursor",
			Description: "AI-first code editor",
			Command:     "cursor",
			Provider:    "vscode", // Uses VS Code extension format
			HasProvider: true,
			InstallHint: "https://cursor.com/",
		},
		{
			ID:          "emacs",
			Name:        "Emacs",
			Description: "Extensible text editor",
			Command:     "emacs",
			AltCommands: []string{"emacsclient"},
			HasProvider: false,
			InstallHint: "brew install emacs",
		},
		{
			ID:          "helix",
			Name:        "Helix",
			Description: "Post-modern modal editor",
			Command:     "hx",
			AltCommands: []string{"helix"},
			HasProvider: false,
			InstallHint: "brew install helix",
		},
		{
			ID:          "intellij",
			Name:        "IntelliJ IDEA",
			Description: "JetBrains Java/Kotlin IDE",
			Command:     "idea",
			HasProvider: false,
			InstallHint: "brew install --cask intellij-idea",
		},
		{
			ID:          "nano",
			Name:        "Nano",
			Description: "Simple terminal editor",
			Command:     "nano",
			HasProvider: false,
			InstallHint: "Usually pre-installed",
		},
		{
			ID:          "neovim",
			Name:        "Neovim",
			Description: "Hyperextensible Vim-based editor",
			Command:     "nvim",
			Provider:    "nvim",
			HasProvider: true,
			InstallHint: "brew install neovim",
		},
		{
			ID:          "sublime",
			Name:        "Sublime Text",
			Description: "Sophisticated text editor",
			Command:     "subl",
			HasProvider: false,
			InstallHint: "brew install --cask sublime-text",
		},
		{
			ID:          "vim",
			Name:        "Vim",
			Description: "Ubiquitous modal editor",
			Command:     "vim",
			HasProvider: false,
			InstallHint: "brew install vim",
		},
		{
			ID:          "vscode",
			Name:        "VS Code",
			Description: "Popular code editor",
			Command:     "code",
			Provider:    "vscode",
			HasProvider: true,
			InstallHint: "brew install --cask visual-studio-code",
		},
		{
			ID:          "vscodium",
			Name:        "VSCodium",
			Description: "Open-source VS Code build",
			Command:     "codium",
			Provider:    "vscode", // Compatible with VS Code provider
			HasProvider: true,
			InstallHint: "brew install --cask vscodium",
		},
		{
			ID:          "webstorm",
			Name:        "WebStorm",
			Description: "JetBrains JavaScript IDE",
			Command:     "webstorm",
			HasProvider: false,
			InstallHint: "brew install --cask webstorm",
		},
		{
			ID:          "zed",
			Name:        "Zed",
			Description: "High-performance editor",
			Command:     "zed",
			HasProvider: false,
			InstallHint: "brew install --cask zed",
		},
	}

	// Detect installation status for each editor
	for i := range editors {
		editors[i].IsInstalled = detectEditor(&editors[i])
	}

	// Sort alphabetically by name for unbiased presentation
	sort.Slice(editors, func(i, j int) bool {
		return editors[i].Name < editors[j].Name
	})

	return editors
}

// detectEditor checks if an editor is installed on the system.
func detectEditor(e *Editor) bool {
	// Check primary command
	if _, err := exec.LookPath(e.Command); err == nil {
		return true
	}

	// Check alternative commands
	for _, cmd := range e.AltCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}

	return false
}

// InstalledEditors returns only editors that are currently installed.
func InstalledEditors() []Editor {
	all := AvailableEditors()
	installed := make([]Editor, 0, len(all))

	for _, e := range all {
		if e.IsInstalled {
			installed = append(installed, e)
		}
	}

	return installed
}

// EditorsWithProviders returns editors that have preflight provider support.
func EditorsWithProviders() []Editor {
	all := AvailableEditors()
	withProviders := make([]Editor, 0, len(all))

	for _, e := range all {
		if e.HasProvider {
			withProviders = append(withProviders, e)
		}
	}

	return withProviders
}

// GetEditorByID returns an editor by its ID.
func GetEditorByID(id string) (Editor, bool) {
	for _, e := range AvailableEditors() {
		if e.ID == id {
			return e, true
		}
	}
	return Editor{}, false
}
