package app

import (
	"os"
	"path/filepath"
)

// DotfilesResolver resolves config_source paths with per-target override support.
// It checks for target-specific dotfiles first (dotfiles.{target}/), then falls back
// to shared dotfiles (dotfiles/).
type DotfilesResolver struct {
	configRoot string
	target     string
}

// NewDotfilesResolver creates a new dotfiles resolver.
func NewDotfilesResolver(configRoot, target string) *DotfilesResolver {
	return &DotfilesResolver{
		configRoot: configRoot,
		target:     target,
	}
}

// Resolve resolves a relative config_source path to an absolute path.
// It checks for per-target override first, then falls back to shared dotfiles.
//
// Resolution order:
// 1. dotfiles.{target}/{relativePath}
// 2. dotfiles/{relativePath}
//
// Returns empty string if neither path exists.
func (r *DotfilesResolver) Resolve(relativePath string) string {
	if relativePath == "" {
		return ""
	}

	// 1. Check per-target directory first
	if r.target != "" {
		targetPath := filepath.Join(r.configRoot, "dotfiles."+r.target, relativePath)
		if r.exists(targetPath) {
			return targetPath
		}
	}

	// 2. Fall back to shared dotfiles directory
	sharedPath := filepath.Join(r.configRoot, "dotfiles", relativePath)
	if r.exists(sharedPath) {
		return sharedPath
	}

	return ""
}

// ResolveWithFallback resolves a config_source path, using the relativePath itself
// as a fallback if it already looks like an absolute path or a dotfiles/ path.
func (r *DotfilesResolver) ResolveWithFallback(relativePath string) string {
	if relativePath == "" {
		return ""
	}

	// If it's already an absolute path, return as-is
	if filepath.IsAbs(relativePath) {
		return relativePath
	}

	// If it starts with "dotfiles/", it's already a relative reference
	if len(relativePath) >= 9 && relativePath[:9] == "dotfiles/" {
		// Extract the part after "dotfiles/" and resolve normally
		subPath := relativePath[9:]
		resolved := r.Resolve(subPath)
		if resolved != "" {
			return resolved
		}
		// Fall back to original path under configRoot
		return filepath.Join(r.configRoot, relativePath)
	}

	// Try normal resolution
	resolved := r.Resolve(relativePath)
	if resolved != "" {
		return resolved
	}

	// Last resort: return path under configRoot
	return filepath.Join(r.configRoot, relativePath)
}

// ResolveDirectory resolves a config_source directory path.
// Returns the resolved path and whether it exists as a directory.
func (r *DotfilesResolver) ResolveDirectory(relativePath string) (string, bool) {
	resolved := r.Resolve(relativePath)
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
func (r *DotfilesResolver) ResolveFile(relativePath string) (string, bool) {
	resolved := r.Resolve(relativePath)
	if resolved == "" {
		return "", false
	}
	if r.isDirectory(resolved) {
		return "", false
	}
	return resolved, true
}

// GetTargetDir returns the target-specific dotfiles directory path.
// Returns the shared dotfiles directory if no target is set.
func (r *DotfilesResolver) GetTargetDir() string {
	if r.target != "" {
		return filepath.Join(r.configRoot, "dotfiles."+r.target)
	}
	return filepath.Join(r.configRoot, "dotfiles")
}

// GetSharedDir returns the shared dotfiles directory path.
func (r *DotfilesResolver) GetSharedDir() string {
	return filepath.Join(r.configRoot, "dotfiles")
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
