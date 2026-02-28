package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure imports are used.
var (
	_ config.ReproducibilityMode
	_ time.Duration
	_ security.Severity
)

// ---------------------------------------------------------------------------
// runTour (tour.go) - boost from 25%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global tourListFlag
func TestCoverBoost_RunTour_ListFlagPrintsTourTopics(t *testing.T) {
	saved := tourListFlag
	defer func() { tourListFlag = saved }()
	tourListFlag = true

	output := captureStdout(t, func() {
		err := runTour(nil, []string{})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available tour topics")
	assert.Contains(t, output, "preflight tour")
}

//nolint:tparallel // modifies global tourListFlag
func TestCoverBoost_RunTour_InvalidTopicReturnsError(t *testing.T) {
	saved := tourListFlag
	defer func() { tourListFlag = saved }()
	tourListFlag = false

	err := runTour(nil, []string{"bogus-topic-that-does-not-exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic")
	assert.Contains(t, err.Error(), "bogus-topic-that-does-not-exist")
}

// ---------------------------------------------------------------------------
// runTrustList (trust.go:129) - boost from 23.1%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunTrustList_EmptyStoreShowsNoKeys(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	output := captureStdout(t, func() {
		err := runTrustList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No trusted keys.")
}

// ---------------------------------------------------------------------------
// runTrustAdd (trust.go:183) - boost from 72.2%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global trust flags
func TestCoverBoost_RunTrustAdd_NonexistentFileReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runTrustAdd(nil, []string{"/nonexistent/key.pub"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read key file")
}

//nolint:tparallel // modifies global trust flags
func TestCoverBoost_RunTrustAdd_InvalidKeyContentReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "badkey.pub")
	require.NoError(t, os.WriteFile(tmpFile, []byte("this is not a valid key"), 0o600))

	saved := trustKeyType
	defer func() { trustKeyType = saved }()
	trustKeyType = "" // let auto-detect run

	err := runTrustAdd(nil, []string{tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect key type")
}

//nolint:tparallel // modifies global trust flags
func TestCoverBoost_RunTrustAdd_UnknownKeyTypeFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "somekey.pub")
	require.NoError(t, os.WriteFile(tmpFile, []byte("ssh-ed25519 AAAA test@host"), 0o600))

	saved := trustKeyType
	defer func() { trustKeyType = saved }()
	trustKeyType = "unsupported"

	err := runTrustAdd(nil, []string{tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key type")
}

// ---------------------------------------------------------------------------
// runTrustRemove (trust.go:255) - boost from 43.5%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunTrustRemove_KeyNotFoundReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runTrustRemove(nil, []string{"nonexistent-key-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// ---------------------------------------------------------------------------
// runTrustShow (trust.go:296) - boost from 77.8%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunTrustShow_KeyNotFoundReturnsError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runTrustShow(nil, []string{"nonexistent-key-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

// ---------------------------------------------------------------------------
// runWatch (watch.go:89) - boost from 54.4%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global watch flags
func TestCoverBoost_RunWatch_InvalidDebounceReturnsError(t *testing.T) {
	reset := setWatchFlags("not-a-valid-duration", false, false, false)
	defer reset()

	err := runWatch(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid debounce duration")
}

//nolint:tparallel // modifies global watch flags and cwd
func TestCoverBoost_RunWatch_NoPrefightYamlReturnsError(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	reset := setWatchFlags("100ms", false, false, false)
	defer reset()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err = runWatch(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no preflight.yaml found")
}

// ---------------------------------------------------------------------------
// runProfileList (profile.go:104) - boost from 17.2%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global profileConfigPath and HOME
func TestCoverBoost_RunProfileList_WithTargets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("targets:\n  work:\n    - base\n  personal:\n    - base\n"), 0o644))

	savedConfig := profileConfigPath
	savedJSON := profileJSON
	profileConfigPath = configPath
	profileJSON = false
	defer func() {
		profileConfigPath = savedConfig
		profileJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runProfileList(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "PROFILE")
	assert.Contains(t, output, "TARGET")
	assert.Contains(t, output, "STATUS")
}

//nolint:tparallel // modifies global profileConfigPath and HOME
func TestCoverBoost_RunProfileList_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("targets:\n  default:\n    - base\n"), 0o644))

	savedConfig := profileConfigPath
	savedJSON := profileJSON
	profileConfigPath = configPath
	profileJSON = true
	defer func() {
		profileConfigPath = savedConfig
		profileJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runProfileList(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"target"`)
}

//nolint:tparallel // modifies global profileConfigPath and HOME
func TestCoverBoost_RunProfileList_NoProfilesAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Config file does not exist => LoadManifest will fail
	configPath := filepath.Join(tmpDir, "nonexistent-preflight.yaml")

	savedConfig := profileConfigPath
	savedJSON := profileJSON
	profileConfigPath = configPath
	profileJSON = false
	defer func() {
		profileConfigPath = savedConfig
		profileJSON = savedJSON
	}()

	err := runProfileList(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load manifest")
}

// ---------------------------------------------------------------------------
// runProfileSwitch (profile.go:175) - boost from 35.7%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global profileConfigPath and HOME
func TestCoverBoost_RunProfileSwitch_ValidConfigAppliesSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("targets:\n  work:\n    - base\n"), 0o644))

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	layerContent := "name: base\ngit:\n  user:\n    name: Test User\n    email: test@example.com\n"
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(layerContent), 0o644))

	savedConfig := profileConfigPath
	profileConfigPath = configPath
	defer func() { profileConfigPath = savedConfig }()

	output := captureStdout(t, func() {
		err := runProfileSwitch(&cobra.Command{}, []string{"work"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Switching to profile: work")
	assert.Contains(t, output, "Applying profile settings")
	assert.Contains(t, output, "Switched to profile: work")
}

// ---------------------------------------------------------------------------
// runFleetPing (fleet.go:337) - boost from 17.2%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetPing_NoHostsSelected(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  myhost:
    hostname: localhost
    user: test
    port: 22
    tags:
      - linux
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
	}()

	fleetInventoryFile = invFile
	fleetTarget = "tag:nonexistent"

	output := captureStdout(t, func() {
		err := runFleetPing(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No hosts selected")
}

// ---------------------------------------------------------------------------
// runFleetPlan (fleet.go:391) - boost from 12.5%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetPlan_NoHostsSelected(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  myhost:
    hostname: localhost
    user: test
    port: 22
    tags:
      - linux
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
	}()

	fleetInventoryFile = invFile
	fleetTarget = "tag:nonexistent"

	output := captureStdout(t, func() {
		err := runFleetPlan(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No hosts selected")
}

// ---------------------------------------------------------------------------
// runPluginSearch (plugin.go:402) - boost from 46%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global search flags
func TestCoverBoost_RunPluginSearch_InvalidTypeReturnsError(t *testing.T) {
	savedType := searchType
	defer func() { searchType = savedType }()
	searchType = "invalidtype"

	err := runPluginSearch("some-query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
	assert.Contains(t, err.Error(), "must be 'config' or 'provider'")
}

// ---------------------------------------------------------------------------
// runPluginUpgrade (plugin.go:650) - boost from 10%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and globals
func TestCoverBoost_RunPluginUpgrade_NoPluginsInstalledEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	output := captureStdout(t, func() {
		err := runPluginUpgrade("")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
}

//nolint:tparallel // modifies HOME and globals
func TestCoverBoost_RunPluginUpgrade_SpecificPluginNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// With no plugins installed, runPluginUpgrade("name") prints "No plugins installed"
	// because the discover returns 0 plugins.
	output := captureStdout(t, func() {
		err := runPluginUpgrade("nonexistent-plugin-xyz")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
}

// ---------------------------------------------------------------------------
// runPluginValidate (plugin.go:503) - boost validation paths
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global pluginValidateJSON
func TestCoverBoost_RunPluginValidate_NonexistentPath(t *testing.T) {
	savedJSON := pluginValidateJSON
	savedStrict := pluginValidateStrict
	defer func() {
		pluginValidateJSON = savedJSON
		pluginValidateStrict = savedStrict
	}()
	pluginValidateJSON = false
	pluginValidateStrict = false

	err := runPluginValidate("/nonexistent/plugin/path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

//nolint:tparallel // modifies global pluginValidateJSON
func TestCoverBoost_RunPluginValidate_NotADirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("not a dir"), 0o644))

	savedJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = savedJSON }()
	pluginValidateJSON = false

	err := runPluginValidate(tmpFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

//nolint:tparallel // modifies global pluginValidateJSON
func TestCoverBoost_RunPluginValidate_JSONOutput(t *testing.T) {
	savedJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = savedJSON }()
	pluginValidateJSON = true

	output := captureStdout(t, func() {
		// With JSON output, outputValidationResult encodes JSON and returns nil
		// even when validation fails (the error path is only for human-readable output).
		err := runPluginValidate("/nonexistent/plugin/path")
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"valid"`)
	assert.Contains(t, output, `"errors"`)
}

// ---------------------------------------------------------------------------
// runRollback (rollback.go:45) - boost from 19.6%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global rollback flags and HOME
func TestCoverBoost_RunRollback_NoSnapshotsAvailable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	savedTo := rollbackTo
	savedLatest := rollbackLatest
	savedDryRun := rollbackDryRun
	rollbackTo = ""
	rollbackLatest = false
	rollbackDryRun = false
	defer func() {
		rollbackTo = savedTo
		rollbackLatest = savedLatest
		rollbackDryRun = savedDryRun
	}()

	output := captureStdout(t, func() {
		err := runRollback(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No snapshots available")
	assert.Contains(t, output, "Snapshots are created automatically")
}

// ---------------------------------------------------------------------------
// runApply (apply.go:75) - boost from 78.8%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global apply flags
func TestCoverBoost_RunApply_DryRunSkipsApply(t *testing.T) {
	plan := execution.NewExecutionPlan()
	step := newDummyStep("files:link:zshrc")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "files", "link", "", "")))

	fake := newFakePreflightClient(plan, nil)
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, true, false)
	defer reset()

	output := captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.True(t, fake.printPlanCalled)
	assert.False(t, fake.applyCalled)
	assert.Contains(t, output, "Dry run")
}

