package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DotfilesResolver resolves config_source paths with per-target override support.
// It uses a home-mirrored structure where the repository mirrors $HOME:
//   - Shared files: .config/nvim, .gitconfig
//   - Target-specific: .config.{target}/nvim, .gitconfig.{target}
type DotfilesResolver struct {
	configRoot string
	target     string
}

// isPathWithinRoot validates that a path stays within the config root.
// Returns false if the path escapes the root via ".." or other traversal.
func isPathWithinRoot(root, path string) bool {
	return ports.IsPathWithinRoot(root, path)
}

// NewDotfilesResolver creates a new dotfiles resolver.
func NewDotfilesResolver(configRoot, target string) *DotfilesResolver {
	return &DotfilesResolver{
		configRoot: configRoot,
		target:     target,
	}
}

// Resolve resolves a home-relative config_source path to an absolute path.
// It checks for per-target override first, then falls back to shared path.
//
// Resolution order for path ".config/nvim":
// 1. {configRoot}/.config.{target}/nvim
// 2. {configRoot}/.config/nvim
//
// Resolution order for path ".gitconfig":
// 1. {configRoot}/.gitconfig.{target}
// 2. {configRoot}/.gitconfig
//
// Returns empty string if neither path exists or if path traversal is detected.
func (r *DotfilesResolver) Resolve(homeRelPath string) string {
	if homeRelPath == "" {
		return ""
	}

	// Security: reject paths that could escape configRoot
	if strings.Contains(homeRelPath, "..") {
		return ""
	}

	// 1. Check per-target path first (suffixed first component)
	if r.target != "" && r.target != "default" {
		targetPath := r.applyTargetSuffix(homeRelPath)
		if isPathWithinRoot(r.configRoot, targetPath) && r.exists(targetPath) {
			return targetPath
		}
	}

	// 2. Fall back to shared path
	sharedPath := filepath.Join(r.configRoot, homeRelPath)
	if isPathWithinRoot(r.configRoot, sharedPath) && r.exists(sharedPath) {
		return sharedPath
	}

	return ""
}

// applyTargetSuffix adds the target suffix to the first path component.
// Examples:
//   - ".gitconfig" -> ".gitconfig.work"
//   - ".config/nvim" -> ".config.work/nvim"
func (r *DotfilesResolver) applyTargetSuffix(homeRelPath string) string {
	return ports.ApplyTargetSuffix(homeRelPath, r.configRoot, r.target)
}

// ResolveWithFallback resolves a config_source path, using the path itself
// as a fallback if it already looks like an absolute path.
// Also handles legacy "dotfiles/" paths for backward compatibility.
// Returns empty string if path traversal is detected.
func (r *DotfilesResolver) ResolveWithFallback(configSourcePath string) string {
	if configSourcePath == "" {
		return ""
	}

	// Security: reject paths that could escape configRoot
	if strings.Contains(configSourcePath, "..") {
		return ""
	}

	// If it's already an absolute path, return as-is
	if filepath.IsAbs(configSourcePath) {
		return configSourcePath
	}

	// Handle legacy "dotfiles/" prefix for backward compatibility
	if strings.HasPrefix(configSourcePath, "dotfiles/") {
		// Extract the provider name after "dotfiles/"
		subPath := configSourcePath[9:]
		// Convert legacy path to home-relative by looking up known mappings
		homeRelPath := r.legacyToHomeRelPath(subPath)
		resolved := r.Resolve(homeRelPath)
		if resolved != "" {
			return resolved
		}
		// Also try the legacy path structure for migration period
		legacyPath := filepath.Join(r.configRoot, configSourcePath)
		if isPathWithinRoot(r.configRoot, legacyPath) && r.exists(legacyPath) {
			return legacyPath
		}
	}

	// Try normal resolution (assumes home-relative path)
	resolved := r.Resolve(configSourcePath)
	if resolved != "" {
		return resolved
	}

	// Last resort: return path under configRoot (with validation)
	fallbackPath := filepath.Join(r.configRoot, configSourcePath)
	if !isPathWithinRoot(r.configRoot, fallbackPath) {
		return ""
	}
	return fallbackPath
}

