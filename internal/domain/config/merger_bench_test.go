package config_test

import (
	"fmt"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
)

// createBenchmarkLayers creates layers for merger benchmarks.
func createBenchmarkLayers(numLayers, itemsPerLayer int) []config.Layer {
	layers := make([]config.Layer, numLayers)

	for i := 0; i < numLayers; i++ {
		layerYAML := fmt.Sprintf(`
name: layer%d
packages:
  brew:
    formulae:
`, i)
		for j := 0; j < itemsPerLayer; j++ {
			layerYAML += fmt.Sprintf("      - formula%d_%d\n", i, j)
		}
		layerYAML += "    casks:\n"
		for j := 0; j < itemsPerLayer/2; j++ {
			layerYAML += fmt.Sprintf("      - cask%d_%d\n", i, j)
		}

		layerYAML += `git:
  user:
`
		layerYAML += fmt.Sprintf("    name: \"User %d\"\n", i)
		layerYAML += fmt.Sprintf("    email: \"user%d@example.com\"\n", i)
		layerYAML += "  alias:\n"
		for j := 0; j < itemsPerLayer; j++ {
			layerYAML += fmt.Sprintf("    alias%d_%d: \"command%d_%d\"\n", i, j, i, j)
		}

		layerYAML += `shell:
  env:
`
		for j := 0; j < itemsPerLayer; j++ {
			layerYAML += fmt.Sprintf("    VAR%d_%d: \"value%d_%d\"\n", i, j, i, j)
		}
		layerYAML += "  aliases:\n"
		for j := 0; j < itemsPerLayer; j++ {
			layerYAML += fmt.Sprintf("    sh%d_%d: \"shellcmd%d_%d\"\n", i, j, i, j)
		}

		layer, err := config.ParseLayer([]byte(layerYAML))
		if err != nil {
			panic(err)
		}
		layers[i] = *layer
	}

	return layers
}

func BenchmarkMerger_Merge_3Layers_Small(b *testing.B) {
	layers := createBenchmarkLayers(3, 5)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_3Layers_Medium(b *testing.B) {
	layers := createBenchmarkLayers(3, 20)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_3Layers_Large(b *testing.B) {
	layers := createBenchmarkLayers(3, 50)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_10Layers_Small(b *testing.B) {
	layers := createBenchmarkLayers(10, 5)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_10Layers_Medium(b *testing.B) {
	layers := createBenchmarkLayers(10, 20)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_10Layers_Large(b *testing.B) {
	layers := createBenchmarkLayers(10, 50)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_HighOverlap(b *testing.B) {
	// Create layers with overlapping content to test deduplication
	numLayers := 5
	layers := make([]config.Layer, numLayers)

	for i := 0; i < numLayers; i++ {
		layerYAML := fmt.Sprintf(`
name: layer%d
packages:
  brew:
    formulae:
      - ripgrep
      - fzf
      - bat
      - fd
      - jq
      - git
      - gh
      - neovim
      - tmux
      - unique%d
    casks:
      - docker
      - visual-studio-code
      - wezterm
      - unique%d
git:
  user:
    name: "User %d"
    email: "user%d@example.com"
  alias:
    co: checkout
    br: branch
    ci: commit
    st: status
    custom%d: "custom command %d"
shell:
  default: zsh
  env:
    EDITOR: nvim
    VISUAL: nvim
    CUSTOM%d: "value%d"
  aliases:
    ll: "ls -la"
    la: "ls -A"
    custom%d: "command%d"
`, i, i, i, i, i, i, i, i, i, i, i)

		layer, err := config.ParseLayer([]byte(layerYAML))
		if err != nil {
			b.Fatal(err)
		}
		layers[i] = *layer
	}

	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMerger_Merge_ProvenanceTracking(b *testing.B) {
	// Test with provenance tracking overhead
	layers := createBenchmarkLayers(5, 30)
	merger := config.NewMerger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		merged, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
		// Access provenance to ensure it's computed
		_ = merged.GetProvenance("packages.brew.formulae", "formula0_0")
	}
}

func BenchmarkMerger_Merge_MemoryAllocation(b *testing.B) {
	layers := createBenchmarkLayers(5, 20)
	merger := config.NewMerger()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := merger.Merge(layers)
		if err != nil {
			b.Fatal(err)
		}
	}
}
