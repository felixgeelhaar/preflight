package ports

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo contains file metadata.
type FileInfo struct {
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// FileSystem provides file system operations.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Exists(path string) bool
	IsSymlink(path string) (isLink bool, target string)
	CreateSymlink(target, link string) error
	Remove(path string) error
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldPath, newPath string) error
	FileHash(path string) (string, error)
	IsDir(path string) bool
	CopyFile(src, dest string) error
	GetFileInfo(path string) (FileInfo, error)

	// Junction support for Windows
	// IsJunction checks if a path is a junction point (Windows) or symlink to directory (Unix).
	IsJunction(path string) (isJunction bool, target string)
	// CreateJunction creates a junction point on Windows, or a directory symlink on Unix.
	CreateJunction(target, link string) error
	// CreateLink creates the appropriate link type based on the target:
	// - On Windows: junction for directories (no admin required), symlink for files
	// - On Unix: symlink for both files and directories
	CreateLink(target, link string) error
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ApplyTargetSuffix adds a target suffix to the first path component.
// This enables per-target configuration overrides in a home-mirrored structure.
//
// Examples:
//   - ".config/nvim" with target "work" -> configRoot/.config.work/nvim
//   - ".gitconfig" with target "work" -> configRoot/.gitconfig.work
func ApplyTargetSuffix(path, configRoot, target string) string {
	if path == "" || target == "" {
		return filepath.Join(configRoot, path)
	}

	parts := strings.SplitN(path, string(filepath.Separator), 2)
	if len(parts) == 0 {
		return filepath.Join(configRoot, path)
	}

	// Add suffix to first component
	parts[0] = parts[0] + "." + target

	if len(parts) == 1 {
		return filepath.Join(configRoot, parts[0])
	}
	return filepath.Join(configRoot, parts[0], parts[1])
}

// IsPathWithinRoot validates that a path stays within the given root directory.
// Returns false if the path escapes the root via ".." or other traversal.
// This is a fast path check without symlink resolution - use IsPathWithinRootSecure
// for security-critical operations.
func IsPathWithinRoot(root, path string) bool {
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)

	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..") && rel != ".."
}

// IsPathWithinRootSecure validates that a path stays within the given root directory,
// resolving symlinks to prevent symlink attacks. This is the secure version that
// should be used for any security-sensitive file operations.
//
// Returns false if:
// - The path escapes the root via ".." or symlinks pointing outside
// - The path contains symlinks that can't be resolved
// - Either path doesn't exist (for full resolution)
//
// For paths that may not exist yet, use IsPathWithinRoot with additional validation.
func IsPathWithinRootSecure(root, path string) bool {
	// First, try to resolve symlinks in the root
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		// Root must exist and be resolvable
		return false
	}
	cleanRoot := filepath.Clean(resolvedRoot)

	// Try to resolve the full path
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Path doesn't exist yet, check parent directories
		// Walk up until we find an existing path
		checkPath := filepath.Clean(path)
		for {
			parent := filepath.Dir(checkPath)
			if parent == checkPath {
				// Reached filesystem root without finding existing path
				break
			}

			resolvedParent, err := filepath.EvalSymlinks(parent)
			if err == nil {
				// Parent exists, verify it's within root
				cleanParent := filepath.Clean(resolvedParent)
				rel, err := filepath.Rel(cleanRoot, cleanParent)
				if err != nil {
					return false
				}
				if strings.HasPrefix(rel, "..") || rel == ".." {
					return false
				}
				// Parent is within root, now check remaining path components
				// for any suspicious patterns
				remaining := strings.TrimPrefix(path, parent)
				return !strings.Contains(remaining, "..")
			}
			checkPath = parent
		}
		// No existing parent found, fall back to basic check
		return IsPathWithinRoot(root, path)
	}

	cleanPath := filepath.Clean(resolvedPath)

	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..") && rel != ".."
}