// legacyToHomeRelPath converts legacy provider-based paths to home-relative paths.
// For example: "nvim" -> ".config/nvim", "ssh" -> ".ssh"
func (r *DotfilesResolver) legacyToHomeRelPath(providerPath string) string {
	// Map of legacy provider paths to home-relative paths
	mappings := map[string]string{
		"nvim":     ".config/nvim",
		"shell":    ".zshrc", // Shell files are typically at root
		"starship": ".config/starship.toml",
		"tmux":     ".tmux.conf",
		"vscode":   ".config/Code/User", // Linux path
		"ssh":      ".ssh",
		"git":      ".gitconfig",
	}

	// Check if it's a known provider
	parts := strings.SplitN(providerPath, string(filepath.Separator), 2)
	if len(parts) > 0 {
		// Terminal provider needs special handling - try multiple locations
		if parts[0] == "terminal" {
			return r.resolveTerminalPath(parts)
		}
		if mappedPath, ok := mappings[parts[0]]; ok {
			if len(parts) == 1 {
				return mappedPath
			}
			// Append remaining path
			return filepath.Join(mappedPath, parts[1])
		}
	}

	// Unknown mapping, return as-is
	return providerPath
}

// resolveTerminalPath handles legacy terminal provider paths by checking multiple locations.
// Terminal emulators store configs at different locations:
//   - WezTerm: ~/.wezterm.lua or ~/.config/wezterm/
//   - Alacritty: ~/.alacritty.toml, ~/.alacritty.yml, or ~/.config/alacritty/
//   - Kitty: ~/.config/kitty/
//   - Ghostty: ~/.config/ghostty/
func (r *DotfilesResolver) resolveTerminalPath(parts []string) string {
	// Terminal paths to check in order of preference
	terminalPaths := []string{
		".config/wezterm",
		".wezterm.lua",
		".config/alacritty",
		".alacritty.toml",
		".alacritty.yml",
		".config/kitty",
		".config/ghostty",
		".hyper.js",
		".config/hyper",
	}

	// If there's a subpath (e.g., "terminal/.wezterm.lua"), use it directly
	if len(parts) > 1 {
		subPath := parts[1]
		// Check if subpath is a known terminal file/dir
		for _, termPath := range terminalPaths {
			if strings.HasPrefix(subPath, filepath.Base(termPath)) {
				// Return the subpath as-is since it's already home-relative
				return subPath
			}
		}
		// For unknown subpaths, try to find a matching terminal path
		for _, termPath := range terminalPaths {
			testPath := filepath.Join(r.configRoot, termPath)
			if r.exists(testPath) {
				if len(parts) > 1 {
					return filepath.Join(termPath, parts[1])
				}
				return termPath
			}
		}
		// Fall back to the subpath as-is
		return subPath
	}

	// No subpath - find the first existing terminal config
	for _, termPath := range terminalPaths {
		testPath := filepath.Join(r.configRoot, termPath)
		if r.exists(testPath) {
			return termPath
		}
	}

	// Default to wezterm if nothing exists
	return ".config/wezterm"
}

// ResolveDirectory resolves a config_source directory path.
// Returns the resolved path and whether it exists as a directory.
func (r *DotfilesResolver) ResolveDirectory(homeRelPath string) (string, bool) {
	resolved := r.Resolve(homeRelPath)
	if resolved == "" {
		return "", false
	}
	if !r.isDirectory(resolved) {
		return "", false
	}
	return resolved, true
}

// ResolveFile resolves a config_source file path.
// Returns the resolved path and whether it exists as a file.
func (r *DotfilesResolver) ResolveFile(homeRelPath string) (string, bool) {
	resolved := r.Resolve(homeRelPath)
	if resolved == "" {
		return "", false
	}
	if r.isDirectory(resolved) {
		return "", false
	}
	return resolved, true
}

// GetTargetDir returns the config root directory.
// With home-mirrored structure, this is just the config root.
func (r *DotfilesResolver) GetTargetDir() string {
	return r.configRoot
}

// GetSharedDir returns the config root directory.
// With home-mirrored structure, this is the same as GetTargetDir.
func (r *DotfilesResolver) GetSharedDir() string {
	return r.configRoot
}

// Target returns the current target name.
func (r *DotfilesResolver) Target() string {
	return r.target
}

// ConfigRoot returns the configuration root directory.
func (r *DotfilesResolver) ConfigRoot() string {
	return r.configRoot
}

func (r *DotfilesResolver) exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *DotfilesResolver) isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
