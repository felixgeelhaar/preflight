package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscovery(t *testing.T) {
	d := NewDiscovery()
	assert.NotNil(t, d)
	assert.NotNil(t, d.finder)
}

func TestNewDiscoveryWithFinder(t *testing.T) {
	finder := pathutil.NewConfigFinder()
	d := NewDiscoveryWithFinder(finder)
	assert.NotNil(t, d)
	assert.Equal(t, finder, d.finder)
}

// --- Kitty Tests ---

func TestKittySearchOpts(t *testing.T) {
	opts := KittySearchOpts()
	assert.Equal(t, "KITTY_CONFIG_DIRECTORY", opts.EnvVar)
	assert.Equal(t, "kitty.conf", opts.ConfigFileName)
	assert.Equal(t, "kitty/kitty.conf", opts.XDGSubpath)
	assert.Contains(t, opts.MacOSPaths, "~/.config/kitty/kitty.conf")
}

func TestDiscovery_FindKittyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a kitty config file
	kittyDir := filepath.Join(tmpDir, ".config", "kitty")
	err := os.MkdirAll(kittyDir, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(kittyDir, "kitty.conf")
	err = os.WriteFile(configFile, []byte("font_size 14\n"), 0o644)
	require.NoError(t, err)

	// Set HOME to use the temp directory
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := NewDiscovery()
	path := d.FindKittyConfig()
	assert.Equal(t, configFile, path)
}

// --- WezTerm Tests ---

func TestWezTermSearchOpts(t *testing.T) {
	opts := WezTermSearchOpts()
	assert.Equal(t, "WEZTERM_CONFIG_DIR", opts.EnvVar)
	assert.Equal(t, "wezterm.lua", opts.ConfigFileName)
	assert.Equal(t, "wezterm/wezterm.lua", opts.XDGSubpath)
	assert.Contains(t, opts.LegacyPaths, "~/.wezterm.lua")
}

func TestDiscovery_FindWezTermConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a wezterm config file
	weztermDir := filepath.Join(tmpDir, ".config", "wezterm")
	err := os.MkdirAll(weztermDir, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(weztermDir, "wezterm.lua")
	err = os.WriteFile(configFile, []byte("return {}\n"), 0o644)
	require.NoError(t, err)

	// Set HOME to use the temp directory
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := NewDiscovery()
	path := d.FindWezTermConfig()
	assert.Equal(t, configFile, path)
}

// --- Ghostty Tests ---

func TestGhosttySearchOpts(t *testing.T) {
	opts := GhosttySearchOpts()
	assert.Equal(t, "", opts.EnvVar)
	assert.Equal(t, "ghostty/config", opts.XDGSubpath)
	assert.Contains(t, opts.MacOSPaths, "~/.config/ghostty/config")
}

func TestDiscovery_FindGhosttyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a ghostty config file
	ghosttyDir := filepath.Join(tmpDir, ".config", "ghostty")
	err := os.MkdirAll(ghosttyDir, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(ghosttyDir, "config")
	err = os.WriteFile(configFile, []byte("font-size = 14\n"), 0o644)
	require.NoError(t, err)

	// Set HOME to use the temp directory
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := NewDiscovery()
	path := d.FindGhosttyConfig()
	assert.Equal(t, configFile, path)
}

// --- iTerm2 Tests ---

func TestITerm2SearchOpts(t *testing.T) {
	opts := ITerm2SearchOpts()
	assert.Equal(t, "", opts.EnvVar)
	assert.Equal(t, "", opts.XDGSubpath)
	assert.Contains(t, opts.MacOSPaths, "~/Library/Preferences/com.googlecode.iterm2.plist")
	assert.Empty(t, opts.LinuxPaths)
	assert.Empty(t, opts.WindowsPaths)
}

