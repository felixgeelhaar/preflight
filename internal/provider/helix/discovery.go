package helix

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Helix editor.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new Helix discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a Helix discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// HelixSearchOpts returns the search options for Helix config.
func HelixSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// HELIX_CONFIG_DIR can override the config directory
		EnvVar:         "HELIX_CONFIG_DIR",
		ConfigFileName: "config.toml",
		XDGSubpath:     "helix",
		// macOS also uses XDG by default in Helix
		MacOSPaths: []string{
			"~/.config/helix",
		},
		LinuxPaths: []string{
			"~/.config/helix",
		},
		WindowsPaths: []string{
			"$APPDATA/helix",
		},
	}
}

// FindConfigDir discovers the Helix configuration directory.
// Checks: 1) HELIX_CONFIG_DIR env var, 2) XDG_CONFIG_HOME/helix, 3) platform-specific paths.
func (d *Discovery) FindConfigDir() string {
	// Check explicit env var first
	if configDir := os.Getenv("HELIX_CONFIG_DIR"); configDir != "" {
		if pathutil.DirExists(configDir) {
			return configDir
		}
	}

	// Check XDG_CONFIG_HOME
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}
	helixDir := filepath.Join(xdgConfig, "helix")
	if pathutil.DirExists(helixDir) {
		return helixDir
	}

	// Platform-specific fallbacks
	home, _ := os.UserHomeDir()
	switch d.goos {
	case "darwin", "linux":
		return filepath.Join(home, ".config", "helix")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "helix")
	default:
		return filepath.Join(home, ".config", "helix")
	}
}

// BestPracticePath returns the canonical path for Helix config directory.
func (d *Discovery) BestPracticePath() string {
	home, _ := os.UserHomeDir()
	switch d.goos {
	case "darwin", "linux":
		return filepath.Join(home, ".config", "helix")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "helix")
	default:
		return filepath.Join(home, ".config", "helix")
	}
}

// FindConfigPath returns the path to config.toml.
func (d *Discovery) FindConfigPath() string {
	return filepath.Join(d.FindConfigDir(), "config.toml")
}

// FindLanguagesPath returns the path to languages.toml.
func (d *Discovery) FindLanguagesPath() string {
	return filepath.Join(d.FindConfigDir(), "languages.toml")
}

// FindThemesDir returns the path to the themes directory.
func (d *Discovery) FindThemesDir() string {
	return filepath.Join(d.FindConfigDir(), "themes")
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(HelixSearchOpts())
}
