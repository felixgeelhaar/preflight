package discover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternMatcher_MatchFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		files        []string
		wantPatterns []string
		wantTypes    []PatternType
	}{
		{
			name: "zsh configuration",
			files: []string{
				".zshrc",
				".zshenv",
				".zsh/",
			},
			wantPatterns: []string{"zsh"},
			wantTypes:    []PatternType{PatternTypeShell},
		},
		{
			name: "bash configuration",
			files: []string{
				".bashrc",
				".bash_profile",
				".bash_aliases",
			},
			wantPatterns: []string{"bash"},
			wantTypes:    []PatternType{PatternTypeShell},
		},
		{
			name: "oh-my-zsh",
			files: []string{
				".zshrc",
				".oh-my-zsh/",
				".oh-my-zsh/custom/",
			},
			wantPatterns: []string{"oh-my-zsh", "zsh"},
			wantTypes:    []PatternType{PatternTypeShell, PatternTypeShell},
		},
		{
			name: "neovim configuration",
			files: []string{
				".config/nvim/",
				".config/nvim/init.lua",
				".config/nvim/lua/",
			},
			wantPatterns: []string{"neovim"},
			wantTypes:    []PatternType{PatternTypeEditor},
		},
		{
			name: "vim configuration",
			files: []string{
				".vimrc",
				".vim/",
			},
			wantPatterns: []string{"vim"},
			wantTypes:    []PatternType{PatternTypeEditor},
		},
		{
			name: "vscode configuration",
			files: []string{
				".vscode/",
				".vscode/settings.json",
				".vscode/extensions.json",
			},
			wantPatterns: []string{"vscode"},
			wantTypes:    []PatternType{PatternTypeEditor},
		},
		{
			name: "git configuration",
			files: []string{
				".gitconfig",
				".gitignore_global",
			},
			wantPatterns: []string{"git"},
			wantTypes:    []PatternType{PatternTypeGit},
		},
		{
			name: "ssh configuration",
			files: []string{
				".ssh/",
				".ssh/config",
			},
			wantPatterns: []string{"ssh"},
			wantTypes:    []PatternType{PatternTypeSSH},
		},
		{
			name: "tmux configuration",
			files: []string{
				".tmux.conf",
				".tmux/",
			},
			wantPatterns: []string{"tmux"},
			wantTypes:    []PatternType{PatternTypeTmux},
		},
		{
			name: "homebrew bundle",
			files: []string{
				"Brewfile",
				"Brewfile.lock.json",
			},
			wantPatterns: []string{"homebrew"},
			wantTypes:    []PatternType{PatternTypePackageManager},
		},
		{
			name: "multiple patterns",
			files: []string{
				".zshrc",
				".config/nvim/init.lua",
				".gitconfig",
				".tmux.conf",
			},
			wantPatterns: []string{"zsh", "neovim", "git", "tmux"},
			wantTypes:    []PatternType{PatternTypeShell, PatternTypeEditor, PatternTypeGit, PatternTypeTmux},
		},
		{
			name:         "no matching files",
			files:        []string{"README.md", "LICENSE", "install.sh"},
			wantPatterns: nil,
			wantTypes:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matcher := NewPatternMatcher()
			patterns := matcher.MatchFiles(tt.files)

			if tt.wantPatterns == nil {
				assert.Empty(t, patterns)
				return
			}

			require.Len(t, patterns, len(tt.wantPatterns))
			for i, p := range patterns {
				assert.Equal(t, tt.wantPatterns[i], p.Name)
				assert.Equal(t, tt.wantTypes[i], p.Type)
			}
		})
	}
}

func TestPatternMatcher_GetPatternDefinitions(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()
	defs := matcher.GetPatternDefinitions()

	// Ensure we have definitions for all pattern types
	typesSeen := make(map[PatternType]bool)
	for _, def := range defs {
		typesSeen[def.Type] = true
		assert.NotEmpty(t, def.Name)
		assert.NotEmpty(t, def.FilePatterns)
	}

	assert.True(t, typesSeen[PatternTypeShell])
	assert.True(t, typesSeen[PatternTypeEditor])
	assert.True(t, typesSeen[PatternTypeGit])
	assert.True(t, typesSeen[PatternTypeSSH])
	assert.True(t, typesSeen[PatternTypeTmux])
	assert.True(t, typesSeen[PatternTypePackageManager])
}