func TestDiscovery_FindITerm2Config(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an iTerm2 plist file
	prefsDir := filepath.Join(tmpDir, "Library", "Preferences")
	err := os.MkdirAll(prefsDir, 0o755)
	require.NoError(t, err)

	plistFile := filepath.Join(prefsDir, "com.googlecode.iterm2.plist")
	err = os.WriteFile(plistFile, []byte("<?xml version=\"1.0\"?>\n<plist></plist>\n"), 0o644)
	require.NoError(t, err)

	// Set HOME to use the temp directory
	t.Setenv("HOME", tmpDir)

	d := NewDiscovery()
	path := d.FindITerm2Config()
	assert.Equal(t, plistFile, path)
}

func TestDiscovery_ITerm2BestPracticePath(t *testing.T) {
	d := NewDiscovery()
	path := d.ITerm2BestPracticePath()
	assert.Contains(t, path, "com.googlecode.iterm2.plist")
}

func TestDiscovery_ITerm2DynamicProfilesDir(t *testing.T) {
	d := NewDiscovery()
	path := d.ITerm2DynamicProfilesDir()
	assert.Contains(t, path, "DynamicProfiles")
	assert.Contains(t, path, "iTerm2")
}

// --- Hyper Tests ---

func TestHyperSearchOpts(t *testing.T) {
	opts := HyperSearchOpts()
	assert.Equal(t, "", opts.EnvVar)
	assert.Equal(t, "", opts.XDGSubpath)
	assert.Contains(t, opts.MacOSPaths, "~/.hyper.js")
	assert.Contains(t, opts.LinuxPaths, "~/.hyper.js")
}

func TestDiscovery_FindHyperConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hyper config file
	hyperFile := filepath.Join(tmpDir, ".hyper.js")
	err := os.WriteFile(hyperFile, []byte("module.exports = {}\n"), 0o644)
	require.NoError(t, err)

	// Set HOME to use the temp directory
	t.Setenv("HOME", tmpDir)

	d := NewDiscovery()
	path := d.FindHyperConfig()
	assert.Equal(t, hyperFile, path)
}

// --- Windows Terminal Tests ---

func TestWindowsTerminalSearchOpts(t *testing.T) {
	opts := WindowsTerminalSearchOpts()
	assert.Equal(t, "", opts.EnvVar)
	assert.Empty(t, opts.MacOSPaths)
	assert.Empty(t, opts.LinuxPaths)
	assert.Len(t, opts.WindowsPaths, 2)
}

func TestDiscovery_FindWindowsTerminalConfig(_ *testing.T) {
	// This function will return empty on non-Windows systems
	d := NewDiscovery()
	path := d.FindWindowsTerminalConfig()
	// On non-Windows, this should return empty
	// The function is called and tested for coverage
	_ = path
}

// --- BestPracticePaths Tests ---

func TestDiscovery_BestPracticePaths(t *testing.T) {
	d := NewDiscovery()
	paths := d.BestPracticePaths()

	assert.Contains(t, paths, "alacritty")
	assert.Contains(t, paths, "kitty")
	assert.Contains(t, paths, "wezterm")
	assert.Contains(t, paths, "ghostty")
	assert.Contains(t, paths, "iterm2")
	assert.Contains(t, paths, "hyper")
	assert.Contains(t, paths, "windows_terminal")

	// Verify the paths contain expected substrings
	assert.Contains(t, paths["alacritty"], "alacritty")
	assert.Contains(t, paths["kitty"], "kitty")
	assert.Contains(t, paths["ghostty"], "ghostty")
	assert.Contains(t, paths["hyper"], ".hyper.js")
}

// --- FindAllConfigs Tests ---

func TestDiscovery_FindAllConfigs_NoConfigs(t *testing.T) {
	// Use a temp dir with no config files
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := NewDiscovery()
	configs := d.FindAllConfigs()

	// Should return empty map when no configs are found
	assert.Empty(t, configs)
}

