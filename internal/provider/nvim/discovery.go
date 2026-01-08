package nvim

import (
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Neovim.
type Discovery struct {
	finder *pathutil.ConfigFinder
}

// NewDiscovery creates a new Neovim discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
	}
}

// SearchOpts returns the search options for Neovim config.
// Supports NVIM_APPNAME for custom config directory names.
func SearchOpts() pathutil.ConfigSearchOpts {
	appName := os.Getenv("NVIM_APPNAME")
	if appName == "" {
		appName = "nvim"
	}

	return pathutil.ConfigSearchOpts{
		// XDG_CONFIG_HOME/NVIM_APPNAME (or nvim if not set)
		XDGSubpath: appName,
		MacOSPaths: []string{
			"~/.config/" + appName,
		},
		LinuxPaths: []string{
			"~/.config/" + appName,
		},
		WindowsPaths: []string{
			"$LOCALAPPDATA/" + appName,
		},
		LegacyPaths: []string{
			"~/.config/nvim", // Fallback to standard nvim if NVIM_APPNAME not found
		},
	}
}

// DataSearchOpts returns the search options for Neovim data directory.
func DataSearchOpts() pathutil.ConfigSearchOpts {
	appName := os.Getenv("NVIM_APPNAME")
	if appName == "" {
		appName = "nvim"
	}

	return pathutil.ConfigSearchOpts{
		// XDG_DATA_HOME/NVIM_APPNAME (or nvim if not set)
		// Note: This uses XDG_DATA_HOME, not XDG_CONFIG_HOME
		LegacyPaths: []string{
			"~/.local/share/" + appName,
		},
		MacOSPaths: []string{
			"~/.local/share/" + appName,
		},
		LinuxPaths: []string{
			"~/.local/share/" + appName,
		},
		WindowsPaths: []string{
			"$LOCALAPPDATA/" + appName + "-data",
		},
	}
}

// FindConfig discovers the Neovim configuration directory.
// Checks: 1) XDG_CONFIG_HOME/NVIM_APPNAME, 2) platform-specific paths, 3) legacy paths.
func (d *Discovery) FindConfig() string {
	opts := SearchOpts()

	// First check if there's a custom XDG_CONFIG_HOME set
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}

	// Check NVIM_APPNAME directory first
	appName := os.Getenv("NVIM_APPNAME")
	if appName != "" {
		appPath := filepath.Join(xdgConfig, appName)
		if pathutil.DirExists(appPath) {
			return appPath
		}
	}

	// Check standard nvim directory
	nvimPath := filepath.Join(xdgConfig, "nvim")
	if pathutil.DirExists(nvimPath) {
		return nvimPath
	}

	// Use ConfigFinder for remaining candidates
	return d.finder.FindConfig(opts)
}

// BestPracticePath returns the canonical path for Neovim config.
// Uses XDG_CONFIG_HOME/nvim or NVIM_APPNAME if set.
func (d *Discovery) BestPracticePath() string {
	appName := os.Getenv("NVIM_APPNAME")
	if appName == "" {
		appName = "nvim"
	}

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}

	return filepath.Join(xdgConfig, appName)
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(SearchOpts())
}

// LazyLockPath returns the path to lazy-lock.json.
func (d *Discovery) LazyLockPath() string {
	configDir := d.FindConfig()
	if configDir == "" {
		configDir = d.BestPracticePath()
	}
	return filepath.Join(configDir, "lazy-lock.json")
}