func TestPatternDefinition_Matches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		definition PatternDefinition
		files      []string
		wantMatch  bool
		wantFiles  []string
	}{
		{
			name: "exact match",
			definition: PatternDefinition{
				Name:         "zsh",
				Type:         PatternTypeShell,
				FilePatterns: []string{".zshrc", ".zshenv"},
			},
			files:     []string{".zshrc", "README.md"},
			wantMatch: true,
			wantFiles: []string{".zshrc"},
		},
		{
			name: "directory match",
			definition: PatternDefinition{
				Name:         "oh-my-zsh",
				Type:         PatternTypeShell,
				FilePatterns: []string{".oh-my-zsh/"},
			},
			files:     []string{".oh-my-zsh/", ".oh-my-zsh/custom/"},
			wantMatch: true,
			wantFiles: []string{".oh-my-zsh/", ".oh-my-zsh/custom/"},
		},
		{
			name: "prefix match",
			definition: PatternDefinition{
				Name:         "neovim",
				Type:         PatternTypeEditor,
				FilePatterns: []string{".config/nvim/"},
			},
			files:     []string{".config/nvim/init.lua", ".config/nvim/lua/plugins.lua"},
			wantMatch: true,
			wantFiles: []string{".config/nvim/init.lua", ".config/nvim/lua/plugins.lua"},
		},
		{
			name: "no match",
			definition: PatternDefinition{
				Name:         "zsh",
				Type:         PatternTypeShell,
				FilePatterns: []string{".zshrc"},
			},
			files:     []string{".bashrc", "README.md"},
			wantMatch: false,
			wantFiles: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matched, matchedFiles := tt.definition.Matches(tt.files)
			assert.Equal(t, tt.wantMatch, matched)
			assert.Equal(t, tt.wantFiles, matchedFiles)
		})
	}
}

func TestPatternMatcher_FilterByTypes(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()

	// Filter to shell only
	filtered := matcher.FilterByTypes([]PatternType{PatternTypeShell})
	for _, def := range filtered.GetPatternDefinitions() {
		assert.Equal(t, PatternTypeShell, def.Type)
	}

	// Filter to multiple types
	filtered = matcher.FilterByTypes([]PatternType{PatternTypeEditor, PatternTypeGit})
	for _, def := range filtered.GetPatternDefinitions() {
		assert.True(t, def.Type == PatternTypeEditor || def.Type == PatternTypeGit)
	}
}

func TestStarshipPattern(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()
	patterns := matcher.MatchFiles([]string{".config/starship.toml"})

	require.Len(t, patterns, 1)
	assert.Equal(t, "starship", patterns[0].Name)
	assert.Equal(t, PatternTypeShell, patterns[0].Type)
}

func TestFishShellPattern(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()
	patterns := matcher.MatchFiles([]string{".config/fish/config.fish", ".config/fish/functions/"})

	require.Len(t, patterns, 1)
	assert.Equal(t, "fish", patterns[0].Name)
	assert.Equal(t, PatternTypeShell, patterns[0].Type)
}

func TestAlacrittyPattern(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()
	patterns := matcher.MatchFiles([]string{".config/alacritty/alacritty.toml"})

	require.Len(t, patterns, 1)
	assert.Equal(t, "alacritty", patterns[0].Name)
	assert.Equal(t, PatternTypeShell, patterns[0].Type)
}

func TestKittyPattern(t *testing.T) {
	t.Parallel()

	matcher := NewPatternMatcher()
	patterns := matcher.MatchFiles([]string{".config/kitty/kitty.conf"})

	require.Len(t, patterns, 1)
	assert.Equal(t, "kitty", patterns[0].Name)
	assert.Equal(t, PatternTypeShell, patterns[0].Type)
}
