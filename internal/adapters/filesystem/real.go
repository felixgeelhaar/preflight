// Package filesystem provides file system adapters.
package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
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
	return os.ReadFile(path)
}

// WriteFile writes data to a file.
func (fs *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
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
	return os.Symlink(target, link)
}

// Remove removes a file or empty directory.
func (fs *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// MkdirAll creates a directory and all necessary parents.
func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Rename renames (moves) a file.
func (fs *RealFileSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// FileHash returns a SHA256 hash of a file's contents.
func (fs *RealFileSystem) FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
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
		return err
	}

	// Get source file permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dest, data, info.Mode())
}

// GetFileInfo returns metadata about a file.
func (fs *RealFileSystem) GetFileInfo(path string) (ports.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ports.FileInfo{}, err
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
