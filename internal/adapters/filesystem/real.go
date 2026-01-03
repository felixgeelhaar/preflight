// Package filesystem provides file system adapters.
package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// RealFileSystem implements ports.FileSystem using actual file system operations.
type RealFileSystem struct{}

// NewRealFileSystem creates a new RealFileSystem.
func NewRealFileSystem() *RealFileSystem {
	return &RealFileSystem{}
}

// ReadFile reads a file and returns its contents.
func (fs *RealFileSystem) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return data, nil
}

// WriteFile writes data to a file.
func (fs *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}
	return nil
}

// Exists checks if a file or directory exists.
func (fs *RealFileSystem) Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// IsSymlink checks if a path is a symbolic link and returns its target.
func (fs *RealFileSystem) IsSymlink(path string) (bool, string) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, ""
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, ""
	}
	target, err := os.Readlink(path)
	if err != nil {
		return true, ""
	}
	return true, target
}

// CreateSymlink creates a symbolic link.
func (fs *RealFileSystem) CreateSymlink(target, link string) error {
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to create symlink %q -> %q: %w", link, target, err)
	}
	return nil
}

// Remove removes a file or empty directory.
func (fs *RealFileSystem) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove %q: %w", path, err)
	}
	return nil
}

// MkdirAll creates a directory and all necessary parents.
func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}
	return nil
}

// Rename renames (moves) a file.
func (fs *RealFileSystem) Rename(oldPath, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename %q to %q: %w", oldPath, newPath, err)
	}
	return nil
}

// FileHash returns a SHA256 hash of a file's contents.
func (fs *RealFileSystem) FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to hash file %q: %w", path, err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// IsDir checks if a path is a directory.
func (fs *RealFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CopyFile copies a file from src to dest.
func (fs *RealFileSystem) CopyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %q: %w", src, err)
	}

	// Get source file permissions
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file %q: %w", src, err)
	}

	if err := os.WriteFile(dest, data, info.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file %q: %w", dest, err)
	}
	return nil
}

// GetFileInfo returns metadata about a file.
func (fs *RealFileSystem) GetFileInfo(path string) (ports.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ports.FileInfo{}, fmt.Errorf("failed to get file info for %q: %w", path, err)
	}

	return ports.FileInfo{
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

// Ensure RealFileSystem implements ports.FileSystem.
var _ ports.FileSystem = (*RealFileSystem)(nil)

// Note: IsJunction, CreateJunction, and CreateLink are implemented in
// real_unix.go and real_windows.go with build constraints.
