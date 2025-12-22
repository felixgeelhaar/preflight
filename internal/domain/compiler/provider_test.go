package compiler

import (
	"errors"
	"testing"
)

// mockProvider is a test double for Provider interface.
type mockProvider struct {
	name      string
	compileFn func(CompileContext) ([]Step, error)
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name: name,
		compileFn: func(_ CompileContext) ([]Step, error) {
			return []Step{}, nil
		},
	}
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Compile(ctx CompileContext) ([]Step, error) {
	return m.compileFn(ctx)
}

func TestProvider_Name(t *testing.T) {
	provider := newMockProvider("brew")
	if provider.Name() != "brew" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "brew")
	}
}

func TestProvider_Compile_EmptySteps(t *testing.T) {
	provider := newMockProvider("brew")
	ctx := NewCompileContext(nil)

	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("Compile() len = %d, want 0", len(steps))
	}
}

func TestProvider_Compile_WithSteps(t *testing.T) {
	provider := newMockProvider("brew")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		step1 := newMockStep("brew:install:git")
		step2 := newMockStep("brew:install:curl")
		return []Step{step1, step2}, nil
	}

	ctx := NewCompileContext(nil)
	steps, err := provider.Compile(ctx)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("Compile() len = %d, want 2", len(steps))
	}
}

func TestProvider_Compile_Error(t *testing.T) {
	provider := newMockProvider("brew")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return nil, errors.New("compilation failed")
	}

	ctx := NewCompileContext(nil)
	_, err := provider.Compile(ctx)
	if err == nil {
		t.Fatal("expected error from Compile()")
	}
}

func TestCompileContext_Config(t *testing.T) {
	config := map[string]interface{}{
		"packages": []string{"git", "curl"},
	}
	ctx := NewCompileContext(config)

	if ctx.Config() == nil {
		t.Error("Config() should not be nil")
	}

	packages, ok := ctx.Config()["packages"].([]string)
	if !ok {
		t.Fatal("Config() should contain packages")
	}
	if len(packages) != 2 {
		t.Errorf("packages len = %d, want 2", len(packages))
	}
}

func TestCompileContext_GetSection(t *testing.T) {
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"packages": []string{"git", "curl"},
		},
	}
	ctx := NewCompileContext(config)

	section := ctx.GetSection("brew")
	if section == nil {
		t.Fatal("GetSection(brew) should not be nil")
	}

	packages, ok := section["packages"].([]string)
	if !ok {
		t.Fatal("section should contain packages")
	}
	if len(packages) != 2 {
		t.Errorf("packages len = %d, want 2", len(packages))
	}
}

func TestCompileContext_GetSection_Missing(t *testing.T) {
	ctx := NewCompileContext(nil)

	section := ctx.GetSection("nonexistent")
	if section != nil {
		t.Error("GetSection for missing key should return nil")
	}
}

func TestCompileContext_Provenance(t *testing.T) {
	ctx := NewCompileContext(nil).WithProvenance("layers/base.yaml")

	if ctx.Provenance() != "layers/base.yaml" {
		t.Errorf("Provenance() = %q, want %q", ctx.Provenance(), "layers/base.yaml")
	}
}
