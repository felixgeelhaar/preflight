package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Loader discovers and loads plugins from the filesystem.
type Loader struct {
	// SearchPaths are directories to search for plugins
	SearchPaths []string
}

// NewLoader creates a new plugin loader with default search paths.
func NewLoader() *Loader {
	home, _ := os.UserHomeDir()
	return &Loader{
		SearchPaths: []string{
			filepath.Join(home, ".preflight", "plugins"),
			"/usr/local/share/preflight/plugins",
		},
	}
}

// WithSearchPaths sets custom search paths.
func (l *Loader) WithSearchPaths(paths ...string) *Loader {
	l.SearchPaths = paths
	return l
}

// Discover finds all plugins in search paths.
func (l *Loader) Discover() ([]*Plugin, error) {
	var plugins []*Plugin

	for _, searchPath := range l.SearchPaths {
		found, err := l.discoverInPath(searchPath)
		if err != nil {
			// Skip paths that don't exist
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("searching %s: %w", searchPath, err)
		}
		plugins = append(plugins, found...)
	}

	return plugins, nil
}

// discoverInPath finds plugins in a single directory.
func (l *Loader) discoverInPath(searchPath string) ([]*Plugin, error) {
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		return nil, err
	}

	plugins := make([]*Plugin, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(searchPath, entry.Name())
		plugin, err := l.LoadFromPath(pluginPath)
		if err != nil {
			// Skip invalid plugins but log the error
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// LoadFromPath loads a plugin from a directory.
func (l *Loader) LoadFromPath(path string) (*Plugin, error) {
	manifestPath := filepath.Join(path, "plugin.yaml")

	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin.yaml not found in %s", path)
	}

	// Read manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading plugin.yaml: %w", err)
	}

	// Parse manifest
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing plugin.yaml: %w", err)
	}

	// Validate manifest
	if err := ValidateManifest(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &Plugin{
		Manifest: manifest,
		Path:     path,
		Enabled:  true,
		LoadedAt: time.Now(),
	}, nil
}

// LoadFromGit clones and loads a plugin from a Git repository.
func (l *Loader) LoadFromGit(repoURL, _ string) (*Plugin, error) {
	// Determine install path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	// Extract repo name from URL
	repoName := filepath.Base(repoURL)
	repoName = repoName[:len(repoName)-len(filepath.Ext(repoName))] // Remove .git

	installPath := filepath.Join(home, ".preflight", "plugins", repoName)

	// Check if already installed
	if _, err := os.Stat(installPath); err == nil {
		// Already exists, try to load
		return l.LoadFromPath(installPath)
	}

	// Clone would happen here - for now, return an error
	// In production, this would use git2go or shell out to git
	return nil, fmt.Errorf("git clone not implemented: install manually to %s", installPath)
}

// InstallPath returns the default plugin installation directory.
func InstallPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".preflight", "plugins"), nil
}

// EnsureInstallPath creates the plugin installation directory if it doesn't exist.
func EnsureInstallPath() (string, error) {
	path, err := InstallPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("creating plugin directory: %w", err)
	}
	return path, nil
}
