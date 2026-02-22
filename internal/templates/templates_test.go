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

	t.Run("contains secret patterns", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, ".env")
		assert.Contains(t, templates.GitignoreTemplate, ".env.*")
		assert.Contains(t, templates.GitignoreTemplate, "*.key")
		assert.Contains(t, templates.GitignoreTemplate, "*.pem")
		assert.Contains(t, templates.GitignoreTemplate, "credentials.json")
		assert.Contains(t, templates.GitignoreTemplate, "secrets.yaml")
		assert.Contains(t, templates.GitignoreTemplate, "secrets.yml")
		assert.Contains(t, templates.GitignoreTemplate, "*.secret")
	})

	t.Run("contains SSH key patterns", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, "id_rsa*")
		assert.Contains(t, templates.GitignoreTemplate, "id_ed25519*")
		assert.Contains(t, templates.GitignoreTemplate, "id_ecdsa*")
		assert.Contains(t, templates.GitignoreTemplate, "*.pub")
	})

	t.Run("contains GPG patterns", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, "*.gpg")
		assert.Contains(t, templates.GitignoreTemplate, "secring.*")
		assert.Contains(t, templates.GitignoreTemplate, "trustdb.gpg")
	})

	t.Run("contains OS generated files", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, ".DS_Store")
		assert.Contains(t, templates.GitignoreTemplate, "Thumbs.db")
		assert.Contains(t, templates.GitignoreTemplate, "Desktop.ini")
	})

	t.Run("contains editor patterns", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, "*.swp")
		assert.Contains(t, templates.GitignoreTemplate, "*.swo")
		assert.Contains(t, templates.GitignoreTemplate, ".idea/")
	})

	t.Run("contains preflight state", func(t *testing.T) {
		t.Parallel()

		assert.Contains(t, templates.GitignoreTemplate, "preflight.lock")
		assert.Contains(t, templates.GitignoreTemplate, ".preflight/")
		assert.Contains(t, templates.GitignoreTemplate, "*.local.yaml")
	})

	t.Run("is non-empty", func(t *testing.T) {
		t.Parallel()

		assert.NotEmpty(t, templates.GitignoreTemplate)
		lines := strings.Split(templates.GitignoreTemplate, "\n")
		assert.Greater(t, len(lines), 10, "gitignore should have many patterns")
	})
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
		// Custom description should replace default
		assert.NotContains(t, readme, "Dotfiles and machine configuration managed by")
	})

	t.Run("without owner uses placeholder", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "# dotfiles")
		assert.Contains(t, readme, "<username>")
		// Should not contain SSH URL when no owner
		assert.NotContains(t, readme, "git@github.com:")
	})

	t.Run("default description when empty", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "Dotfiles and machine configuration managed by")
		assert.Contains(t, readme, "preflight")
	})

	t.Run("custom description replaces default", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName:    "my-config",
			Description: "Custom workstation setup",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "Custom workstation setup")
		assert.NotContains(t, readme, "Dotfiles and machine configuration managed by")
	})

	t.Run("contains repository structure", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "preflight.yaml")
		assert.Contains(t, readme, "layers/")
		assert.Contains(t, readme, "dotfiles/")
		assert.Contains(t, readme, "preflight.lock")
	})

	t.Run("contains all commands", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "preflight plan")
		assert.Contains(t, readme, "preflight apply")
		assert.Contains(t, readme, "preflight doctor")
		assert.Contains(t, readme, "preflight capture")
		assert.Contains(t, readme, "preflight explain")
	})

	t.Run("contains targets section", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "## Targets")
		assert.Contains(t, readme, "work")
		assert.Contains(t, readme, "personal")
		assert.Contains(t, readme, "minimal")
	})

	t.Run("contains install instructions", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
			Owner:    "alice",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "brew install felixgeelhaar/tap/preflight")
		assert.Contains(t, readme, "## Quick Start")
		assert.Contains(t, readme, "### On a new machine")
		assert.Contains(t, readme, "### On an existing machine")
	})

	t.Run("contains learn more links", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "## Learn More")
		assert.Contains(t, readme, "Preflight Documentation")
		assert.Contains(t, readme, "Configuration Reference")
	})

	t.Run("zero value ReadmeData", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{})

		require.NoError(t, err)
		assert.NotEmpty(t, readme)
		// Should use defaults for empty RepoName
		assert.Contains(t, readme, "Dotfiles and machine configuration managed by")
	})

	t.Run("special characters in fields", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName:    "my-dotfiles-2.0",
			Description: "Config with special chars: & < > \"quotes\"",
			Owner:       "user-name_123",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "# my-dotfiles-2.0")
		assert.Contains(t, readme, "Config with special chars: & < > \"quotes\"")
		assert.Contains(t, readme, "user-name_123")
	})

	t.Run("owner with SSH URL format", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "dotfiles",
			Owner:    "myorg",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "git@github.com:myorg/dotfiles.git")
	})

	t.Run("repo name in structure section", func(t *testing.T) {
		t.Parallel()

		readme, err := templates.GenerateReadme(templates.ReadmeData{
			RepoName: "workspace-config",
		})

		require.NoError(t, err)
		assert.Contains(t, readme, "workspace-config/")
	})
}
