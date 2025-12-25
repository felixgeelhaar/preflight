package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpgradeChecker(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	checker := NewUpgradeChecker(registry)

	assert.NotNil(t, checker)
	assert.NotNil(t, checker.registry)
	assert.NotNil(t, checker.cloner)
}

func TestUpgradeChecker_CheckUpgrade(t *testing.T) {
	t.Parallel()

	t.Run("nil registry returns error", func(t *testing.T) {
		t.Parallel()
		checker := &UpgradeChecker{}
		_, err := checker.CheckUpgrade(context.Background(), "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
	})

	t.Run("plugin not found returns error", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		checker := NewUpgradeChecker(registry)
		_, err := checker.CheckUpgrade(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("builtin plugin shows no upgrade", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		plugin := &Plugin{
			Manifest: Manifest{
				Name:    "builtin-plugin",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(plugin))

		checker := NewUpgradeChecker(registry)
		info, err := checker.CheckUpgrade(context.Background(), "builtin-plugin")
		require.NoError(t, err)
		assert.Equal(t, "builtin-plugin", info.Name)
		assert.Equal(t, "1.0.0", info.CurrentVersion)
		assert.Equal(t, "1.0.0", info.LatestVersion)
		assert.False(t, info.UpgradeAvailable)
	})

	t.Run("plugin without source shows no upgrade", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		plugin := &Plugin{
			Manifest: Manifest{
				Name:    "local-plugin",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(plugin))

		checker := NewUpgradeChecker(registry)
		info, err := checker.CheckUpgrade(context.Background(), "local-plugin")
		require.NoError(t, err)
		assert.False(t, info.UpgradeAvailable)
	})
}

func TestUpgradeChecker_CheckAllUpgrades(t *testing.T) {
	t.Parallel()

	t.Run("nil registry returns error", func(t *testing.T) {
		t.Parallel()
		checker := &UpgradeChecker{}
		_, err := checker.CheckAllUpgrades(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
	})

	t.Run("empty registry returns empty list", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()
		checker := NewUpgradeChecker(registry)
		infos, err := checker.CheckAllUpgrades(context.Background())
		require.NoError(t, err)
		assert.Empty(t, infos)
	})

	t.Run("returns info for all plugins", func(t *testing.T) {
		t.Parallel()
		registry := NewRegistry()

		plugin1 := &Plugin{
			Manifest: Manifest{
				Name:    "plugin-1",
				Version: "1.0.0",
				Type:    TypeConfig,
			},
		}
		plugin2 := &Plugin{
			Manifest: Manifest{
				Name:    "plugin-2",
				Version: "2.0.0",
				Type:    TypeConfig,
			},
		}
		require.NoError(t, registry.Register(plugin1))
		require.NoError(t, registry.Register(plugin2))

		checker := NewUpgradeChecker(registry)
		infos, err := checker.CheckAllUpgrades(context.Background())
		require.NoError(t, err)
		assert.Len(t, infos, 2)
	})
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"", "v0.0.0"},
		{"2.3.4", "v2.3.4"},
		{"v2.3.4", "v2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := normalizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/user/repo", true},
		{"https://github.com/user/repo.git", true},
		{"git@github.com:user/repo.git", true},
		{"https://gitlab.com/user/repo", false},
		{"https://example.com", false},
		{"short", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			result := isGitHubURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstructChangelogURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		repoURL  string
		version  string
		expected string
	}{
		{
			"https://github.com/user/repo",
			"v1.0.0",
			"https://github.com/user/repo/releases/tag/v1.0.0",
		},
		{
			"https://github.com/user/repo.git",
			"v2.0.0",
			"https://github.com/user/repo/releases/tag/v2.0.0",
		},
		{
			"git@github.com:user/repo.git",
			"v3.0.0",
			"https://github.com/user/repo/releases/tag/v3.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.repoURL, func(t *testing.T) {
			t.Parallel()
			result := constructChangelogURL(tt.repoURL, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatUpgradeInfo(t *testing.T) {
	t.Parallel()

	t.Run("nil returns empty", func(t *testing.T) {
		t.Parallel()
		result := FormatUpgradeInfo(nil)
		assert.Empty(t, result)
	})

	t.Run("no upgrade available", func(t *testing.T) {
		t.Parallel()
		info := &UpgradeInfo{
			Name:             "test-plugin",
			CurrentVersion:   "1.0.0",
			LatestVersion:    "1.0.0",
			UpgradeAvailable: false,
		}
		result := FormatUpgradeInfo(info)
		assert.Contains(t, result, "test-plugin")
		assert.Contains(t, result, "1.0.0")
		assert.Contains(t, result, "up to date")
	})

	t.Run("upgrade available", func(t *testing.T) {
		t.Parallel()
		info := &UpgradeInfo{
			Name:             "test-plugin",
			CurrentVersion:   "1.0.0",
			LatestVersion:    "2.0.0",
			UpgradeAvailable: true,
		}
		result := FormatUpgradeInfo(info)
		assert.Contains(t, result, "test-plugin")
		assert.Contains(t, result, "1.0.0")
		assert.Contains(t, result, "2.0.0")
		assert.Contains(t, result, "→")
	})
}

func TestSplitLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"multi-line", "one\ntwo\nthree", []string{"one", "two", "three"}},
		{"single", "single", []string{"single"}},
		{"trailing newline", "line\n", []string{"line"}},
		{"just newline", "\n", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		result := splitLines("")
		assert.Nil(t, result)
	})
}

func TestTrimSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"spaces", "  hello  ", "hello"},
		{"tabs", "\thello\t", "hello"},
		{"no trim", "hello", "hello"},
		{"empty", "", ""},
		{"only spaces", "  ", ""},
		{"trailing cr", "hello\r", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := trimSpace(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s        string
		prefix   string
		expected bool
	}{
		{"hello world", "hello", true},
		{"hello", "hello world", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.prefix, func(t *testing.T) {
			t.Parallel()
			result := hasPrefix(tt.s, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetGitRemoteURL(t *testing.T) {
	t.Parallel()

	t.Run("returns empty for non-git directory", func(t *testing.T) {
		t.Parallel()
		result := getGitRemoteURL("/nonexistent/path")
		assert.Empty(t, result)
	})
}

func TestFormatUpgradeList(t *testing.T) {
	t.Parallel()

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		result := FormatUpgradeList(nil)
		assert.Equal(t, "No plugins installed.", result)
	})

	t.Run("all up to date", func(t *testing.T) {
		t.Parallel()
		infos := []UpgradeInfo{
			{Name: "plugin-1", CurrentVersion: "1.0.0", LatestVersion: "1.0.0", UpgradeAvailable: false},
			{Name: "plugin-2", CurrentVersion: "2.0.0", LatestVersion: "2.0.0", UpgradeAvailable: false},
		}
		result := FormatUpgradeList(infos)
		assert.Equal(t, "All plugins are up to date.", result)
	})

	t.Run("with upgrades", func(t *testing.T) {
		t.Parallel()
		infos := []UpgradeInfo{
			{Name: "plugin-1", CurrentVersion: "1.0.0", LatestVersion: "2.0.0", UpgradeAvailable: true},
			{Name: "plugin-2", CurrentVersion: "2.0.0", LatestVersion: "2.0.0", UpgradeAvailable: false},
		}
		result := FormatUpgradeList(infos)
		assert.Contains(t, result, "Available upgrades")
		assert.Contains(t, result, "plugin-1")
		assert.NotContains(t, result, "plugin-2")
	})

	t.Run("with changelog URL", func(t *testing.T) {
		t.Parallel()
		infos := []UpgradeInfo{
			{
				Name:             "plugin-1",
				CurrentVersion:   "1.0.0",
				LatestVersion:    "2.0.0",
				UpgradeAvailable: true,
				ChangelogURL:     "https://github.com/user/repo/releases/tag/v2.0.0",
			},
		}
		result := FormatUpgradeList(infos)
		assert.Contains(t, result, "https://github.com/user/repo/releases/tag/v2.0.0")
	})
}
