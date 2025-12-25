package plugin

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// maxManifestSize limits manifest file size to prevent memory exhaustion (256KB).
	maxManifestSize int64 = 256 * 1024
)

// Loader discovers and loads plugins from the filesystem.
type Loader struct {
	// SearchPaths are directories to search for plugins
	SearchPaths []string
}

// NewLoader creates a new plugin loader with default search paths.
func NewLoader() *Loader {
	paths := []string{"/usr/local/share/preflight/plugins"}

	home, err := os.UserHomeDir()
	if err == nil {
		// Prepend user path (higher priority than system path)
		paths = append([]string{filepath.Join(home, ".preflight", "plugins")}, paths...)
	}

	return &Loader{SearchPaths: paths}
}

// WithSearchPaths sets custom search paths.
func (l *Loader) WithSearchPaths(paths ...string) *Loader {
	l.SearchPaths = paths
	return l
}

// Discover finds all plugins in search paths.
// Returns a DiscoveryResult containing both successfully loaded plugins and any errors.
// The context can be used for cancellation.
func (l *Loader) Discover(ctx context.Context) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		Plugins: make([]*Plugin, 0),
		Errors:  make([]DiscoveryError, 0),
	}

	for _, searchPath := range l.SearchPaths {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		plugins, errors := l.discoverInPath(ctx, searchPath)
		result.Plugins = append(result.Plugins, plugins...)
		result.Errors = append(result.Errors, errors...)
	}

	return result, nil
}

// discoverInPath finds plugins in a single directory.
// Returns both plugins and any discovery errors encountered.
func (l *Loader) discoverInPath(ctx context.Context, searchPath string) ([]*Plugin, []DiscoveryError) {
	entries, err := os.ReadDir(searchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Path doesn't exist, not an error
		}
		return nil, []DiscoveryError{{Path: searchPath, Err: err}}
	}

	plugins := make([]*Plugin, 0, len(entries))
	errors := make([]DiscoveryError, 0)

	for _, entry := range entries {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return plugins, errors
		default:
		}

		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(searchPath, entry.Name())
		plugin, err := l.LoadFromPath(pluginPath)
		if err != nil {
			// Collect errors but continue discovering
			errors = append(errors, DiscoveryError{Path: pluginPath, Err: err})
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins, errors
}

// Ensure Loader implements Discoverer.
var _ Discoverer = (*Loader)(nil)

// LoadFromPath loads a plugin from a directory.
func (l *Loader) LoadFromPath(path string) (*Plugin, error) {
	manifestPath := filepath.Join(path, "plugin.yaml")

	// Check if manifest exists and get file info
	info, err := os.Stat(manifestPath)
	if os.IsNotExist(err) {
		return nil, ErrManifestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("checking plugin.yaml: %w", err)
	}

	// Check manifest size limit
	if info.Size() > maxManifestSize {
		return nil, &ManifestSizeError{
			Size:  info.Size(),
			Limit: maxManifestSize,
		}
	}

	// Read manifest with size limit (defense in depth)
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("opening plugin.yaml: %w", err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxManifestSize))
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

	// Validate WASM capabilities for provider plugins
	if manifest.IsProviderPlugin() && manifest.WASM != nil {
		if err := ValidateCapabilities(manifest.WASM.Capabilities); err != nil {
			return nil, fmt.Errorf("capability validation: %w", err)
		}
	}

	return &Plugin{
		Manifest: manifest,
		Path:     path,
		Enabled:  true,
		LoadedAt: time.Now(),
	}, nil
}

// LoadFromGit clones and loads a plugin from a Git repository.
// The ref parameter can be a tag, branch, or empty for the default branch.
func (l *Loader) LoadFromGit(repoURL, ref string) (*Plugin, error) {
	return l.LoadFromGitWithContext(context.Background(), repoURL, ref)
}

// LoadFromGitWithContext clones and loads a plugin from a Git repository with context support.
func (l *Loader) LoadFromGitWithContext(ctx context.Context, repoURL, ref string) (*Plugin, error) {
	// Validate and parse the URL
	repoName, err := extractRepoName(repoURL)
	if err != nil {
		return nil, err
	}

	// Validate the repo name doesn't contain path traversal
	if err := validatePluginName(repoName); err != nil {
		return nil, err
	}

	// Determine install path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	installPath := filepath.Join(home, ".preflight", "plugins", repoName)

	// Ensure the resolved path is within the plugins directory (defense in depth)
	// Use filepath.Rel which properly handles path traversal attempts
	pluginsDir := filepath.Join(home, ".preflight", "plugins")
	absPluginsDir, err := filepath.Abs(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("resolving plugins directory: %w", err)
	}
	absInstallPath, err := filepath.Abs(installPath)
	if err != nil {
		return nil, fmt.Errorf("resolving install path: %w", err)
	}
	relPath, err := filepath.Rel(absPluginsDir, absInstallPath)
	if err != nil {
		return nil, &PathTraversalError{Path: repoName}
	}
	// Check if relPath starts with ".." which would indicate escaping the plugins directory
	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return nil, &PathTraversalError{Path: repoName}
	}

	// Check if already installed
	if _, err := os.Stat(installPath); err == nil {
		// Already exists, try to load
		return l.LoadFromPath(installPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(pluginsDir, 0700); err != nil {
		return nil, fmt.Errorf("creating plugins directory: %w", err)
	}

	// Clone the repository
	cloner := NewGitCloner()
	if err := cloner.Clone(ctx, repoURL, ref, installPath); err != nil {
		return nil, err
	}

	// Load the plugin from the cloned path
	return l.LoadFromPath(installPath)
}

// extractRepoName safely extracts the repository name from a Git URL.
func extractRepoName(repoURL string) (string, error) {
	// Parse the URL
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", &InvalidURLError{URL: repoURL, Reason: "malformed URL"}
	}

	// Validate scheme
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "git" {
		return "", &InvalidURLError{URL: repoURL, Reason: "unsupported scheme (use https, http, or git)"}
	}

	// Get the path and extract the last component
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" {
		return "", &InvalidURLError{URL: repoURL, Reason: "empty path"}
	}

	// Get the base name
	name := filepath.Base(path)
	if name == "" || name == "." || name == "/" {
		return "", &InvalidURLError{URL: repoURL, Reason: "invalid repository path"}
	}

	// Remove .git suffix if present
	name = strings.TrimSuffix(name, ".git")
	if name == "" {
		return "", &InvalidURLError{URL: repoURL, Reason: "empty repository name"}
	}

	return name, nil
}

// validatePluginName checks if a plugin name is safe to use as a directory name.
func validatePluginName(name string) error {
	// Check for path traversal attempts
	if strings.Contains(name, "..") ||
		strings.Contains(name, "/") ||
		strings.Contains(name, "\\") ||
		strings.HasPrefix(name, ".") {
		return &PathTraversalError{Path: name}
	}

	// Check for empty name
	if name == "" {
		return ErrEmptyPluginName
	}

	return nil
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
// Uses 0700 permissions to restrict access to the owner only.
func EnsureInstallPath() (string, error) {
	path, err := InstallPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", fmt.Errorf("creating plugin directory: %w", err)
	}
	return path, nil
}
