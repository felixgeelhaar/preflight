package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateGitBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		branch  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"empty allowed", "", false, ""},
		{"simple branch", "main", false, ""},
		{"feature branch", "feature/add-login", false, ""},
		{"with hyphen", "fix-bug-123", false, ""},
		{"with underscore", "release_v1.0", false, ""},
		{"with dots", "v1.0.0", false, ""},

		// Invalid cases - command injection attempts
		{"semicolon injection", "main; rm -rf /", true, "invalid character"},
		{"ampersand injection", "main && evil", true, "invalid character"},
		{"pipe injection", "main | cat /etc/passwd", true, "invalid character"},
		{"backtick injection", "main`whoami`", true, "invalid character"},
		{"dollar injection", "main$(whoami)", true, "invalid character"},
		{"newline injection", "main\nrm -rf /", true, "invalid character"},

		// Invalid cases - other
		{"path traversal", "main/../../../etc", true, "cannot contain '..'"},
		{"too long", strings.Repeat("a", 256), true, "too long"},
		{"special chars", "main<>!", true, "invalid character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGitBranch(tt.branch)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitRemoteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"empty allowed", "", false, ""},
		{"https with .git", "https://github.com/user/repo.git", false, ""},
		{"https without .git", "https://github.com/user/repo", false, ""},
		{"ssh format", "git@github.com:user/repo.git", false, ""},
		{"ssh protocol", "ssh://git@github.com/user/repo.git", false, ""},
		{"gitlab https", "https://gitlab.com/user/project.git", false, ""},
		{"nested path", "https://github.com/org/team/repo.git", false, ""},

		// Invalid cases - command injection
		{"semicolon injection", "https://evil.com/repo.git; rm -rf /", true, "invalid character"},
		{"backtick injection", "https://evil.com/`whoami`.git", true, "invalid character"},
		{"pipe injection", "https://evil.com/repo | cat", true, "invalid character"},

		// Local paths - valid
		{"file protocol", "file:///path/to/repo", false, ""},
		{"unix absolute path", "/path/to/repo", false, ""},

		// Invalid cases - format
		{"no protocol", "github.com/user/repo", true, "invalid git remote URL"},
		{"ftp protocol", "ftp://evil.com/repo", true, "invalid git remote URL"},
		{"too long", "https://github.com/" + strings.Repeat("a", 2048), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGitRemoteURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitRemoteName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		remote  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"simple", "origin", false, ""},
		{"with dash", "upstream-1", false, ""},
		{"with slash", "forks/dev", false, ""},
		{"with dot", "origin.dev", false, ""},

		// Invalid cases
		{"empty", "", true, "cannot be empty"},
		{"path traversal", "../origin", true, "cannot contain"},
		{"semicolon injection", "origin; rm -rf /", true, "invalid character"},
		{"too long", strings.Repeat("a", 256), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGitRemoteName(tt.remote)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"simple path", "/home/user/repo", false, ""},
		{"relative path", "./repo", false, ""},
		{"with dots", "/home/user/my.dotfiles", false, ""},

		// Invalid cases
		{"empty", "", true, "cannot be empty"},
		{"null byte", "/home/user\x00/repo", true, "null byte"},
		{"semicolon", "/home/user; rm -rf /", true, "invalid character"},
		{"pipe", "/home/user | cat", true, "invalid character"},
		{"too long", "/" + strings.Repeat("a", 4096), true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGitPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{"simple", "myrepo", false, ""},
		{"with hyphen", "my-repo", false, ""},
		{"with underscore", "my_repo", false, ""},
		{"with numbers", "repo123", false, ""},
		{"with dots", "my.repo", false, ""},

		// Invalid cases
		{"empty", "", true, "cannot be empty"},
		{"starts with hyphen", "-myrepo", true, "must start with alphanumeric"},
		{"starts with dot", ".myrepo", true, "must start with alphanumeric"},
		{"too long", strings.Repeat("a", 101), true, "too long"},
		{"special chars", "my<repo>", true, "must start with alphanumeric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGitRepoName(tt.repo)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
