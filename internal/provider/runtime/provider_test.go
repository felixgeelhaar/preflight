package runtime

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestProvider_Name(t *testing.T) {
	p := NewProvider(mocks.NewFileSystem())
	if p.Name() != "runtime" {
		t.Errorf("Name() = %q, want %q", p.Name(), "runtime")
	}
}

func TestProvider_Compile_WithTools_ReturnsToolVersionStep(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"runtime": map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{
					"name":    "node",
					"version": "20.10.0",
				},
			},
		},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Should have 1 step (tool-versions)
	if len(steps) != 1 {
		t.Fatalf("Compile() returned %d steps, want 1", len(steps))
	}

	if steps[0].ID().String() != "runtime:tool-versions" {
		t.Errorf("steps[0].ID() = %q, want %q", steps[0].ID().String(), "runtime:tool-versions")
	}
}

func TestProvider_Compile_WithPlugins_ReturnsPluginSteps(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"runtime": map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{
					"name": "golang",
					"url":  "https://github.com/asdf-community/asdf-golang.git",
				},
				map[string]interface{}{
					"name": "rust",
				},
			},
			"tools": []interface{}{
				map[string]interface{}{
					"name":    "golang",
					"version": "1.21.5",
				},
			},
		},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Should have 3 steps (2 plugins + 1 tool-versions)
	if len(steps) != 3 {
		t.Fatalf("Compile() returned %d steps, want 3", len(steps))
	}
}

func TestProvider_Compile_NoConfig_ReturnsNoSteps(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if len(steps) != 0 {
		t.Errorf("Compile() returned %d steps, want 0", len(steps))
	}
}

func TestProvider_Compile_EmptyTools_ReturnsNoSteps(t *testing.T) {
	fs := mocks.NewFileSystem()
	p := NewProvider(fs)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"runtime": map[string]interface{}{
			"tools": []interface{}{},
		},
	})

	steps, err := p.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if len(steps) != 0 {
		t.Errorf("Compile() returned %d steps, want 0", len(steps))
	}
}
