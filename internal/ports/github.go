// Package ports defines interfaces for external dependencies.
package ports

import "context"

// GitHubCreateOptions contains options for creating a GitHub repository.
type GitHubCreateOptions struct {
	Name        string
	Description string
	Private     bool
}

// GitHubRepoInfo contains information about a GitHub repository.
type GitHubRepoInfo struct {
	Name     string
	URL      string
	CloneURL string
	SSHURL   string
	Owner    string
}

// GitHubPort defines the interface for GitHub operations.
type GitHubPort interface {
	// IsAuthenticated checks if the user is authenticated with GitHub.
	IsAuthenticated(ctx context.Context) (bool, error)

	// CreateRepository creates a new GitHub repository.
	CreateRepository(ctx context.Context, opts GitHubCreateOptions) (*GitHubRepoInfo, error)

	// SetRemote adds or updates the origin remote for a local repository.
	SetRemote(ctx context.Context, path, url string) error

	// GetAuthenticatedUser returns the username of the authenticated user.
	GetAuthenticatedUser(ctx context.Context) (string, error)
}
