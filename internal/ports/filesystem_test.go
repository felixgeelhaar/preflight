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
