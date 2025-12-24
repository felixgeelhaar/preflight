package discover

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SuggestionGenerator generates configuration suggestions from detected patterns.
type SuggestionGenerator struct {
	snippets     map[string]string
	descriptions map[string]string
	docLinks     map[string][]string
}

// NewSuggestionGenerator creates a new suggestion generator.
func NewSuggestionGenerator() *SuggestionGenerator {
	return &SuggestionGenerator{
		snippets:     defaultConfigSnippets(),
		descriptions: defaultDescriptions(),
		docLinks:     defaultDocLinks(),
	}
}

// Generate creates suggestions from detected patterns.
func (g *SuggestionGenerator) Generate(patterns []Pattern, totalRepos int) []Suggestion {
	if len(patterns) == 0 {
		return nil
	}

	suggestions := make([]Suggestion, 0, len(patterns))
	for _, p := range patterns {
		s := Suggestion{
			Title:         g.getTitle(p.Name),
			Description:   g.getDescription(p.Name),
			PatternType:   p.Type,
			ConfigSnippet: g.getConfigSnippet(p.Name, p.Type),
			Confidence:    g.calculateConfidence(p.Occurrences, totalRepos),
			Occurrences:   p.Occurrences,
			Links:         g.getDocLinks(p.Name),
			Reasons:       g.getReasons(p, totalRepos),
		}
		suggestions = append(suggestions, s)
	}

	return suggestions
}

// getTitle returns a human-readable title for the pattern.
func (g *SuggestionGenerator) getTitle(patternName string) string {
	titles := map[string]string{
		"oh-my-zsh": "Oh My Zsh Framework",
		"prezto":    "Prezto Framework",
		"zsh":       "Zsh Shell",
		"bash":      "Bash Shell",
		"fish":      "Fish Shell",
		"starship":  "Starship Prompt",
		"alacritty": "Alacritty Terminal",
		"kitty":     "Kitty Terminal",
		"wezterm":   "WezTerm Terminal",
		"neovim":    "Neovim Editor",
		"vim":       "Vim Editor",
		"vscode":    "Visual Studio Code",
		"emacs":     "Emacs Editor",
		"helix":     "Helix Editor",
		"git":       "Git Configuration",
		"ssh":       "SSH Configuration",
		"tmux":      "Tmux Multiplexer",
		"tpm":       "Tmux Plugin Manager",
		"homebrew":  "Homebrew Bundle",
		"nix":       "Nix Package Manager",
		"asdf":      "asdf Version Manager",
		"mise":      "mise Runtime Executor",
	}

	if title, ok := titles[patternName]; ok {
		return title
	}
	caser := cases.Title(language.English)
	return caser.String(patternName)
}

// getDescription returns a description for the pattern.
func (g *SuggestionGenerator) getDescription(patternName string) string {
	if desc, ok := g.descriptions[patternName]; ok {
		return desc
	}
	return fmt.Sprintf("Configuration for %s", patternName)
}

// getConfigSnippet returns a sample configuration snippet.
func (g *SuggestionGenerator) getConfigSnippet(patternName string, patternType PatternType) string {
	if snippet, ok := g.snippets[patternName]; ok {
		return snippet
	}

	// Generate a generic snippet based on pattern type
	switch patternType {
	case PatternTypeShell:
		return fmt.Sprintf("shell:\n  framework: %s", patternName)
	case PatternTypeEditor:
		return fmt.Sprintf("editor:\n  preset: %s", patternName)
	case PatternTypeGit:
		return "git:\n  user:\n    name: \"Your Name\"\n    email: \"your@email.com\""
	case PatternTypeSSH:
		return "ssh:\n  config: true"
	case PatternTypeTmux:
		return fmt.Sprintf("shell:\n  tmux:\n    config: %s", patternName)
	case PatternTypePackageManager:
		return "brew:\n  formulae:\n    - git\n    - ripgrep"
	default:
		return fmt.Sprintf("# Add %s configuration", patternName)
	}
}

// getDocLinks returns documentation links for the pattern.
func (g *SuggestionGenerator) getDocLinks(patternName string) []string {
	if links, ok := g.docLinks[patternName]; ok {
		return links
	}
	return nil
}

