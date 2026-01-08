// Package pathutil provides utilities for dynamic config path discovery.
package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigFinder provides methods for discovering configuration file locations.
type ConfigFinder struct {
	homeDir string
	goos    string
}

// NewConfigFinder creates a new ConfigFinder.
func NewConfigFinder() *ConfigFinder {
	home, _ := os.UserHomeDir()
	return &ConfigFinder{
		homeDir: home,
		goos:    runtime.GOOS,
	}
}

// NewConfigFinderWithHome creates a ConfigFinder with a custom home directory (for testing).
func NewConfigFinderWithHome(home, goos string) *ConfigFinder {
	return &ConfigFinder{
		homeDir: home,
		goos:    goos,
	}
}

// FindConfig searches for a config file using dynamic discovery.
// It checks: 1) explicit env var, 2) XDG paths, 3) legacy/default paths.
// Returns the first existing path, or empty string if none found.
func (f *ConfigFinder) FindConfig(opts ConfigSearchOpts) string {
	candidates := f.GetCandidatePaths(opts)
	for _, path := range candidates {
		if path != "" && fileExists(path) {
			return path
		}
	}
	return ""
}

// GetCandidatePaths returns all candidate paths in priority order (for capture).
// Does not check if files exist - returns all possible locations.
func (f *ConfigFinder) GetCandidatePaths(opts ConfigSearchOpts) []string {
	// Pre-allocate with estimated capacity to avoid reallocations
	paths := make([]string, 0, 8)

	// 1. Explicit environment variable override
	if opts.EnvVar != "" {
		if envPath := os.Getenv(opts.EnvVar); envPath != "" {
			if opts.ConfigFileName != "" {
				paths = append(paths, filepath.Join(envPath, opts.ConfigFileName))
			} else {
				paths = append(paths, envPath)
			}
		}
	}

	// 2. XDG_CONFIG_HOME based path
	if opts.XDGSubpath != "" {
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(f.homeDir, ".config")
		}
		paths = append(paths, filepath.Join(xdgConfig, opts.XDGSubpath))
	}

	// 3. Platform-specific paths
	switch f.goos {
	case "darwin":
		for _, p := range opts.MacOSPaths {
			paths = append(paths, f.expandPath(p))
		}
	case "linux":
		for _, p := range opts.LinuxPaths {
			paths = append(paths, f.expandPath(p))
		}
	case "windows":
		for _, p := range opts.WindowsPaths {
			paths = append(paths, f.expandPath(p))
		}
	}

	// 4. Legacy/fallback paths (cross-platform)
	for _, p := range opts.LegacyPaths {
		paths = append(paths, f.expandPath(p))
	}

	return paths
}

// BestPracticePath returns the canonical best-practice path for a config.
// This is where apply should symlink to (normalized path).
func (f *ConfigFinder) BestPracticePath(opts ConfigSearchOpts) string {
	// Prefer XDG path if specified
	if opts.XDGSubpath != "" {
		return filepath.Join(f.homeDir, ".config", opts.XDGSubpath)
	}

	// Fall back to platform-specific best practice
	switch f.goos {
	case "darwin":
		if len(opts.MacOSPaths) > 0 {
			return f.expandPath(opts.MacOSPaths[0])
		}
	case "linux":
		if len(opts.LinuxPaths) > 0 {
			return f.expandPath(opts.LinuxPaths[0])
		}
	case "windows":
		if len(opts.WindowsPaths) > 0 {
			return f.expandPath(opts.WindowsPaths[0])
		}
	}

	// Ultimate fallback
	if len(opts.LegacyPaths) > 0 {
		return f.expandPath(opts.LegacyPaths[0])
	}

	return ""
}

// ConfigSearchOpts defines options for config file discovery.
type ConfigSearchOpts struct {
	// EnvVar is the environment variable to check first (e.g., "ALACRITTY_CONFIG_DIR")
	EnvVar string

	// ConfigFileName is appended to EnvVar path if set (e.g., "alacritty.toml")
	ConfigFileName string

	// XDGSubpath is the path relative to XDG_CONFIG_HOME (e.g., "alacritty/alacritty.toml")
	XDGSubpath string

	// MacOSPaths are macOS-specific paths to check (supports ~ expansion)
	MacOSPaths []string

	// LinuxPaths are Linux-specific paths to check
	LinuxPaths []string

	// WindowsPaths are Windows-specific paths to check (supports %APPDATA% etc.)
	WindowsPaths []string

	// LegacyPaths are legacy/fallback paths to check on any platform
	LegacyPaths []string
}

// expandPath expands ~ and environment variables in a path.
func (f *ConfigFinder) expandPath(path string) string {
	if len(path) == 0 {
		return path
	}

	// Expand ~
	if path[0] == '~' {
		path = filepath.Join(f.homeDir, path[1:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ExpandPath expands ~ and environment variables in a path.
func ExpandPath(path string) string {
	finder := NewConfigFinder()
	return finder.expandPath(path)
}
