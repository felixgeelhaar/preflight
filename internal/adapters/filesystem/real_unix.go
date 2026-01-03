//go:build !windows

package filesystem

import (
	"fmt"
	"os"
)

// IsJunction checks if a path is a symlink to a directory.
// On Unix, there are no junctions - this checks for directory symlinks.
func (fs *RealFileSystem) IsJunction(path string) (bool, string) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, ""
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		return false, ""
	}

	// Read the target
	target, err := os.Readlink(path)
	if err != nil {
		return false, ""
	}

	// Check if target is a directory
	targetInfo, err := os.Stat(path)
	if err != nil {
		return true, target // Symlink exists but target may be broken
	}

	if targetInfo.IsDir() {
		return true, target
	}

	return false, ""
}

// CreateJunction creates a directory symlink on Unix.
// On Unix, this is the same as CreateSymlink.
func (fs *RealFileSystem) CreateJunction(target, link string) error {
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to create junction %q -> %q: %w", link, target, err)
	}
	return nil
}

// CreateLink creates a symlink to the target.
// On Unix, symlinks work for both files and directories.
func (fs *RealFileSystem) CreateLink(target, link string) error {
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to create link %q -> %q: %w", link, target, err)
	}
	return nil
}
