package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid package names
		{name: "simple name", input: "git", wantErr: nil},
		{name: "with hyphen", input: "node-lts", wantErr: nil},
		{name: "with underscore", input: "python_dev", wantErr: nil},
		{name: "with dot", input: "python3.11", wantErr: nil},
		{name: "with plus", input: "g++", wantErr: nil},
		{name: "numeric start", input: "7zip", wantErr: nil},

		// Invalid package names - regex catches invalid characters first
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "with semicolon", input: "git;rm -rf", wantErr: ErrInvalidPackageName},
		{name: "with pipe", input: "git|cat", wantErr: ErrInvalidPackageName},
		{name: "with ampersand", input: "git&&rm", wantErr: ErrInvalidPackageName},
		{name: "with dollar", input: "git$PATH", wantErr: ErrInvalidPackageName},
		{name: "with backtick", input: "git`whoami`", wantErr: ErrInvalidPackageName},
		{name: "with newline", input: "git\nrm", wantErr: ErrInvalidPackageName},
		{name: "with space", input: "git repo", wantErr: ErrInvalidPackageName},
		{name: "starts with hyphen", input: "-git", wantErr: ErrInvalidPackageName},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidPackageName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageName(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTapName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid tap names
		{name: "homebrew core", input: "homebrew/core", wantErr: nil},
		{name: "github gh", input: "github/gh", wantErr: nil},
		{name: "with underscore", input: "some_user/some_repo", wantErr: nil},
		{name: "with hyphen", input: "some-user/some-repo", wantErr: nil},

		// Invalid tap names
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "no slash", input: "homebrew", wantErr: ErrInvalidTapName},
		{name: "multiple slashes", input: "home/brew/core", wantErr: ErrInvalidTapName},
		{name: "with semicolon", input: "user;rm/repo", wantErr: ErrInvalidTapName},
		{name: "with space", input: "user name/repo", wantErr: ErrInvalidTapName},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidTapName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTapName(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePPA(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid PPAs
		{name: "with ppa prefix", input: "ppa:deadsnakes/ppa", wantErr: nil},
		{name: "without ppa prefix", input: "git-core/ppa", wantErr: nil},
		{name: "with underscore", input: "ppa:some_user/some_ppa", wantErr: nil},

		// Invalid PPAs
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "no slash", input: "ppa:deadsnakes", wantErr: ErrInvalidPPA},
		{name: "with semicolon", input: "ppa:user;rm/ppa", wantErr: ErrInvalidPPA},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidPPA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePPA(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid paths
		{name: "simple path", input: "/home/user/file.txt", wantErr: nil},
		{name: "relative path", input: "config/settings.yaml", wantErr: nil},
		{name: "home path", input: "~/.config/preflight", wantErr: nil},
		{name: "with dots in name", input: "/path/to/file.tar.gz", wantErr: nil},

		// Invalid paths
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "path traversal", input: "../../../etc/passwd", wantErr: ErrPathTraversal},
		{name: "encoded traversal", input: "%2e%2e/%2e%2e/etc/passwd", wantErr: ErrPathTraversal},
		{name: "null byte", input: "/etc/passwd\x00.txt", wantErr: ErrInvalidPath},
		// Note: /home/user/../../etc/passwd normalizes to /etc/passwd (no ..)
		// Use ValidatePathWithBase to catch these - it verifies final path is within base
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePathWithBase(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		basePath string
		wantErr  error
	}{
		// Valid paths within base
		{name: "within base", path: "/home/user/config/file.txt", basePath: "/home/user", wantErr: nil},
		{name: "exact base", path: "/home/user", basePath: "/home/user", wantErr: nil},

		// Invalid paths - escaping base
		{name: "escapes base", path: "/home/other/file.txt", basePath: "/home/user", wantErr: ErrPathTraversal},
		{name: "traversal escape", path: "/home/user/../other/file.txt", basePath: "/home/user", wantErr: ErrPathTraversal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathWithBase(tt.path, tt.basePath)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid hostnames
		{name: "simple domain", input: "github.com", wantErr: nil},
		{name: "subdomain", input: "api.github.com", wantErr: nil},
		{name: "wildcard", input: "*.github.com", wantErr: nil},
		{name: "IP address", input: "192.168.1.1", wantErr: nil},
		{name: "localhost", input: "localhost", wantErr: nil},

		// Invalid hostnames - regex catches invalid characters first
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "with semicolon", input: "github.com;rm", wantErr: ErrInvalidHostname},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidHostname},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGitConfigValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid values
		{name: "simple string", input: "John Doe", wantErr: nil},
		{name: "email", input: "john@example.com", wantErr: nil},
		{name: "path", input: "/usr/bin/vim", wantErr: nil},
		{name: "with special chars", input: "user@host:/path", wantErr: nil},

		// Invalid values
		{name: "with newline", input: "value\ninjected=bad", wantErr: ErrNewlineInjection},
		{name: "with carriage return", input: "value\rinjected", wantErr: ErrNewlineInjection},
		{name: "with control char", input: "value\x00null", wantErr: ErrInvalidGitConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitConfigValue(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHProxyCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid commands
		{name: "empty", input: "", wantErr: nil},
		{name: "ssh jump", input: "ssh -W %h:%p jump-host", wantErr: nil},
		{name: "netcat", input: "nc -X 5 -x proxy:port %h %p", wantErr: nil},
		{name: "simple nc", input: "nc %h %p", wantErr: nil},

		// Invalid commands
		{name: "with semicolon", input: "ssh host; rm -rf /", wantErr: ErrCommandInjection},
		{name: "with pipe", input: "ssh host | cat", wantErr: ErrCommandInjection},
		{name: "with ampersand", input: "ssh host && rm", wantErr: ErrCommandInjection},
		{name: "with dollar", input: "ssh $HOST", wantErr: ErrCommandInjection},
		{name: "with backtick", input: "ssh `whoami`@host", wantErr: ErrCommandInjection},
		{name: "with newline", input: "ssh host\nrm -rf /", wantErr: ErrNewlineInjection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHProxyCommand(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHParameter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid parameters
		{name: "empty", input: "", wantErr: nil},
		{name: "simple value", input: "yes", wantErr: nil},
		{name: "path", input: "~/.ssh/id_rsa", wantErr: nil},
		{name: "port forward", input: "8080:localhost:80", wantErr: nil},

		// Invalid parameters
		{name: "with newline", input: "value\ninjected", wantErr: ErrNewlineInjection},
		{name: "with control char", input: "value\x00null", wantErr: ErrInvalidSSHParameter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHParameter(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePluginName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid plugin names
		{name: "simple name", input: "git", wantErr: nil},
		{name: "github repo", input: "zsh-users/zsh-autosuggestions", wantErr: nil},
		{name: "with hyphen", input: "zsh-syntax-highlighting", wantErr: nil},

		// Invalid plugin names - regex catches invalid characters first
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "with semicolon", input: "plugin;rm", wantErr: ErrInvalidPackageName},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidPackageName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginName(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContainsShellMeta(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"safe-string", false},
		{"with;semicolon", true},
		{"with|pipe", true},
		{"with&ampersand", true},
		{"with$dollar", true},
		{"with`backtick`", true},
		{"with(parens)", true},
		{"with{braces}", true},
		{"with<angle>", true},
		{"with\nnewline", true},
		{"with\\backslash", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsShellMeta(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/normal/path/file.txt", false},
		{"relative/path/file.txt", false},
		{"../etc/passwd", true},
		// Note: /path/../etc/passwd normalizes to /etc/passwd (no ..)
		// The path traversal is caught by ValidatePathWithBase instead
		{"/path/../etc/passwd", false}, // filepath.Clean removes the ..
		{"%2e%2e/etc/passwd", true},
		{"%2E%2E/etc/passwd", true},
		// Windows paths not applicable on Unix - filepath.Separator is /
		{"file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsPathTraversal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCaskName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid cask", input: "visual-studio-code", wantErr: nil},
		{name: "valid cask with dot", input: "firefox", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection attempt", input: "app;rm -rf", wantErr: ErrInvalidPackageName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCaskName(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBrewArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "HEAD flag", input: "--HEAD", wantErr: nil},
		{name: "with-openssl", input: "--with-openssl", wantErr: nil},
		{name: "force flag", input: "--force", wantErr: nil},
		{name: "short flag", input: "-v", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "no dash prefix", input: "HEAD", wantErr: ErrInvalidBrewArg},
		{name: "injection attempt", input: "--flag;rm", wantErr: ErrInvalidBrewArg},
		{name: "too long", input: "--" + strings.Repeat("a", 300), wantErr: ErrInvalidBrewArg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBrewArg(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWingetID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid ID", input: "Microsoft.VisualStudioCode", wantErr: nil},
		{name: "git", input: "Git.Git", wantErr: nil},
		{name: "7zip", input: "7zip.7zip", wantErr: nil},
		{name: "with hyphen", input: "Some-Publisher.Some-App", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "no dot", input: "MicrosoftVisualStudioCode", wantErr: ErrInvalidWingetID},
		{name: "injection", input: "Publisher.App;rm", wantErr: ErrInvalidWingetID},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidWingetID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateWingetID(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWingetSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty allowed", input: "", wantErr: nil},
		{name: "winget", input: "winget", wantErr: nil},
		{name: "msstore", input: "msstore", wantErr: nil},
		{name: "with hyphen", input: "my-source", wantErr: nil},
		{name: "injection", input: "source;rm", wantErr: ErrInvalidWingetSource},
		{name: "too long", input: strings.Repeat("a", 200), wantErr: ErrInvalidWingetSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateWingetSource(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateScoopBucket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "extras", input: "extras", wantErr: nil},
		{name: "versions", input: "versions", wantErr: nil},
		{name: "github repo", input: "ScoopInstaller/Main", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "bucket;rm", wantErr: ErrInvalidScoopBucket},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidScoopBucket},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateScoopBucket(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateChocoPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "git", input: "git", wantErr: nil},
		{name: "nodejs", input: "nodejs", wantErr: nil},
		{name: "with dot", input: "7zip.install", wantErr: nil},
		{name: "python3", input: "python3", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "pkg;rm", wantErr: ErrInvalidChocoPackage},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidChocoPackage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateChocoPackage(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateChocoSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "chocolatey", input: "chocolatey", wantErr: nil},
		{name: "internal", input: "internal", wantErr: nil},
		{name: "with hyphen", input: "my-feed", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "source;rm", wantErr: ErrInvalidChocoSource},
		{name: "too long", input: strings.Repeat("a", 200), wantErr: ErrInvalidChocoSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateChocoSource(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "https URL", input: "https://community.chocolatey.org/api/v2/", wantErr: nil},
		{name: "http URL", input: "http://nuget.internal.com/v3/", wantErr: nil},
		{name: "simple URL", input: "https://example.com", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "no scheme", input: "example.com", wantErr: ErrInvalidURL},
		{name: "ftp scheme", input: "ftp://example.com", wantErr: ErrInvalidURL},
		{name: "too long", input: "https://" + strings.Repeat("a", 2100), wantErr: ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateURL(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty allowed", input: "", wantErr: nil},
		{name: "yaml extension", input: "config/preflight.yaml", wantErr: nil},
		{name: "yml extension", input: "config/preflight.yml", wantErr: nil},
		{name: "wrong extension", input: "config/preflight.json", wantErr: ErrInvalidPath},
		{name: "no extension", input: "config/preflight", wantErr: ErrInvalidPath},
		{name: "shell metachar", input: "config;rm.yaml", wantErr: ErrCommandInjection},
		{name: "null byte", input: "config\x00.yaml", wantErr: ErrInvalidPath},
		{name: "too long", input: strings.Repeat("a", 5000) + ".yaml", wantErr: ErrInvalidPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateConfigPath(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty allowed", input: "", wantErr: nil},
		{name: "default", input: "default", wantErr: nil},
		{name: "with hyphen", input: "my-target", wantErr: nil},
		{name: "with underscore", input: "my_target", wantErr: nil},
		{name: "with dot", input: "work.laptop", wantErr: nil},
		{name: "alphanumeric", input: "target123", wantErr: nil},
		{name: "invalid char space", input: "my target", wantErr: ErrInvalidPath},
		{name: "invalid char semicolon", input: "target;rm", wantErr: ErrInvalidPath},
		{name: "too long", input: strings.Repeat("a", 200), wantErr: ErrInvalidPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTarget(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSnapshotID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "empty allowed", input: "", wantErr: nil},
		{name: "simple id", input: "snapshot-1", wantErr: nil},
		{name: "with underscore", input: "snap_20240101", wantErr: nil},
		{name: "alphanumeric", input: "abc123", wantErr: nil},
		{name: "invalid char dot", input: "snap.1", wantErr: ErrInvalidPath},
		{name: "invalid char space", input: "snap 1", wantErr: ErrInvalidPath},
		{name: "too long", input: strings.Repeat("a", 200), wantErr: ErrInvalidPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSnapshotID(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNpmPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "simple package", input: "lodash", wantErr: nil},
		{name: "scoped package", input: "@types/node", wantErr: nil},
		{name: "scoped with version", input: "@anthropic-ai/claude-code@2.0.0", wantErr: nil},
		{name: "with version", input: "pnpm@10.24.0", wantErr: nil},
		{name: "with dots", input: "socket.io", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "pkg;rm", wantErr: ErrInvalidNpmPackage},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidNpmPackage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateNpmPackage(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGoTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "gopls", input: "golang.org/x/tools/gopls@latest", wantErr: nil},
		{name: "golangci-lint", input: "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.0", wantErr: nil},
		{name: "without version", input: "github.com/user/tool/cmd/tool", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "simple name", input: "gopls", wantErr: ErrInvalidGoTool},
		{name: "injection", input: "github.com/user/tool;rm", wantErr: ErrInvalidGoTool},
		{name: "too long", input: strings.Repeat("a", 600), wantErr: ErrInvalidGoTool},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGoTool(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePipPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "simple package", input: "requests", wantErr: nil},
		{name: "with exact version", input: "black==23.1.0", wantErr: nil},
		{name: "with min version", input: "ruff>=0.1.0", wantErr: ErrCommandInjection},
		{name: "with compat version", input: "numpy~=1.24.0", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "pkg;rm", wantErr: ErrInvalidPipPackage},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidPipPackage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePipPackage(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGemName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "rails", input: "rails", wantErr: nil},
		{name: "bundler with version", input: "bundler@2.4.0", wantErr: nil},
		{name: "rake", input: "rake", wantErr: nil},
		{name: "with hyphen", input: "ruby-lint", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "gem;rm", wantErr: ErrInvalidGemName},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidGemName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGemName(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCargoCrate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "ripgrep", input: "ripgrep", wantErr: nil},
		{name: "bat with version", input: "bat@0.22.1", wantErr: nil},
		{name: "tokio", input: "tokio", wantErr: nil},
		{name: "with underscore", input: "cargo_watch", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "injection", input: "crate;rm", wantErr: ErrInvalidCargoCrate},
		{name: "with dot", input: "crate.name", wantErr: ErrInvalidCargoCrate},
		{name: "too long", input: strings.Repeat("a", 300), wantErr: ErrInvalidCargoCrate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCargoCrate(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
