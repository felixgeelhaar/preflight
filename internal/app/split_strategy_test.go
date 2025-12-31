package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSplitStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected SplitStrategy
		wantErr  bool
	}{
		{"category", "category", SplitByCategory, false},
		{"categories alias", "categories", SplitByCategory, false},
		{"language", "language", SplitByLanguage, false},
		{"languages alias", "languages", SplitByLanguage, false},
		{"lang alias", "lang", SplitByLanguage, false},
		{"stack", "stack", SplitByStack, false},
		{"stacks alias", "stacks", SplitByStack, false},
		{"provider", "provider", SplitByProvider, false},
		{"providers alias", "providers", SplitByProvider, false},
		{"case insensitive", "CATEGORY", SplitByCategory, false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ParseSplitStrategy(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidSplitStrategies(t *testing.T) {
	t.Parallel()

	strategies := ValidSplitStrategies()
	require.Len(t, strategies, 4)
	assert.Contains(t, strategies, "category")
	assert.Contains(t, strategies, "language")
	assert.Contains(t, strategies, "stack")
	assert.Contains(t, strategies, "provider")
}

func TestStrategyCategorizer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		strategy         SplitStrategy
		items            []CapturedItem
		expectedLayers   []string
		unexpectedLayers []string
	}{
		{
			name:     "category strategy",
			strategy: SplitByCategory,
			items: []CapturedItem{
				{Name: "go", Provider: "brew"},
				{Name: "trivy", Provider: "brew"},
				{Name: "docker", Provider: "brew"},
			},
			expectedLayers:   []string{"dev-go", "security", "containers"},
			unexpectedLayers: []string{"backend", "frontend"},
		},
		{
			name:     "language strategy",
			strategy: SplitByLanguage,
			items: []CapturedItem{
				{Name: "go", Provider: "brew"},
				{Name: "gopls", Provider: "brew"},
				{Name: "node", Provider: "brew"},
				{Name: "python", Provider: "brew"},
			},
			expectedLayers:   []string{"go", "node", "python"},
			unexpectedLayers: []string{"dev-go", "dev-node"},
		},
		{
			name:     "stack strategy",
			strategy: SplitByStack,
			items: []CapturedItem{
				{Name: "node", Provider: "brew"},
				{Name: "vite", Provider: "brew"},
				{Name: "docker", Provider: "brew"},
				{Name: "kubectl", Provider: "brew"},
			},
			expectedLayers:   []string{"frontend", "devops"},
			unexpectedLayers: []string{"dev-node", "containers"},
		},
		{
			name:     "provider strategy returns empty categorizer",
			strategy: SplitByProvider,
			items: []CapturedItem{
				{Name: "go", Provider: "brew"},
			},
			expectedLayers:   []string{}, // Provider strategy handled separately
			unexpectedLayers: []string{"dev-go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			categorizer := StrategyCategorizer(tt.strategy)
			require.NotNil(t, categorizer)

			result := categorizer.Categorize(tt.items)
			require.NotNil(t, result)

			for _, layer := range tt.expectedLayers {
				_, ok := result.Layers[layer]
				assert.True(t, ok, "expected layer %q to exist", layer)
			}

			for _, layer := range tt.unexpectedLayers {
				_, ok := result.Layers[layer]
				assert.False(t, ok, "did not expect layer %q to exist", layer)
			}
		})
	}
}

func TestLanguageCategories(t *testing.T) {
	t.Parallel()

	categories := languageCategories()
	require.NotEmpty(t, categories)

	// Check for expected language categories
	names := make([]string, len(categories))
	for i, cat := range categories {
		names[i] = cat.Name
	}

	assert.Contains(t, names, "go")
	assert.Contains(t, names, "node")
	assert.Contains(t, names, "python")
	assert.Contains(t, names, "rust")
	assert.Contains(t, names, "java")
	assert.Contains(t, names, "tools")
}

func TestStackCategories(t *testing.T) {
	t.Parallel()

	categories := stackCategories()
	require.NotEmpty(t, categories)

	// Check for expected stack categories
	names := make([]string, len(categories))
	for i, cat := range categories {
		names[i] = cat.Name
	}

	assert.Contains(t, names, "frontend")
	assert.Contains(t, names, "backend")
	assert.Contains(t, names, "devops")
	assert.Contains(t, names, "data")
	assert.Contains(t, names, "security")
	assert.Contains(t, names, "ai")
}

func TestCategorizeWithAI_EmptyUncategorized(t *testing.T) {
	t.Parallel()

	categorized := &CategorizedItems{
		Layers:        map[string][]CapturedItem{"base": {{Name: "git"}}},
		LayerOrder:    []string{"base"},
		Uncategorized: []CapturedItem{}, // No uncategorized items
	}

	// Should return immediately without calling AI
	err := CategorizeWithAI(context.Background(), categorized, nil, SplitByCategory)
	require.NoError(t, err)
}

// MockAICategorizer for testing
type MockAICategorizer struct {
	result *AICategorizationResult
	err    error
}

func (m *MockAICategorizer) Categorize(_ context.Context, _ AICategorizationRequest) (*AICategorizationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestCategorizeWithAI_AppliesCategorization(t *testing.T) {
	t.Parallel()

	categorized := &CategorizedItems{
		Layers:     map[string][]CapturedItem{},
		LayerOrder: []string{},
		Uncategorized: []CapturedItem{
			{Name: "unknown-tool", Provider: "brew"},
		},
	}

	mockAI := &MockAICategorizer{
		result: &AICategorizationResult{
			Categorizations: map[string]string{
				"unknown-tool": "custom-layer",
			},
			Reasoning: map[string]string{
				"unknown-tool": "AI determined this belongs in custom-layer",
			},
		},
	}

	err := CategorizeWithAI(context.Background(), categorized, mockAI, SplitByCategory)
	require.NoError(t, err)

	// Check that the item was moved to the new layer
	items, ok := categorized.Layers["custom-layer"]
	require.True(t, ok, "custom-layer should exist")
	require.Len(t, items, 1)
	assert.Equal(t, "unknown-tool", items[0].Name)

	// Check uncategorized is empty
	assert.Empty(t, categorized.Uncategorized)
}

func TestCategorizeWithAI_HandlesExistingLayers(t *testing.T) {
	t.Parallel()

	categorized := &CategorizedItems{
		Layers: map[string][]CapturedItem{
			"dev-go": {{Name: "go", Provider: "brew"}},
		},
		LayerOrder: []string{"dev-go", "misc"},
		Uncategorized: []CapturedItem{
			{Name: "gopls", Provider: "brew"},
		},
	}

	mockAI := &MockAICategorizer{
		result: &AICategorizationResult{
			Categorizations: map[string]string{
				"gopls": "dev-go", // Categorize into existing layer
			},
		},
	}

	err := CategorizeWithAI(context.Background(), categorized, mockAI, SplitByCategory)
	require.NoError(t, err)

	// Check that the item was added to existing layer
	items := categorized.Layers["dev-go"]
	require.Len(t, items, 2)
}
