package vscode

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for VSCode.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new VSCode discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a VSCode discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// VSCodeSearchOpts returns the search options for VSCode config.
func VSCodeSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// VSCODE_PORTABLE overrides default locations for portable installations
		EnvVar:         "VSCODE_PORTABLE",
		ConfigFileName: "data/user-data/User",
		// XDG not used by VSCode (it has its own platform-specific paths)
		MacOSPaths: []string{
			"~/Library/Application Support/Code/User",
		},
		LinuxPaths: []string{
			"~/.config/Code/User",
		},
		WindowsPaths: []string{
			"$APPDATA/Code/User",
		},
	}
}

// FindSettingsPath discovers the VSCode settings.json location.
// Checks: 1) VSCODE_PORTABLE env var, 2) platform-specific paths.
func (d *Discovery) FindSettingsPath() string {
	// Check portable installation first
	if portable := os.Getenv("VSCODE_PORTABLE"); portable != "" {
		settingsPath := filepath.Join(portable, "data", "user-data", "User", "settings.json")
		if fileExists(settingsPath) {
			return settingsPath
		}
	}

	// Use platform-specific paths
	userDir := d.getUserDir()
	settingsPath := filepath.Join(userDir, "settings.json")
	if fileExists(settingsPath) {
		return settingsPath
	}

	// Return best-practice path for new configs
	return filepath.Join(d.BestPracticeUserDir(), "settings.json")
}

// FindKeybindingsPath discovers the VSCode keybindings.json location.
func (d *Discovery) FindKeybindingsPath() string {
	// Check portable installation first
	if portable := os.Getenv("VSCODE_PORTABLE"); portable != "" {
		kbPath := filepath.Join(portable, "data", "user-data", "User", "keybindings.json")
		if fileExists(kbPath) {
			return kbPath
		}
	}

	// Use platform-specific paths
	userDir := d.getUserDir()
	kbPath := filepath.Join(userDir, "keybindings.json")
	if fileExists(kbPath) {
		return kbPath
	}

	// Return best-practice path for new configs
	return filepath.Join(d.BestPracticeUserDir(), "keybindings.json")
}

// BestPracticeUserDir returns the canonical User directory for VSCode.
func (d *Discovery) BestPracticeUserDir() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		return filepath.Join(home, ".config", "Code", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Code", "User")
	default:
		return filepath.Join(home, ".config", "Code", "User")
	}
}

// getUserDir returns the User config directory for the current platform.
func (d *Discovery) getUserDir() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		return filepath.Join(home, ".config", "Code", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Code", "User")
	default:
		return filepath.Join(home, ".config", "Code", "User")
	}
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(VSCodeSearchOpts())
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
