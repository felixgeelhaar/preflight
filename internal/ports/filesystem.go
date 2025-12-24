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
