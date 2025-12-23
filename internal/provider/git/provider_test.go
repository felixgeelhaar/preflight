package git

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestGitProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "git" {
		t.Errorf("Name() = %q, want %q", got, "git")
	}
}

func TestGitProvider_Compile_Empty(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestGitProvider_Compile_NoGitSection(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestGitProvider_Compile_WithUserConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"git": map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
}

func TestGitProvider_Compile_WithAliases(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"git": map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"alias": map[string]interface{}{
				"co": "checkout",
				"st": "status",
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
}

func TestGitProvider_Compile_WithIncludes(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"git": map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"includes": []interface{}{
				map[string]interface{}{
					"path":     "~/.gitconfig.work",
					"ifconfig": "gitdir:~/work/",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}
}

func TestGitProvider_Compile_InvalidConfig(t *testing.T) {
	fs := mocks.NewFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"git": map[string]interface{}{
			"includes": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}
