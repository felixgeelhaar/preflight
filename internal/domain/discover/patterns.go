package discover

import "strings"

// PatternDefinition describes a configuration pattern to detect.
type PatternDefinition struct {
	Name         string      // Name of the pattern (e.g., "oh-my-zsh")
	Type         PatternType // Category (shell, editor, etc.)
	FilePatterns []string    // File/directory patterns to match
	Description  string      // Human-readable description
	Priority     int         // Higher priority patterns are checked first
}

// Matches checks if the pattern matches any of the given files.
// Returns true if matched and the list of files that matched.
func (d PatternDefinition) Matches(files []string) (bool, []string) {
	var matched []string

	for _, file := range files {
		for _, pattern := range d.FilePatterns {
			if matchesPattern(file, pattern) {
				matched = append(matched, file)
				break
			}
		}
	}

	return len(matched) > 0, matched
}

// matchesPattern checks if a file matches a pattern.
// Patterns ending with "/" match directories and their contents.
func matchesPattern(file, pattern string) bool {
	// Directory pattern: matches if file starts with the pattern
	if strings.HasSuffix(pattern, "/") {
		return file == pattern || strings.HasPrefix(file, pattern)
	}
	// Exact match
	return file == pattern
}

// PatternMatcher detects configuration patterns in file lists.
type PatternMatcher struct {
	definitions []PatternDefinition
}

// NewPatternMatcher creates a new pattern matcher with default definitions.
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		definitions: defaultPatternDefinitions(),
	}
}

// GetPatternDefinitions returns all pattern definitions.
func (m *PatternMatcher) GetPatternDefinitions() []PatternDefinition {
	return m.definitions
}

