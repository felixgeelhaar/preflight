package ports

import (
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