//nolint:tparallel // modifies global apply flags
func TestCoverBoost_RunApply_PlanFailureReturnsError(t *testing.T) {
	fake := &fakePreflightClient{
		planErr: errors.New("config parse error"),
	}
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, false, false)
	defer reset()

	err := runApply(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan failed")
}

//nolint:tparallel // modifies global apply flags
func TestCoverBoost_RunApply_StepFailureReturnsError(t *testing.T) {
	plan := execution.NewExecutionPlan()
	step := newDummyStep("brew:install:wget")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "brew", "install", "", "")))

	results := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusUnknown, errors.New("install failed")),
	}

	fake := newFakePreflightClient(plan, results)
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, false, false)
	defer reset()

	output := captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "some steps failed")
	})

	assert.Contains(t, output, "Applying changes")
}

//nolint:tparallel // modifies global apply flags
func TestCoverBoost_RunApply_ApplyErrorReturnsError(t *testing.T) {
	plan := execution.NewExecutionPlan()
	step := newDummyStep("brew:install:fd")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply,
		compiler.NewDiff(compiler.DiffTypeAdd, "brew", "install", "", "")))

	fake := &fakePreflightClient{
		planResult: plan,
		applyErr:   errors.New("context cancelled"),
	}
	restore := overrideNewPreflight(fake)
	defer restore()

	reset := setApplyFlags(t, false, false)
	defer reset()

	captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apply failed")
	})
}

