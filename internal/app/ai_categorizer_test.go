package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCategorizationPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		strategy       SplitStrategy
		expectedText   []string
		unexpectedText []string
	}{
		{
			name:     "category strategy prompt",
			strategy: SplitByCategory,
			expectedText: []string{
				"BY CATEGORY",
				"functional category",
			},
			unexpectedText: []string{
				"BY LANGUAGE",
				"BY STACK",
			},
		},
		{
			name:     "language strategy prompt",
			strategy: SplitByLanguage,
			expectedText: []string{
				"BY LANGUAGE",
				"programming language",
			},
			unexpectedText: []string{
				"BY CATEGORY",
				"BY STACK",
			},
		},
		{
			name:     "stack strategy prompt",
			strategy: SplitByStack,
			expectedText: []string{
				"BY STACK",
				"tech stack role",
			},
			unexpectedText: []string{
				"BY CATEGORY",
				"BY LANGUAGE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := AICategorizationRequest{
				Items: []CapturedItem{
					{Name: "go", Provider: "brew"},
					{Name: "node", Provider: "brew"},
				},
				AvailableLayers: []string{"base", "dev-go"},
				Strategy:        tt.strategy,
			}

			prompt := buildCategorizationPrompt(req)
			require.NotEmpty(t, prompt)

			for _, text := range tt.expectedText {
				assert.Contains(t, prompt, text)
			}

			for _, text := range tt.unexpectedText {
				assert.NotContains(t, prompt, text)
			}

			// Check that items are included
			assert.Contains(t, prompt, "go")
			assert.Contains(t, prompt, "node")

			// Check that layers are included
			assert.Contains(t, prompt, "base")
			assert.Contains(t, prompt, "dev-go")
		})
	}
}

func TestExtractJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "json in markdown code block",
			input: `Here's the categorization:
` + "```json" + `
{"categorizations": {"go": "dev-go"}}
` + "```" + `
Hope this helps!`,
			expected: `{"categorizations": {"go": "dev-go"}}`,
		},
		{
			name:     "json in plain code block",
			input:    "```\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "raw json",
			input:    `Some text {"key": "value"} more text`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested json",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no json",
			input:    "just plain text",
			expected: "",
		},
		{
			name:     "incomplete json",
			input:    `{"key": "value"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCategorizationResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		response               string
		items                  []CapturedItem
		expectedCategorization map[string]string
		expectedReasoning      map[string]string
	}{
		{
			name: "valid response",
			response: `{
				"categorizations": {"go": "dev-go", "node": "dev-node"},
				"reasoning": {"go": "Go tool", "node": "Node.js tool"}
			}`,
			items: []CapturedItem{
				{Name: "go"},
				{Name: "node"},
			},
			expectedCategorization: map[string]string{
				"go":   "dev-go",
				"node": "dev-node",
			},
			expectedReasoning: map[string]string{
				"go":   "Go tool",
				"node": "Node.js tool",
			},
		},
		{
			name:     "empty response returns empty result",
			response: "",
			items: []CapturedItem{
				{Name: "test"},
			},
			expectedCategorization: map[string]string{},
			expectedReasoning:      map[string]string{},
		},
		{
			name:     "invalid json returns empty result",
			response: "not json at all",
			items: []CapturedItem{
				{Name: "test"},
			},
			expectedCategorization: map[string]string{},
			expectedReasoning:      map[string]string{},
		},
		{
			name: "filters out unknown packages",
			response: `{
				"categorizations": {"go": "dev-go", "unknown": "misc"},
				"reasoning": {"go": "Go tool", "unknown": "Unknown"}
			}`,
			items: []CapturedItem{
				{Name: "go"},
				// "unknown" is not in items
			},
			expectedCategorization: map[string]string{
				"go": "dev-go",
			},
			expectedReasoning: map[string]string{
				"go": "Go tool",
			},
		},
		{
			name: "case insensitive matching",
			response: `{
				"categorizations": {"GO": "dev-go"},
				"reasoning": {"GO": "Go tool"}
			}`,
			items: []CapturedItem{
				{Name: "go"},
			},
			expectedCategorization: map[string]string{
				"GO": "dev-go",
			},
			expectedReasoning: map[string]string{
				"GO": "Go tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseCategorizationResponse(tt.response, tt.items)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedCategorization, result.Categorizations)
			assert.Equal(t, tt.expectedReasoning, result.Reasoning)
		})
	}
}

func TestProviderCategorizer_EmptyItems(t *testing.T) {
	t.Parallel()

	// Test that empty items returns empty result without calling provider
	categorizer := NewProviderCategorizer(nil)

	result, err := categorizer.Categorize(context.Background(), AICategorizationRequest{
		Items: []CapturedItem{}, // Empty
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Categorizations)
}
