package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDiscoverer_Discover_FindsKnownConfigs(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create some known config files
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".zshrc"), []byte("# zsh"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("[user]"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".config", "nvim"), 0755))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.Discover()
	require.NoError(t, err)

	// Should find all three configs
	assert.GreaterOrEqual(t, len(result.Configs), 3)

	// Check for specific configs
	found := make(map[string]bool)
	for _, cfg := range result.Configs {
		found[cfg.HomeRelPath] = true
	}

	assert.True(t, found[".zshrc"], "should find .zshrc")
	assert.True(t, found[".gitconfig"], "should find .gitconfig")
	assert.True(t, found[".config/nvim"], "should find .config/nvim")
}

func TestConfigDiscoverer_Discover_SetsCorrectProvider(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create configs for different providers
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".zshrc"), []byte("# zsh"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("[user]"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".wezterm.lua"), []byte("return {}"), 0644))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.Discover()
	require.NoError(t, err)

	providers := make(map[string]string)
	for _, cfg := range result.Configs {
		providers[cfg.HomeRelPath] = cfg.Provider
	}

	assert.Equal(t, "shell", providers[".zshrc"])
	assert.Equal(t, "git", providers[".gitconfig"])
	assert.Equal(t, "terminal", providers[".wezterm.lua"])
}

func TestConfigDiscoverer_Discover_SkipsSensitiveFiles(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create sensitive files
	sshDir := filepath.Join(homeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "id_rsa"), []byte("PRIVATE KEY"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *"), 0644))

	// Create history file
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".zsh_history"), []byte("history"), 0600))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.Discover()
	require.NoError(t, err)

	// SSH config should be found but history should be skipped
	foundSSHConfig := false
	foundHistory := false
	for _, cfg := range result.Configs {
		if cfg.HomeRelPath == ".ssh/config" {
			foundSSHConfig = true
		}
		if cfg.HomeRelPath == ".zsh_history" {
			foundHistory = true
		}
	}

	assert.True(t, foundSSHConfig, "SSH config should be found")
	assert.False(t, foundHistory, "History file should not be found")
}

func TestConfigDiscoverer_Discover_CalculatesSize(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create a file with known size
	content := "test content for size"
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte(content), 0644))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.Discover()
	require.NoError(t, err)

	// Find the gitconfig
	var gitconfig *DiscoveredConfig
	for i := range result.Configs {
		if result.Configs[i].HomeRelPath == ".gitconfig" {
			gitconfig = &result.Configs[i]
			break
		}
	}

	require.NotNil(t, gitconfig)
	assert.Equal(t, int64(len(content)), gitconfig.Size)
}

func TestConfigDiscoverer_DiscoverUnknown_FindsUnknownConfigs(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create an unknown config directory in .config
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".config", "myapp"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(homeDir, ".config", "myapp", "config.yaml"),
		[]byte("key: value"),
		0644,
	))

	// Create an unknown dotfile
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".myapprc"), []byte("config"), 0644))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.DiscoverUnknown()
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, cfg := range result.Configs {
		found[cfg.HomeRelPath] = true
	}

	assert.True(t, found[".config/myapp"], "should find unknown .config/myapp")
	assert.True(t, found[".myapprc"], "should find unknown .myapprc")
}

func TestConfigDiscoverer_DiscoverUnknown_SkipsKnownPatterns(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create a known config (nvim)
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".config", "nvim"), 0755))

	// Create an unknown config
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".config", "myapp"), 0755))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.DiscoverUnknown()
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, cfg := range result.Configs {
		found[cfg.HomeRelPath] = true
	}

	assert.False(t, found[".config/nvim"], "should NOT find known nvim config")
	assert.True(t, found[".config/myapp"], "should find unknown myapp config")
}

func TestConfigDiscoverer_DiscoverUnknown_SkipsNonConfigDirs(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()

	// Create common non-config directories
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".cache"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".npm"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".cargo"), 0755))

	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	result, err := discoverer.DiscoverUnknown()
	require.NoError(t, err)

	found := make(map[string]bool)
	for _, cfg := range result.Configs {
		found[cfg.HomeRelPath] = true
	}

	assert.False(t, found[".cache"], "should NOT find .cache")
	assert.False(t, found[".npm"], "should NOT find .npm")
	assert.False(t, found[".cargo"], "should NOT find .cargo")
}

func TestDiscoveryResult_ToCapture(t *testing.T) {
	t.Parallel()

	result := &DiscoveryResult{
		Configs: []DiscoveredConfig{
			{HomeRelPath: ".zshrc", Provider: "shell"},
			{HomeRelPath: ".zshenv", Provider: "shell"},
			{HomeRelPath: ".gitconfig", Provider: "git"},
			{HomeRelPath: ".config/nvim", Provider: "nvim"},
		},
	}

	captureConfigs := result.ToCapture()

	// Should group by provider
	byProvider := make(map[string][]string)
	for _, cfg := range captureConfigs {
		byProvider[cfg.Provider] = cfg.SourcePaths
	}

	assert.Len(t, byProvider["shell"], 2)
	assert.Contains(t, byProvider["shell"], "~/.zshrc")
	assert.Contains(t, byProvider["shell"], "~/.zshenv")
	assert.Len(t, byProvider["git"], 1)
	assert.Contains(t, byProvider["git"], "~/.gitconfig")
	assert.Len(t, byProvider["nvim"], 1)
	assert.Contains(t, byProvider["nvim"], "~/.config/nvim")
}

func TestGetDiscoveryPatterns_HasExpectedProviders(t *testing.T) {
	t.Parallel()

	patterns := getDiscoveryPatterns()

	providers := make(map[string]bool)
	for _, p := range patterns {
		providers[p.Provider] = true
	}

	// Should have all major providers
	expectedProviders := []string{
		"shell", "git", "ssh", "terminal", "nvim", "vscode", "tmux", "starship",
	}

	for _, expected := range expectedProviders {
		assert.True(t, providers[expected], "should have provider: %s", expected)
	}
}

func TestGetSensitivePatterns_IncludesPrivateKeys(t *testing.T) {
	t.Parallel()

	patterns := getSensitivePatterns()

	// Should include private key patterns
	hasIDPattern := false
	for _, p := range patterns {
		if p == ".ssh/id_*" {
			hasIDPattern = true
			break
		}
	}

	assert.True(t, hasIDPattern, "should include .ssh/id_* pattern")
}

func TestConfigDiscoverer_LooksLikeConfig(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	fs := filesystem.NewRealFileSystem()
	discoverer := NewConfigDiscoverer(fs, homeDir)

	tests := []struct {
		name     string
		expected bool
	}{
		{".zshrc", true},
		{".bashrc", true},
		{".vimrc", true},
		{".config", true},
		{"config.json", true},
		{"settings.yaml", true},
		{"app.toml", true},
		{".random", false},
		{"readme.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := discoverer.looksLikeConfig(tt.name)
			assert.Equal(t, tt.expected, result, "looksLikeConfig(%q)", tt.name)
		})
	}
}
