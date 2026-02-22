package helix_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/helix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewDiscoveryWithOS Tests
// =============================================================================

func TestNewDiscoveryWithOS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
	}{
		{name: "darwin", goos: "darwin"},
		{name: "linux", goos: "linux"},
		{name: "windows", goos: "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := helix.NewDiscoveryWithOS(tt.goos)
			assert.NotNil(t, d)
		})
	}
}

// =============================================================================
// SearchOpts Tests
// =============================================================================

func TestSearchOpts(t *testing.T) {
	t.Parallel()

	opts := helix.SearchOpts()

	assert.Equal(t, "HELIX_CONFIG_DIR", opts.EnvVar)
	assert.Equal(t, "config.toml", opts.ConfigFileName)
	assert.Equal(t, "helix", opts.XDGSubpath)
	assert.NotEmpty(t, opts.MacOSPaths)
	assert.NotEmpty(t, opts.LinuxPaths)
	assert.NotEmpty(t, opts.WindowsPaths)
}

// =============================================================================
// FindConfigDir Tests
// =============================================================================

func TestDiscovery_FindConfigDir_HelixConfigDirEnv(t *testing.T) {
	tmpDir := t.TempDir()

	helixDir := filepath.Join(tmpDir, "custom-helix")
	require.NoError(t, os.MkdirAll(helixDir, 0o755))

	t.Setenv("HELIX_CONFIG_DIR", helixDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := helix.NewDiscovery()
	result := d.FindConfigDir()

	assert.Equal(t, helixDir, result)
}

func TestDiscovery_FindConfigDir_HelixConfigDirEnv_NotExist(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HELIX_CONFIG_DIR", filepath.Join(tmpDir, "nonexistent"))
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	d := helix.NewDiscovery()
	result := d.FindConfigDir()
	// Should fall through to XDG or platform path
	assert.NotEmpty(t, result)
}

func TestDiscovery_FindConfigDir_XDGConfigHome(t *testing.T) {
	tmpDir := t.TempDir()

	xdgDir := filepath.Join(tmpDir, "custom-xdg")
	helixDir := filepath.Join(xdgDir, "helix")
	require.NoError(t, os.MkdirAll(helixDir, 0o755))

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", tmpDir)

	d := helix.NewDiscovery()
	result := d.FindConfigDir()

	assert.Equal(t, helixDir, result)
}

func TestDiscovery_FindConfigDir_XDGDefault(t *testing.T) {
	tmpDir := t.TempDir()

	// Create default XDG path
	helixDir := filepath.Join(tmpDir, ".config", "helix")
	require.NoError(t, os.MkdirAll(helixDir, 0o755))

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpDir)

	d := helix.NewDiscovery()
	result := d.FindConfigDir()

	assert.Equal(t, helixDir, result)
}

func TestDiscovery_FindConfigDir_PlatformFallback_DarwinLinux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Test for darwin/linux only")
	}

	tmpDir := t.TempDir()

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "empty-xdg"))
	t.Setenv("HOME", tmpDir)

	d := helix.NewDiscovery()
	result := d.FindConfigDir()

	assert.Contains(t, result, filepath.Join(".config", "helix"))
}

func TestDiscovery_FindConfigDir_WindowsFallback(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "empty-xdg"))
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))

	d := helix.NewDiscoveryWithOS("windows")
	result := d.FindConfigDir()

	assert.Contains(t, result, "helix")
}

func TestDiscovery_FindConfigDir_WindowsFallback_NoAPPDATA(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "empty-xdg"))
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", "")

	d := helix.NewDiscoveryWithOS("windows")
	result := d.FindConfigDir()

	assert.Contains(t, result, filepath.Join("AppData", "Roaming", "helix"))
}

func TestDiscovery_FindConfigDir_UnknownOS(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "empty-xdg"))
	t.Setenv("HOME", tmpDir)

	d := helix.NewDiscoveryWithOS("freebsd")
	result := d.FindConfigDir()

	assert.Contains(t, result, filepath.Join(".config", "helix"))
}

// =============================================================================
// BestPracticePath Tests
// =============================================================================

func TestDiscovery_BestPracticePath_DarwinLinux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			d := helix.NewDiscoveryWithOS(goos)
			result := d.BestPracticePath()

			assert.Contains(t, result, filepath.Join(".config", "helix"))
		})
	}
}

func TestDiscovery_BestPracticePath_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))

	d := helix.NewDiscoveryWithOS("windows")
	result := d.BestPracticePath()

	assert.Contains(t, result, "helix")
}

func TestDiscovery_BestPracticePath_WindowsNoAPPDATA(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", "")

	d := helix.NewDiscoveryWithOS("windows")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join("AppData", "Roaming", "helix"))
}

func TestDiscovery_BestPracticePath_UnknownOS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := helix.NewDiscoveryWithOS("freebsd")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join(".config", "helix"))
}

// =============================================================================
// FindConfigPath Tests
// =============================================================================

func TestDiscovery_FindConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := helix.NewDiscovery()
	result := d.FindConfigPath()

	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, "config.toml", filepath.Base(result))
}

// =============================================================================
// FindLanguagesPath Tests
// =============================================================================

func TestDiscovery_FindLanguagesPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := helix.NewDiscovery()
	result := d.FindLanguagesPath()

	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, "languages.toml", filepath.Base(result))
}

// =============================================================================
// FindThemesDir Tests
// =============================================================================

func TestDiscovery_FindThemesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("HELIX_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := helix.NewDiscovery()
	result := d.FindThemesDir()

	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, "themes", filepath.Base(result))
}

// =============================================================================
// GetCandidatePaths Tests
// =============================================================================

func TestDiscovery_GetCandidatePaths(t *testing.T) {
	t.Parallel()

	d := helix.NewDiscovery()
	paths := d.GetCandidatePaths()

	assert.NotEmpty(t, paths)
}
