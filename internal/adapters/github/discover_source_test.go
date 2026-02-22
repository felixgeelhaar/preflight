package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
)

func TestDiscoverSource_SearchDotfileRepos(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles stars:>=10",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "50",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stdout: `[
			{
				"name": "dotfiles",
				"fullName": "mathiasbynens/dotfiles",
				"description": "Sensible hacker defaults for macOS",
				"url": "https://github.com/mathiasbynens/dotfiles",
				"stargazerCount": 30000,
				"primaryLanguage": "Shell",
				"owner": {"login": "mathiasbynens"}
			},
			{
				"name": "dotfiles",
				"fullName": "holman/dotfiles",
				"description": "My dotfiles",
				"url": "https://github.com/holman/dotfiles",
				"stargazerCount": 7000,
				"primaryLanguage": "Shell",
				"owner": {"login": "holman"}
			}
		]`,
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	repos, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		MinStars:   10,
		MaxResults: 50,
	})

	require.NoError(t, err)
	require.Len(t, repos, 2)

	assert.Equal(t, "mathiasbynens", repos[0].Owner)
	assert.Equal(t, "dotfiles", repos[0].Name)
	assert.Equal(t, 30000, repos[0].Stars)

	assert.Equal(t, "holman", repos[1].Owner)
	assert.Equal(t, "dotfiles", repos[1].Name)
}

func TestDiscoverSource_SearchDotfileRepos_WithLanguage(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles language:go stars:>=10",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "50",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stdout:   `[]`,
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	repos, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		Language:   "go",
		MinStars:   10,
		MaxResults: 50,
	})

	require.NoError(t, err)
	assert.Empty(t, repos)
}

func TestDiscoverSource_SearchDotfileRepos_Error(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "50",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stderr:   "gh: Not logged in",
		ExitCode: 1,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		MaxResults: 50,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Not logged in")
}

func TestDiscoverSource_GetRepoFiles(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"api", "repos/user/dotfiles/git/trees/HEAD?recursive=1",
	}, ports.CommandResult{
		Stdout: `{
			"sha": "abc123",
			"tree": [
				{"path": ".zshrc", "type": "blob"},
				{"path": ".config", "type": "tree"},
				{"path": ".config/nvim", "type": "tree"},
				{"path": ".config/nvim/init.lua", "type": "blob"},
				{"path": ".gitconfig", "type": "blob"}
			]
		}`,
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	files, err := source.GetRepoFiles(ctx, "user", "dotfiles")

	require.NoError(t, err)
	require.Len(t, files, 5)

	assert.Equal(t, ".zshrc", files[0])
	assert.Equal(t, ".config/", files[1])      // Directory gets trailing slash
	assert.Equal(t, ".config/nvim/", files[2]) // Directory gets trailing slash
	assert.Equal(t, ".config/nvim/init.lua", files[3])
	assert.Equal(t, ".gitconfig", files[4])
}

func TestDiscoverSource_GetRepoFiles_Error(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"api", "repos/user/nonexistent/git/trees/HEAD?recursive=1",
	}, ports.CommandResult{
		Stderr:   "gh: Not Found (HTTP 404)",
		ExitCode: 1,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.GetRepoFiles(ctx, "user", "nonexistent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Not Found")
}

func TestDiscoverSource_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ discover.RepoSource = (*DiscoverSource)(nil)
}

func TestDiscoverSource_SearchDotfileRepos_DefaultQuery(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "10",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stdout: `[
			{
				"name": "dotfiles",
				"fullName": "user/dotfiles",
				"description": "My config",
				"url": "https://github.com/user/dotfiles",
				"stargazerCount": 100,
				"primaryLanguage": "Shell",
				"owner": {"login": "user"}
			}
		]`,
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	// Empty query should default to "dotfiles"
	repos, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "",
		MaxResults: 10,
	})

	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "user", repos[0].Owner)
}

func TestDiscoverSource_SearchDotfileRepos_OwnerFallbackFromFullName(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "10",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stdout: `[
			{
				"name": "dotfiles",
				"fullName": "fallbackuser/dotfiles",
				"description": "",
				"url": "https://github.com/fallbackuser/dotfiles",
				"stargazerCount": 5,
				"primaryLanguage": "",
				"owner": {"login": ""}
			}
		]`,
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	repos, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		MaxResults: 10,
	})

	require.NoError(t, err)
	require.Len(t, repos, 1)
	// Owner should be extracted from fullName since owner.login is empty
	assert.Equal(t, "fallbackuser", repos[0].Owner)
}

func TestDiscoverSource_SearchDotfileRepos_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("gh", []string{
		"search", "repos",
		"dotfiles",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "10",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, assert.AnError)

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		MaxResults: 10,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search repositories")
}

func TestDiscoverSource_SearchDotfileRepos_InvalidJSON(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"search", "repos",
		"dotfiles",
		"--sort", "stars",
		"--order", "desc",
		"--limit", "10",
		"--json", "name,fullName,description,url,stargazerCount,primaryLanguage,owner",
	}, ports.CommandResult{
		Stdout:   "not valid json",
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.SearchDotfileRepos(ctx, discover.SearchOptions{
		Query:      "dotfiles",
		MaxResults: 10,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse search results")
}

func TestDiscoverSource_GetRepoFiles_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("gh", []string{
		"api", "repos/user/dotfiles/git/trees/HEAD?recursive=1",
	}, assert.AnError)

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.GetRepoFiles(ctx, "user", "dotfiles")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get repository files")
}

func TestDiscoverSource_GetRepoFiles_InvalidJSON(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddResult("gh", []string{
		"api", "repos/user/dotfiles/git/trees/HEAD?recursive=1",
	}, ports.CommandResult{
		Stdout:   "not valid json at all",
		ExitCode: 0,
	})

	source := NewDiscoverSource(runner)
	ctx := context.Background()

	_, err := source.GetRepoFiles(ctx, "user", "dotfiles")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse tree response")
}