// calculateConfidence calculates a confidence score based on adoption rate.
func (g *SuggestionGenerator) calculateConfidence(occurrences, totalRepos int) float64 {
	if totalRepos <= 0 || occurrences <= 0 {
		return 0.0
	}

	// Calculate adoption rate
	rate := float64(occurrences) / float64(totalRepos)

	// Apply a slight boost for higher absolute numbers
	// to account for the fact that popular tools are likely good choices
	boost := 1.0
	if occurrences >= 50 {
		boost = 1.1
	}

	confidence := rate * boost
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// getReasons returns reasons why this pattern is suggested.
func (g *SuggestionGenerator) getReasons(p Pattern, totalRepos int) []string {
	var reasons []string

	// Popularity reason
	percentage := 0
	if totalRepos > 0 {
		percentage = (p.Occurrences * 100) / totalRepos
	}

	switch {
	case percentage >= 50:
		reasons = append(reasons, fmt.Sprintf("Used by %d%% of analyzed repositories", percentage))
	case percentage >= 20:
		reasons = append(reasons, fmt.Sprintf("Popular choice found in %d%% of repositories", percentage))
	case p.Occurrences >= 10:
		reasons = append(reasons, fmt.Sprintf("Found in %d repositories", p.Occurrences))
	}

	// Type-specific reasons
	switch p.Type {
	case PatternTypeShell:
		reasons = append(reasons, "Enhances shell productivity and customization")
	case PatternTypeEditor:
		reasons = append(reasons, "Provides consistent editing experience across machines")
	case PatternTypeGit:
		reasons = append(reasons, "Standardizes version control configuration")
	case PatternTypeSSH:
		reasons = append(reasons, "Simplifies remote connection management")
	case PatternTypeTmux:
		reasons = append(reasons, "Enables persistent terminal sessions")
	case PatternTypePackageManager:
		reasons = append(reasons, "Automates software installation")
	}

	return reasons
}

// defaultConfigSnippets returns the default configuration snippets.
func defaultConfigSnippets() map[string]string {
	return map[string]string{
		"oh-my-zsh": `shell:
  zsh:
    framework: oh-my-zsh
    plugins:
      - git
      - docker
      - kubectl`,
		"prezto": `shell:
  zsh:
    framework: prezto
    modules:
      - environment
      - terminal
      - editor
      - history`,
		"zsh": `shell:
  zsh:
    config: true`,
		"bash": `shell:
  bash:
    config: true`,
		"fish": `shell:
  fish:
    config: true`,
		"starship": `shell:
  prompt: starship
  starship:
    preset: nerd-font-symbols`,
		"neovim": `editor:
  preset: neovim
  neovim:
    config: lazy`,
		"vim": `editor:
  preset: vim`,
		"vscode": `editor:
  preset: vscode
  vscode:
    extensions:
      - ms-vscode.go
      - vscodevim.vim`,
		"git": `git:
  user:
    name: "Your Name"
    email: "your@email.com"
  core:
    editor: nvim`,
		"ssh": `ssh:
  config: true
  hosts: []`,
		"tmux": `shell:
  tmux:
    config: true
    plugins:
      - tpm`,
		"homebrew": `brew:
  taps:
    - homebrew/cask-fonts
  formulae:
    - git
    - ripgrep
    - fzf`,
		"nix": `# Nix configuration
# home-manager recommended for user-level packages`,
		"asdf": `runtime:
  manager: asdf
  tools:
    node: lts
    python: 3.11`,
		"mise": `runtime:
  manager: mise
  tools:
    node: lts
    go: latest`,
	}
}

// defaultDescriptions returns the default pattern descriptions.
func defaultDescriptions() map[string]string {
	return map[string]string{
		"oh-my-zsh": "Community-driven framework for managing your zsh configuration with themes and plugins",
		"prezto":    "Instantly Awesome Zsh - configuration framework with sensible defaults",
		"zsh":       "Z shell - powerful shell with advanced features and scripting capabilities",
		"bash":      "Bourne Again Shell - the GNU Project's shell",
		"fish":      "Friendly Interactive Shell with autosuggestions and syntax highlighting",
		"starship":  "Minimal, blazing-fast, and infinitely customizable cross-shell prompt",
		"alacritty": "GPU-accelerated terminal emulator with sensible defaults",
		"kitty":     "Fast, feature-rich, GPU based terminal emulator",
		"wezterm":   "GPU-accelerated terminal emulator with multiplexing",
		"neovim":    "Hyperextensible Vim-based text editor with modern plugin ecosystem",
		"vim":       "Highly configurable text editor built for efficiency",
		"vscode":    "Microsoft's Visual Studio Code - popular extensible editor",
		"emacs":     "Extensible, customizable text editor and computing environment",
		"helix":     "Post-modern modal text editor with built-in LSP support",
		"git":       "Distributed version control system configuration",
		"ssh":       "Secure Shell client configuration for remote connections",
		"tmux":      "Terminal multiplexer for managing multiple sessions",
		"tpm":       "Tmux Plugin Manager for easy plugin installation",
		"homebrew":  "The package manager for macOS - manages formulae and casks",
		"nix":       "Purely functional package manager with reproducible builds",
		"asdf":      "Multiple runtime version manager (node, python, etc.)",
		"mise":      "Polyglot runtime executor - successor to rtx/asdf",
	}
}

// defaultDocLinks returns the default documentation links.
func defaultDocLinks() map[string][]string {
	return map[string][]string{
		"oh-my-zsh": {"https://ohmyz.sh/", "https://github.com/ohmyzsh/ohmyzsh"},
		"prezto":    {"https://github.com/sorin-ionescu/prezto"},
		"zsh":       {"https://zsh.sourceforge.io/Doc/"},
		"starship":  {"https://starship.rs/"},
		"alacritty": {"https://alacritty.org/", "https://github.com/alacritty/alacritty"},
		"kitty":     {"https://sw.kovidgoyal.net/kitty/"},
		"wezterm":   {"https://wezfurlong.org/wezterm/"},
		"neovim":    {"https://neovim.io/", "https://github.com/neovim/neovim"},
		"vim":       {"https://www.vim.org/docs.php"},
		"vscode":    {"https://code.visualstudio.com/docs"},
		"emacs":     {"https://www.gnu.org/software/emacs/"},
		"helix":     {"https://helix-editor.com/", "https://docs.helix-editor.com/"},
		"tmux":      {"https://github.com/tmux/tmux/wiki"},
		"homebrew":  {"https://brew.sh/", "https://docs.brew.sh/"},
		"nix":       {"https://nixos.org/", "https://nix.dev/"},
		"asdf":      {"https://asdf-vm.com/"},
		"mise":      {"https://mise.jdx.dev/"},
	}
}