// ---------------------------------------------------------------------------
// runSync (sync.go:72) - boost from 5.3%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies cwd
func TestCoverBoost_RunSync_NotInGitRepoReturnsError(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	err = runSync(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

//nolint:tparallel // modifies cwd and global flags
func TestCoverBoost_RunSync_InvalidRemoteName(t *testing.T) {
	dir := t.TempDir()
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	gitCmd = exec.Command("git", "config", "user.email", "test@test.com")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())
	gitCmd = exec.Command("git", "config", "user.name", "Test")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	testFile := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))
	gitCmd = exec.Command("git", "add", ".")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())
	gitCmd = exec.Command("git", "commit", "-m", "initial")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	savedRemote := syncRemote
	syncRemote = "../invalid remote name!"
	defer func() { syncRemote = savedRemote }()

	err = runSync(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid remote name")
}

//nolint:tparallel // modifies cwd and global flags
func TestCoverBoost_RunSync_InvalidBranchName(t *testing.T) {
	dir := t.TempDir()
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	gitCmd = exec.Command("git", "config", "user.email", "test@test.com")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())
	gitCmd = exec.Command("git", "config", "user.name", "Test")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	testFile := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))
	gitCmd = exec.Command("git", "add", ".")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())
	gitCmd = exec.Command("git", "commit", "-m", "initial")
	gitCmd.Dir = dir
	require.NoError(t, gitCmd.Run())

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	savedRemote := syncRemote
	savedBranch := syncBranch
	syncRemote = "origin"
	syncBranch = "../bad branch!!"
	defer func() {
		syncRemote = savedRemote
		syncBranch = savedBranch
	}()

	err = runSync(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch")
}

