package compiler

import (
	"errors"
	"testing"
)

func TestStepGraph_Empty(t *testing.T) {
	graph := NewStepGraph()

	if graph.Len() != 0 {
		t.Errorf("Len() = %d, want 0", graph.Len())
	}

	steps := graph.Steps()
	if len(steps) != 0 {
		t.Errorf("Steps() len = %d, want 0", len(steps))
	}
}

func TestStepGraph_AddStep(t *testing.T) {
	graph := NewStepGraph()
	step := newMockStep("brew:install:git")

	err := graph.Add(step)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if graph.Len() != 1 {
		t.Errorf("Len() = %d, want 1", graph.Len())
	}
}

func TestStepGraph_AddDuplicate(t *testing.T) {
	graph := NewStepGraph()
	step1 := newMockStep("brew:install:git")
	step2 := newMockStep("brew:install:git")

	_ = graph.Add(step1)
	err := graph.Add(step2)

	if !errors.Is(err, ErrDuplicateStep) {
		t.Errorf("Add() error = %v, want %v", err, ErrDuplicateStep)
	}
}

func TestStepGraph_Get(t *testing.T) {
	graph := NewStepGraph()
	step := newMockStep("brew:install:git")
	_ = graph.Add(step)

	id, _ := NewStepID("brew:install:git")
	retrieved, ok := graph.Get(id)
	if !ok {
		t.Fatal("Get() should find the step")
	}
	if retrieved.ID().String() != "brew:install:git" {
		t.Errorf("Get() ID = %q, want %q", retrieved.ID().String(), "brew:install:git")
	}
}

func TestStepGraph_Get_NotFound(t *testing.T) {
	graph := NewStepGraph()

	id, _ := NewStepID("nonexistent:step:id")
	_, ok := graph.Get(id)
	if ok {
		t.Error("Get() should not find nonexistent step")
	}
}

func TestStepGraph_TopologicalSort_NoDeps(t *testing.T) {
	graph := NewStepGraph()
	_ = graph.Add(newMockStep("brew:install:git"))
	_ = graph.Add(newMockStep("brew:install:curl"))
	_ = graph.Add(newMockStep("brew:install:wget"))

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("TopologicalSort() len = %d, want 3", len(sorted))
	}
}

func TestStepGraph_TopologicalSort_WithDeps(t *testing.T) {
	graph := NewStepGraph()

	// nvim plugin depends on nvim
	nvim := newMockStep("brew:install:nvim")
	plugin := newMockStep("nvim:install:plugin", "brew:install:nvim")

	_ = graph.Add(nvim)
	_ = graph.Add(plugin)

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("TopologicalSort() len = %d, want 2", len(sorted))
	}

	// nvim must come before plugin
	nvimIdx := -1
	pluginIdx := -1
	for i, step := range sorted {
		if step.ID().String() == "brew:install:nvim" {
			nvimIdx = i
		}
		if step.ID().String() == "nvim:install:plugin" {
			pluginIdx = i
		}
	}

	if nvimIdx >= pluginIdx {
		t.Errorf("nvim (idx %d) should come before plugin (idx %d)", nvimIdx, pluginIdx)
	}
}

func TestStepGraph_TopologicalSort_ComplexDeps(t *testing.T) {
	graph := NewStepGraph()

	// A -> B -> D
	// A -> C -> D
	a := newMockStep("step:a")
	b := newMockStep("step:b", "step:a")
	c := newMockStep("step:c", "step:a")
	d := newMockStep("step:d", "step:b", "step:c")

	_ = graph.Add(a)
	_ = graph.Add(b)
	_ = graph.Add(c)
	_ = graph.Add(d)

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}

	// Find indices
	indices := make(map[string]int)
	for i, step := range sorted {
		indices[step.ID().String()] = i
	}

	// Verify ordering constraints
	if indices["step:a"] >= indices["step:b"] {
		t.Error("a should come before b")
	}
	if indices["step:a"] >= indices["step:c"] {
		t.Error("a should come before c")
	}
	if indices["step:b"] >= indices["step:d"] {
		t.Error("b should come before d")
	}
	if indices["step:c"] >= indices["step:d"] {
		t.Error("c should come before d")
	}
}

func TestStepGraph_TopologicalSort_Cycle(t *testing.T) {
	graph := NewStepGraph()

	// Create a cycle: A -> B -> A
	a := newMockStep("step:a", "step:b")
	b := newMockStep("step:b", "step:a")

	_ = graph.Add(a)
	_ = graph.Add(b)

	_, err := graph.TopologicalSort()
	if !errors.Is(err, ErrCyclicDependency) {
		t.Errorf("TopologicalSort() error = %v, want %v", err, ErrCyclicDependency)
	}
}

func TestStepGraph_Validate_MissingDep(t *testing.T) {
	graph := NewStepGraph()

	// Step with dependency on nonexistent step
	step := newMockStep("nvim:install:plugin", "brew:install:nvim")
	_ = graph.Add(step)

	err := graph.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing dependency")
	}
}

func TestStepGraph_Validate_Valid(t *testing.T) {
	graph := NewStepGraph()

	nvim := newMockStep("brew:install:nvim")
	plugin := newMockStep("nvim:install:plugin", "brew:install:nvim")

	_ = graph.Add(nvim)
	_ = graph.Add(plugin)

	err := graph.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestStepGraph_Roots(t *testing.T) {
	graph := NewStepGraph()

	// root1 and root2 have no deps, child depends on both
	root1 := newMockStep("root:one")
	root2 := newMockStep("root:two")
	child := newMockStep("child:step", "root:one", "root:two")

	_ = graph.Add(root1)
	_ = graph.Add(root2)
	_ = graph.Add(child)

	roots := graph.Roots()
	if len(roots) != 2 {
		t.Errorf("Roots() len = %d, want 2", len(roots))
	}
}

func TestStepGraph_Leaves(t *testing.T) {
	graph := NewStepGraph()

	// root has two children (leaves)
	root := newMockStep("root:step")
	leaf1 := newMockStep("leaf:one", "root:step")
	leaf2 := newMockStep("leaf:two", "root:step")

	_ = graph.Add(root)
	_ = graph.Add(leaf1)
	_ = graph.Add(leaf2)

	leaves := graph.Leaves()
	if len(leaves) != 2 {
		t.Errorf("Leaves() len = %d, want 2", len(leaves))
	}
}
