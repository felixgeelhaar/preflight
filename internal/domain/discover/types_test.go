package discover

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		patternType PatternType
		valid       bool
	}{
		{PatternTypeShell, true},
		{PatternTypeEditor, true},
		{PatternTypeGit, true},
		{PatternTypeSSH, true},
		{PatternTypeTmux, true},
		{PatternTypePackageManager, true},
		{PatternType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.patternType), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.patternType.IsValid())
		})
	}
}

func TestRepo_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    Repo
		wantErr bool
	}{
		{
			name: "valid repo",
			repo: Repo{
				Owner: "user",
				Name:  "dotfiles",
				URL:   "https://github.com/user/dotfiles",
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			repo: Repo{
				Name: "dotfiles",
				URL:  "https://github.com/user/dotfiles",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			repo: Repo{
				Owner: "user",
				URL:   "https://github.com/user/dotfiles",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.repo.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepo_FullName(t *testing.T) {
	t.Parallel()

	repo := Repo{Owner: "user", Name: "dotfiles"}
	assert.Equal(t, "user/dotfiles", repo.FullName())
}

func TestPattern_Score(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  Pattern
		minScore float64
		maxScore float64
	}{
		{
			name: "popular config",
			pattern: Pattern{
				Type:        PatternTypeShell,
				Name:        "oh-my-zsh",
				Occurrences: 100,
			},
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name: "rare config",
			pattern: Pattern{
				Type:        PatternTypeShell,
				Name:        "custom-shell",
				Occurrences: 1,
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score := tt.pattern.Score()
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSuggestion_Priority(t *testing.T) {
	t.Parallel()

	high := Suggestion{
		Title:       "Install oh-my-zsh",
		Confidence:  0.9,
		Occurrences: 100,
	}
	low := Suggestion{
		Title:       "Try custom config",
		Confidence:  0.3,
		Occurrences: 5,
	}

	assert.Greater(t, high.Priority(), low.Priority())
}

func TestDiscoveryResult_TopSuggestions(t *testing.T) {
	t.Parallel()

	result := DiscoveryResult{
		Suggestions: []Suggestion{
			{Title: "Low", Confidence: 0.1, Occurrences: 1},
			{Title: "High", Confidence: 0.9, Occurrences: 100},
			{Title: "Medium", Confidence: 0.5, Occurrences: 50},
		},
	}

	top := result.TopSuggestions(2)
	assert.Len(t, top, 2)
	assert.Equal(t, "High", top[0].Title)
	assert.Equal(t, "Medium", top[1].Title)
}