// FilterByTypes returns a new matcher with only the specified pattern types.
func (m *PatternMatcher) FilterByTypes(types []PatternType) *PatternMatcher {
	typeSet := make(map[PatternType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	var filtered []PatternDefinition
	for _, def := range m.definitions {
		if typeSet[def.Type] {
			filtered = append(filtered, def)
		}
	}

	return &PatternMatcher{definitions: filtered}
}

// MatchFiles detects patterns in a list of files.
// Returns matched patterns sorted by priority.
func (m *PatternMatcher) MatchFiles(files []string) []Pattern {
	var patterns []Pattern
	seen := make(map[string]bool)

	// Check definitions in order (higher priority first)
	for _, def := range m.definitions {
		if seen[def.Name] {
			continue
		}

		matched, matchedFiles := def.Matches(files)
		if matched {
			patterns = append(patterns, Pattern{
				Type:    def.Type,
				Name:    def.Name,
				Files:   matchedFiles,
				Details: def.Description,
			})
			seen[def.Name] = true
		}
	}

	return patterns
}

// defaultPatternDefinitions returns the built-in pattern definitions.
func defaultPatternDefinitions() []PatternDefinition {
	return []PatternDefinition{
		// Shell - Frameworks (higher priority)
		{
			Name:         "oh-my-zsh",
			Type:         PatternTypeShell,
			FilePatterns: []string{".oh-my-zsh/"},
			Description:  "Oh My Zsh framework for managing zsh configuration",
			Priority:     100,
		},
		{
			Name:         "prezto",
			Type:         PatternTypeShell,
			FilePatterns: []string{".zprezto/", ".zpreztorc"},
			Description:  "Prezto - Instantly Awesome Zsh",
			Priority:     100,
		},

		// Shell - Shells
		{
			Name:         "zsh",
			Type:         PatternTypeShell,
			FilePatterns: []string{".zshrc", ".zshenv", ".zprofile", ".zsh/"},
			Description:  "Zsh shell configuration",
			Priority:     50,
		},
		{
			Name:         "bash",
			Type:         PatternTypeShell,
			FilePatterns: []string{".bashrc", ".bash_profile", ".bash_aliases", ".bash_logout"},
			Description:  "Bash shell configuration",
			Priority:     50,
		},
		{
			Name:         "fish",
			Type:         PatternTypeShell,
			FilePatterns: []string{".config/fish/"},
			Description:  "Fish shell configuration",
			Priority:     50,
		},

		// Shell - Prompts
		{
			Name:         "starship",
			Type:         PatternTypeShell,
			FilePatterns: []string{".config/starship.toml", "starship.toml"},
			Description:  "Starship cross-shell prompt",
			Priority:     80,
		},

		// Shell - Terminals
		{
			Name:         "alacritty",
			Type:         PatternTypeShell,
			FilePatterns: []string{".config/alacritty/", ".alacritty.yml", ".alacritty.toml"},
			Description:  "Alacritty GPU-accelerated terminal",
			Priority:     60,
		},
		{
			Name:         "kitty",
			Type:         PatternTypeShell,
			FilePatterns: []string{".config/kitty/"},
			Description:  "Kitty terminal emulator",
			Priority:     60,
		},
		{
			Name:         "wezterm",
			Type:         PatternTypeShell,
			FilePatterns: []string{".config/wezterm/", ".wezterm.lua"},
			Description:  "WezTerm terminal emulator",
			Priority:     60,
		},

		// Editors
		{
			Name:         "neovim",
			Type:         PatternTypeEditor,
			FilePatterns: []string{".config/nvim/"},
			Description:  "Neovim configuration",
			Priority:     90,
		},
		{
			Name:         "vim",
			Type:         PatternTypeEditor,
			FilePatterns: []string{".vimrc", ".vim/", ".gvimrc"},
			Description:  "Vim configuration",
			Priority:     80,
		},
		{
			Name:         "vscode",
			Type:         PatternTypeEditor,
			FilePatterns: []string{".vscode/", "Library/Application Support/Code/User/"},
			Description:  "Visual Studio Code configuration",
			Priority:     80,
		},
		{
			Name:         "emacs",
			Type:         PatternTypeEditor,
			FilePatterns: []string{".emacs", ".emacs.d/", ".doom.d/", ".spacemacs"},
			Description:  "Emacs configuration",
			Priority:     80,
		},
		{
			Name:         "helix",
			Type:         PatternTypeEditor,
			FilePatterns: []string{".config/helix/"},
			Description:  "Helix editor configuration",
			Priority:     80,
		},

		// Git
		{
			Name:         "git",
			Type:         PatternTypeGit,
			FilePatterns: []string{".gitconfig", ".gitignore_global", ".config/git/"},
			Description:  "Git version control configuration",
			Priority:     70,
		},

		// SSH
		{
			Name:         "ssh",
			Type:         PatternTypeSSH,
			FilePatterns: []string{".ssh/config", ".ssh/"},
			Description:  "SSH configuration",
			Priority:     70,
		},

		// Tmux
		{
			Name:         "tmux",
			Type:         PatternTypeTmux,
			FilePatterns: []string{".tmux.conf", ".tmux/", ".config/tmux/"},
			Description:  "Tmux terminal multiplexer configuration",
			Priority:     70,
		},
		{
			Name:         "tpm",
			Type:         PatternTypeTmux,
			FilePatterns: []string{".tmux/plugins/tpm/"},
			Description:  "Tmux Plugin Manager",
			Priority:     80,
		},

		// Package Managers
		{
			Name:         "homebrew",
			Type:         PatternTypePackageManager,
			FilePatterns: []string{"Brewfile"},
			Description:  "Homebrew package manager bundle",
			Priority:     70,
		},
		{
			Name:         "nix",
			Type:         PatternTypePackageManager,
			FilePatterns: []string{".config/nix/", "flake.nix", "home.nix", ".config/home-manager/"},
			Description:  "Nix/NixOS configuration",
			Priority:     70,
		},
		{
			Name:         "asdf",
			Type:         PatternTypePackageManager,
			FilePatterns: []string{".tool-versions", ".asdfrc"},
			Description:  "asdf version manager",
			Priority:     60,
		},
		{
			Name:         "mise",
			Type:         PatternTypePackageManager,
			FilePatterns: []string{".mise.toml", ".config/mise/"},
			Description:  "mise (rtx) runtime executor",
			Priority:     60,
		},
	}
}
