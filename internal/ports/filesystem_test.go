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
