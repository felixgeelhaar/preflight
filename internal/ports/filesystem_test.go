package ports

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/.zshrc", filepath.Join(home, ".zshrc")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := ExpandPath(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExpandPath_NotHomePrefix(t *testing.T) {
	// Test that ~ in the middle of a path is not expanded
	result := ExpandPath("/path/with~tilde")
	if result != "/path/with~tilde" {
		t.Errorf("ExpandPath should not expand ~ in middle of path, got %q", result)
	}
}

func TestApplyTargetSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		configRoot string
		target     string
		expected   string
	}{
		{
			name:       "directory with target",
			path:       ".config/nvim",
			configRoot: "/home/user/dotfiles",
			target:     "work",
			expected:   "/home/user/dotfiles/.config.work/nvim",
		},
		{
			name:       "file with target",
			path:       ".gitconfig",
			configRoot: "/home/user/dotfiles",
			target:     "work",
			expected:   "/home/user/dotfiles/.gitconfig.work",
		},
		{
			name:       "empty target returns path under root",
			path:       ".config/nvim",
			configRoot: "/home/user/dotfiles",
			target:     "",
			expected:   "/home/user/dotfiles/.config/nvim",
		},
		{
			name:       "empty path returns root",
			path:       "",
			configRoot: "/home/user/dotfiles",
			target:     "work",
			expected:   "/home/user/dotfiles",
		},
		{
			name:       "deep path with target",
			path:       ".config/nvim/lua/plugins",
			configRoot: "/home/user/dotfiles",
			target:     "personal",
			expected:   "/home/user/dotfiles/.config.personal/nvim/lua/plugins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ApplyTargetSuffix(tt.path, tt.configRoot, tt.target)
			if result != tt.expected {
				t.Errorf("ApplyTargetSuffix(%q, %q, %q) = %q, want %q",
					tt.path, tt.configRoot, tt.target, result, tt.expected)
			}
		})
	}
}

func TestIsPathWithinRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		root     string
		path     string
		expected bool
	}{
		{"valid subpath", "/home/user/config", "/home/user/config/.config/nvim", true},
		{"valid file", "/home/user/config", "/home/user/config/.gitconfig", true},
		{"exact root", "/home/user/config", "/home/user/config", true},
		{"escapes root", "/home/user/config", "/home/user/other", false},
		{"parent traversal", "/home/user/config", "/home/user/config/../other", false},
		{"deep traversal", "/home/user/config", "/home/user/config/../../etc/passwd", false},
		{"absolute escape", "/home/user/config", "/etc/passwd", false},
		{"empty path", "/home/user/config", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsPathWithinRoot(tt.root, tt.path)
			if result != tt.expected {
				t.Errorf("IsPathWithinRoot(%q, %q) = %v, want %v",
					tt.root, tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsPathWithinRootSecure(t *testing.T) {
	t.Parallel()

	// Create a temp directory structure for testing
	tmpDir := t.TempDir()
	safeRoot := filepath.Join(tmpDir, "safe")
	unsafeDir := filepath.Join(tmpDir, "unsafe")

	if err := os.MkdirAll(safeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unsafeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a safe subdir
	safeSubdir := filepath.Join(safeRoot, "subdir")
	if err := os.MkdirAll(safeSubdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file in safe root
	safeFile := filepath.Join(safeRoot, "file.txt")
	if err := os.WriteFile(safeFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a file outside safe root
	unsafeFile := filepath.Join(unsafeDir, "secret.txt")
	if err := os.WriteFile(unsafeFile, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside safe root pointing outside
	evilSymlink := filepath.Join(safeRoot, "evil")
	if err := os.Symlink(unsafeDir, evilSymlink); err != nil {
		t.Fatal(err)
	}

	// Create a safe symlink (points to path within root)
	safeSymlink := filepath.Join(safeRoot, "safe-link")
	if err := os.Symlink(safeSubdir, safeSymlink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		root     string
		path     string
		expected bool
	}{
		{
			name:     "valid file in root",
			root:     safeRoot,
			path:     safeFile,
			expected: true,
		},
		{
			name:     "valid subdir in root",
			root:     safeRoot,
			path:     safeSubdir,
			expected: true,
		},
		{
			name:     "safe symlink stays within root",
			root:     safeRoot,
			path:     safeSymlink,
			expected: true,
		},
		{
			name:     "evil symlink escapes root",
			root:     safeRoot,
			path:     evilSymlink,
			expected: false,
		},
		{
			name:     "path through evil symlink",
			root:     safeRoot,
			path:     filepath.Join(evilSymlink, "secret.txt"),
			expected: false,
		},
		{
			name:     "direct path outside root",
			root:     safeRoot,
			path:     unsafeFile,
			expected: false,
		},
		{
			name:     "non-existent path within root",
			root:     safeRoot,
			path:     filepath.Join(safeRoot, "newfile.txt"),
			expected: true,
		},
		{
			name:     "non-existent root",
			root:     filepath.Join(tmpDir, "nonexistent"),
			path:     filepath.Join(tmpDir, "nonexistent", "file.txt"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IsPathWithinRootSecure(tt.root, tt.path)
			if result != tt.expected {
				t.Errorf("IsPathWithinRootSecure(%q, %q) = %v, want %v",
					tt.root, tt.path, result, tt.expected)
			}
		})
	}
}
