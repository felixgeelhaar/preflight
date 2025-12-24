package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/discover"
)

func TestDiscover_PatternMatcher_DetectsShellPatterns(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".zshrc",
		".zshenv",
		".oh-my-zsh/",
		".oh-my-zsh/custom/plugins/",
	}

	patterns := matcher.MatchFiles(files)

	// Should detect both zsh and oh-my-zsh
	patternNames := make(map[string]bool)
	for _, p := range patterns {
		patternNames[p.Name] = true
	}

	assert.True(t, patternNames["zsh"], "should detect zsh")
	assert.True(t, patternNames["oh-my-zsh"], "should detect oh-my-zsh")
}

func TestDiscover_PatternMatcher_DetectsEditorPatterns(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".config/nvim/init.lua",
		".config/nvim/lua/",
		".vimrc",
		".vim/",
	}

	patterns := matcher.MatchFiles(files)

	patternNames := make(map[string]bool)
	for _, p := range patterns {
		patternNames[p.Name] = true
	}

	assert.True(t, patternNames["neovim"], "should detect neovim")
	assert.True(t, patternNames["vim"], "should detect vim")
}

func TestDiscover_PatternMatcher_DetectsGitPattern(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".gitconfig",
		".gitignore_global",
	}

	patterns := matcher.MatchFiles(files)

	var gitPattern *discover.Pattern
	for i := range patterns {
		if patterns[i].Name == "git" {
			gitPattern = &patterns[i]
			break
		}
	}

	require.NotNil(t, gitPattern, "should detect git pattern")
	assert.Equal(t, discover.PatternTypeGit, gitPattern.Type)
}

func TestDiscover_PatternMatcher_DetectsSSHPattern(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".ssh/config",
		".ssh/known_hosts",
	}

	patterns := matcher.MatchFiles(files)

	var sshPattern *discover.Pattern
	for i := range patterns {
		if patterns[i].Name == "ssh" {
			sshPattern = &patterns[i]
			break
		}
	}

	require.NotNil(t, sshPattern, "should detect ssh pattern")
	assert.Equal(t, discover.PatternTypeSSH, sshPattern.Type)
}

func TestDiscover_PatternMatcher_DetectsTmuxPattern(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".tmux.conf",
		".tmux/plugins/tpm/",
	}

	patterns := matcher.MatchFiles(files)

	patternNames := make(map[string]bool)
	for _, p := range patterns {
		patternNames[p.Name] = true
	}

	assert.True(t, patternNames["tmux"], "should detect tmux")
	assert.True(t, patternNames["tpm"], "should detect tpm")
}

func TestDiscover_PatternMatcher_DetectsPackageManagerPatterns(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		"Brewfile",
		".config/nix/",
	}

	patterns := matcher.MatchFiles(files)

	patternNames := make(map[string]bool)
	for _, p := range patterns {
		patternNames[p.Name] = true
	}

	assert.True(t, patternNames["homebrew"], "should detect homebrew")
	assert.True(t, patternNames["nix"], "should detect nix")
}

func TestDiscover_PatternMatcher_FilterByType(t *testing.T) {
	t.Parallel()

	matcher := discover.NewPatternMatcher()

	files := []string{
		".zshrc",
		".config/nvim/init.lua",
		".gitconfig",
		".tmux.conf",
	}

	// Filter to only shell patterns
	shellMatcher := matcher.FilterByTypes([]discover.PatternType{discover.PatternTypeShell})
	patterns := shellMatcher.MatchFiles(files)

	for _, p := range patterns {
		assert.Equal(t, discover.PatternTypeShell, p.Type, "should only return shell patterns")
	}
}

func TestDiscover_SuggestionGenerator_GeneratesValidSuggestions(t *testing.T) {
	t.Parallel()

	generator := discover.NewSuggestionGenerator()

	patterns := []discover.Pattern{
		{Name: "oh-my-zsh", Type: discover.PatternTypeShell, Occurrences: 80},
		{Name: "neovim", Type: discover.PatternTypeEditor, Occurrences: 50},
	}

	suggestions := generator.Generate(patterns, 100)

	require.Len(t, suggestions, 2)

	for _, s := range suggestions {
		assert.NotEmpty(t, s.Title, "suggestion should have title")
		assert.NotEmpty(t, s.Description, "suggestion should have description")
		assert.NotEmpty(t, s.ConfigSnippet, "suggestion should have config snippet")
		assert.Greater(t, s.Confidence, 0.0, "suggestion should have positive confidence")
		assert.LessOrEqual(t, s.Confidence, 1.0, "confidence should not exceed 1.0")
	}
}

func TestDiscover_SuggestionGenerator_HigherOccurrencesHaveHigherConfidence(t *testing.T) {
	t.Parallel()

	generator := discover.NewSuggestionGenerator()

	patterns := []discover.Pattern{
		{Name: "popular", Type: discover.PatternTypeShell, Occurrences: 90},
		{Name: "rare", Type: discover.PatternTypeShell, Occurrences: 5},
	}

	suggestions := generator.Generate(patterns, 100)

	require.Len(t, suggestions, 2)

	var popular, rare *discover.Suggestion
	for i := range suggestions {
		if suggestions[i].Occurrences == 90 {
			popular = &suggestions[i]
		} else {
			rare = &suggestions[i]
		}
	}

	require.NotNil(t, popular)
	require.NotNil(t, rare)
	assert.Greater(t, popular.Confidence, rare.Confidence)
}

func TestDiscover_DiscoveryResult_TopSuggestions(t *testing.T) {
	t.Parallel()

	result := &discover.DiscoveryResult{
		ReposAnalyzed: 100,
		Suggestions: []discover.Suggestion{
			{Title: "First", Occurrences: 90},
			{Title: "Second", Occurrences: 70},
			{Title: "Third", Occurrences: 50},
			{Title: "Fourth", Occurrences: 30},
			{Title: "Fifth", Occurrences: 10},
		},
	}

	top3 := result.TopSuggestions(3)

	require.Len(t, top3, 3)
	assert.Equal(t, "First", top3[0].Title)
	assert.Equal(t, "Second", top3[1].Title)
	assert.Equal(t, "Third", top3[2].Title)
}

func TestDiscover_SearchOptions_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    discover.SearchOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: discover.SearchOptions{
				Query:      "dotfiles",
				MinStars:   10,
				MaxResults: 50,
			},
			wantErr: false,
		},
		{
			name: "negative min stars",
			opts: discover.SearchOptions{
				Query:      "dotfiles",
				MinStars:   -1,
				MaxResults: 50,
			},
			wantErr: true,
		},
		{
			name: "zero max results",
			opts: discover.SearchOptions{
				Query:      "dotfiles",
				MinStars:   10,
				MaxResults: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiscover_DefaultOptions_AreValid(t *testing.T) {
	t.Parallel()

	opts := discover.DefaultOptions()

	assert.Equal(t, "github", opts.Source)
	assert.Equal(t, 50, opts.MaxRepos)
	assert.Equal(t, 10, opts.MinStars)
}
