package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLayerCategorizer(t *testing.T) {
	t.Parallel()
	c := NewLayerCategorizer()
	require.NotNil(t, c)
	assert.NotEmpty(t, c.categories)
}

func TestLayerCategorizer_Categorize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		items          []CapturedItem
		expectedLayers map[string][]string // layer name -> item names
	}{
		{
			name: "categorizes base CLI tools",
			items: []CapturedItem{
				{Name: "git", Provider: "brew"},
				{Name: "curl", Provider: "brew"},
				{Name: "jq", Provider: "brew"},
				{Name: "ripgrep", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"base": {"git", "curl", "jq", "ripgrep"},
			},
		},
		{
			name: "categorizes Go development tools",
			items: []CapturedItem{
				{Name: "go", Provider: "brew"},
				{Name: "gopls", Provider: "brew"},
				{Name: "golangci-lint", Provider: "brew"},
				{Name: "go@1.24", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"dev-go": {"go", "gopls", "golangci-lint", "go@1.24"},
			},
		},
		{
			name: "categorizes Node.js tools",
			items: []CapturedItem{
				{Name: "node", Provider: "brew"},
				{Name: "pnpm", Provider: "brew"},
				{Name: "typescript", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"dev-node": {"node", "pnpm", "typescript"},
			},
		},
		{
			name: "categorizes Python tools",
			items: []CapturedItem{
				{Name: "python@3.12", Provider: "brew"},
				{Name: "poetry", Provider: "brew"},
				{Name: "ruff", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"dev-python": {"python@3.12", "poetry", "ruff"},
			},
		},
		{
			name: "categorizes security tools",
			items: []CapturedItem{
				{Name: "trivy", Provider: "brew"},
				{Name: "grype", Provider: "brew"},
				{Name: "nmap", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"security": {"trivy", "grype", "nmap"},
			},
		},
		{
			name: "categorizes container tools",
			items: []CapturedItem{
				{Name: "docker", Provider: "brew"},
				{Name: "kubectl", Provider: "brew"},
				{Name: "helm", Provider: "brew"},
				{Name: "k9s", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"containers": {"docker", "kubectl", "helm", "k9s"},
			},
		},
		{
			name: "categorizes mixed items into multiple layers",
			items: []CapturedItem{
				{Name: "git", Provider: "brew"},
				{Name: "go", Provider: "brew"},
				{Name: "node", Provider: "brew"},
				{Name: "trivy", Provider: "brew"},
				{Name: "docker", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"base":       {"git"},
				"dev-go":     {"go"},
				"dev-node":   {"node"},
				"security":   {"trivy"},
				"containers": {"docker"},
			},
		},
		{
			name: "puts uncategorized items in misc",
			items: []CapturedItem{
				{Name: "some-unknown-tool", Provider: "brew"},
				{Name: "another-custom-thing", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"misc": {"some-unknown-tool", "another-custom-thing"},
			},
		},
		{
			name: "categorizes fonts",
			items: []CapturedItem{
				{Name: "font-jetbrains-mono-nerd-font", Provider: "brew"},
				{Name: "font-fira-code", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"fonts": {"font-jetbrains-mono-nerd-font", "font-fira-code"},
			},
		},
		{
			name: "categorizes git tools",
			items: []CapturedItem{
				{Name: "gh", Provider: "brew"},
				{Name: "lazygit", Provider: "brew"},
				{Name: "git-delta", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"git": {"gh", "lazygit", "git-delta"},
			},
		},
		{
			name: "categorizes shell tools",
			items: []CapturedItem{
				{Name: "zsh", Provider: "brew"},
				{Name: "starship", Provider: "brew"},
				{Name: "alacritty", Provider: "brew"},
			},
			expectedLayers: map[string][]string{
				"shell": {"zsh", "starship", "alacritty"},
			},
		},
		{
			name:           "handles empty input",
			items:          []CapturedItem{},
			expectedLayers: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewLayerCategorizer()
			result := c.Categorize(tt.items)

			require.NotNil(t, result)
			assert.Len(t, result.Layers, len(tt.expectedLayers))

			for layerName, expectedNames := range tt.expectedLayers {
				items, ok := result.Layers[layerName]
				assert.True(t, ok, "expected layer %q to exist", layerName)

				actualNames := make([]string, len(items))
				for i, item := range items {
					actualNames[i] = item.Name
				}
				assert.ElementsMatch(t, expectedNames, actualNames, "layer %q mismatch", layerName)
			}
		})
	}
}

func TestLayerCategorizer_VersionedPackages(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()

	tests := []struct {
		name     string
		pkg      string
		expected string
	}{
		{"go@1.24", "go@1.24", "dev-go"},
		{"python@3.12", "python@3.12", "dev-python"},
		{"node@20", "node@20", "dev-node"},
		{"openjdk@21", "openjdk@21", "dev-java"},
		{"mysql@8.4", "mysql@8.4", "database"},
		{"postgresql@16", "postgresql@16", "database"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			items := []CapturedItem{{Name: tt.pkg, Provider: "brew"}}
			result := c.Categorize(items)

			_, ok := result.Layers[tt.expected]
			assert.True(t, ok, "expected %q to be in layer %q", tt.pkg, tt.expected)
		})
	}
}

func TestCategorizedItems_Summary(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()
	items := []CapturedItem{
		{Name: "git", Provider: "brew"},
		{Name: "curl", Provider: "brew"},
		{Name: "jq", Provider: "brew"},
		{Name: "go", Provider: "brew"},
		{Name: "gopls", Provider: "brew"},
		{Name: "node", Provider: "brew"},
	}

	result := c.Categorize(items)
	summary := result.Summary()

	require.NotEmpty(t, summary)

	// Find base summary
	var baseSummary *LayerSummary
	for i := range summary {
		if summary[i].Name == "base" {
			baseSummary = &summary[i]
			break
		}
	}

	require.NotNil(t, baseSummary)
	assert.Equal(t, 3, baseSummary.Count)
}

func TestCategorizedItems_SortItemsAlphabetically(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()
	items := []CapturedItem{
		{Name: "zsh", Provider: "brew"},
		{Name: "alacritty", Provider: "brew"},
		{Name: "starship", Provider: "brew"},
	}

	result := c.Categorize(items)
	result.SortItemsAlphabetically()

	shellItems := result.Layers["shell"]
	require.Len(t, shellItems, 3)
	assert.Equal(t, "alacritty", shellItems[0].Name)
	assert.Equal(t, "starship", shellItems[1].Name)
	assert.Equal(t, "zsh", shellItems[2].Name)
}

func TestLayerCategorizer_GetLayerDescription(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()

	tests := []struct {
		layer    string
		expected string
	}{
		{"base", "Core CLI utilities and essential tools"},
		{"dev-go", "Go development ecosystem"},
		{"security", "Security scanning and analysis tools"},
		{"misc", "Uncategorized items"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.layer, func(t *testing.T) {
			t.Parallel()
			desc := c.GetLayerDescription(tt.layer)
			assert.Equal(t, tt.expected, desc)
		})
	}
}

func TestLayerCategorizer_AvailableCategories(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()
	categories := c.AvailableCategories()

	require.NotEmpty(t, categories)
	assert.Contains(t, categories, "base")
	assert.Contains(t, categories, "dev-go")
	assert.Contains(t, categories, "dev-node")
	assert.Contains(t, categories, "security")
	assert.Contains(t, categories, "containers")
}

func TestLayerCategorizer_LayerOrder(t *testing.T) {
	t.Parallel()

	c := NewLayerCategorizer()
	items := []CapturedItem{
		{Name: "docker", Provider: "brew"}, // containers
		{Name: "git", Provider: "brew"},    // base
		{Name: "go", Provider: "brew"},     // dev-go
	}

	result := c.Categorize(items)

	// Verify layer order is maintained (base should come before dev-go and containers)
	require.Len(t, result.LayerOrder, 3)

	baseIdx := -1
	goIdx := -1
	dockerIdx := -1
	for i, name := range result.LayerOrder {
		switch name {
		case "base":
			baseIdx = i
		case "dev-go":
			goIdx = i
		case "containers":
			dockerIdx = i
		}
	}

	assert.Less(t, baseIdx, goIdx, "base should come before dev-go")
	assert.Less(t, goIdx, dockerIdx, "dev-go should come before containers")
}
