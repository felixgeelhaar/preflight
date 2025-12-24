package discover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepoSource is a mock implementation of RepoSource for testing.
type mockRepoSource struct {
	repos map[string][]string // repo full name -> files
}

func newMockRepoSource() *mockRepoSource {
	return &mockRepoSource{
		repos: make(map[string][]string),
	}
}

func (m *mockRepoSource) AddRepo(owner, name string, files []string) {
	fullName := owner + "/" + name
	m.repos[fullName] = files
}

func (m *mockRepoSource) SearchDotfileRepos(_ context.Context, opts SearchOptions) ([]Repo, error) {
	repos := make([]Repo, 0, len(m.repos))
	i := 0
	for fullName := range m.repos {
		if i >= opts.MaxResults {
			break
		}
		parts := splitFullName(fullName)
		repos = append(repos, Repo{
			Owner: parts[0],
			Name:  parts[1],
			URL:   "https://github.com/" + fullName,
			Stars: 100 - i*10, // Decreasing stars
		})
		i++
	}
	return repos, nil
}

func (m *mockRepoSource) GetRepoFiles(_ context.Context, owner, name string) ([]string, error) {
	fullName := owner + "/" + name
	if files, ok := m.repos[fullName]; ok {
		return files, nil
	}
	return nil, nil
}

func splitFullName(fullName string) []string {
	for i, c := range fullName {
		if c == '/' {
			return []string{fullName[:i], fullName[i+1:]}
		}
	}
	return []string{fullName, ""}
}

func TestAnalyzer_Analyze_SingleRepo(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	source.AddRepo("user", "dotfiles", []string{
		".zshrc",
		".config/nvim/init.lua",
		".gitconfig",
	})

	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, DefaultOptions())

	require.NoError(t, err)
	assert.Equal(t, 1, result.ReposAnalyzed)
	assert.NotEmpty(t, result.Patterns)

	// Should detect zsh, neovim, and git patterns
	patternNames := make(map[string]bool)
	for _, p := range result.Patterns {
		patternNames[p.Name] = true
	}
	assert.True(t, patternNames["zsh"])
	assert.True(t, patternNames["neovim"])
	assert.True(t, patternNames["git"])
}

func TestAnalyzer_Analyze_MultipleRepos(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	source.AddRepo("user1", "dotfiles", []string{".zshrc", ".oh-my-zsh/"})
	source.AddRepo("user2", "dotfiles", []string{".zshrc", ".oh-my-zsh/"})
	source.AddRepo("user3", "dotfiles", []string{".bashrc"})

	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, DefaultOptions())

	require.NoError(t, err)
	assert.Equal(t, 3, result.ReposAnalyzed)

	// Find oh-my-zsh pattern and check occurrences
	var ohmyzsh *Pattern
	for i := range result.Patterns {
		if result.Patterns[i].Name == "oh-my-zsh" {
			ohmyzsh = &result.Patterns[i]
			break
		}
	}
	require.NotNil(t, ohmyzsh)
	assert.Equal(t, 2, ohmyzsh.Occurrences) // Found in 2 repos
}

func TestAnalyzer_Analyze_PatternTypeFilter(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	source.AddRepo("user", "dotfiles", []string{
		".zshrc",
		".config/nvim/init.lua",
		".gitconfig",
		".tmux.conf",
	})

	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	opts := DefaultOptions()
	opts.PatternTypes = []PatternType{PatternTypeShell}

	result, err := analyzer.Analyze(ctx, opts)

	require.NoError(t, err)
	// Should only detect shell patterns
	for _, p := range result.Patterns {
		assert.Equal(t, PatternTypeShell, p.Type)
	}
}

func TestAnalyzer_Analyze_MaxRepos(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	for i := 0; i < 10; i++ {
		source.AddRepo("user", "dotfiles"+string(rune('0'+i)), []string{".zshrc"})
	}

	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	opts := DefaultOptions()
	opts.MaxRepos = 3

	result, err := analyzer.Analyze(ctx, opts)

	require.NoError(t, err)
	assert.Equal(t, 3, result.ReposAnalyzed)
}

func TestAnalyzer_Analyze_EmptySource(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, DefaultOptions())

	require.NoError(t, err)
	assert.Equal(t, 0, result.ReposAnalyzed)
	assert.Empty(t, result.Patterns)
}

func TestAnalyzer_Analyze_NoPatterns(t *testing.T) {
	t.Parallel()

	source := newMockRepoSource()
	source.AddRepo("user", "dotfiles", []string{
		"README.md",
		"LICENSE",
		"install.sh",
	})

	analyzer := NewAnalyzer(source)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, DefaultOptions())

	require.NoError(t, err)
	assert.Equal(t, 1, result.ReposAnalyzed)
	assert.Empty(t, result.Patterns)
}

func TestAnalyzer_AggregatePatterns(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)

	repoPatterns := [][]Pattern{
		{
			{Name: "zsh", Type: PatternTypeShell, Files: []string{".zshrc"}},
			{Name: "neovim", Type: PatternTypeEditor, Files: []string{".config/nvim/"}},
		},
		{
			{Name: "zsh", Type: PatternTypeShell, Files: []string{".zshrc", ".zshenv"}},
			{Name: "git", Type: PatternTypeGit, Files: []string{".gitconfig"}},
		},
		{
			{Name: "zsh", Type: PatternTypeShell, Files: []string{".zshrc"}},
		},
	}

	aggregated := analyzer.aggregatePatterns(repoPatterns)

	// zsh should have 3 occurrences
	var zsh *Pattern
	for i := range aggregated {
		if aggregated[i].Name == "zsh" {
			zsh = &aggregated[i]
			break
		}
	}
	require.NotNil(t, zsh)
	assert.Equal(t, 3, zsh.Occurrences)

	// neovim should have 1 occurrence
	var neovim *Pattern
	for i := range aggregated {
		if aggregated[i].Name == "neovim" {
			neovim = &aggregated[i]
			break
		}
	}
	require.NotNil(t, neovim)
	assert.Equal(t, 1, neovim.Occurrences)
}

func TestAnalyzer_SortPatternsByOccurrence(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)

	patterns := []Pattern{
		{Name: "rare", Occurrences: 1},
		{Name: "popular", Occurrences: 100},
		{Name: "medium", Occurrences: 50},
	}

	sorted := analyzer.sortPatternsByOccurrence(patterns)

	require.Len(t, sorted, 3)
	assert.Equal(t, "popular", sorted[0].Name)
	assert.Equal(t, "medium", sorted[1].Name)
	assert.Equal(t, "rare", sorted[2].Name)
}

func TestSearchOptions_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    SearchOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: SearchOptions{
				Query:      "dotfiles",
				MinStars:   10,
				MaxResults: 50,
			},
			wantErr: false,
		},
		{
			name: "negative min stars",
			opts: SearchOptions{
				Query:      "dotfiles",
				MinStars:   -1,
				MaxResults: 50,
			},
			wantErr: true,
		},
		{
			name: "zero max results",
			opts: SearchOptions{
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
