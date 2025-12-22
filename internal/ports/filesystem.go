package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
}

// RealFileSystem implements FileSystem using actual file system operations.
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

// MockFileSystem implements FileSystem for testing.
type MockFileSystem struct {
	files    map[string][]byte
	symlinks map[string]string
	dirs     map[string]bool
}

// NewMockFileSystem creates a new MockFileSystem.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files:    make(map[string][]byte),
		symlinks: make(map[string]string),
		dirs:     make(map[string]bool),
	}
}

// AddFile adds a file to the mock filesystem.
func (fs *MockFileSystem) AddFile(path string, content string) {
	fs.files[path] = []byte(content)
}

// AddSymlink adds a symlink to the mock filesystem.
func (fs *MockFileSystem) AddSymlink(link, target string) {
	fs.symlinks[link] = target
}

// ReadFile reads a file from the mock filesystem.
func (fs *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if content, ok := fs.files[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// WriteFile writes a file to the mock filesystem.
func (fs *MockFileSystem) WriteFile(path string, data []byte, _ os.FileMode) error {
	fs.files[path] = data
	return nil
}

// Exists checks if a file exists in the mock filesystem.
func (fs *MockFileSystem) Exists(path string) bool {
	_, fileExists := fs.files[path]
	_, linkExists := fs.symlinks[path]
	_, dirExists := fs.dirs[path]
	return fileExists || linkExists || dirExists
}

// IsSymlink checks if a path is a symlink in the mock filesystem.
func (fs *MockFileSystem) IsSymlink(path string) (bool, string) {
	if target, ok := fs.symlinks[path]; ok {
		return true, target
	}
	return false, ""
}

// CreateSymlink creates a symlink in the mock filesystem.
func (fs *MockFileSystem) CreateSymlink(target, link string) error {
	fs.symlinks[link] = target
	return nil
}

// Remove removes a file from the mock filesystem.
func (fs *MockFileSystem) Remove(path string) error {
	delete(fs.files, path)
	delete(fs.symlinks, path)
	delete(fs.dirs, path)
	return nil
}

// MkdirAll creates a directory in the mock filesystem.
func (fs *MockFileSystem) MkdirAll(path string, _ os.FileMode) error {
	fs.dirs[path] = true
	return nil
}

// Rename renames a file in the mock filesystem.
func (fs *MockFileSystem) Rename(oldPath, newPath string) error {
	if content, ok := fs.files[oldPath]; ok {
		fs.files[newPath] = content
		delete(fs.files, oldPath)
		return nil
	}
	if target, ok := fs.symlinks[oldPath]; ok {
		fs.symlinks[newPath] = target
		delete(fs.symlinks, oldPath)
		return nil
	}
	return fmt.Errorf("file not found: %s", oldPath)
}

// FileHash returns a hash of a file in the mock filesystem.
func (fs *MockFileSystem) FileHash(path string) (string, error) {
	content, err := fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// IsDir checks if a path is a directory in the mock filesystem.
func (fs *MockFileSystem) IsDir(path string) bool {
	return fs.dirs[path]
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
