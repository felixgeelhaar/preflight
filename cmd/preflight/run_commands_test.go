package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// runCompare
// ---------------------------------------------------------------------------

func TestRunCompare_NoArgs_ReturnsUsageError(t *testing.T) { //nolint:tparallel // modifies global flags
	saved := compareSecondConfigPath
	compareSecondConfigPath = ""
	defer func() { compareSecondConfigPath = saved }()

	err := runCompare(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage")
}

func TestRunCompare_TwoArgs_NonexistentConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	saved := compareConfigPath
	compareConfigPath = "/nonexistent/preflight.yaml"
	defer func() { compareConfigPath = saved }()

	err := runCompare(&cobra.Command{}, []string{"work", "personal"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load source config")
}

func TestRunCompare_SecondConfigPath_NoArgs(t *testing.T) { //nolint:tparallel // modifies global flags
	savedConfig := compareConfigPath
	savedSecond := compareSecondConfigPath
	compareConfigPath = "/nonexistent/source.yaml"
	compareSecondConfigPath = "/nonexistent/dest.yaml"
	defer func() {
		compareConfigPath = savedConfig
		compareSecondConfigPath = savedSecond
	}()

	err := runCompare(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load source config")
}

func TestRunCompare_OneArgWithSecondConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	savedConfig := compareConfigPath
	savedSecond := compareSecondConfigPath
	compareConfigPath = "/nonexistent/source.yaml"
	compareSecondConfigPath = "/nonexistent/dest.yaml"
	defer func() {
		compareConfigPath = savedConfig
		compareSecondConfigPath = savedSecond
	}()

	err := runCompare(&cobra.Command{}, []string{"default"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load source config")
}

// ---------------------------------------------------------------------------
// runDoctor
// ---------------------------------------------------------------------------

func TestRunDoctor_MissingConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	saved := cfgFile
	cfgFile = "/nonexistent/preflight.yaml"
	defer func() { cfgFile = saved }()

	err := runDoctor(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doctor check failed")
}

func TestRunDoctor_EmptyCfgFile_DefaultsToPreflightYaml(t *testing.T) { //nolint:tparallel // modifies global flags
	saved := cfgFile
	cfgFile = ""
	defer func() { cfgFile = saved }()

	// Running in a temp dir with no preflight.yaml should fail
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runDoctor(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doctor check failed")
}

// ---------------------------------------------------------------------------
// runExport
// ---------------------------------------------------------------------------

func TestRunExport_NonexistentConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	savedConfig := exportConfigPath
	savedTarget := exportTarget
	savedFormat := exportFormat
	exportConfigPath = "/nonexistent/preflight.yaml"
	exportTarget = "default"
	exportFormat = "yaml"
	defer func() {
		exportConfigPath = savedConfig
		exportTarget = savedTarget
		exportFormat = savedFormat
	}()

	err := runExport(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

// ---------------------------------------------------------------------------
// runClean
// ---------------------------------------------------------------------------

func TestRunClean_NonexistentConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	savedConfig := cleanConfigPath
	savedTarget := cleanTarget
	cleanConfigPath = "/nonexistent/preflight.yaml"
	cleanTarget = "default"
	defer func() {
		cleanConfigPath = savedConfig
		cleanTarget = savedTarget
	}()

	err := runClean(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

// ---------------------------------------------------------------------------
// runHistory
// ---------------------------------------------------------------------------

func TestRunHistory_EmptyHistoryDir(t *testing.T) { //nolint:tparallel // modifies HOME and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedJSON := historyJSON
	savedProvider := historyProvider
	savedSince := historySince
	savedLimit := historyLimit
	historyJSON = false
	historyProvider = ""
	historySince = ""
	historyLimit = 20
	defer func() {
		historyJSON = savedJSON
		historyProvider = savedProvider
		historySince = savedSince
		historyLimit = savedLimit
	}()

	output := captureStdout(t, func() {
		err := runHistory(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No history entries found")
}

func TestRunHistory_WithEntries(t *testing.T) { //nolint:tparallel // modifies HOME, globals, and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save a history entry
	entry := HistoryEntry{
		ID:      "test-run-history-001",
		Command: "apply",
		Status:  "success",
	}
	require.NoError(t, SaveHistoryEntry(entry))

	savedJSON := historyJSON
	savedProvider := historyProvider
	savedSince := historySince
	savedLimit := historyLimit
	historyJSON = false
	historyProvider = ""
	historySince = ""
	historyLimit = 20
	defer func() {
		historyJSON = savedJSON
		historyProvider = savedProvider
		historySince = savedSince
		historyLimit = savedLimit
	}()

	output := captureStdout(t, func() {
		err := runHistory(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "Showing 1 entries")
}

func TestRunHistory_InvalidSinceDuration(t *testing.T) { //nolint:tparallel // modifies globals
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a history entry so loadHistory succeeds
	entry := HistoryEntry{
		ID:      "since-test-001",
		Command: "apply",
		Status:  "success",
	}
	require.NoError(t, SaveHistoryEntry(entry))

	savedSince := historySince
	savedProvider := historyProvider
	savedLimit := historyLimit
	savedJSON := historyJSON
	historySince = "invalid"
	historyProvider = ""
	historyLimit = 20
	historyJSON = false
	defer func() {
		historySince = savedSince
		historyProvider = savedProvider
		historyLimit = savedLimit
		historyJSON = savedJSON
	}()

	err := runHistory(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestRunHistory_JSONOutput(t *testing.T) { //nolint:tparallel // modifies HOME, globals, and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:      "json-hist-001",
		Command: "doctor",
		Status:  "success",
	}
	require.NoError(t, SaveHistoryEntry(entry))

	savedJSON := historyJSON
	savedSince := historySince
	savedProvider := historyProvider
	savedLimit := historyLimit
	historyJSON = true
	historySince = ""
	historyProvider = ""
	historyLimit = 20
	defer func() {
		historyJSON = savedJSON
		historySince = savedSince
		historyProvider = savedProvider
		historyLimit = savedLimit
	}()

	output := captureStdout(t, func() {
		err := runHistory(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	// JSON output should contain the entry ID
	assert.Contains(t, output, "json-hist-001")
	assert.Contains(t, output, "doctor")
}

func TestRunHistory_FilterByProvider(t *testing.T) { //nolint:tparallel // modifies HOME, globals, and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry1 := HistoryEntry{
		ID:      "prov-filter-001",
		Command: "apply",
		Status:  "success",
		Changes: []Change{{Provider: "brew", Action: "install", Item: "git"}},
	}
	entry2 := HistoryEntry{
		ID:      "prov-filter-002",
		Command: "apply",
		Status:  "success",
		Changes: []Change{{Provider: "apt", Action: "install", Item: "curl"}},
	}
	require.NoError(t, SaveHistoryEntry(entry1))
	require.NoError(t, SaveHistoryEntry(entry2))

	savedJSON := historyJSON
	savedProvider := historyProvider
	savedSince := historySince
	savedLimit := historyLimit
	historyJSON = false
	historyProvider = "brew"
	historySince = ""
	historyLimit = 20
	defer func() {
		historyJSON = savedJSON
		historyProvider = savedProvider
		historySince = savedSince
		historyLimit = savedLimit
	}()

	output := captureStdout(t, func() {
		err := runHistory(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	// Should only show 1 entry (filtered to brew provider)
	assert.Contains(t, output, "Showing 1 entries")
}

// ---------------------------------------------------------------------------
// runRollback
// ---------------------------------------------------------------------------

func TestRunRollback_NoSnapshots(t *testing.T) { //nolint:tparallel // modifies HOME, globals, and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedTo := rollbackTo
	savedLatest := rollbackLatest
	rollbackTo = ""
	rollbackLatest = false
	defer func() {
		rollbackTo = savedTo
		rollbackLatest = savedLatest
	}()

	output := captureStdout(t, func() {
		err := runRollback(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No snapshots available")
}

// ---------------------------------------------------------------------------
// runCapture
// ---------------------------------------------------------------------------

func TestRunCapture_NonexistentProvider(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedAll := captureAll
	savedProvider := captureProvider
	savedOutput := captureOutput
	savedTarget := captureTarget
	savedYes := yesFlag
	captureAll = true
	captureProvider = "nonexistent-provider-xyz"
	captureOutput = tmpDir
	captureTarget = "default"
	yesFlag = false
	defer func() {
		captureAll = savedAll
		captureProvider = savedProvider
		captureOutput = savedOutput
		captureTarget = savedTarget
		yesFlag = savedYes
	}()

	captureStdout(t, func() {
		err := runCapture(&cobra.Command{}, nil)
		// runCapture may succeed with "No items found" or fail with "capture failed"
		if err != nil {
			assert.Contains(t, err.Error(), "capture failed")
		}
	})
}

// ---------------------------------------------------------------------------
// runAnalyze
// ---------------------------------------------------------------------------

func TestRunAnalyze_NoLayerFiles(t *testing.T) { //nolint:tparallel // modifies globals and cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedJSON := analyzeJSON
	savedTools := analyzeTools
	savedNoAI := analyzeNoAI
	analyzeJSON = false
	analyzeTools = false
	analyzeNoAI = true
	defer func() {
		analyzeJSON = savedJSON
		analyzeTools = savedTools
		analyzeNoAI = savedNoAI
	}()

	output := captureStdout(t, func() {
		err := runAnalyze(&cobra.Command{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no layers found")
	})

	assert.Contains(t, output, "No layer files found")
}

func TestRunAnalyze_NoLayerFiles_JSON(t *testing.T) { //nolint:tparallel // modifies globals and cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedJSON := analyzeJSON
	savedTools := analyzeTools
	savedNoAI := analyzeNoAI
	analyzeJSON = true
	analyzeTools = false
	analyzeNoAI = true
	defer func() {
		analyzeJSON = savedJSON
		analyzeTools = savedTools
		analyzeNoAI = savedNoAI
	}()

	output := captureStdout(t, func() {
		err := runAnalyze(&cobra.Command{}, nil)
		require.Error(t, err)
	})

	// JSON output should contain the error
	assert.Contains(t, output, "no layers found")
}

func TestRunAnalyze_WithLayerFile(t *testing.T) { //nolint:tparallel // modifies globals and cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create a layers directory with a simple layer
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	layerContent := `name: base
packages:
  brew:
    formulae:
      - ripgrep
      - fd
`
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(layerContent), 0o644))

	savedJSON := analyzeJSON
	savedTools := analyzeTools
	savedNoAI := analyzeNoAI
	savedQuiet := analyzeQuiet
	analyzeJSON = false
	analyzeTools = false
	analyzeNoAI = true
	analyzeQuiet = false
	defer func() {
		analyzeJSON = savedJSON
		analyzeTools = savedTools
		analyzeNoAI = savedNoAI
		analyzeQuiet = savedQuiet
	}()

	output := captureStdout(t, func() {
		err := runAnalyze(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Layer Analysis Report")
	assert.Contains(t, output, "base")
}

func TestRunAnalyze_ToolsMode_NoLayers(t *testing.T) { //nolint:tparallel // modifies globals and cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	savedTools := analyzeTools
	savedJSON := analyzeJSON
	savedNoAI := analyzeNoAI
	analyzeTools = true
	analyzeJSON = false
	analyzeNoAI = true
	defer func() {
		analyzeTools = savedTools
		analyzeJSON = savedJSON
		analyzeNoAI = savedNoAI
	}()

	err := runAnalyze(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no layers found")
}

func TestRunAnalyze_ToolsMode_WithToolArgs(t *testing.T) { //nolint:tparallel // modifies globals
	savedTools := analyzeTools
	savedJSON := analyzeJSON
	savedNoAI := analyzeNoAI
	savedAI := analyzeAI
	savedQuiet := analyzeQuiet
	analyzeTools = true
	analyzeJSON = false
	analyzeNoAI = true
	analyzeAI = false
	analyzeQuiet = false
	defer func() {
		analyzeTools = savedTools
		analyzeJSON = savedJSON
		analyzeNoAI = savedNoAI
		analyzeAI = savedAI
		analyzeQuiet = savedQuiet
	}()

	output := captureStdout(t, func() {
		err := runAnalyze(&cobra.Command{}, []string{"ripgrep", "fd"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Tool Configuration Analysis")
}

func TestRunAnalyze_WithSpecificLayerPath(t *testing.T) { //nolint:tparallel // modifies globals and cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create a layer file in a layers/ directory
	layerContent := `name: test-layer
packages:
  brew:
    formulae:
      - git
`
	layerPath := filepath.Join(tmpDir, "layers", "test.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(layerPath), 0o755))
	require.NoError(t, os.WriteFile(layerPath, []byte(layerContent), 0o644))

	savedJSON := analyzeJSON
	savedTools := analyzeTools
	savedNoAI := analyzeNoAI
	analyzeJSON = false
	analyzeTools = false
	analyzeNoAI = true
	defer func() {
		analyzeJSON = savedJSON
		analyzeTools = savedTools
		analyzeNoAI = savedNoAI
	}()

	output := captureStdout(t, func() {
		err := runAnalyze(&cobra.Command{}, []string{layerPath})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "test-layer")
}

// ---------------------------------------------------------------------------
// runProfileList
// ---------------------------------------------------------------------------

func TestRunProfileList_NonexistentConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedConfig := profileConfigPath
	profileConfigPath = "/nonexistent/preflight.yaml"
	defer func() { profileConfigPath = savedConfig }()

	err := runProfileList(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

// ---------------------------------------------------------------------------
// runProfileSwitch
// ---------------------------------------------------------------------------

func TestRunProfileSwitch_NonexistentConfig(t *testing.T) { //nolint:tparallel // modifies global flags
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedConfig := profileConfigPath
	profileConfigPath = "/nonexistent/preflight.yaml"
	defer func() { profileConfigPath = savedConfig }()

	output := captureStdout(t, func() {
		err := runProfileSwitch(&cobra.Command{}, []string{"work"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	assert.Contains(t, output, "Switching to profile: work")
}

// ---------------------------------------------------------------------------
// runPluginUpgrade
// ---------------------------------------------------------------------------

func TestRunPluginUpgrade_NoPluginsInstalled(t *testing.T) { //nolint:tparallel // modifies HOME and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := captureStdout(t, func() {
		err := runPluginUpgrade("")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
}

// ---------------------------------------------------------------------------
// runPluginList
// ---------------------------------------------------------------------------

func TestRunPluginList_NoPlugins(t *testing.T) { //nolint:tparallel // modifies HOME and stdout
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := captureStdout(t, func() {
		err := runPluginList()
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
}

// ---------------------------------------------------------------------------
// runSyncConflicts
// ---------------------------------------------------------------------------

func TestRunSyncConflicts_NotInGitRepo(t *testing.T) { //nolint:tparallel // modifies cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runSyncConflicts(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

// ---------------------------------------------------------------------------
// runSyncResolve
// ---------------------------------------------------------------------------

func TestRunSyncResolve_NotInGitRepo(t *testing.T) { //nolint:tparallel // modifies cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	err := runSyncResolve(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

// ---------------------------------------------------------------------------
// getRemoteLockfilePath
// ---------------------------------------------------------------------------

func TestGetRemoteLockfilePath_NotGitRepo(t *testing.T) { //nolint:tparallel // reads filesystem
	tmpDir := t.TempDir()

	// Using a non-git directory should fail getting branch
	_, err := getRemoteLockfilePath(tmpDir, "origin", "preflight.lock")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// runDiscover
// ---------------------------------------------------------------------------

func TestRunDiscover_FailsGracefully(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	// Discovery depends on `gh` CLI. Without it or network, it should fail.
	savedMaxRepos := discoverMaxRepos
	savedMinStars := discoverMinStars
	discoverMaxRepos = 1
	discoverMinStars = 1
	defer func() {
		discoverMaxRepos = savedMaxRepos
		discoverMinStars = savedMinStars
	}()

	output := captureStdout(t, func() {
		err := runDiscover(&cobra.Command{}, nil)
		// Expected to fail with "discovery failed" when gh is not available
		if err != nil {
			assert.Contains(t, err.Error(), "discovery failed")
		}
	})

	// The function prints "Analyzing popular dotfile repositories..." before the error
	assert.Contains(t, output, "Analyzing popular dotfile repositories")
}

// ---------------------------------------------------------------------------
// runMCP - verify command and flags exist (cannot test stdio blocking)
// ---------------------------------------------------------------------------

func TestMCPCmd_Exists(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "mcp" {
			found = true
			break
		}
	}
	assert.True(t, found, "mcp command should be registered on root")
}

func TestMCPCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"http", "config", "target"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := mcpCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestMCPCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"http", ""},
		{"config", "preflight.yaml"},
		{"target", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			f := mcpCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

// ---------------------------------------------------------------------------
// runSecurity - exercise the --list-scanners path
// ---------------------------------------------------------------------------

func TestRunSecurity_ListScanners(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	savedListScanners := securityListIgnore
	securityListIgnore = true
	defer func() { securityListIgnore = savedListScanners }()

	output := captureStdout(t, func() {
		err := runSecurity(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available security scanners")
}

// ---------------------------------------------------------------------------
// Outdated command flag tests
// ---------------------------------------------------------------------------

func TestOutdatedCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"all", "fail-on", "ignore", "json", "quiet", "upgrade", "major", "dry-run"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := outdatedCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestOutdatedCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"all", "false"},
		{"fail-on", "minor"},
		{"json", "false"},
		{"quiet", "false"},
		{"upgrade", "false"},
		{"major", "false"},
		{"dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			f := outdatedCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

// ---------------------------------------------------------------------------
// Deprecated command flag tests
// ---------------------------------------------------------------------------

func TestDeprecatedCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"ignore", "json", "quiet"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := deprecatedCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

// ---------------------------------------------------------------------------
// Cleanup command flag tests
// ---------------------------------------------------------------------------

func TestCleanupCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"remove", "autoremove", "all", "dry-run", "json", "quiet", "ignore", "keep", "no-orphans", "no-overlaps"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := cleanupCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestCleanupCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"autoremove", "false"},
		{"all", "false"},
		{"dry-run", "false"},
		{"json", "false"},
		{"quiet", "false"},
		{"no-orphans", "false"},
		{"no-overlaps", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			f := cleanupCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

// ---------------------------------------------------------------------------
// compareConfigs
// ---------------------------------------------------------------------------

func TestCompareConfigs_NoDifferences(t *testing.T) {
	t.Parallel()

	source := map[string]interface{}{
		"brew": map[string]interface{}{"formulae": []interface{}{"git"}},
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{"formulae": []interface{}{"git"}},
	}

	diffs := compareConfigs(source, dest, nil)
	assert.Empty(t, diffs)
}

func TestCompareConfigs_WithDifferences(t *testing.T) {
	t.Parallel()

	source := map[string]interface{}{
		"brew":  map[string]interface{}{"formulae": []interface{}{"git"}},
		"files": "some-value",
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{"formulae": []interface{}{"git", "fd"}},
		"ssh":  "new-section",
	}

	diffs := compareConfigs(source, dest, nil)
	assert.NotEmpty(t, diffs)

	// Should detect: files removed, ssh added, brew changed
	typeMap := make(map[string]bool)
	for _, d := range diffs {
		typeMap[d.Type] = true
	}
	assert.True(t, typeMap["removed"], "should detect removed provider")
	assert.True(t, typeMap["added"], "should detect added provider")
}

func TestCompareConfigs_WithProviderFilter(t *testing.T) {
	t.Parallel()

	source := map[string]interface{}{
		"brew":  "v1",
		"files": "v1",
	}
	dest := map[string]interface{}{
		"brew":  "v2",
		"files": "v2",
	}

	diffs := compareConfigs(source, dest, []string{"brew"})
	// Should only report changes for brew, not files
	for _, d := range diffs {
		assert.Equal(t, "brew", d.Provider)
	}
}

// ---------------------------------------------------------------------------
// compareProviderConfig
// ---------------------------------------------------------------------------

func TestCompareProviderConfig_NonMapValues(t *testing.T) {
	t.Parallel()

	diffs := compareProviderConfig("test", "value1", "value2")
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
}

func TestCompareProviderConfig_EqualNonMaps(t *testing.T) {
	t.Parallel()

	diffs := compareProviderConfig("test", "same", "same")
	assert.Empty(t, diffs)
}

func TestCompareProviderConfig_MapDifferences(t *testing.T) {
	t.Parallel()

	source := map[string]interface{}{
		"name":    "Alice",
		"email":   "alice@example.com",
		"removed": "gone",
	}
	dest := map[string]interface{}{
		"name":  "Bob",
		"email": "alice@example.com",
		"added": "new",
	}

	diffs := compareProviderConfig("git", source, dest)
	assert.NotEmpty(t, diffs)

	typeCount := map[string]int{}
	for _, d := range diffs {
		typeCount[d.Type]++
	}
	assert.Equal(t, 1, typeCount["changed"], "name changed")
	assert.Equal(t, 1, typeCount["removed"], "removed field gone")
	assert.Equal(t, 1, typeCount["added"], "added field new")
}

// ---------------------------------------------------------------------------
// outputCompareJSON
// ---------------------------------------------------------------------------

func TestRunCmd_OutputCompareJSON_EmptyDiffs(t *testing.T) {
	output := captureStdout(t, func() {
		err := outputCompareJSON(nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "[]")
}

func TestRunCmd_OutputCompareJSON_WithDiffs(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Source: nil, Dest: "fd"},
	}

	output := captureStdout(t, func() {
		err := outputCompareJSON(diffs)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "added")
}

// ---------------------------------------------------------------------------
// outputCompareText
// ---------------------------------------------------------------------------

func TestRunCmd_OutputCompareText_NoDiffs(t *testing.T) {
	output := captureStdout(t, func() {
		outputCompareText("source", "dest", nil)
	})

	assert.Contains(t, output, "No differences")
}

func TestRunCmd_OutputCompareText_WithDiffs(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Source: nil, Dest: "fd"},
		{Provider: "git", Key: "", Type: "removed", Source: "config", Dest: nil},
	}

	output := captureStdout(t, func() {
		outputCompareText("work", "personal", diffs)
	})

	assert.Contains(t, output, "Comparing work")
	assert.Contains(t, output, "personal")
	assert.Contains(t, output, "Total: 2 difference(s)")
}

// ---------------------------------------------------------------------------
// findLayerFiles
// ---------------------------------------------------------------------------

func TestRunCmd_FindLayerFiles_WithYamlAndYml(t *testing.T) { //nolint:tparallel // modifies cwd
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "extra.yml"), []byte("name: extra"), 0o644))

	paths, err := findLayerFiles()
	require.NoError(t, err)
	assert.Len(t, paths, 2)
}

// ---------------------------------------------------------------------------
// extractAllTools
// ---------------------------------------------------------------------------

func TestExtractAllTools_EmptyLayer(t *testing.T) { //nolint:tparallel // reads filesystem
	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "empty.yaml")
	require.NoError(t, os.WriteFile(layerPath, []byte("name: empty\n"), 0o644))

	tools, err := extractAllTools([]string{layerPath})
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestExtractAllTools_WithTools(t *testing.T) { //nolint:tparallel // reads filesystem
	tmpDir := t.TempDir()
	content := `packages:
  brew:
    formulae:
      - ripgrep
      - fd
runtime:
  tools:
    go: "1.22"
    node: "20"
shell:
  plugins:
    - zsh-autosuggestions
`
	layerPath := filepath.Join(tmpDir, "tools.yaml")
	require.NoError(t, os.WriteFile(layerPath, []byte(content), 0o644))

	tools, err := extractAllTools([]string{layerPath})
	require.NoError(t, err)
	// Should include: ripgrep, fd, go, node, zsh-autosuggestions (5 total)
	assert.Len(t, tools, 5)
}

func TestExtractAllTools_NonexistentFile(t *testing.T) { //nolint:tparallel // reads filesystem
	_, err := extractAllTools([]string{"/nonexistent/layer.yaml"})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// handleRemove (cleanup.go) - dry run mode
// ---------------------------------------------------------------------------

func TestHandleRemove_DryRun(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	cleanupDryRun = true
	cleanupJSON = false
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := handleRemove(nil, nil, []string{"test-package"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "test-package")
}

func TestHandleRemove_DryRun_JSON(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	cleanupDryRun = true
	cleanupJSON = true
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := handleRemove(nil, nil, []string{"pkg1", "pkg2"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "pkg1")
	assert.Contains(t, output, "pkg2")
	assert.Contains(t, output, "dry_run")
}

// ---------------------------------------------------------------------------
// handleCleanupAll - dry run mode with empty removal list
// ---------------------------------------------------------------------------

func TestHandleCleanupAll_EmptyRedundancies(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	cleanupDryRun = false
	cleanupJSON = false
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	result := &security.RedundancyResult{
		Checker:      "test",
		Redundancies: nil,
	}

	output := captureStdout(t, func() {
		err := handleCleanupAll(nil, nil, result)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Nothing to clean up")
}

func TestHandleCleanupAll_DryRun(t *testing.T) { //nolint:tparallel // modifies globals and stdout
	savedDryRun := cleanupDryRun
	savedJSON := cleanupJSON
	cleanupDryRun = true
	cleanupJSON = false
	defer func() {
		cleanupDryRun = savedDryRun
		cleanupJSON = savedJSON
	}()

	result := &security.RedundancyResult{
		Checker: "test",
		Redundancies: security.Redundancies{
			{
				Type:     security.RedundancyDuplicate,
				Packages: []string{"go", "go@1.22"},
				Remove:   []string{"go@1.22"},
			},
		},
	}

	output := captureStdout(t, func() {
		err := handleCleanupAll(nil, nil, result)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "go@1.22")
}

// ---------------------------------------------------------------------------
// Export format helpers
// ---------------------------------------------------------------------------

func TestExportToNix_EmptyConfig(t *testing.T) {
	t.Parallel()

	data, err := exportToNix(map[string]interface{}{})
	require.NoError(t, err)
	assert.Contains(t, string(data), "Generated by preflight export")
}

func TestExportToNix_WithBrewAndGit(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"ripgrep", "fd"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
		"shell": map[string]interface{}{
			"shell":   "zsh",
			"plugins": []interface{}{"zsh-autosuggestions"},
		},
	}

	data, err := exportToNix(config)
	require.NoError(t, err)
	output := string(data)
	assert.Contains(t, output, "home.packages")
	assert.Contains(t, output, "programs.git")
	assert.Contains(t, output, "programs.zsh")
}

func TestExportToBrewfile_EmptyConfig(t *testing.T) {
	t.Parallel()

	data, err := exportToBrewfile(map[string]interface{}{})
	require.NoError(t, err)
	assert.Contains(t, string(data), "Generated by preflight export")
}

func TestExportToBrewfile_WithPackages(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"ripgrep"},
			"casks":    []interface{}{"iterm2"},
		},
	}

	data, err := exportToBrewfile(config)
	require.NoError(t, err)
	output := string(data)
	assert.Contains(t, output, `tap "homebrew/cask"`)
	assert.Contains(t, output, `brew "ripgrep"`)
	assert.Contains(t, output, `cask "iterm2"`)
}

func TestExportToShell_EmptyConfig(t *testing.T) {
	t.Parallel()

	data, err := exportToShell(map[string]interface{}{})
	require.NoError(t, err)
	output := string(data)
	assert.Contains(t, output, "#!/usr/bin/env bash")
	assert.Contains(t, output, "Setup complete!")
}

func TestExportToShell_WithPackages(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"felixgeelhaar/tap"},
			"formulae": []interface{}{"ripgrep", "fd"},
			"casks":    []interface{}{"firefox"},
		},
		"git": map[string]interface{}{
			"name":  "Test",
			"email": "test@example.com",
		},
	}

	data, err := exportToShell(config)
	require.NoError(t, err)
	output := string(data)
	assert.Contains(t, output, "brew tap felixgeelhaar/tap")
	assert.Contains(t, output, "brew install")
	assert.Contains(t, output, "ripgrep")
	assert.Contains(t, output, "brew install --cask")
	assert.Contains(t, output, "firefox")
	assert.Contains(t, output, `git config --global user.name "Test"`)
}

// ---------------------------------------------------------------------------
// Compare command flag tests
// ---------------------------------------------------------------------------

func TestCompareCmd_HasAllFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"config", "config2", "remote", "providers", "json", "verbose"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := compareCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

func TestCompareCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"config", "preflight.yaml"},
		{"config2", ""},
		{"remote", ""},
		{"providers", ""},
		{"json", "false"},
		{"verbose", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			f := compareCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

// ---------------------------------------------------------------------------
// Security command flag tests
// ---------------------------------------------------------------------------

func TestSecurityCmd_HasAllFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"path", "scanner", "severity", "fail-on", "ignore", "json", "quiet", "list-scanners"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := securityCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

// ---------------------------------------------------------------------------
// Clean command flag tests
// ---------------------------------------------------------------------------

func TestCleanCmd_HasAllFlags(t *testing.T) {
	t.Parallel()

	flags := []string{"config", "target", "apply", "providers", "ignore", "json", "force"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			f := cleanCmd.Flags().Lookup(flag)
			assert.NotNil(t, f, "flag %s should exist", flag)
		})
	}
}

// ---------------------------------------------------------------------------
// Export command flag defaults
// ---------------------------------------------------------------------------

func TestExportCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"config", "preflight.yaml"},
		{"target", "default"},
		{"format", "yaml"},
		{"output", ""},
		{"flatten", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			t.Parallel()
			f := exportCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

// ---------------------------------------------------------------------------
// formatAge (rollback.go)
// ---------------------------------------------------------------------------

func TestFormatAge_AllRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ago      time.Duration
		expected string
	}{
		{"just_now", 10 * time.Second, "just now"},
		{"1_min", 90 * time.Second, "1 min ago"},
		{"5_mins", 5 * time.Minute, "5 mins ago"},
		{"1_hour", time.Hour + time.Minute, "1 hour ago"},
		{"3_hours", 3*time.Hour + time.Minute, "3 hours ago"},
		{"1_day", 25 * time.Hour, "1 day ago"},
		{"3_days", 73 * time.Hour, "3 days ago"},
		{"1_week", 8 * 24 * time.Hour, "1 week ago"},
		{"3_weeks", 22 * 24 * time.Hour, "3 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatAge(time.Now().Add(-tt.ago))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// parseUpdateType (outdated.go)
// ---------------------------------------------------------------------------

func TestParseUpdateType_AllValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"major", "major"},
		{"minor", "minor"},
		{"patch", "patch"},
		{"Major", "major"},
		{"MINOR", "minor"},
		{"unknown", "minor"}, // defaults to minor
		{"", "minor"},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_value", func(t *testing.T) {
			t.Parallel()
			result := parseUpdateType(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

// ---------------------------------------------------------------------------
// JSON output helpers
// ---------------------------------------------------------------------------

func TestRunCmd_OutputCleanupJSON_NoError(t *testing.T) {
	output := captureStdout(t, func() {
		outputCleanupJSON(nil, nil, nil)
	})

	// Should produce valid JSON even with nil inputs
	assert.Contains(t, output, "{")
}

func TestRunCmd_OutputDeprecatedJSON_NilResult(t *testing.T) {
	output := captureStdout(t, func() {
		outputDeprecatedJSON(nil, nil)
	})

	assert.Contains(t, output, "{")
}

// ---------------------------------------------------------------------------
// printJSONOutput (sync_conflicts.go)
// ---------------------------------------------------------------------------

func TestPrintJSONOutput_ValidOutput(t *testing.T) {
	output := captureStdout(t, func() {
		err := printJSONOutput(ConflictsOutputJSON{
			Relation:        "equal (in sync)",
			TotalConflicts:  0,
			AutoResolvable:  0,
			ManualConflicts: []ConflictJSON{},
			NeedsMerge:      false,
		})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "equal (in sync)")
	assert.Contains(t, output, "needs_merge")
}

// ---------------------------------------------------------------------------
// loadLayerInfos
// ---------------------------------------------------------------------------

func TestRunCmd_LoadLayerInfos_ValidLayer(t *testing.T) { //nolint:tparallel // reads filesystem
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	content := `name: test
packages:
  brew:
    formulae:
      - ripgrep
git:
  name: Test
ssh:
  config: test
shell:
  framework: oh-my-zsh
nvim:
  preset: kickstart
vscode:
  extensions:
    - ms-python.python
`
	layerPath := filepath.Join(layersDir, "test.yaml")
	require.NoError(t, os.WriteFile(layerPath, []byte(content), 0o644))

	layers, err := loadLayerInfos([]string{layerPath})
	require.NoError(t, err)
	require.Len(t, layers, 1)

	assert.Equal(t, "test", layers[0].Name)
	assert.Equal(t, []string{"ripgrep"}, layers[0].Packages)
	assert.True(t, layers[0].HasGitConfig)
	assert.True(t, layers[0].HasSSHConfig)
	assert.True(t, layers[0].HasShellConfig)
	assert.True(t, layers[0].HasEditorConfig)
}

// ---------------------------------------------------------------------------
// readLayerFile
// ---------------------------------------------------------------------------

func TestReadLayerFile_Nonexistent(t *testing.T) {
	t.Parallel()

	_, err := readLayerFile("/nonexistent/path/layer.yaml")
	require.Error(t, err)
}

func TestReadLayerFile_ValidFile(t *testing.T) { //nolint:tparallel // reads filesystem
	tmpDir := t.TempDir()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	path := filepath.Join(layersDir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: test"), 0o644))

	data, err := readLayerFile(path)
	require.NoError(t, err)
	assert.Equal(t, "name: test", string(data))
}

// ---------------------------------------------------------------------------
// parseAIToolInsights
// ---------------------------------------------------------------------------

func TestRunCmd_ParseAIToolInsights_EmptyInsights(t *testing.T) {
	t.Parallel()

	content := `{"insights": []}`
	findings := parseAIToolInsights(content)
	assert.Empty(t, findings)
}

func TestRunCmd_ParseAIToolInsights_MultipleSeverities(t *testing.T) {
	t.Parallel()

	content := `{
  "insights": [
    {"type": "rec", "severity": "warning", "tools": ["a"], "message": "warn msg", "suggestion": "fix"},
    {"type": "rec", "severity": "error", "tools": ["b"], "message": "err msg", "suggestion": "fix2"},
    {"type": "rec", "severity": "info", "tools": ["c"], "message": "info msg", "suggestion": "fix3"}
  ]
}`

	findings := parseAIToolInsights(content)
	require.Len(t, findings, 3)
	assert.Equal(t, security.SeverityWarning, findings[0].Severity)
	assert.Equal(t, security.SeverityError, findings[1].Severity)
	assert.Equal(t, security.SeverityInfo, findings[2].Severity)
}

// ---------------------------------------------------------------------------
// validateLayerPath
// ---------------------------------------------------------------------------

func TestValidateLayerPath_InvalidPath(t *testing.T) {
	t.Parallel()

	err := validateLayerPath("../../../etc/passwd")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// collectEvaluatedItems
// ---------------------------------------------------------------------------

func TestRunCmd_CollectEvaluatedItems_Nil(t *testing.T) {
	t.Parallel()

	items := collectEvaluatedItems(nil)
	assert.Nil(t, items)
}

// ---------------------------------------------------------------------------
// outputComplianceError
// ---------------------------------------------------------------------------

func TestRunCmd_OutputComplianceError(t *testing.T) {
	output := captureStdout(t, func() {
		outputComplianceError(assert.AnError)
	})

	assert.Contains(t, output, "error")
}

// ---------------------------------------------------------------------------
// getPatternIcon (discover.go) - default case
// ---------------------------------------------------------------------------

func TestRunCmd_GetPatternIcon_Default(t *testing.T) {
	t.Parallel()

	result := getPatternIcon("unknown-type")
	assert.Equal(t, "\u2022", result) // bullet character for default case
}

// ---------------------------------------------------------------------------
// formatCategory (cleanup.go)
// ---------------------------------------------------------------------------

func TestRunCmd_FormatCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"version_duplicates", "Version Duplicates"},
		{"orphan", "Orphan"},
		{"overlapping_tools", "Overlapping Tools"},
		{"", ""},
		{"single", "Single"},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_category", func(t *testing.T) {
			t.Parallel()
			result := formatCategory(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
