package templates_test

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitignoreTemplate(t *testing.T) {
	t.Parallel()

	// Verify template contains essential patterns
	assert.Contains(t, templates.GitignoreTemplate, ".env")
	assert.Contains(t, templates.GitignoreTemplate, "*.key")
	assert.Contains(t, templates.GitignoreTemplate, "id_rsa*")
	assert.Contains(t, templates.GitignoreTemplate, ".DS_Store")
	assert.Contains(t, templates.GitignoreTemplate, "preflight.lock")
}

func TestGenerateReadme(t *testing.T) {
	t.Parallel()

	t.Run("with all fields", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName:    "my-dotfiles",
			Description: "My personal dotfiles",
			Owner:       "testuser",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "# my-dotfiles")
		assert.Contains(t, readme, "My personal dotfiles")
		assert.Contains(t, readme, "git@github.com:testuser/my-dotfiles.git")
		assert.Contains(t, readme, "preflight repo clone")
	})

	t.Run("without owner", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "# dotfiles")
		assert.Contains(t, readme, "<username>")
	})

	t.Run("default description", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "preflight")
		assert.True(t, strings.Contains(readme, "Dotfiles and machine configuration"))
	})

	t.Run("contains structure", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "preflight.yaml")
		assert.Contains(t, readme, "layers/")
		assert.Contains(t, readme, "dotfiles/")
	})

	t.Run("contains commands", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "preflight plan")
		assert.Contains(t, readme, "preflight apply")
		assert.Contains(t, readme, "preflight doctor")
	})
}
