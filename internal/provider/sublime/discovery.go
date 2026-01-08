package sublime

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Sublime Text.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new Sublime Text discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a Sublime Text discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// SublimeSearchOpts returns the search options for Sublime Text config.
func SublimeSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// SUBLIME_DATA can override the config directory
		EnvVar:         "SUBLIME_DATA",
		ConfigFileName: "Packages/User/Preferences.sublime-settings",
		MacOSPaths: []string{
			"~/Library/Application Support/Sublime Text/Packages/User",
			"~/Library/Application Support/Sublime Text 3/Packages/User",
		},
		LinuxPaths: []string{
			"~/.config/sublime-text/Packages/User",
			"~/.config/sublime-text-3/Packages/User",
		},
		WindowsPaths: []string{
			"$APPDATA/Sublime Text/Packages/User",
			"$APPDATA/Sublime Text 3/Packages/User",
		},
	}
}

// FindConfigDir discovers the Sublime Text User configuration directory.
// Checks: 1) SUBLIME_DATA env var, 2) platform-specific paths.
func (d *Discovery) FindConfigDir() string {
	// Check explicit env var first
	if dataDir := os.Getenv("SUBLIME_DATA"); dataDir != "" {
		userDir := filepath.Join(dataDir, "Packages", "User")
		if pathutil.DirExists(userDir) {
			return userDir
		}
	}

	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		// Check Sublime Text 4 first, then 3
		paths := []string{
			filepath.Join(home, "Library", "Application Support", "Sublime Text", "Packages", "User"),
			filepath.Join(home, "Library", "Application Support", "Sublime Text 3", "Packages", "User"),
		}
		for _, p := range paths {
			if pathutil.DirExists(p) {
				return p
			}
		}
		return paths[0] // Default to Sublime Text 4 path

	case "linux":
		paths := []string{
			filepath.Join(home, ".config", "sublime-text", "Packages", "User"),
			filepath.Join(home, ".config", "sublime-text-3", "Packages", "User"),
		}
		for _, p := range paths {
			if pathutil.DirExists(p) {
				return p
			}
		}
		return paths[0]

	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		paths := []string{
			filepath.Join(appData, "Sublime Text", "Packages", "User"),
			filepath.Join(appData, "Sublime Text 3", "Packages", "User"),
		}
		for _, p := range paths {
			if pathutil.DirExists(p) {
				return p
			}
		}
		return paths[0]

	default:
		return filepath.Join(home, ".config", "sublime-text", "Packages", "User")
	}
}

// BestPracticePath returns the canonical path for Sublime Text config.
func (d *Discovery) BestPracticePath() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Sublime Text", "Packages", "User")
	case "linux":
		return filepath.Join(home, ".config", "sublime-text", "Packages", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Sublime Text", "Packages", "User")
	default:
		return filepath.Join(home, ".config", "sublime-text", "Packages", "User")
	}
}

// FindPreferencesPath returns the path to Preferences.sublime-settings.
func (d *Discovery) FindPreferencesPath() string {
	return filepath.Join(d.FindConfigDir(), "Preferences.sublime-settings")
}

// FindKeybindingsPath returns the path to Default (OS).sublime-keymap.
func (d *Discovery) FindKeybindingsPath() string {
	var filename string
	switch d.goos {
	case "darwin":
		filename = "Default (OSX).sublime-keymap"
	case "linux":
		filename = "Default (Linux).sublime-keymap"
	case "windows":
		filename = "Default (Windows).sublime-keymap"
	default:
		filename = "Default (Linux).sublime-keymap"
	}
	return filepath.Join(d.FindConfigDir(), filename)
}

// FindPackageControlPath returns the path to Package Control.sublime-settings.
func (d *Discovery) FindPackageControlPath() string {
	return filepath.Join(d.FindConfigDir(), "Package Control.sublime-settings")
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(SublimeSearchOpts())
}