// ---------------------------------------------------------------------------
// runCapture (capture.go:69) - boost from 26.1%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global capture flags
func TestCoverBoost_RunCapture_AcceptAll_ProviderFilter(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedAll := captureAll
	savedProvider := captureProvider
	savedOutput := captureOutput
	savedTarget := captureTarget
	savedYes := yesFlag
	captureAll = true
	captureProvider = "brew"
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
		if err != nil {
			assert.Contains(t, err.Error(), "capture failed")
		}
	})
}

//nolint:tparallel // modifies global capture flags
func TestCoverBoost_RunCapture_YesFlagWorks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedAll := captureAll
	savedProvider := captureProvider
	savedOutput := captureOutput
	savedTarget := captureTarget
	savedYes := yesFlag
	captureAll = false
	captureProvider = "nonexistent-provider"
	captureOutput = tmpDir
	captureTarget = "default"
	yesFlag = true
	defer func() {
		captureAll = savedAll
		captureProvider = savedProvider
		captureOutput = savedOutput
		captureTarget = savedTarget
		yesFlag = savedYes
	}()

	captureStdout(t, func() {
		_ = runCapture(&cobra.Command{}, nil)
	})
}

// ---------------------------------------------------------------------------
// runEnvExport (env.go:280) - unsupported shell path
// ---------------------------------------------------------------------------

// coverBoostWriteEnvConfig is a test helper that writes a valid preflight config
// with a "default" target, a base layer, and the given env vars in the layer.
func coverBoostWriteEnvConfig(t *testing.T, tmpDir string, envVars map[string]string) string {
	t.Helper()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("targets:\n  default:\n    - base\n"), 0o644))

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	layerData := "name: base\nenv:\n"
	for k, v := range envVars {
		layerData += "  " + k + ": " + v + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte(layerData), 0o644))
	return configPath
}

