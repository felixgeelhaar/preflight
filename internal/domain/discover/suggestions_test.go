package discover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestionGenerator_Generate(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	patterns := []Pattern{
		{
			Name:        "oh-my-zsh",
			Type:        PatternTypeShell,
			Occurrences: 80,
		},
		{
			Name:        "neovim",
			Type:        PatternTypeEditor,
			Occurrences: 50,
		},
		{
			Name:        "homebrew",
			Type:        PatternTypePackageManager,
			Occurrences: 90,
		},
	}

	suggestions := generator.Generate(patterns, 100)

	require.Len(t, suggestions, 3)
	// All suggestions should have required fields
	for _, s := range suggestions {
		assert.NotEmpty(t, s.Title)
		assert.NotEmpty(t, s.Description)
		assert.NotEmpty(t, s.ConfigSnippet)
		assert.Greater(t, s.Confidence, 0.0)
		assert.LessOrEqual(t, s.Confidence, 1.0)
	}
}

func TestSuggestionGenerator_Generate_ConfidenceBasedOnOccurrence(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	patterns := []Pattern{
		{Name: "popular", Type: PatternTypeShell, Occurrences: 90},
		{Name: "rare", Type: PatternTypeShell, Occurrences: 5},
	}

	suggestions := generator.Generate(patterns, 100)

	require.Len(t, suggestions, 2)

	var popular, rare *Suggestion
	for i := range suggestions {
		if suggestions[i].Title == "popular" || suggestions[i].Occurrences == 90 {
			popular = &suggestions[i]
		} else {
			rare = &suggestions[i]
		}
	}

	require.NotNil(t, popular)
	require.NotNil(t, rare)
	assert.Greater(t, popular.Confidence, rare.Confidence)
}

func TestSuggestionGenerator_Generate_EmptyPatterns(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()
	suggestions := generator.Generate(nil, 100)

	assert.Empty(t, suggestions)
}

func TestSuggestionGenerator_GetConfigSnippet(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	tests := []struct {
		patternName  string
		patternType  PatternType
		wantContains string
	}{
		{
			patternName:  "oh-my-zsh",
			patternType:  PatternTypeShell,
			wantContains: "shell:",
		},
		{
			patternName:  "neovim",
			patternType:  PatternTypeEditor,
			wantContains: "editor:",
		},
		{
			patternName:  "homebrew",
			patternType:  PatternTypePackageManager,
			wantContains: "brew:",
		},
		{
			patternName:  "git",
			patternType:  PatternTypeGit,
			wantContains: "git:",
		},
		{
			patternName:  "tmux",
			patternType:  PatternTypeTmux,
			wantContains: "shell:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.patternName, func(t *testing.T) {
			t.Parallel()

			snippet := generator.getConfigSnippet(tt.patternName, tt.patternType)
			assert.Contains(t, snippet, tt.wantContains)
		})
	}
}

func TestSuggestionGenerator_GetDocLinks(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	tests := []struct {
		patternName string
		wantLink    bool
	}{
		{"oh-my-zsh", true},
		{"neovim", true},
		{"homebrew", true},
		{"unknown-pattern", false},
	}

	for _, tt := range tests {
		t.Run(tt.patternName, func(t *testing.T) {
			t.Parallel()

			links := generator.getDocLinks(tt.patternName)
			if tt.wantLink {
				assert.NotEmpty(t, links)
				assert.Contains(t, links[0], "http")
			} else {
				assert.Empty(t, links)
			}
		})
	}
}

func TestSuggestionGenerator_CalculateConfidence(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	tests := []struct {
		name        string
		occurrences int
		totalRepos  int
		minScore    float64
		maxScore    float64
	}{
		{
			name:        "high adoption",
			occurrences: 90,
			totalRepos:  100,
			minScore:    0.8,
			maxScore:    1.0,
		},
		{
			name:        "medium adoption",
			occurrences: 50,
			totalRepos:  100,
			minScore:    0.4,
			maxScore:    0.7,
		},
		{
			name:        "low adoption",
			occurrences: 5,
			totalRepos:  100,
			minScore:    0.0,
			maxScore:    0.2,
		},
		{
			name:        "zero repos",
			occurrences: 0,
			totalRepos:  100,
			minScore:    0.0,
			maxScore:    0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			confidence := generator.calculateConfidence(tt.occurrences, tt.totalRepos)
			assert.GreaterOrEqual(t, confidence, tt.minScore)
			assert.LessOrEqual(t, confidence, tt.maxScore)
		})
	}
}

func TestSuggestionGenerator_GetDescription(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	tests := []struct {
		patternName  string
		wantContains string
	}{
		{"oh-my-zsh", "framework"},
		{"neovim", "editor"},
		{"homebrew", "package"},
		{"starship", "prompt"},
	}

	for _, tt := range tests {
		t.Run(tt.patternName, func(t *testing.T) {
			t.Parallel()

			desc := generator.getDescription(tt.patternName)
			assert.Contains(t, desc, tt.wantContains)
		})
	}
}

func TestSuggestionGenerator_GetReasons(t *testing.T) {
	t.Parallel()

	generator := NewSuggestionGenerator()

	pattern := Pattern{
		Name:        "oh-my-zsh",
		Type:        PatternTypeShell,
		Occurrences: 80,
	}

	reasons := generator.getReasons(pattern, 100)

	require.NotEmpty(t, reasons)
	// Should include popularity reason
	found := false
	for _, r := range reasons {
		if r != "" {
			found = true
		}
	}
	assert.True(t, found)
}
