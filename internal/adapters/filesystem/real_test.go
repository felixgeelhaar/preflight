package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

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
	err = fs.WriteFile(testFile, []byte("hello world"), 0o644)
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

	// Test IsDir
	if !fs.IsDir(tmpDir) {
		t.Error("IsDir() should return true for directory")
	}
	if fs.IsDir(testFile) {
		t.Error("IsDir() should return false for file")
	}

	// Test MkdirAll
	nestedDir := filepath.Join(tmpDir, "nested", "dir")
	err = fs.MkdirAll(nestedDir, 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if !fs.Exists(nestedDir) {
		t.Error("MkdirAll() should create nested directories")
	}

	// Test Rename
	newPath := filepath.Join(tmpDir, "renamed.txt")
	err = fs.Rename(testFile, newPath)
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if fs.Exists(testFile) {
		t.Error("Rename() should remove original file")
	}
	if !fs.Exists(newPath) {
		t.Error("Rename() should create new file")
	}

	// Test Remove
	err = fs.Remove(newPath)
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if fs.Exists(newPath) {
		t.Error("Remove() should delete the file")
	}
}

func TestRealFileSystem_NotSymlink(t *testing.T) {
	fs := NewRealFileSystem()

	tmpDir, err := os.MkdirTemp("", "preflight-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "regular.txt")
	err = fs.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	isLink, _ := fs.IsSymlink(testFile)
	if isLink {
		t.Error("IsSymlink() should return false for regular file")
	}
}

func TestRealFileSystem_ReadFile_NotFound(t *testing.T) {
	fs := NewRealFileSystem()

	_, err := fs.ReadFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("ReadFile() should return error for non-existent file")
	}
}

func TestRealFileSystem_FileHash_NotFound(t *testing.T) {
	fs := NewRealFileSystem()

	_, err := fs.FileHash("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("FileHash() should return error for non-existent file")
	}
}
