package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/ports"
)

// DiscoverSource implements discover.RepoSource using the gh CLI.
type DiscoverSource struct {
	runner ports.CommandRunner
}

// NewDiscoverSource creates a new GitHub discover source.
func NewDiscoverSource(runner ports.CommandRunner) *DiscoverSource {
	return &DiscoverSource{
		runner: runner,
	}
}

// ghSearchResult represents a repository from GitHub search.
type ghSearchResult struct {
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Stars       int    `json:"stargazerCount"`
	Language    string `json:"primaryLanguage"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// ghTreeEntry represents a file/directory entry in a repo tree.
type ghTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
}

// ghTreeResponse represents the JSON response from the tree API.
type ghTreeResponse struct {
	Tree []ghTreeEntry `json:"tree"`
}

// SearchDotfileRepos searches for dotfile repositories on GitHub.
func (s *DiscoverSource) SearchDotfileRepos(ctx context.Context, opts discover.SearchOptions) ([]discover.Repo, error) {
	// Build the search query
	query := opts.Query
	if query == "" {
		query = "dotfiles"
	}

	// Add language filter if specified
	if opts.Language != "" {
		query = fmt.Sprintf("%s language:%s", query, opts.Language)
	}

	// Add minimum stars filter
	if opts.MinStars > 0 {
		query = fmt.Sprintf("%s stars:>=%d", query, opts.MinStars)
	}

	// Use gh search to find repositories
	args := []string{
		"search", "repos",
		query,
		"--sort", "stars",
		"--order", "desc",
		"--limit", fmt.Sprintf("%d", opts.MaxResults),
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}

	result, err := s.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search repositories: %w", err)
	}
	if !result.Success() {
		return nil, fmt.Errorf("failed to search repositories: %s", result.Stderr)
	}

	// Parse JSON response
	var items []ghSearchResult
	if err := json.Unmarshal([]byte(result.Stdout), &items); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	// Convert to discover.Repo
	repos := make([]discover.Repo, 0, len(items))
	for _, item := range items {
		// Extract owner from fullName if not in owner field
		owner := item.Owner.Login
		if owner == "" && item.FullName != "" {
			parts := strings.SplitN(item.FullName, "/", 2)
			if len(parts) == 2 {
				owner = parts[0]
			}
		}

		repos = append(repos, discover.Repo{
			Owner:       owner,
			Name:        item.Name,
			URL:         item.URL,
			Description: item.Description,
			Stars:       item.Stars,
			Language:    item.Language,
		})
	}

	return repos, nil
}

// GetRepoFiles returns the list of files in a repository.
func (s *DiscoverSource) GetRepoFiles(ctx context.Context, owner, name string) ([]string, error) {
	// Use gh api to get the repository tree
	// We get the default branch first, then fetch the tree
	endpoint := fmt.Sprintf("repos/%s/%s/git/trees/HEAD?recursive=1", owner, name)

	result, err := s.runner.Run(ctx, "gh", "api", endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository files: %w", err)
	}
	if !result.Success() {
		return nil, fmt.Errorf("failed to get repository files: %s", result.Stderr)
	}

	// Parse JSON response
	var tree ghTreeResponse
	if err := json.Unmarshal([]byte(result.Stdout), &tree); err != nil {
		return nil, fmt.Errorf("failed to parse tree response: %w", err)
	}

	// Extract file paths
	files := make([]string, 0, len(tree.Tree))
	for _, entry := range tree.Tree {
		if entry.Type == "tree" {
			// Directory - add with trailing slash
			files = append(files, entry.Path+"/")
		} else {
			files = append(files, entry.Path)
		}
	}

	return files, nil
}

// Ensure DiscoverSource implements discover.RepoSource.
var _ discover.RepoSource = (*DiscoverSource)(nil)