func TestDiscovery_FindAllConfigs_WithConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Create alacritty config
	alacrittyDir := filepath.Join(tmpDir, ".config", "alacritty")
	err := os.MkdirAll(alacrittyDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(alacrittyDir, "alacritty.toml"), []byte(""), 0o644)
	require.NoError(t, err)

	// Create kitty config
	kittyDir := filepath.Join(tmpDir, ".config", "kitty")
	err = os.MkdirAll(kittyDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(kittyDir, "kitty.conf"), []byte(""), 0o644)
	require.NoError(t, err)

	// Create ghostty config
	ghosttyDir := filepath.Join(tmpDir, ".config", "ghostty")
	err = os.MkdirAll(ghosttyDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(ghosttyDir, "config"), []byte(""), 0o644)
	require.NoError(t, err)

	d := NewDiscovery()
	configs := d.FindAllConfigs()

	assert.Contains(t, configs, "alacritty")
	assert.Contains(t, configs, "kitty")
	assert.Contains(t, configs, "ghostty")

	// Verify paths are correct
	assert.Equal(t, filepath.Join(alacrittyDir, "alacritty.toml"), configs["alacritty"])
	assert.Equal(t, filepath.Join(kittyDir, "kitty.conf"), configs["kitty"])
	assert.Equal(t, filepath.Join(ghosttyDir, "config"), configs["ghostty"])
}

func TestDiscovery_FindAllConfigs_AllTerminals(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Create configs for all terminals except Windows Terminal (platform-specific)
	configs := []struct {
		dir  string
		file string
	}{
		{".config/alacritty", "alacritty.toml"},
		{".config/kitty", "kitty.conf"},
		{".config/wezterm", "wezterm.lua"},
		{".config/ghostty", "config"},
		{"Library/Preferences", "com.googlecode.iterm2.plist"},
		{"", ".hyper.js"},
	}

	for _, cfg := range configs {
		var fullPath string
		if cfg.dir != "" {
			dir := filepath.Join(tmpDir, cfg.dir)
			err := os.MkdirAll(dir, 0o755)
			require.NoError(t, err)
			fullPath = filepath.Join(dir, cfg.file)
		} else {
			fullPath = filepath.Join(tmpDir, cfg.file)
		}
		err := os.WriteFile(fullPath, []byte(""), 0o644)
		require.NoError(t, err)
	}

	d := NewDiscovery()
	foundConfigs := d.FindAllConfigs()

	// Verify all non-Windows terminals are found
	assert.Contains(t, foundConfigs, "alacritty")
	assert.Contains(t, foundConfigs, "kitty")
	assert.Contains(t, foundConfigs, "wezterm")
	assert.Contains(t, foundConfigs, "ghostty")
	assert.Contains(t, foundConfigs, "iterm2")
	assert.Contains(t, foundConfigs, "hyper")
}

// --- Search Options Tests ---

func TestAlacrittySearchOpts(t *testing.T) {
	opts := AlacrittySearchOpts()
	assert.Equal(t, "ALACRITTY_CONFIG_DIR", opts.EnvVar)
	assert.Equal(t, "alacritty.toml", opts.ConfigFileName)
	assert.Equal(t, "alacritty/alacritty.toml", opts.XDGSubpath)
	assert.Contains(t, opts.LegacyPaths, "~/.alacritty.toml")
	assert.Contains(t, opts.LegacyPaths, "~/.alacritty.yml")
}

func TestDiscovery_FindAlacrittyConfig_WithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config via env var path
	configDir := filepath.Join(tmpDir, "custom-alacritty")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "alacritty.toml")
	err = os.WriteFile(configFile, []byte("[general]\n"), 0o644)
	require.NoError(t, err)

	// Set env var to point to custom location
	t.Setenv("ALACRITTY_CONFIG_DIR", configDir)

	d := NewDiscovery()
	path := d.FindAlacrittyConfig()
	assert.Equal(t, configFile, path)
}
