package mocks

import (
	"sync"
	"testing"
)

func TestFileSystem_ReadFile(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/home/user/.zshrc", "export PATH=$PATH:/usr/local/bin")

	content, err := fs.ReadFile("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "export PATH=$PATH:/usr/local/bin" {
		t.Errorf("ReadFile() = %q, want %q", string(content), "export PATH=$PATH:/usr/local/bin")
	}
}

func TestFileSystem_ReadFile_NotFound(t *testing.T) {
	fs := NewFileSystem()

	_, err := fs.ReadFile("/nonexistent")
	if err == nil {
		t.Error("ReadFile() should return error for non-existent file")
	}
}

func TestFileSystem_WriteFile(t *testing.T) {
	fs := NewFileSystem()

	err := fs.WriteFile("/home/user/.config/test", []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	content, _ := fs.ReadFile("/home/user/.config/test")
	if string(content) != "content" {
		t.Errorf("WriteFile() content = %q, want %q", string(content), "content")
	}
}

func TestFileSystem_Exists(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	if !fs.Exists("/home/user/.zshrc") {
		t.Error("Exists() should return true for existing file")
	}
	if fs.Exists("/nonexistent") {
		t.Error("Exists() should return false for non-existent file")
	}
}

func TestFileSystem_IsSymlink(t *testing.T) {
	fs := NewFileSystem()
	fs.AddSymlink("/home/user/.zshrc", "/dotfiles/.zshrc")

	isLink, target := fs.IsSymlink("/home/user/.zshrc")
	if !isLink {
		t.Error("IsSymlink() should return true for symlink")
	}
	if target != "/dotfiles/.zshrc" {
		t.Errorf("IsSymlink() target = %q, want %q", target, "/dotfiles/.zshrc")
	}
}

func TestFileSystem_IsSymlink_NotSymlink(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	isLink, _ := fs.IsSymlink("/home/user/.zshrc")
	if isLink {
		t.Error("IsSymlink() should return false for regular file")
	}
}

func TestFileSystem_CreateSymlink(t *testing.T) {
	fs := NewFileSystem()

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

func TestFileSystem_Remove(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/home/user/.zshrc", "content")

	err := fs.Remove("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if fs.Exists("/home/user/.zshrc") {
		t.Error("Remove() should delete the file")
	}
}

func TestFileSystem_MkdirAll(t *testing.T) {
	fs := NewFileSystem()

	err := fs.MkdirAll("/home/user/.config/app", 0755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if !fs.IsDir("/home/user/.config/app") {
		t.Error("MkdirAll() should create directory")
	}
}

func TestFileSystem_Rename(t *testing.T) {
	fs := NewFileSystem()
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

func TestFileSystem_FileHash(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/home/user/.zshrc", "export PATH=$PATH:/usr/local/bin")

	hash, err := fs.FileHash("/home/user/.zshrc")
	if err != nil {
		t.Fatalf("FileHash() error = %v", err)
	}
	if hash == "" {
		t.Error("FileHash() should return non-empty hash")
	}
}

func TestFileSystem_Reset(t *testing.T) {
	fs := NewFileSystem()
	fs.AddFile("/test.txt", "content")
	fs.AddSymlink("/link", "/target")
	fs.AddDir("/dir")

	fs.Reset()

	if fs.Exists("/test.txt") || fs.Exists("/link") || fs.Exists("/dir") {
		t.Error("Reset() should clear all entries")
	}
}

func TestFileSystem_ThreadSafety(_ *testing.T) {
	fs := NewFileSystem()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path := "/file" + string(rune('a'+idx%26))
			_ = fs.WriteFile(path, []byte("content"), 0644)
			_, _ = fs.ReadFile(path)
			_ = fs.Exists(path)
		}(i)
	}

	wg.Wait()
	// Should not panic or have data races
}
