// Package github provides a GitHub adapter using the gh CLI.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// Client implements ports.GitHubPort using the gh CLI.
type Client struct {
	runner ports.CommandRunner
}

// NewClient creates a new GitHub client.
func NewClient(runner ports.CommandRunner) *Client {
	return &Client{
		runner: runner,
	}
}

// IsAuthenticated checks if the user is authenticated with GitHub.
func (c *Client) IsAuthenticated(ctx context.Context) (bool, error) {
	result, err := c.runner.Run(ctx, "gh", "auth", "status")
	if err != nil {
		return false, fmt.Errorf("failed to check auth status: %w", err)
	}
	return result.Success(), nil
}

// GetAuthenticatedUser returns the username of the authenticated user.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (string, error) {
	result, err := c.runner.Run(ctx, "gh", "api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if !result.Success() {
		return "", fmt.Errorf("failed to get user: %s", result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ghRepoResponse represents the JSON response from gh repo create.
type ghRepoResponse struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// CreateRepository creates a new GitHub repository.
func (c *Client) CreateRepository(ctx context.Context, opts ports.GitHubCreateOptions) (*ports.GitHubRepoInfo, error) {
	args := []string{"repo", "create", opts.Name, "--confirm"}

	if opts.Private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}

	if opts.Description != "" {
		args = append(args, "--description", opts.Description)
	}

	// Add JSON output for parsing
	args = append(args, "--json", "name,url,cloneUrl,sshUrl,owner")

	result, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	if !result.Success() {
		return nil, fmt.Errorf("failed to create repository: %s", result.Stderr)
	}

	// Parse JSON response
	var resp ghRepoResponse
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		// If JSON parsing fails, try to extract info from the output
		// gh repo create sometimes just outputs the URL
		url := strings.TrimSpace(result.Stdout)
		return &ports.GitHubRepoInfo{ //nolint:nilerr // Fallback to URL parsing when JSON fails
			Name:     opts.Name,
			URL:      url,
			CloneURL: url + ".git",
			SSHURL:   fmt.Sprintf("git@github.com:%s.git", opts.Name),
		}, nil
	}

	return &ports.GitHubRepoInfo{
		Name:     resp.Name,
		URL:      resp.URL,
		CloneURL: resp.CloneURL,
		SSHURL:   resp.SSHURL,
		Owner:    resp.Owner.Login,
	}, nil
}

// SetRemote adds or updates the origin remote for a local repository.
func (c *Client) SetRemote(ctx context.Context, path, url string) error {
	// Try to add remote first
	result, err := c.runner.Run(ctx, "git", "-C", path, "remote", "add", "origin", url)
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// If remote already exists, update it
	if !result.Success() && strings.Contains(result.Stderr, "already exists") {
		result, err = c.runner.Run(ctx, "git", "-C", path, "remote", "set-url", "origin", url)
		if err != nil {
			return fmt.Errorf("failed to set remote URL: %w", err)
		}
		if !result.Success() {
			return fmt.Errorf("failed to set remote URL: %s", result.Stderr)
		}
	} else if !result.Success() {
		return fmt.Errorf("failed to add remote: %s", result.Stderr)
	}

	return nil
}

// Ensure Client implements ports.GitHubPort.
var _ ports.GitHubPort = (*Client)(nil)
