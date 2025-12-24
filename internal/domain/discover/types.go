// Package discover provides dotfile repository discovery and analysis.
package discover

import (
	"fmt"
	"sort"
)

// PatternType represents the type of configuration pattern detected.
type PatternType string

// Pattern types for configuration detection.
const (
	PatternTypeShell          PatternType = "shell"
	PatternTypeEditor         PatternType = "editor"
	PatternTypeGit            PatternType = "git"
	PatternTypeSSH            PatternType = "ssh"
	PatternTypeTmux           PatternType = "tmux"
	PatternTypePackageManager PatternType = "package_manager"
)

// validPatternTypes is the set of valid pattern types.
var validPatternTypes = map[PatternType]bool{
	PatternTypeShell:          true,
	PatternTypeEditor:         true,
	PatternTypeGit:            true,
	PatternTypeSSH:            true,
	PatternTypeTmux:           true,
	PatternTypePackageManager: true,
}

// IsValid returns true if the pattern type is valid.
func (p PatternType) IsValid() bool {
	return validPatternTypes[p]
}

// Repo represents a discovered dotfiles repository.
type Repo struct {
	Owner       string // Repository owner (user or org)
	Name        string // Repository name
	URL         string // Full URL to the repository
	Description string // Repository description
	Stars       int    // Star count
	Language    string // Primary language
}

// Validate checks if the repo has required fields.
func (r Repo) Validate() error {
	if r.Owner == "" {
		return fmt.Errorf("owner is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// FullName returns the full repository name (owner/name).
func (r Repo) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// Pattern represents a detected configuration pattern in a repository.
type Pattern struct {
	Type        PatternType // Type of pattern (shell, editor, etc.)
	Name        string      // Name of the pattern (e.g., "oh-my-zsh", "neovim")
	Files       []string    // Files that matched this pattern
	Occurrences int         // Number of repos with this pattern
	Details     string      // Additional details about the pattern
}

// Score calculates a relevance score for this pattern.
// Score ranges from 0.0 to 1.0 based on popularity.
func (p Pattern) Score() float64 {
	// Score based on occurrences using a sigmoid-like curve
	// 1 occurrence = ~0.1, 10 = ~0.4, 50 = ~0.7, 100 = ~0.85, 500+ = ~1.0
	if p.Occurrences <= 0 {
		return 0.0
	}

	// Use a formula that gives higher scores for moderate popularity
	// score = 1 - (1 / (1 + occurrences/50))
	score := 1.0 - (1.0 / (1.0 + float64(p.Occurrences)/50.0))
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// Suggestion represents a configuration suggestion for the user.
type Suggestion struct {
	Title         string      // Short title for the suggestion
	Description   string      // Detailed description
	PatternType   PatternType // Type of configuration
	ConfigSnippet string      // Sample YAML config to add
	Confidence    float64     // Confidence score (0.0 to 1.0)
	Occurrences   int         // How many repos use this
	Links         []string    // Documentation links
	Reasons       []string    // Why this is suggested
}

// Priority calculates the suggestion priority based on confidence and popularity.
func (s Suggestion) Priority() float64 {
	return s.Confidence * (1.0 + float64(s.Occurrences)/100.0)
}

// DiscoveryResult contains the results of analyzing dotfile repositories.
type DiscoveryResult struct {
	ReposAnalyzed int          // Number of repos analyzed
	Patterns      []Pattern    // Detected patterns across repos
	Suggestions   []Suggestion // Generated suggestions
	Source        string       // Source of discovery (e.g., "github")
}

// TopSuggestions returns the top n suggestions sorted by priority.
func (r DiscoveryResult) TopSuggestions(n int) []Suggestion {
	suggestions := make([]Suggestion, len(r.Suggestions))
	copy(suggestions, r.Suggestions)

	// Sort by priority (descending)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority() > suggestions[j].Priority()
	})

	if n > len(suggestions) {
		n = len(suggestions)
	}
	return suggestions[:n]
}

// DiscoveryOptions configures the discovery process.
type DiscoveryOptions struct {
	Source       string        // Source to search (e.g., "github")
	Language     string        // Filter by primary language
	MinStars     int           // Minimum star count
	MaxRepos     int           // Maximum repos to analyze
	PatternTypes []PatternType // Types of patterns to look for
}

// DefaultOptions returns the default discovery options.
func DefaultOptions() DiscoveryOptions {
	return DiscoveryOptions{
		Source:   "github",
		MinStars: 10,
		MaxRepos: 50,
		PatternTypes: []PatternType{
			PatternTypeShell,
			PatternTypeEditor,
			PatternTypeGit,
			PatternTypeSSH,
			PatternTypeTmux,
			PatternTypePackageManager,
		},
	}
}
