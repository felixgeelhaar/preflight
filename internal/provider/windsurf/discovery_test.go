package windsurf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/provider/windsurf"
	"github.com/stretchr/testify/assert"
)

func TestDiscovery_FindConfigDir_Portable(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()

	// Create portable installation directory structure
	portableUserDir := filepath.Join(tmpDir, "data", "user-data", "User")
	err := os.MkdirAll(portableUserDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create portable dir: %v", err)
	}

	t.Setenv("WINDSURF_PORTABLE", tmpDir)

	discovery := windsurf.NewDiscovery()
	configDir := discovery.FindConfigDir()

	assert.Equal(t, portableUserDir, configDir)
}

func TestDiscovery_FindConfigDir_PlatformDefault(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()

	// Clear portable env var
	t.Setenv("WINDSURF_PORTABLE", "")
	t.Setenv("HOME", tmpDir)

	// Create platform-specific config directory
	var expectedPath string
	switch runtime.GOOS {
	case "darwin":
		expectedPath = filepath.Join(tmpDir, "Library", "Application Support", "Windsurf", "User")
	case "linux":
		expectedPath = filepath.Join(tmpDir, ".config", "Windsurf", "User")
	default: // windows
		t.Setenv("APPDATA", tmpDir)
		expectedPath = filepath.Join(tmpDir, "Windsurf", "User")
	}

	err := os.MkdirAll(expectedPath, 0o755)
	if err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	discovery := windsurf.NewDiscovery()
	configDir := discovery.FindConfigDir()

	assert.Equal(t, expectedPath, configDir)
}

func TestDiscovery_FindConfigDir_FallbackToBestPractice(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()

	// Clear portable env var
	t.Setenv("WINDSURF_PORTABLE", "")
	t.Setenv("HOME", tmpDir)

	// Don't create any config directory - should return best practice path

	var expectedPath string
	switch runtime.GOOS {
	case "darwin":
		expectedPath = filepath.Join(tmpDir, "Library", "Application Support", "Windsurf", "User")
	case "linux":
		expectedPath = filepath.Join(tmpDir, ".config", "Windsurf", "User")
	default: // windows
		t.Setenv("APPDATA", tmpDir)
		expectedPath = filepath.Join(tmpDir, "Windsurf", "User")
	}

	discovery := windsurf.NewDiscovery()
	configDir := discovery.FindConfigDir()

	assert.Equal(t, expectedPath, configDir)
}

func TestDiscovery_BestPracticePath(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	var expectedPath string
	switch runtime.GOOS {
	case "darwin":
		expectedPath = filepath.Join(tmpDir, "Library", "Application Support", "Windsurf", "User")
	case "linux":
		expectedPath = filepath.Join(tmpDir, ".config", "Windsurf", "User")
	default: // windows
		t.Setenv("APPDATA", tmpDir)
		expectedPath = filepath.Join(tmpDir, "Windsurf", "User")
	}

	discovery := windsurf.NewDiscovery()
	path := discovery.BestPracticePath()

	assert.Equal(t, expectedPath, path)
}

func TestDiscovery_FindSettingsPath(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("WINDSURF_PORTABLE", "")

	discovery := windsurf.NewDiscovery()
	settingsPath := discovery.FindSettingsPath()

	assert.Contains(t, settingsPath, "settings.json")
}

func TestDiscovery_FindKeybindingsPath(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("WINDSURF_PORTABLE", "")

	discovery := windsurf.NewDiscovery()
	keybindingsPath := discovery.FindKeybindingsPath()

	assert.Contains(t, keybindingsPath, "keybindings.json")
}

func TestNewDiscoveryWithOS(t *testing.T) {
	t.Parallel()

	// Test macOS discovery
	darwinDiscovery := windsurf.NewDiscoveryWithOS("darwin")
	assert.NotNil(t, darwinDiscovery)

	// Test Linux discovery
	linuxDiscovery := windsurf.NewDiscoveryWithOS("linux")
	assert.NotNil(t, linuxDiscovery)

	// Test Windows discovery
	windowsDiscovery := windsurf.NewDiscoveryWithOS("windows")
	assert.NotNil(t, windowsDiscovery)
}

func TestWindsurfSearchOpts(t *testing.T) {
	t.Parallel()

	opts := windsurf.WindsurfSearchOpts()

	assert.Equal(t, "WINDSURF_PORTABLE", opts.EnvVar)
	assert.Contains(t, opts.ConfigFileName, "data/user-data/User")
	assert.NotEmpty(t, opts.MacOSPaths)
	assert.NotEmpty(t, opts.LinuxPaths)
	assert.NotEmpty(t, opts.WindowsPaths)
}
