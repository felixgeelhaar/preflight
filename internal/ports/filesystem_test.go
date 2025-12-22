package ports

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMockFileSystem_ReadFile(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "export PATH=$PATH:/usr/local/bin")

	content, err := fs.ReadFile("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "export PATH=$PATH:/usr/local/bin" {
		t.Errorf("ReadFile() = %q, want %q", string(content), "export PATH=$PATH:/usr/local/bin")
	}
}

func TestMockFileSystem_ReadFile_NotFound(t *testing.T) {
	fs := NewMockFileSystem()

	_, err := fs.ReadFile("/nonexistent")
	if err == nil {
		t.Error("ReadFile() should return error for non-existent file")
	}
}

func TestMockFileSystem_WriteFile(t *testing.T) {
	fs := NewMockFileSystem()

	err := fs.WriteFile("/home/user/.config/test", []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	content, _ := fs.ReadFile("/home/user/.config/test")
	if string(content) != "content" {
		t.Errorf("WriteFile() content = %q, want %q", string(content), "content")
	}
}

func TestMockFileSystem_Exists(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	if !fs.Exists("/home/user/.zshrc") {
		t.Error("Exists() should return true for existing file")
	}
	if fs.Exists("/nonexistent") {
		t.Error("Exists() should return false for non-existent file")
	}
}

func TestMockFileSystem_IsSymlink(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddSymlink("/home/user/.zshrc", "/dotfiles/.zshrc")

	isLink, target := fs.IsSymlink("/home/user/.zshrc")
	if !isLink {
		t.Error("IsSymlink() should return true for symlink")
	}
	if target != "/dotfiles/.zshrc" {
		t.Errorf("IsSymlink() target = %q, want %q", target, "/dotfiles/.zshrc")
	}
}

func TestMockFileSystem_IsSymlink_NotSymlink(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	isLink, _ := fs.IsSymlink("/home/user/.zshrc")
	if isLink {
		t.Error("IsSymlink() should return false for regular file")
	}
}

func TestMockFileSystem_CreateSymlink(t *testing.T) {
	fs := NewMockFileSystem()

	err := fs.CreateSymlink("/dotfiles/.zshrc", "/home/user/.zshrc")
	if err != nil {
		t.Fatalf("CreateSymlink() error = %v", err)
	}

	isLink, target := fs.IsSymlink("/home/user/.zshrc")
	if !isLink {
		t.Error("CreateSymlink() should create a symlink")
	}
	if target != "/dotfiles/.zshrc" {
		t.Errorf("CreateSymlink() target = %q, want %q", target, "/dotfiles/.zshrc")
	}
}

func TestMockFileSystem_Remove(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	err := fs.Remove("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if fs.Exists("/home/user/.zshrc") {
		t.Error("Remove() should delete the file")
	}
}

func TestMockFileSystem_MkdirAll(t *testing.T) {
	fs := NewMockFileSystem()

	err := fs.MkdirAll("/home/user/.config/app", 0755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if !fs.IsDir("/home/user/.config/app") {
		t.Error("MkdirAll() should create directory")
	}
}

func TestMockFileSystem_Rename(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	err := fs.Rename("/home/user/.zshrc", "/home/user/.zshrc.bak")
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if fs.Exists("/home/user/.zshrc") {
		t.Error("Rename() should remove original file")
	}
	if !fs.Exists("/home/user/.zshrc.bak") {
		t.Error("Rename() should create new file")
	}
}

func TestMockFileSystem_FileHash(t *testing.T) {
	fs := NewMockFileSystem()
	fs.AddFile("/home/user/.zshrc", "export PATH=$PATH:/usr/local/bin")

	hash, err := fs.FileHash("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("FileHash() error = %v", err)
	}
	if hash == "" {
		t.Error("FileHash() should return non-empty hash")
	}
}

func TestNewRealFileSystem(t *testing.T) {
	fs := NewRealFileSystem()
	if fs == nil {
		t.Error("NewRealFileSystem() should not return nil")
	}
}

func TestRealFileSystem_Integration(t *testing.T) {
	fs := NewRealFileSystem()

	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "preflight-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test WriteFile and ReadFile
	testFile := filepath.Join(tmpDir, "test.txt")
	err = fs.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("ReadFile() = %q, want %q", string(content), "hello world")
	}

	// Test Exists
	if !fs.Exists(testFile) {
		t.Error("Exists() should return true")
	}

	// Test CreateSymlink and IsSymlink
	linkPath := filepath.Join(tmpDir, "link.txt")
	err = fs.CreateSymlink(testFile, linkPath)
	if err != nil {
		t.Fatalf("CreateSymlink() error = %v", err)
	}

	isLink, target := fs.IsSymlink(linkPath)
	if !isLink {
		t.Error("IsSymlink() should return true for symlink")
	}
	if target != testFile {
		t.Errorf("IsSymlink() target = %q, want %q", target, testFile)
	}

	// Test FileHash
	hash, err := fs.FileHash(testFile)
	if err != nil {
		t.Fatalf("FileHash() error = %v", err)
	}
	if hash == "" {
		t.Error("FileHash() should return non-empty hash")
	}
}

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
