package github_test

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/adapters/github"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_IsAuthenticated(t *testing.T) {
	t.Parallel()

	t.Run("authenticated", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{"auth", "status"}, ports.CommandResult{
			ExitCode: 0,
			Stdout:   "✓ Logged in to github.com as testuser",
		})

		client := github.NewClient(runner)
		authed, err := client.IsAuthenticated(context.Background())

		require.NoError(t, err)
		assert.True(t, authed)
	})

	t.Run("not authenticated", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{"auth", "status"}, ports.CommandResult{
			ExitCode: 1,
			Stderr:   "You are not logged into any GitHub hosts",
		})

		client := github.NewClient(runner)
		authed, err := client.IsAuthenticated(context.Background())

		require.NoError(t, err)
		assert.False(t, authed)
	})
}

func TestClient_GetAuthenticatedUser(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{"api", "user", "--jq", ".login"}, ports.CommandResult{
			ExitCode: 0,
			Stdout:   "testuser\n",
		})

		client := github.NewClient(runner)
		user, err := client.GetAuthenticatedUser(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "testuser", user)
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{"api", "user", "--jq", ".login"}, ports.CommandResult{
			ExitCode: 1,
			Stderr:   "not authenticated",
		})

		client := github.NewClient(runner)
		_, err := client.GetAuthenticatedUser(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
	})
}

func TestClient_CreateRepository(t *testing.T) {
	t.Parallel()

	t.Run("private repository", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{
			"repo", "create", "my-dotfiles", "--confirm", "--private",
			"--description", "My dotfiles",
			"--json", "name,url,cloneUrl,sshUrl,owner",
		}, ports.CommandResult{
			ExitCode: 0,
			Stdout: `{
				"name": "my-dotfiles",
				"url": "https://github.com/testuser/my-dotfiles",
				"clone_url": "https://github.com/testuser/my-dotfiles.git",
				"ssh_url": "git@github.com:testuser/my-dotfiles.git",
				"owner": {"login": "testuser"}
			}`,
		})

		client := github.NewClient(runner)
		info, err := client.CreateRepository(context.Background(), ports.GitHubCreateOptions{
			Name:        "my-dotfiles",
			Description: "My dotfiles",
			Private:     true,
		})

		require.NoError(t, err)
		assert.Equal(t, "my-dotfiles", info.Name)
		assert.Equal(t, "testuser", info.Owner)
		assert.Contains(t, info.SSHURL, "git@github.com")
	})

	t.Run("public repository", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{
			"repo", "create", "my-dotfiles", "--confirm", "--public",
			"--json", "name,url,cloneUrl,sshUrl,owner",
		}, ports.CommandResult{
			ExitCode: 0,
			Stdout: `{
				"name": "my-dotfiles",
				"url": "https://github.com/testuser/my-dotfiles",
				"clone_url": "https://github.com/testuser/my-dotfiles.git",
				"ssh_url": "git@github.com:testuser/my-dotfiles.git",
				"owner": {"login": "testuser"}
			}`,
		})

		client := github.NewClient(runner)
		info, err := client.CreateRepository(context.Background(), ports.GitHubCreateOptions{
			Name:    "my-dotfiles",
			Private: false,
		})

		require.NoError(t, err)
		assert.Equal(t, "my-dotfiles", info.Name)
	})

	t.Run("creation failure", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("gh", []string{
			"repo", "create", "my-dotfiles", "--confirm", "--private",
			"--json", "name,url,cloneUrl,sshUrl,owner",
		}, ports.CommandResult{
			ExitCode: 1,
			Stderr:   "repository already exists",
		})

		client := github.NewClient(runner)
		_, err := client.CreateRepository(context.Background(), ports.GitHubCreateOptions{
			Name:    "my-dotfiles",
			Private: true,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository already exists")
	})
}

func TestClient_SetRemote(t *testing.T) {
	t.Parallel()

	t.Run("add new remote", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("git", []string{"-C", "/path/to/repo", "remote", "add", "origin", "git@github.com:user/repo.git"}, ports.CommandResult{
			ExitCode: 0,
		})

		client := github.NewClient(runner)
		err := client.SetRemote(context.Background(), "/path/to/repo", "git@github.com:user/repo.git")

		require.NoError(t, err)
	})

	t.Run("update existing remote", func(t *testing.T) {
		t.Parallel()

		runner := mocks.NewCommandRunner()
		runner.AddResult("git", []string{"-C", "/path/to/repo", "remote", "add", "origin", "git@github.com:user/repo.git"}, ports.CommandResult{
			ExitCode: 128,
			Stderr:   "error: remote origin already exists.",
		})
		runner.AddResult("git", []string{"-C", "/path/to/repo", "remote", "set-url", "origin", "git@github.com:user/repo.git"}, ports.CommandResult{
			ExitCode: 0,
		})

		client := github.NewClient(runner)
		err := client.SetRemote(context.Background(), "/path/to/repo", "git@github.com:user/repo.git")

		require.NoError(t, err)
	})
}
