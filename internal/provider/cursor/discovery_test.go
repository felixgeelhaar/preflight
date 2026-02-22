package cursor_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/provider/cursor"
	"github.com/felixgeelhaar/preflight/internal/testutil/mocks"
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

			d := cursor.NewDiscoveryWithOS(tt.goos)
			assert.NotNil(t, d)
		})
	}
}

// =============================================================================
// SearchOpts Tests
// =============================================================================

func TestSearchOpts(t *testing.T) {
	t.Parallel()

	opts := cursor.SearchOpts()

	assert.Equal(t, "CURSOR_PORTABLE", opts.EnvVar)
	assert.Equal(t, "data/user-data/User", opts.ConfigFileName)
	assert.NotEmpty(t, opts.MacOSPaths)
	assert.NotEmpty(t, opts.LinuxPaths)
	assert.NotEmpty(t, opts.WindowsPaths)
}

// =============================================================================
// FindConfigDir Tests
// =============================================================================

func TestDiscovery_FindConfigDir_PortableEnv(t *testing.T) {
	tmpDir := t.TempDir()

	portableDir := filepath.Join(tmpDir, "portable")
	userDir := filepath.Join(portableDir, "data", "user-data", "User")
	require.NoError(t, os.MkdirAll(userDir, 0o755))

	t.Setenv("CURSOR_PORTABLE", portableDir)
	t.Setenv("HOME", tmpDir)

	d := cursor.NewDiscovery()
	result := d.FindConfigDir()

	assert.Equal(t, userDir, result)
}

func TestDiscovery_FindConfigDir_PortableEnv_DirNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("CURSOR_PORTABLE", filepath.Join(tmpDir, "nonexistent"))
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := cursor.NewDiscovery()
	result := d.FindConfigDir()
	assert.NotEmpty(t, result)
}

func TestDiscovery_FindConfigDir_PlatformSpecific(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("CURSOR_PORTABLE", "")
	t.Setenv("HOME", tmpDir)

	var expectedDir string
	switch runtime.GOOS {
	case "darwin":
		expectedDir = filepath.Join(tmpDir, "Library", "Application Support", "Cursor", "User")
	case "linux":
		expectedDir = filepath.Join(tmpDir, ".config", "Cursor", "User")
	default:
		t.Setenv("APPDATA", tmpDir)
		expectedDir = filepath.Join(tmpDir, "Cursor", "User")
	}
	require.NoError(t, os.MkdirAll(expectedDir, 0o755))

	d := cursor.NewDiscovery()
	result := d.FindConfigDir()

	assert.Equal(t, expectedDir, result)
}

func TestDiscovery_FindConfigDir_FallsThroughToBestPractice(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("CURSOR_PORTABLE", "")
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := cursor.NewDiscovery()
	result := d.FindConfigDir()

	expected := d.BestPracticePath()
	assert.Equal(t, expected, result)
}

// =============================================================================
// BestPracticePath Tests
// =============================================================================

func TestDiscovery_BestPracticePath_Darwin(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := cursor.NewDiscoveryWithOS("darwin")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join("Library", "Application Support", "Cursor", "User"))
}

func TestDiscovery_BestPracticePath_Linux(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := cursor.NewDiscoveryWithOS("linux")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join(".config", "Cursor", "User"))
}

func TestDiscovery_BestPracticePath_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))

	d := cursor.NewDiscoveryWithOS("windows")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join("Cursor", "User"))
}

func TestDiscovery_BestPracticePath_WindowsNoAPPDATA(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", "")

	d := cursor.NewDiscoveryWithOS("windows")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join("AppData", "Roaming", "Cursor", "User"))
}

func TestDiscovery_BestPracticePath_UnknownOS(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	d := cursor.NewDiscoveryWithOS("freebsd")
	result := d.BestPracticePath()

	assert.Contains(t, result, filepath.Join(".config", "Cursor", "User"))
}

// =============================================================================
// FindSettingsPath Tests
// =============================================================================

func TestDiscovery_FindSettingsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("CURSOR_PORTABLE", "")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := cursor.NewDiscovery()
	result := d.FindSettingsPath()

	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, "settings.json", filepath.Base(result))
}

// =============================================================================
// FindKeybindingsPath Tests
// =============================================================================

func TestDiscovery_FindKeybindingsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("CURSOR_PORTABLE", "")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	}

	d := cursor.NewDiscovery()
	result := d.FindKeybindingsPath()

	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, "keybindings.json", filepath.Base(result))
}

// =============================================================================
// GetCandidatePaths Tests
// =============================================================================

func TestDiscovery_GetCandidatePaths(t *testing.T) {
	t.Parallel()

	d := cursor.NewDiscovery()
	paths := d.GetCandidatePaths()

	assert.NotEmpty(t, paths)
}

// =============================================================================
// Additional Step Edge Cases
// =============================================================================

func TestExtensionStep_Check_RunnerError(t *testing.T) {
	t.Parallel()

	runner := mocks.NewCommandRunner()
	runner.AddError("cursor", []string{"--list-extensions"}, errors.New("command execution failed"))

	step := cursor.NewExtensionStep("ms-python.python", runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	assert.Error(t, err)
	assert.Equal(t, compiler.StatusUnknown, status)
}

func TestSettingsStep_Check_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	configPath := getCursorConfigPathForTest(tmpDir)
	require.NoError(t, os.MkdirAll(configPath, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(configPath, "settings.json"),
		[]byte("{invalid json content}"),
		0o644,
	))

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Check_MissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	configPath := getCursorConfigPathForTest(tmpDir)
	require.NoError(t, os.MkdirAll(configPath, 0o755))

	existingSettings := map[string]interface{}{"editor.tabSize": 4}
	data, _ := json.Marshal(existingSettings)
	require.NoError(t, os.WriteFile(
		filepath.Join(configPath, "settings.json"),
		data,
		0o644,
	))

	runner := mocks.NewCommandRunner()
	settings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(settings, runner)

	ctx := compiler.NewRunContext(context.Background())
	status, err := step.Check(ctx)

	require.NoError(t, err)
	assert.Equal(t, compiler.StatusNeedsApply, status)
}

func TestSettingsStep_Apply_MergesWithExisting(t *testing.T) {
	tmpDir := t.TempDir()
	setupCursorConfigPath(t, tmpDir)

	configPath := getCursorConfigPathForTest(tmpDir)
	require.NoError(t, os.MkdirAll(configPath, 0o755))
	existingSettings := map[string]interface{}{"editor.tabSize": 2}
	data, _ := json.Marshal(existingSettings)
	require.NoError(t, os.WriteFile(
		filepath.Join(configPath, "settings.json"),
		data,
		0o644,
	))

	runner := mocks.NewCommandRunner()
	newSettings := map[string]interface{}{"editor.fontSize": 14}
	step := cursor.NewSettingsStep(newSettings, runner)

	ctx := compiler.NewRunContext(context.Background())
	err := step.Apply(ctx)
	require.NoError(t, err)

	written, err := os.ReadFile(filepath.Join(configPath, "settings.json"))
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(written, &result))
	assert.InDelta(t, float64(14), result["editor.fontSize"], 0.001)
	assert.InDelta(t, float64(2), result["editor.tabSize"], 0.001)
}
