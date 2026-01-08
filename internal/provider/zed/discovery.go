package zed

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Zed editor.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new Zed discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a Zed discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// ZedSearchOpts returns the search options for Zed config.
func ZedSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// Zed uses XDG_CONFIG_HOME/zed on Linux
		XDGSubpath: "zed",
		MacOSPaths: []string{
			// macOS uses Application Support, not XDG
			"~/Library/Application Support/Zed",
		},
		LinuxPaths: []string{
			"~/.config/zed",
		},
	}
}

// FindConfigDir discovers the Zed configuration directory.
// Checks: 1) XDG_CONFIG_HOME/zed (Linux), 2) platform-specific paths.
func (d *Discovery) FindConfigDir() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		// macOS uses Application Support
		appSupport := filepath.Join(home, "Library", "Application Support", "Zed")
		if pathutil.DirExists(appSupport) {
			return appSupport
		}
	case "linux":
		// Linux uses XDG_CONFIG_HOME/zed
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		zedPath := filepath.Join(xdgConfig, "zed")
		if pathutil.DirExists(zedPath) {
			return zedPath
		}
	}

	// Return best-practice path for new configs
	return d.BestPracticePath()
}

// BestPracticePath returns the canonical path for Zed config.
func (d *Discovery) BestPracticePath() string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Zed")
	case "linux":
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgConfig, "zed")
	default:
		return filepath.Join(home, ".config", "zed")
	}
}

// FindSettingsPath returns the path to settings.json.
func (d *Discovery) FindSettingsPath() string {
	configDir := d.FindConfigDir()
	return filepath.Join(configDir, "settings.json")
}

// FindKeymapPath returns the path to keymap.json.
func (d *Discovery) FindKeymapPath() string {
	configDir := d.FindConfigDir()
	return filepath.Join(configDir, "keymap.json")
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(ZedSearchOpts())
}
