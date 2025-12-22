package files

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestFilesProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "files" {
		t.Errorf("Name() = %q, want %q", got, "files")
	}
}

func TestFilesProvider_Compile_Empty(t *testing.T) {
	fs := ports.NewMockFileSystem()
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

func TestFilesProvider_Compile_NoFilesSection(t *testing.T) {
	fs := ports.NewMockFileSystem()
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

func TestFilesProvider_Compile_Links(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"links": []interface{}{
				map[string]interface{}{
					"src":  "dotfiles/.zshrc",
					"dest": "~/.zshrc",
				},
				map[string]interface{}{
					"src":  "dotfiles/.vimrc",
					"dest": "~/.vimrc",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}
}

func TestFilesProvider_Compile_Templates(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"templates": []interface{}{
				map[string]interface{}{
					"src":  "templates/gitconfig.tmpl",
					"dest": "~/.gitconfig",
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

func TestFilesProvider_Compile_Copies(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"copies": []interface{}{
				map[string]interface{}{
					"src":  "files/script.sh",
					"dest": "~/.local/bin/script.sh",
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

func TestFilesProvider_Compile_Full(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"links": []interface{}{
				map[string]interface{}{
					"src":  "dotfiles/.zshrc",
					"dest": "~/.zshrc",
				},
			},
			"templates": []interface{}{
				map[string]interface{}{
					"src":  "templates/gitconfig.tmpl",
					"dest": "~/.gitconfig",
				},
			},
			"copies": []interface{}{
				map[string]interface{}{
					"src":  "files/script.sh",
					"dest": "~/.local/bin/script.sh",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 3 {
		t.Errorf("Compile() len = %d, want 3", len(steps))
	}
}

func TestFilesProvider_Compile_InvalidConfig(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"links": "not-a-list",
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestFilesProvider_Compile_StepsOrder(t *testing.T) {
	fs := ports.NewMockFileSystem()
	provider := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"files": map[string]interface{}{
			"links": []interface{}{
				map[string]interface{}{
					"src":  "dotfiles/.zshrc",
					"dest": "~/.zshrc",
				},
			},
			"templates": []interface{}{
				map[string]interface{}{
					"src":  "templates/config.tmpl",
					"dest": "~/.config/app",
				},
			},
			"copies": []interface{}{
				map[string]interface{}{
					"src":  "files/script.sh",
					"dest": "~/.local/bin/script.sh",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Links should come first, then templates, then copies
	if len(steps) != 3 {
		t.Fatalf("Compile() len = %d, want 3", len(steps))
	}

	// Verify order by checking ID prefixes
	if steps[0].ID().String()[:11] != "files:link:" {
		t.Errorf("First step should be link, got %s", steps[0].ID().String())
	}
	if steps[1].ID().String()[:15] != "files:template:" {
		t.Errorf("Second step should be template, got %s", steps[1].ID().String())
	}
	if steps[2].ID().String()[:11] != "files:copy:" {
		t.Errorf("Third step should be copy, got %s", steps[2].ID().String())
	}
}
