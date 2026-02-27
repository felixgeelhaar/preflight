package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// cleanup.go - handleRemove (dry-run path, no checker needed)
// ===========================================================================

func TestBoostD_HandleRemove_DryRun_Text(t *testing.T) {
	// Save and restore flags
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = true
	cleanupJSON = false

	checker := security.NewBrewRedundancyChecker()
	packages := []string{"go@1.24", "node@18"}

	out := captureStdout(t, func() {
		err := handleRemove(context.Background(), checker, packages)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Would remove:")
	assert.Contains(t, out, "go@1.24")
	assert.Contains(t, out, "node@18")
}

func TestBoostD_HandleRemove_DryRun_JSON(t *testing.T) {
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = true
	cleanupJSON = true

	checker := security.NewBrewRedundancyChecker()
	packages := []string{"go@1.24"}

	out := captureStdout(t, func() {
		err := handleRemove(context.Background(), checker, packages)
		require.NoError(t, err)
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.NotNil(t, parsed.Cleanup)
	assert.True(t, parsed.Cleanup.DryRun)
	assert.Contains(t, parsed.Cleanup.Removed, "go@1.24")
}

// ===========================================================================
// cleanup.go - handleCleanupAll (dry-run paths)
// ===========================================================================

func TestBoostD_HandleCleanupAll_DryRun_Text(t *testing.T) {
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = true
	cleanupJSON = false

	checker := security.NewBrewRedundancyChecker()
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:   security.RedundancyDuplicate,
				Remove: []string{"go@1.24"},
			},
		},
	}

	out := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), checker, result)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Would remove 1 package(s)")
	assert.Contains(t, out, "go@1.24")
}

func TestBoostD_HandleCleanupAll_DryRun_JSON(t *testing.T) {
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = true
	cleanupJSON = true

	checker := security.NewBrewRedundancyChecker()
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:   security.RedundancyDuplicate,
				Remove: []string{"go@1.24", "node@18"},
			},
		},
	}

	out := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), checker, result)
		require.NoError(t, err)
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	require.NotNil(t, parsed.Cleanup)
	assert.True(t, parsed.Cleanup.DryRun)
	assert.Len(t, parsed.Cleanup.Removed, 2)
}

func TestBoostD_HandleCleanupAll_NothingToRemove_Text(t *testing.T) {
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = false
	cleanupJSON = false

	checker := security.NewBrewRedundancyChecker()
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:   security.RedundancyOverlap,
				Remove: []string{}, // empty - nothing to remove
			},
		},
	}

	out := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), checker, result)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Nothing to clean up")
}

func TestBoostD_HandleCleanupAll_NothingToRemove_JSON(t *testing.T) {
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	cleanupDryRun = false
	cleanupJSON = true

	checker := security.NewBrewRedundancyChecker()
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:   security.RedundancyOverlap,
				Remove: []string{},
			},
		},
	}

	out := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), checker, result)
		require.NoError(t, err)
	})

	var parsed security.CleanupResultJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
}

// ===========================================================================
// watch.go - runWatch (with injectable deps)
// ===========================================================================

func TestBoostD_RunWatch_InvalidDebounce(t *testing.T) {
	savedDebounce := watchDebounce
	defer func() { watchDebounce = savedDebounce }()

	watchDebounce = "not-a-duration"

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runWatch(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid debounce duration")
}

func TestBoostD_RunWatch_NoConfigFile(t *testing.T) {
	savedDebounce := watchDebounce
	defer func() { watchDebounce = savedDebounce }()

	watchDebounce = "500ms"

	// Create a temp dir without preflight.yaml
	tmpDir := t.TempDir()

	// Save and restore working dir
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err = runWatch(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no preflight.yaml found")
}

// ===========================================================================
// outdated.go - outputUpgradeJSON edge cases
// ===========================================================================

func TestBoostD_OutputUpgradeJSON_NilResult(t *testing.T) {
	t.Parallel()

	// When result is nil and error is non-nil
	out := captureStdout(t, func() {
		outputUpgradeJSON(nil, fmt.Errorf("checker not available"))
	})

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "checker not available", parsed["error"])
	assert.False(t, parsed["dry_run"].(bool))
}

// ===========================================================================
// deprecated.go - printDeprecatedTable edge cases
// ===========================================================================

func TestBoostD_PrintDeprecatedTable_EmptyVersionAndMessage(t *testing.T) {
	t.Parallel()

	packages := security.DeprecatedPackages{
		{Name: "pkg1", Version: "", Reason: security.ReasonDisabled, Message: ""},
	}

	out := captureStdout(t, func() {
		printDeprecatedTable(packages)
	})

	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "-") // empty version replaced with "-"
}

