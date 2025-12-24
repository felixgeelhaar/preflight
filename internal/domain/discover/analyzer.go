package discover

import (
	"context"
	"fmt"
	"sort"
)

// RepoSource is the interface for fetching dotfile repositories.
type RepoSource interface {
	// SearchDotfileRepos searches for dotfile repositories.
	SearchDotfileRepos(ctx context.Context, opts SearchOptions) ([]Repo, error)

	// GetRepoFiles returns the list of files in a repository.
	GetRepoFiles(ctx context.Context, owner, name string) ([]string, error)
}

// SearchOptions configures repository search.
type SearchOptions struct {
	Query      string // Search query
	MinStars   int    // Minimum star count
	MaxResults int    // Maximum number of results
	Language   string // Filter by primary language
}

// Validate checks if the search options are valid.
func (o SearchOptions) Validate() error {
	if o.MinStars < 0 {
		return fmt.Errorf("min stars must be non-negative")
	}
	if o.MaxResults <= 0 {
		return fmt.Errorf("max results must be positive")
	}
	return nil
}

// DefaultSearchOptions returns default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Query:      "dotfiles",
		MinStars:   10,
		MaxResults: 50,
	}
}

// Analyzer analyzes dotfile repositories to detect configuration patterns.
type Analyzer struct {
	source  RepoSource
	matcher *PatternMatcher
}

// NewAnalyzer creates a new analyzer with the given repository source.
func NewAnalyzer(source RepoSource) *Analyzer {
	return &Analyzer{
		source:  source,
		matcher: NewPatternMatcher(),
	}
}

// Analyze fetches and analyzes dotfile repositories.
func (a *Analyzer) Analyze(ctx context.Context, opts DiscoveryOptions) (*DiscoveryResult, error) {
	// Build search options from discovery options
	searchOpts := SearchOptions{
		Query:      "dotfiles",
		MinStars:   opts.MinStars,
		MaxResults: opts.MaxRepos,
		Language:   opts.Language,
	}

	// Get matcher filtered by requested pattern types
	matcher := a.matcher
	if len(opts.PatternTypes) > 0 {
		matcher = matcher.FilterByTypes(opts.PatternTypes)
	}

	// Search for repositories
	repos, err := a.source.SearchDotfileRepos(ctx, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("searching repositories: %w", err)
	}

	// Limit to MaxRepos
	if len(repos) > opts.MaxRepos {
		repos = repos[:opts.MaxRepos]
	}

	// Analyze each repository
	var allPatterns [][]Pattern
	for _, repo := range repos {
		files, err := a.source.GetRepoFiles(ctx, repo.Owner, repo.Name)
		if err != nil {
			// Log error but continue with other repos
			continue
		}

		patterns := matcher.MatchFiles(files)
		if len(patterns) > 0 {
			allPatterns = append(allPatterns, patterns)
		}
	}

	// Aggregate patterns across all repos
	aggregated := a.aggregatePatterns(allPatterns)
	sorted := a.sortPatternsByOccurrence(aggregated)

	return &DiscoveryResult{
		ReposAnalyzed: len(repos),
		Patterns:      sorted,
		Source:        opts.Source,
	}, nil
}

// aggregatePatterns combines patterns from multiple repos, counting occurrences.
func (a *Analyzer) aggregatePatterns(repoPatterns [][]Pattern) []Pattern {
	counts := make(map[string]*Pattern)

	for _, patterns := range repoPatterns {
		// Track which patterns we've seen in this repo
		seenInRepo := make(map[string]bool)

		for _, p := range patterns {
			if seenInRepo[p.Name] {
				continue
			}
			seenInRepo[p.Name] = true

			if existing, ok := counts[p.Name]; ok {
				existing.Occurrences++
				// Merge files
				existing.Files = append(existing.Files, p.Files...)
			} else {
				newPattern := p
				newPattern.Occurrences = 1
				counts[p.Name] = &newPattern
			}
		}
	}

	result := make([]Pattern, 0, len(counts))
	for _, p := range counts {
		result = append(result, *p)
	}
	return result
}

// sortPatternsByOccurrence sorts patterns by occurrence count (descending).
func (a *Analyzer) sortPatternsByOccurrence(patterns []Pattern) []Pattern {
	sorted := make([]Pattern, len(patterns))
	copy(sorted, patterns)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Occurrences > sorted[j].Occurrences
	})

	return sorted
}
