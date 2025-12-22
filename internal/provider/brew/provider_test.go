package brew

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

func TestBrewProvider_Name(t *testing.T) {
	provider := NewProvider(nil)
	if got := provider.Name(); got != "brew" {
		t.Errorf("Name() = %q, want %q", got, "brew")
	}
}

func TestBrewProvider_Compile_Empty(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestBrewProvider_Compile_NoBrewSection(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"apt": map[string]interface{}{},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestBrewProvider_Compile_Taps(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"taps": []interface{}{"homebrew/cask", "homebrew/core"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}

	// Verify step IDs
	ids := make(map[string]bool)
	for _, s := range steps {
		ids[s.ID().String()] = true
	}
	if !ids["brew:tap:homebrew/cask"] {
		t.Error("Missing brew:tap:homebrew/cask step")
	}
	if !ids["brew:tap:homebrew/core"] {
		t.Error("Missing brew:tap:homebrew/core step")
	}
}

func TestBrewProvider_Compile_Formulae(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
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

func TestBrewProvider_Compile_Casks(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"casks": []interface{}{"docker", "slack"},
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

func TestBrewProvider_Compile_Full(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"git"},
			"casks":    []interface{}{"docker"},
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

func TestBrewProvider_Compile_InvalidConfig(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"taps": "not-a-list", // Invalid: should be a list
		},
	})
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Error("Compile() should return error for invalid config")
	}
}

func TestBrewProvider_Compile_FormulaWithTap(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{
				map[string]interface{}{
					"name": "neovim",
					"tap":  "homebrew/core",
				},
			},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	// Should have formula step only (tap as dependency is declared, not added as explicit step)
	if len(steps) != 1 {
		t.Errorf("Compile() len = %d, want 1", len(steps))
	}

	// Verify formula has tap dependency
	deps := steps[0].DependsOn()
	if len(deps) != 1 {
		t.Fatalf("DependsOn() len = %d, want 1", len(deps))
	}
	if deps[0].String() != "brew:tap:homebrew/core" {
		t.Errorf("DependsOn()[0] = %q, want %q", deps[0].String(), "brew:tap:homebrew/core")
	}
}

func TestBrewProvider_Compile_StepsOrder(t *testing.T) {
	runner := ports.NewMockCommandRunner()
	provider := NewProvider(runner)

	ctx := compiler.NewCompileContext(map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"git", "curl"},
			"casks":    []interface{}{"docker"},
		},
	})
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Taps should come first
	if steps[0].ID().String() != "brew:tap:homebrew/cask" {
		t.Errorf("First step should be tap, got %s", steps[0].ID().String())
	}
}
