package cursor

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Cursor editor.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new Cursor discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a Cursor discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// SearchOpts returns the search options for Cursor config.
func SearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// CURSOR_PORTABLE overrides default locations for portable installations
		EnvVar:         "CURSOR_PORTABLE",
		ConfigFileName: "data/user-data/User",
		MacOSPaths: []string{
			"~/Library/Application Support/Cursor/User",
		},
		LinuxPaths: []string{
			"~/.config/Cursor/User",
		},
		WindowsPaths: []string{
			"$APPDATA/Cursor/User",
		},
	}
}

// FindConfigDir discovers the Cursor User configuration directory.
// Checks: 1) CURSOR_PORTABLE env var, 2) platform-specific paths.
func (d *Discovery) FindConfigDir() string {
	// Check portable installation first
	if portable := os.Getenv("CURSOR_PORTABLE"); portable != "" {
		userDir := filepath.Join(portable, "data", "user-data", "User")
		if pathutil.DirExists(userDir) {
			return userDir
		}
	}

	// Use platform-specific paths
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		appSupport := filepath.Join(home, "Library", "Application Support", "Cursor", "User")
		if pathutil.DirExists(appSupport) {
			return appSupport
		}
	case "linux":
		configDir := filepath.Join(home, ".config", "Cursor", "User")
		if pathutil.DirExists(configDir) {
			return configDir
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		cursorDir := filepath.Join(appData, "Cursor", "User")
		if pathutil.DirExists(cursorDir) {
			return cursorDir
		}
	}

	// Return best-practice path for new configs
	return d.BestPracticePath()
}

// BestPracticePath returns the canonical path for Cursor config.
func (d *Discovery) BestPracticePath() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor", "User")
	case "linux":
		return filepath.Join(home, ".config", "Cursor", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User")
	default:
		return filepath.Join(home, ".config", "Cursor", "User")
	}
}

// FindSettingsPath returns the path to settings.json.
func (d *Discovery) FindSettingsPath() string {
	configDir := d.FindConfigDir()
	return filepath.Join(configDir, "settings.json")
}

// FindKeybindingsPath returns the path to keybindings.json.
func (d *Discovery) FindKeybindingsPath() string {
	configDir := d.FindConfigDir()
	return filepath.Join(configDir, "keybindings.json")
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(SearchOpts())
}