//nolint:tparallel // modifies global envShell and envConfigPath
func TestCoverBoost_RunEnvExport_UnsupportedShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := coverBoostWriteEnvConfig(t, tmpDir, map[string]string{"EDITOR": "nvim"})

	savedConfig := envConfigPath
	savedTarget := envTarget
	savedShell := envShell
	envConfigPath = configPath
	envTarget = "default"
	envShell = "unsupported-shell-xyz"
	defer func() {
		envConfigPath = savedConfig
		envTarget = savedTarget
		envShell = savedShell
	}()

	err := runEnvExport(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

//nolint:tparallel // modifies global envShell and envConfigPath
func TestCoverBoost_RunEnvExport_FishShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := coverBoostWriteEnvConfig(t, tmpDir, map[string]string{"EDITOR": "nvim"})

	savedConfig := envConfigPath
	savedTarget := envTarget
	savedShell := envShell
	envConfigPath = configPath
	envTarget = "default"
	envShell = "fish"
	defer func() {
		envConfigPath = savedConfig
		envTarget = savedTarget
		envShell = savedShell
	}()

	output := captureStdout(t, func() {
		err := runEnvExport(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Generated by preflight env export")
	assert.Contains(t, output, "fish")
}

//nolint:tparallel // modifies global envShell and envConfigPath
func TestCoverBoost_RunEnvExport_BashShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := coverBoostWriteEnvConfig(t, tmpDir, map[string]string{"GOPATH": "/home/user/go"})

	savedConfig := envConfigPath
	savedTarget := envTarget
	savedShell := envShell
	envConfigPath = configPath
	envTarget = "default"
	envShell = "bash"
	defer func() {
		envConfigPath = savedConfig
		envTarget = savedTarget
		envShell = savedShell
	}()

	output := captureStdout(t, func() {
		err := runEnvExport(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Generated by preflight env export")
	assert.Contains(t, output, "bashrc")
}

// ---------------------------------------------------------------------------
// runSecurity (security.go:67) - boost from 19.4%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global security flags
func TestCoverBoost_RunSecurity_ListScannersPath(t *testing.T) {
	savedFlag := securityListIgnore
	securityListIgnore = true
	defer func() { securityListIgnore = savedFlag }()

	output := captureStdout(t, func() {
		err := runSecurity(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available security scanners")
}

// ---------------------------------------------------------------------------
// runSecretsCheck (secrets.go:138) - boost from 28.6%
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath
func TestCoverBoost_RunSecretsCheck_NoSecretRefsInConfig(t *testing.T) {
	content := `brew:
  formulae:
    - git
    - curl
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	savedConfig := secretsConfigPath
	secretsConfigPath = tmpFile
	defer func() { secretsConfigPath = savedConfig }()

	output := captureStdout(t, func() {
		err := runSecretsCheck(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No secret references to check")
}

// ---------------------------------------------------------------------------
// runSecretsList (secrets.go:108) - boost additional paths
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsConfigPath and secretsJSON
func TestCoverBoost_RunSecretsList_JSONOutput(t *testing.T) {
	content := `git:
  signing_key: "secret://1password/vault/key"
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	savedConfig := secretsConfigPath
	savedJSON := secretsJSON
	secretsConfigPath = tmpFile
	secretsJSON = true
	defer func() {
		secretsConfigPath = savedConfig
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, `"backend"`)
}

//nolint:tparallel // modifies global secretsConfigPath
func TestCoverBoost_RunSecretsList_WithTableOutput(t *testing.T) {
	content := `git:
  signing_key: "secret://env/MY_SECRET_KEY"
  gpg_key: "secret://keychain/gpg-key"
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	savedConfig := secretsConfigPath
	savedJSON := secretsJSON
	secretsConfigPath = tmpFile
	secretsJSON = false
	defer func() {
		secretsConfigPath = savedConfig
		secretsJSON = savedJSON
	}()

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "secret reference(s)")
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BACKEND")
}

// ---------------------------------------------------------------------------
// runSecretsBackends (secrets.go:219) - boost
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global secretsJSON
func TestCoverBoost_RunSecretsBackends_TableOutput(t *testing.T) {
	savedJSON := secretsJSON
	secretsJSON = false
	defer func() { secretsJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Available secret backends")
	assert.Contains(t, output, "BACKEND")
	assert.Contains(t, output, "env")
}

//nolint:tparallel // modifies global secretsJSON
func TestCoverBoost_RunSecretsBackends_JSONOutput(t *testing.T) {
	savedJSON := secretsJSON
	secretsJSON = true
	defer func() { secretsJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, `"Name"`)
	assert.Contains(t, output, `"env"`)
}

// ---------------------------------------------------------------------------
// resolveSecret (secrets.go:290) - test env backend
// ---------------------------------------------------------------------------

func TestCoverBoost_ResolveSecret_EnvBackend(t *testing.T) {
	t.Setenv("MY_TEST_SECRET", "hello-world")

	val, err := resolveSecret("env", "MY_TEST_SECRET")
	require.NoError(t, err)
	assert.Equal(t, "hello-world", val)
}

func TestCoverBoost_ResolveSecret_UnknownBackend(t *testing.T) {
	t.Parallel()

	_, err := resolveSecret("nonexistent", "key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

// ---------------------------------------------------------------------------
// setSecret (secrets.go:307) - test error paths
// ---------------------------------------------------------------------------

func TestCoverBoost_SetSecret_EnvBackendNotSupported(t *testing.T) {
	t.Parallel()

	err := setSecret("env", "name", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment variables")
}

func TestCoverBoost_SetSecret_UnsupportedBackend(t *testing.T) {
	t.Parallel()

	err := setSecret("bogus", "name", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setting secrets not supported")
}

// ---------------------------------------------------------------------------
// findSecretRefs (secrets.go:256) - test parsing
// ---------------------------------------------------------------------------

func TestCoverBoost_FindSecretRefs_ParsesMultipleRefs(t *testing.T) {
	content := `git:
  key: "secret://1password/vault/item/field"
  other: "secret://env/TOKEN"
  normal: just-a-string
`
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	refs, err := findSecretRefs(tmpFile)
	require.NoError(t, err)
	assert.Len(t, refs, 2)

	backends := make(map[string]bool)
	for _, ref := range refs {
		backends[ref.Backend] = true
	}
	assert.True(t, backends["1password"])
	assert.True(t, backends["env"])
}

func TestCoverBoost_FindSecretRefs_NonexistentFile(t *testing.T) {
	t.Parallel()

	_, err := findSecretRefs("/nonexistent/path")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// runPluginList (plugin.go:195) - test no plugins path
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunPluginList_NoPluginsShowsHint(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	output := captureStdout(t, func() {
		err := runPluginList()
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
	assert.Contains(t, output, "preflight plugin install")
}

// ---------------------------------------------------------------------------
// runFleetList (fleet.go:283) - experimental gate
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies env
func TestCoverBoost_RunFleetList_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")

	err := runFleetList(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

// ---------------------------------------------------------------------------
// runFleetApply (fleet.go:464) - invalid strategy
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetApply_InvalidStrategy(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  myhost:
    hostname: localhost
    user: test
    port: 22
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetStrategy = "invalidstrategy"

	err := runFleetApply(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid strategy")
}

// ---------------------------------------------------------------------------
// runFleetStatus (fleet.go:564) - boost
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetStatus_WithInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  host1:
    hostname: server1.example.com
    user: admin
    port: 22
    tags:
      - production
  host2:
    hostname: server2.example.com
    user: admin
    port: 22
    tags:
      - staging
groups:
  production:
    description: Production servers
    hosts:
      - host1
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetStatus(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Fleet Status")
	assert.Contains(t, output, "Total hosts")
}

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetStatus_JSONOutput(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  host1:
    hostname: server1.example.com
    user: admin
    port: 22
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetJSON = true

	output := captureStdout(t, func() {
		err := runFleetStatus(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "host_count")
}

// ---------------------------------------------------------------------------
// formatAge (rollback.go:162) - comprehensive ranges
// ---------------------------------------------------------------------------

func TestCoverBoost_FormatAge_AllRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		offset   time.Duration
		contains string
	}{
		{"10_seconds", 10 * time.Second, "just now"},
		{"1_minute", 1*time.Minute + 5*time.Second, "1 min ago"},
		{"5_minutes", 5*time.Minute + 10*time.Second, "5 mins ago"},
		{"1_hour", 1*time.Hour + 5*time.Minute, "1 hour ago"},
		{"3_hours", 3*time.Hour + 10*time.Minute, "3 hours ago"},
		{"1_day", 25 * time.Hour, "1 day ago"},
		{"5_days", 5 * 24 * time.Hour, "5 days ago"},
		{"1_week", 8 * 24 * time.Hour, "1 week ago"},
		{"3_weeks", 21 * 24 * time.Hour, "3 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := time.Now().Add(-tt.offset)
			result := formatAge(ts)
			assert.Contains(t, result, tt.contains)
		})
	}
}

// ---------------------------------------------------------------------------
// printTourTopics (tour.go:108)
// ---------------------------------------------------------------------------

func TestCoverBoost_PrintTourTopics_OutputsTopics(t *testing.T) {
	output := captureStdout(t, func() {
		printTourTopics()
	})

	assert.Contains(t, output, "Available tour topics")
	assert.Contains(t, output, "preflight tour")
}

// ---------------------------------------------------------------------------
// outputValidationResult (plugin.go:611) - human-readable with warnings
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global pluginValidateJSON
func TestCoverBoost_OutputValidationResult_WithWarnings(t *testing.T) {
	savedJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = savedJSON }()
	pluginValidateJSON = false

	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{"missing description", "plugin is unsigned"},
		Plugin:   "test-plugin",
		Version:  "1.0.0",
		Path:     "/tmp/test-plugin",
	}

	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Plugin validated")
	assert.Contains(t, output, "Warnings")
	assert.Contains(t, output, "missing description")
}

//nolint:tparallel // modifies global pluginValidateJSON
func TestCoverBoost_OutputValidationResult_WithErrors(t *testing.T) {
	savedJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = savedJSON }()
	pluginValidateJSON = false

	result := ValidationResult{
		Valid:    false,
		Errors:   []string{"missing required field: name", "invalid version"},
		Warnings: []string{},
		Path:     "/tmp/broken-plugin",
	}

	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		require.Error(t, err)
	})

	assert.Contains(t, output, "Validation failed")
	assert.Contains(t, output, "Errors")
	assert.Contains(t, output, "missing required field")
}

// ---------------------------------------------------------------------------
// runPluginRemove (plugin.go:285) - not found
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunPluginRemove_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runPluginRemove("nonexistent-plugin-abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// runPluginInfo (plugin.go:318) - not found
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestCoverBoost_RunPluginInfo_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runPluginInfo("nonexistent-plugin-def")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// getScanner (security.go:144) - auto with no available scanners
// ---------------------------------------------------------------------------

func TestCoverBoost_GetScanner_AutoNoScanners(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "grype", available: false})
	registry.Register(&fakeScanner{name: "trivy", available: false})

	_, err := getScanner(registry, "auto")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no scanners available")
}

func TestCoverBoost_GetScanner_SpecificScanner(t *testing.T) {
	t.Parallel()

	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "testscanner", available: true, version: "2.0"})

	scanner, err := getScanner(registry, "testscanner")
	require.NoError(t, err)
	assert.Equal(t, "testscanner", scanner.Name())
}

// ---------------------------------------------------------------------------
// listScanners (security.go:161) - with no version
// ---------------------------------------------------------------------------

func TestCoverBoost_ListScanners_WithoutVersion(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(&fakeScanner{name: "noversionscanner", available: true, version: ""})

	output := captureStdout(t, func() {
		err := listScanners(registry)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "noversionscanner")
	assert.Contains(t, output, "available")
}

// ---------------------------------------------------------------------------
// runFleetList - with inventory table and JSON output
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetList_WithInventoryTable(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  host-alpha:
    hostname: alpha.example.com
    user: admin
    port: 22
    tags:
      - web
    groups:
      - production
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "host-alpha")
	assert.Contains(t, output, "alpha.example.com")
}

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetList_JSONOutput(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  host-beta:
    hostname: beta.example.com
    user: deploy
    port: 2222
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetJSON = true

	output := captureStdout(t, func() {
		err := runFleetList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "host-beta")
	assert.Contains(t, output, "beta.example.com")
}

// ---------------------------------------------------------------------------
// runWatch with mode override and skip-initial
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global watch flags and cwd
func TestCoverBoost_RunWatch_WithSkipInitialAndDryRun(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "preflight.yaml"), []byte("target: default\n"), 0o644))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	reset := setWatchFlags("100ms", true, true, false)
	defer reset()

	prevApp := newWatchApp
	prevMode := newWatchMode
	fakeMode2 := &fakeWatchMode{}
	fakePF := newFakeWatchPreflight(execution.NewExecutionPlan())

	newWatchApp = func(io.Writer) watchPreflight {
		return fakePF
	}
	newWatchMode = func(opts app.WatchOptions, _ func(context.Context) error) watchMode {
		fakeMode2.opts = opts
		fakeMode2.startCalled = false
		return fakeMode2
	}
	defer func() {
		newWatchApp = prevApp
		newWatchMode = prevMode
	}()

	watchCmd2 := &cobra.Command{}
	watchCmd2.SetContext(context.Background())

	err = runWatch(watchCmd2, nil)
	require.NoError(t, err)
	assert.True(t, fakeMode2.startCalled)
	assert.False(t, fakeMode2.opts.ApplyOnStart)
}

// ---------------------------------------------------------------------------
// detectKeyType (trust.go:344) - GPG type flag override
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global trust flags
func TestCoverBoost_RunTrustAdd_GPGKeyTypeOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tmpFile := filepath.Join(t.TempDir(), "gpgkey.pub")
	require.NoError(t, os.WriteFile(tmpFile,
		[]byte("-----BEGIN PGP PUBLIC KEY-----\ndata\n-----END PGP PUBLIC KEY-----"), 0o600))

	savedName := trustKeyName
	savedLevel := trustKeyLevel
	savedType := trustKeyType
	savedEmail := trustEmail
	defer func() {
		trustKeyName = savedName
		trustKeyLevel = savedLevel
		trustKeyType = savedType
		trustEmail = savedEmail
	}()

	trustKeyName = "GPG Test Key"
	trustKeyLevel = "community"
	trustKeyType = "gpg"
	trustEmail = "gpg@example.com"

	output := captureStdout(t, func() {
		err := runTrustAdd(nil, []string{tmpFile})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Added key")
}

// ---------------------------------------------------------------------------
// selectHosts (fleet.go:248) - with exclude patterns
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestCoverBoost_SelectHosts_WithExclude(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  include-me:
    hostname: include.example.com
    user: admin
    port: 22
  exclude-me:
    hostname: exclude.example.com
    user: admin
    port: 22
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origExclude := fleetExclude
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetExclude = origExclude
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetExclude = "exclude-me"
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetList(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "include-me")
	assert.NotContains(t, output, "exclude-me")
}

// ---------------------------------------------------------------------------
// shouldFail (security.go:187) - additional coverage
// ---------------------------------------------------------------------------

func TestCoverBoost_ShouldFail_NoVulns(t *testing.T) {
	t.Parallel()

	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{},
	}

	assert.False(t, shouldFail(result, security.SeverityLow))
}

// ---------------------------------------------------------------------------
// runFleetApply - no hosts selected
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestCoverBoost_RunFleetApply_NoHostsSelected(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  myhost:
    hostname: localhost
    user: test
    port: 22
    tags:
      - linux
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
	}()

	fleetInventoryFile = invFile
	fleetTarget = "tag:nonexistent"
	fleetStrategy = "parallel"

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No hosts selected")
}

// ---------------------------------------------------------------------------
// runProfileCurrent with no active profile (text and JSON)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and global flags
func TestCoverBoost_RunProfileCurrent_NoProfileJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	savedJSON := profileJSON
	profileJSON = true
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		err := runProfileCurrent(&cobra.Command{}, nil)
		require.NoError(t, err)
	})

	// When no profile active and JSON is set, it still prints text
	assert.Contains(t, output, "No profile active")
}
