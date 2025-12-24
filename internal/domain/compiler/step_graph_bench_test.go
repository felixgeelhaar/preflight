package compiler

import (
	"fmt"
	"testing"
)

// createLinearGraphForBench creates a graph with a linear dependency chain.
// Step 0 <- Step 1 <- Step 2 <- ... <- Step N-1
func createLinearGraphForBench(size int) *StepGraph {
	graph := NewStepGraph()

	for i := 0; i < size; i++ {
		var deps []string
		if i > 0 {
			deps = []string{fmt.Sprintf("step:%d", i-1)}
		}
		step := newMockStep(fmt.Sprintf("step:%d", i), deps...)
		_ = graph.Add(step)
	}

	return graph
}

// createWideGraphForBench creates a graph where all steps depend on a single root.
// Step 0 (root) <- Step 1, Step 2, ..., Step N-1
func createWideGraphForBench(size int) *StepGraph {
	graph := NewStepGraph()

	// Add root
	root := newMockStep("step:0")
	_ = graph.Add(root)

	// Add children that depend on root
	for i := 1; i < size; i++ {
		step := newMockStep(fmt.Sprintf("step:%d", i), "step:0")
		_ = graph.Add(step)
	}

	return graph
}

// createDiamondGraphForBench creates a diamond dependency pattern.
// Multiple paths converge to a single node.
func createDiamondGraphForBench(depth, width int) *StepGraph {
	graph := NewStepGraph()

	// Layer 0: root
	root := newMockStep("step:root")
	_ = graph.Add(root)

	prevLayer := []string{"step:root"}

	// Middle layers
	for d := 1; d < depth-1; d++ {
		currentLayer := make([]string, width)
		for w := 0; w < width; w++ {
			id := fmt.Sprintf("step:l%d-s%d", d, w)
			currentLayer[w] = id
			step := newMockStep(id, prevLayer...)
			_ = graph.Add(step)
		}
		prevLayer = currentLayer
	}

	// Final layer: sink (depends on all previous layer nodes)
	sink := newMockStep("step:sink", prevLayer...)
	_ = graph.Add(sink)

	return graph
}

// createComplexGraphForBench creates a realistic graph with mixed dependencies.
func createComplexGraphForBench(size int) *StepGraph {
	graph := NewStepGraph()

	// Add roots (no dependencies)
	numRoots := size / 10
	if numRoots < 3 {
		numRoots = 3
	}
	for i := 0; i < numRoots; i++ {
		step := newMockStep(fmt.Sprintf("root:%d", i))
		_ = graph.Add(step)
	}

	// Add middle steps with varying dependencies
	for i := numRoots; i < size-1; i++ {
		// Each step depends on 1-3 previous steps
		numDeps := (i % 3) + 1
		deps := make([]string, 0, numDeps)
		for j := 0; j < numDeps && i-j-1 >= 0; j++ {
			depIdx := i - j - 1
			if depIdx < numRoots {
				deps = append(deps, fmt.Sprintf("root:%d", depIdx))
			} else {
				deps = append(deps, fmt.Sprintf("step:%d", depIdx))
			}
		}
		step := newMockStep(fmt.Sprintf("step:%d", i), deps...)
		_ = graph.Add(step)
	}

	// Add final step that depends on several late steps
	if size > numRoots+1 {
		deps := make([]string, 0, 3)
		for j := 0; j < 3 && size-2-j >= numRoots; j++ {
			deps = append(deps, fmt.Sprintf("step:%d", size-2-j))
		}
		step := newMockStep("step:final", deps...)
		_ = graph.Add(step)
	}

	return graph
}

func BenchmarkStepGraph_Add_10Steps(b *testing.B) {
	for i := 0; i < b.N; i++ {
		graph := NewStepGraph()
		for j := 0; j < 10; j++ {
			step := newMockStep(fmt.Sprintf("step:%d", j))
			_ = graph.Add(step)
		}
	}
}

func BenchmarkStepGraph_Add_100Steps(b *testing.B) {
	for i := 0; i < b.N; i++ {
		graph := NewStepGraph()
		for j := 0; j < 100; j++ {
			step := newMockStep(fmt.Sprintf("step:%d", j))
			_ = graph.Add(step)
		}
	}
}

func BenchmarkStepGraph_Add_1000Steps(b *testing.B) {
	for i := 0; i < b.N; i++ {
		graph := NewStepGraph()
		for j := 0; j < 1000; j++ {
			step := newMockStep(fmt.Sprintf("step:%d", j))
			_ = graph.Add(step)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Linear_10(b *testing.B) {
	graph := createLinearGraphForBench(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Linear_100(b *testing.B) {
	graph := createLinearGraphForBench(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Linear_1000(b *testing.B) {
	graph := createLinearGraphForBench(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Wide_100(b *testing.B) {
	graph := createWideGraphForBench(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Wide_1000(b *testing.B) {
	graph := createWideGraphForBench(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Diamond_5x10(b *testing.B) {
	graph := createDiamondGraphForBench(5, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Diamond_10x20(b *testing.B) {
	graph := createDiamondGraphForBench(10, 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Complex_50(b *testing.B) {
	graph := createComplexGraphForBench(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Complex_200(b *testing.B) {
	graph := createComplexGraphForBench(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_TopologicalSort_Complex_500(b *testing.B) {
	graph := createComplexGraphForBench(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_Validate_100Steps(b *testing.B) {
	graph := createComplexGraphForBench(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := graph.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepGraph_Get_100Steps(b *testing.B) {
	graph := createComplexGraphForBench(100)
	targetID := MustNewStepID("step:50")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, found := graph.Get(targetID)
		if !found {
			b.Fatal("step not found")
		}
	}
}

func BenchmarkStepGraph_Roots_Wide_100(b *testing.B) {
	graph := createWideGraphForBench(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		roots := graph.Roots()
		if len(roots) != 1 {
			b.Fatal("expected 1 root")
		}
	}
}

func BenchmarkStepGraph_Leaves_Wide_100(b *testing.B) {
	graph := createWideGraphForBench(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaves := graph.Leaves()
		if len(leaves) != 99 {
			b.Fatalf("expected 99 leaves, got %d", len(leaves))
		}
	}
}

func BenchmarkStepGraph_MemoryAllocation_Add(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		graph := NewStepGraph()
		for j := 0; j < 100; j++ {
			var deps []string
			if j > 0 {
				deps = []string{fmt.Sprintf("step:%d", j-1)}
			}
			step := newMockStep(fmt.Sprintf("step:%d", j), deps...)
			_ = graph.Add(step)
		}
	}
}

func BenchmarkStepGraph_MemoryAllocation_Sort(b *testing.B) {
	graph := createComplexGraphForBench(100)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = graph.TopologicalSort()
	}
}
