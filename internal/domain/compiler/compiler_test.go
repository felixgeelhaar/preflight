package compiler

import (
	"errors"
	"testing"
)

func TestCompiler_New(t *testing.T) {
	c := NewCompiler()
	if c == nil {
		t.Fatal("NewCompiler() should not return nil")
	}
}

func TestCompiler_RegisterProvider(t *testing.T) {
	c := NewCompiler()
	provider := newMockProvider("brew")

	c.RegisterProvider(provider)

	providers := c.Providers()
	if len(providers) != 1 {
		t.Errorf("Providers() len = %d, want 1", len(providers))
	}
	if providers[0].Name() != "brew" {
		t.Errorf("Provider name = %q, want %q", providers[0].Name(), "brew")
	}
}

func TestCompiler_RegisterMultipleProviders(t *testing.T) {
	c := NewCompiler()
	c.RegisterProvider(newMockProvider("brew"))
	c.RegisterProvider(newMockProvider("apt"))
	c.RegisterProvider(newMockProvider("files"))

	if len(c.Providers()) != 3 {
		t.Errorf("Providers() len = %d, want 3", len(c.Providers()))
	}
}

func TestCompiler_Compile_Empty(t *testing.T) {
	c := NewCompiler()
	config := map[string]interface{}{}

	graph, err := c.Compile(config)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if graph.Len() != 0 {
		t.Errorf("graph.Len() = %d, want 0", graph.Len())
	}
}

func TestCompiler_Compile_SingleProvider(t *testing.T) {
	c := NewCompiler()

	provider := newMockProvider("brew")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("brew:install:git"),
			newMockStep("brew:install:curl"),
		}, nil
	}
	c.RegisterProvider(provider)

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"packages": []string{"git", "curl"},
		},
	}

	graph, err := c.Compile(config)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if graph.Len() != 2 {
		t.Errorf("graph.Len() = %d, want 2", graph.Len())
	}
}

func TestCompiler_Compile_MultipleProviders(t *testing.T) {
	c := NewCompiler()

	brewProvider := newMockProvider("brew")
	brewProvider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("brew:install:nvim"),
		}, nil
	}

	nvimProvider := newMockProvider("nvim")
	nvimProvider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("nvim:install:plugin", "brew:install:nvim"),
		}, nil
	}

	c.RegisterProvider(brewProvider)
	c.RegisterProvider(nvimProvider)

	config := map[string]interface{}{}

	graph, err := c.Compile(config)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if graph.Len() != 2 {
		t.Errorf("graph.Len() = %d, want 2", graph.Len())
	}

	// Verify dependencies are valid
	err = graph.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCompiler_Compile_ProviderError(t *testing.T) {
	c := NewCompiler()

	provider := newMockProvider("failing")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return nil, errors.New("provider failed")
	}
	c.RegisterProvider(provider)

	_, err := c.Compile(map[string]interface{}{})
	if err == nil {
		t.Error("Compile() should return error when provider fails")
	}
}

func TestCompiler_Compile_DuplicateStepError(t *testing.T) {
	c := NewCompiler()

	provider1 := newMockProvider("brew")
	provider1.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{newMockStep("common:step:id")}, nil
	}

	provider2 := newMockProvider("apt")
	provider2.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{newMockStep("common:step:id")}, nil
	}

	c.RegisterProvider(provider1)
	c.RegisterProvider(provider2)

	_, err := c.Compile(map[string]interface{}{})
	if err == nil {
		t.Error("Compile() should return error for duplicate step IDs")
	}
}

func TestCompiler_Compile_ValidatesGraph(t *testing.T) {
	c := NewCompiler()

	// Provider emits step with missing dependency
	provider := newMockProvider("broken")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("broken:step", "nonexistent:dep"),
		}, nil
	}
	c.RegisterProvider(provider)

	_, err := c.Compile(map[string]interface{}{})
	if err == nil {
		t.Error("Compile() should return error for missing dependencies")
	}
}

func TestCompiler_Compile_DetectsCycle(t *testing.T) {
	c := NewCompiler()

	// Provider emits steps with circular dependency
	provider := newMockProvider("cyclic")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("step:a", "step:b"),
			newMockStep("step:b", "step:a"),
		}, nil
	}
	c.RegisterProvider(provider)

	_, err := c.Compile(map[string]interface{}{})
	if err == nil {
		t.Error("Compile() should return error for cyclic dependencies")
	}
}

func TestCompiler_Compile_OrderedSteps(t *testing.T) {
	c := NewCompiler()

	provider := newMockProvider("ordered")
	provider.compileFn = func(_ CompileContext) ([]Step, error) {
		return []Step{
			newMockStep("step:first"),
			newMockStep("step:second", "step:first"),
			newMockStep("step:third", "step:second"),
		}, nil
	}
	c.RegisterProvider(provider)

	graph, err := c.Compile(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}

	// Verify order: first -> second -> third
	indices := make(map[string]int)
	for i, step := range sorted {
		indices[step.ID().String()] = i
	}

	if indices["step:first"] >= indices["step:second"] {
		t.Error("first should come before second")
	}
	if indices["step:second"] >= indices["step:third"] {
		t.Error("second should come before third")
	}
}