// ===========================================================================
// compliance.go - outputComplianceJSON edge case
// ===========================================================================

func TestBoostD_OutputComplianceError_FormatsJSON(t *testing.T) {
	t.Parallel()

	out := captureStdout(t, func() {
		outputComplianceError(fmt.Errorf("policy load failed"))
	})

	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, "policy load failed", parsed["error"])
}

// ===========================================================================
// env.go - runEnvSet and runEnvUnset (file I/O tests)
// ===========================================================================

func TestBoostD_RunEnvSet_CreatesLayerFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layers dir
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "layers"), 0o755))

	// Create a base layer file
	baseLayer := filepath.Join(tmpDir, "layers", "base.yaml")
	require.NoError(t, os.WriteFile(baseLayer, []byte("name: base\n"), 0o644))

	// Save and restore flags
	savedLayer := envLayer
	savedPath := envConfigPath
	defer func() {
		envLayer = savedLayer
		envConfigPath = savedPath
	}()
	envLayer = "base"
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err := runEnvSet(cmd, []string{"EDITOR", "nvim"})
	require.NoError(t, err)
}

func TestBoostD_RunEnvUnset_RemovesFromLayer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create layers dir
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "layers"), 0o755))

	// Create a base layer file with an env var
	baseLayer := filepath.Join(tmpDir, "layers", "base.yaml")
	require.NoError(t, os.WriteFile(baseLayer, []byte("name: base\nenv:\n  EDITOR: nvim\n"), 0o644))

	savedLayer := envLayer
	savedPath := envConfigPath
	defer func() {
		envLayer = savedLayer
		envConfigPath = savedPath
	}()
	envLayer = "base"
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err := runEnvUnset(cmd, []string{"EDITOR"})
	require.NoError(t, err)
}


// ===========================================================================
// init.go - runInitNonInteractive (early error paths)
// ===========================================================================

func TestBoostD_RunInitNonInteractive_NoPreset(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Save and restore flags
	savedPreset := initPreset
	defer func() { initPreset = savedPreset }()
	initPreset = ""

	err = runInitNonInteractive(filepath.Join(tmpDir, "preflight.yaml"))
	// Should error because no preset specified
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--preset is required")
}

func TestBoostD_RunInitNonInteractive_Success(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedPreset := initPreset
	defer func() { initPreset = savedPreset }()
	initPreset = "minimal"

	out := captureStdout(t, func() {
		err := runInitNonInteractive(filepath.Join(tmpDir, "preflight.yaml"))
		require.NoError(t, err)
	})

	assert.Contains(t, out, "preflight.yaml")

	// Verify files were created
	_, err = os.Stat(filepath.Join(tmpDir, "preflight.yaml"))
	assert.NoError(t, err)
}

// ===========================================================================
// sync.go - findRepoRoot (requires git but we test error path)
// ===========================================================================

func TestBoostD_FindRepoRoot_NotInRepo(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	_, err = findRepoRoot()
	// Should fail since tmpDir is not a git repo
	require.Error(t, err)
}


// ===========================================================================
// env.go - runEnvExport and runEnvDiff branches
// ===========================================================================

func TestBoostD_RunEnvExport_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedPath := envConfigPath
	defer func() { envConfigPath = savedPath }()
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err = runEnvExport(cmd, nil)
	require.Error(t, err)
}

func TestBoostD_RunEnvDiff_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Save and restore flag
	savedPath := envConfigPath
	defer func() { envConfigPath = savedPath }()
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err = runEnvDiff(cmd, []string{"default", "work"})
	require.Error(t, err)
}

func TestBoostD_RunEnvGet_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedPath := envConfigPath
	defer func() { envConfigPath = savedPath }()
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err = runEnvGet(cmd, []string{"EDITOR"})
	require.Error(t, err)
}

func TestBoostD_RunEnvList_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedPath := envConfigPath
	defer func() { envConfigPath = savedPath }()
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")

	cmd := &cobra.Command{}
	err = runEnvList(cmd, nil)
	require.Error(t, err)
}

// ===========================================================================
// catalog.go - getRegistry
// ===========================================================================

func TestBoostD_GetRegistry_LoadsBuiltin(t *testing.T) {
	t.Parallel()

	reg, err := getRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg)
}
