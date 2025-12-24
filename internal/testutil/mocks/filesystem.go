package mocks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// FileSystem is a thread-safe test double for ports.FileSystem.
type FileSystem struct {
	mu       sync.RWMutex
	files    map[string][]byte
	symlinks map[string]string
	dirs     map[string]bool
}

// NewFileSystem creates a new FileSystem mock.
func NewFileSystem() *FileSystem {
	return &FileSystem{
		files:    make(map[string][]byte),
		symlinks: make(map[string]string),
		dirs:     make(map[string]bool),
	}
}

// AddFile adds a file to the mock filesystem.
func (fs *FileSystem) AddFile(path string, content string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[path] = []byte(content)
}

// SetFileContent sets file content directly as bytes.
func (fs *FileSystem) SetFileContent(path string, content []byte) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[path] = content
}

// AddSymlink adds a symlink to the mock filesystem.
func (fs *FileSystem) AddSymlink(link, target string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.symlinks[link] = target
}

// AddDir adds a directory to the mock filesystem.
func (fs *FileSystem) AddDir(path string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.dirs[path] = true
}

// ReadFile reads a file from the mock filesystem.
func (fs *FileSystem) ReadFile(path string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if content, ok := fs.files[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// WriteFile writes a file to the mock filesystem.
func (fs *FileSystem) WriteFile(path string, data []byte, _ os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[path] = data
	return nil
}

// Exists checks if a file exists in the mock filesystem.
func (fs *FileSystem) Exists(path string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	_, fileExists := fs.files[path]
	_, linkExists := fs.symlinks[path]
	_, dirExists := fs.dirs[path]
	return fileExists || linkExists || dirExists
}

// IsSymlink checks if a path is a symlink in the mock filesystem.
func (fs *FileSystem) IsSymlink(path string) (bool, string) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if target, ok := fs.symlinks[path]; ok {
		return true, target
	}
	return false, ""
}

// CreateSymlink creates a symlink in the mock filesystem.
func (fs *FileSystem) CreateSymlink(target, link string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.symlinks[link] = target
	return nil
}

// Remove removes a file from the mock filesystem.
func (fs *FileSystem) Remove(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.files, path)
	delete(fs.symlinks, path)
	delete(fs.dirs, path)
	return nil
}

// MkdirAll creates a directory in the mock filesystem.
func (fs *FileSystem) MkdirAll(path string, _ os.FileMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.dirs[path] = true
	return nil
}

// Rename renames a file in the mock filesystem.
func (fs *FileSystem) Rename(oldPath, newPath string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
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
func (fs *FileSystem) FileHash(path string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	content, ok := fs.files[path]
	if !ok {
		return "", fmt.Errorf("file not found: %s", path)
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// IsDir checks if a path is a directory in the mock filesystem.
func (fs *FileSystem) IsDir(path string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.dirs[path]
}

// CopyFile copies a file in the mock filesystem.
func (fs *FileSystem) CopyFile(src, dest string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	content, ok := fs.files[src]
	if !ok {
		return fmt.Errorf("file not found: %s", src)
	}
	fs.files[dest] = append([]byte(nil), content...)
	return nil
}

// GetFileInfo returns metadata about a file in the mock filesystem.
func (fs *FileSystem) GetFileInfo(path string) (ports.FileInfo, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if content, ok := fs.files[path]; ok {
		return ports.FileInfo{
			Size:    int64(len(content)),
			Mode:    0o644,
			ModTime: time.Now(),
			IsDir:   false,
		}, nil
	}

	if fs.dirs[path] {
		return ports.FileInfo{
			Size:    0,
			Mode:    0o755,
			ModTime: time.Now(),
			IsDir:   true,
		}, nil
	}

	return ports.FileInfo{}, fmt.Errorf("file not found: %s", path)
}

// Reset clears all files, symlinks, and directories.
func (fs *FileSystem) Reset() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files = make(map[string][]byte)
	fs.symlinks = make(map[string]string)
	fs.dirs = make(map[string]bool)
}

// Ensure FileSystem implements ports.FileSystem.
var _ ports.FileSystem = (*FileSystem)(nil)
