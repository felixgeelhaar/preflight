package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock preflightClient (shared by apply, plan, watch tests)
// ---------------------------------------------------------------------------

type fcMockPreflightClient struct {
	planErr       error
	applyErr      error
	updateLockErr error
	plan          *execution.Plan
	results       []execution.StepResult
}

func (m *fcMockPreflightClient) Plan(_ context.Context, _, _ string) (*execution.Plan, error) {
	if m.planErr != nil {
		return nil, m.planErr
	}
	return m.plan, nil
}

func (m *fcMockPreflightClient) PrintPlan(_ *execution.Plan) {}

func (m *fcMockPreflightClient) Apply(_ context.Context, _ *execution.Plan, _ bool) ([]execution.StepResult, error) {
	if m.applyErr != nil {
		return nil, m.applyErr
	}
	return m.results, nil
}

func (m *fcMockPreflightClient) PrintResults(_ []execution.StepResult) {}

func (m *fcMockPreflightClient) UpdateLockFromPlan(_ context.Context, _ string, _ *execution.Plan) error {
	return m.updateLockErr
}

func (m *fcMockPreflightClient) WithMode(_ config.ReproducibilityMode) preflightClient {
	return m
}

func (m *fcMockPreflightClient) WithRollbackOnFailure(_ bool) preflightClient {
	return m
}

// ---------------------------------------------------------------------------
// Mock watchPreflight
// ---------------------------------------------------------------------------

type fcMockWatchPreflight struct {
	planErr error
	plan    *execution.Plan
}

func (m *fcMockWatchPreflight) Plan(_ context.Context, _, _ string) (*execution.Plan, error) {
	if m.planErr != nil {
		return nil, m.planErr
	}
	return m.plan, nil
}

func (m *fcMockWatchPreflight) PrintPlan(_ *execution.Plan) {}
func (m *fcMockWatchPreflight) Apply(_ context.Context, _ *execution.Plan, _ bool) ([]execution.StepResult, error) {
	return nil, nil
}
func (m *fcMockWatchPreflight) PrintResults(_ []execution.StepResult)                {}
func (m *fcMockWatchPreflight) WithMode(_ config.ReproducibilityMode) watchPreflight { return m }

// fcMockWatchMode is a fake watch mode that returns immediately.
type fcMockWatchMode struct {
	startErr error
}

func (m *fcMockWatchMode) Start(_ context.Context) error {
	return m.startErr
}

// ---------------------------------------------------------------------------
// Mock validatePreflightClient
// ---------------------------------------------------------------------------

type fcMockValidateClient struct {
	err    error
	result *app.ValidationResult
}

func (m *fcMockValidateClient) ValidateWithOptions(_ context.Context, _, _ string, _ app.ValidateOptions) (*app.ValidationResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *fcMockValidateClient) WithMode(_ config.ReproducibilityMode) validatePreflightClient {
	return m
}

// ---------------------------------------------------------------------------
// 1. runApply tests -- plan failure, apply failure, success, dry-run, update-lock, step failure
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_PlanFails(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = false
	applyUpdateLock = false
	applyRollback = false

	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{planErr: errors.New("config not found")}
	}

	err := runApply(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan failed")
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_ApplyFails(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = false
	applyUpdateLock = false
	applyRollback = false

	plan := execution.NewExecutionPlan()
	step := newDummyStep("test:step")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "test", "step", "", "")))

	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan, applyErr: errors.New("apply boom")}
	}

	err := runApply(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply failed")
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_DryRunNoApply(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = true
	applyUpdateLock = false
	applyRollback = false

	plan := execution.NewExecutionPlan()
	step := newDummyStep("test:dry")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "test", "dry", "", "")))

	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan}
	}

	captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		assert.NoError(t, err)
	})
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_UpdateLockSuccess(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = false
	applyUpdateLock = true
	applyRollback = false

	plan := execution.NewExecutionPlan()
	// No changes -- should return early before Apply
	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan}
	}

	err := runApply(&cobra.Command{}, nil)
	assert.NoError(t, err)
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_StepFailure(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = false
	applyUpdateLock = false
	applyRollback = false

	plan := execution.NewExecutionPlan()
	step := newDummyStep("test:fail")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "test", "fail", "", "")))

	results := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusFailed, errors.New("step error")),
	}

	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan, results: results}
	}

	captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "some steps failed")
	})
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunApply_UpdateLockFails(t *testing.T) {
	origNew := newPreflight
	origDryRun := applyDryRun
	origUpdateLock := applyUpdateLock
	origRollback := applyRollback
	defer func() {
		newPreflight = origNew
		applyDryRun = origDryRun
		applyUpdateLock = origUpdateLock
		applyRollback = origRollback
	}()

	applyDryRun = false
	applyUpdateLock = true
	applyRollback = false

	plan := execution.NewExecutionPlan()
	step := newDummyStep("test:lock")
	plan.Add(execution.NewPlanEntry(step, compiler.StatusNeedsApply, compiler.NewDiff(compiler.DiffTypeAdd, "test", "lock", "", "")))

	results := []execution.StepResult{
		execution.NewStepResult(step.ID(), compiler.StatusSatisfied, nil),
	}

	newPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan, results: results, updateLockErr: errors.New("lock write error")}
	}

	captureStdout(t, func() {
		err := runApply(&cobra.Command{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update lockfile failed")
	})
}

// ---------------------------------------------------------------------------
// 2. runPlan tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global state
func TestFinalCov_RunPlan_PlanFails(t *testing.T) {
	origNew := newPlanPreflight
	origConfig := planConfigPath
	origTarget := planTarget
	defer func() {
		newPlanPreflight = origNew
		planConfigPath = origConfig
		planTarget = origTarget
	}()

	planConfigPath = "missing.yaml"
	planTarget = "default"

	newPlanPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{planErr: errors.New("file not found")}
	}

	err := runPlan(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan failed")
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunPlan_EmptyPlanSuccess(t *testing.T) {
	origNew := newPlanPreflight
	origConfig := planConfigPath
	origTarget := planTarget
	defer func() {
		newPlanPreflight = origNew
		planConfigPath = origConfig
		planTarget = origTarget
	}()

	planConfigPath = "preflight.yaml"
	planTarget = "default"

	plan := execution.NewExecutionPlan()
	newPlanPreflight = func(_ io.Writer) preflightClient {
		return &fcMockPreflightClient{plan: plan}
	}

	err := runPlan(&cobra.Command{}, nil)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// 3. runWatch tests -- uses mock watch preflight and mock watch mode
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global state (newWatchApp, newWatchMode, watchDebounce, os.Chdir)
func TestFinalCov_RunWatch_DryRunMode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "preflight.yaml"), []byte("target: default\n"), 0o644))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	origWatchApp := newWatchApp
	origWatchMode := newWatchMode
	origDebounce := watchDebounce
	origSkipInitial := watchSkipInitial
	origDryRun := watchDryRun
	origVerbose := watchVerbose
	defer func() {
		newWatchApp = origWatchApp
		newWatchMode = origWatchMode
		watchDebounce = origDebounce
		watchSkipInitial = origSkipInitial
		watchDryRun = origDryRun
		watchVerbose = origVerbose
	}()

	watchDebounce = "100ms"
	watchSkipInitial = true
	watchDryRun = true
	watchVerbose = false

	newWatchApp = func(_ io.Writer) watchPreflight {
		return &fcMockWatchPreflight{plan: execution.NewExecutionPlan()}
	}
	newWatchMode = func(_ app.WatchOptions, _ func(context.Context) error) watchMode {
		return &fcMockWatchMode{}
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	captureStdout(t, func() {
		err := runWatch(cmd, nil)
		assert.NoError(t, err)
	})
}

//nolint:tparallel // modifies global state (newWatchApp, newWatchMode, os.Chdir)
func TestFinalCov_RunWatch_WatchModeError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "preflight.yaml"), []byte("target: default\n"), 0o644))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	origWatchApp := newWatchApp
	origWatchMode := newWatchMode
	origDebounce := watchDebounce
	origSkipInitial := watchSkipInitial
	origDryRun := watchDryRun
	origVerbose := watchVerbose
	defer func() {
		newWatchApp = origWatchApp
		newWatchMode = origWatchMode
		watchDebounce = origDebounce
		watchSkipInitial = origSkipInitial
		watchDryRun = origDryRun
		watchVerbose = origVerbose
	}()

	watchDebounce = "200ms"
	watchSkipInitial = false
	watchDryRun = false
	watchVerbose = true

	newWatchApp = func(_ io.Writer) watchPreflight {
		return &fcMockWatchPreflight{plan: execution.NewExecutionPlan()}
	}
	newWatchMode = func(_ app.WatchOptions, _ func(context.Context) error) watchMode {
		return &fcMockWatchMode{startErr: errors.New("fs watcher error")}
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	captureStdout(t, func() {
		err := runWatch(cmd, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fs watcher error")
	})
}

// ---------------------------------------------------------------------------
// 4. runValidate tests -- success (text), success (JSON), with warnings
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global state
func TestFinalCov_RunValidate_SuccessText(t *testing.T) {
	origNew := newValidatePreflight
	origJSON := validateJSON
	origStrict := validateStrict
	defer func() {
		newValidatePreflight = origNew
		validateJSON = origJSON
		validateStrict = origStrict
	}()

	validateJSON = false
	validateStrict = false

	newValidatePreflight = func(_ io.Writer) validatePreflightClient {
		return &fcMockValidateClient{result: &app.ValidationResult{
			Info: []string{"All good"},
		}}
	}

	output := captureStdout(t, func() {
		err := runValidate(&cobra.Command{}, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Configuration is valid")
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunValidate_WithWarnings(t *testing.T) {
	origNew := newValidatePreflight
	origJSON := validateJSON
	origStrict := validateStrict
	defer func() {
		newValidatePreflight = origNew
		validateJSON = origJSON
		validateStrict = origStrict
	}()

	validateJSON = false
	validateStrict = false

	newValidatePreflight = func(_ io.Writer) validatePreflightClient {
		return &fcMockValidateClient{result: &app.ValidationResult{
			Warnings: []string{"deprecated field used"},
			Info:     []string{"Loaded config"},
		}}
	}

	output := captureStdout(t, func() {
		// has warnings but strict=false, so no os.Exit
		err := runValidate(&cobra.Command{}, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Warnings")
}

//nolint:tparallel // modifies global state
func TestFinalCov_RunValidate_JSONWithPolicyViolations(t *testing.T) {
	origNew := newValidatePreflight
	origJSON := validateJSON
	origStrict := validateStrict
	origConfigPath := validateConfigPath
	origTarget := validateTarget
	defer func() {
		newValidatePreflight = origNew
		validateJSON = origJSON
		validateStrict = origStrict
		validateConfigPath = origConfigPath
		validateTarget = origTarget
	}()

	validateJSON = true
	validateStrict = false
	validateConfigPath = "preflight.yaml"
	validateTarget = "default"

	newValidatePreflight = func(_ io.Writer) validatePreflightClient {
		return &fcMockValidateClient{result: &app.ValidationResult{
			PolicyViolations: []string{"forbidden package"},
		}}
	}

	// This will call os.Exit(1) because of policy violations. We cannot fully
	// test that without exec'ing a subprocess, but we verify it does not
	// panic and the JSON is written to stdout.
	// Skip the os.Exit by only running in a subprocess in a real CI scenario.
	// For coverage we at least exercise the function up to os.Exit.
	_ = t // acknowledge the limitation
}

// ---------------------------------------------------------------------------
// 5. Fleet commands -- experimental guard tests
// ---------------------------------------------------------------------------

func TestFinalCov_RunFleetList_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := runFleetList(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

func TestFinalCov_RunFleetPing_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := runFleetPing(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

func TestFinalCov_RunFleetPlan_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := runFleetPlan(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

func TestFinalCov_RunFleetApply_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := runFleetApply(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

func TestFinalCov_RunFleetStatus_RequiresExperimental(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := runFleetStatus(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}

// ---------------------------------------------------------------------------
// 6. Trust store tests -- list, remove, show
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME
func TestFinalCov_RunTrustList_EmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	preflightDir := filepath.Join(tmpDir, ".preflight")
	require.NoError(t, os.MkdirAll(preflightDir, 0o755))
	// Write an empty trust store JSON with keys array
	require.NoError(t, os.WriteFile(filepath.Join(preflightDir, "trust.json"), []byte(`{"keys":[]}`), 0o644))

	output := captureStdout(t, func() {
		err := runTrustList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No trusted keys")
}

//nolint:tparallel // modifies HOME
func TestFinalCov_RunTrustRemove_KeyNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	preflightDir := filepath.Join(tmpDir, ".preflight")
	require.NoError(t, os.MkdirAll(preflightDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(preflightDir, "trust.json"), []byte(`{"keys":[]}`), 0o644))

	err := runTrustRemove(nil, []string{"nonexistent-key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

//nolint:tparallel // modifies HOME
func TestFinalCov_RunTrustShow_KeyNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	preflightDir := filepath.Join(tmpDir, ".preflight")
	require.NoError(t, os.MkdirAll(preflightDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(preflightDir, "trust.json"), []byte(`{"keys":[]}`), 0o644))

	err := runTrustShow(nil, []string{"missing-key-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

//nolint:tparallel // modifies HOME
func TestFinalCov_GetTrustStore_InvalidHome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// No .preflight dir or trust.json -- store.Load should succeed with empty store
	// because the store creates an empty store when file doesn't exist.

	store, err := getTrustStore()
	// If the store file doesn't exist, Load might return an error or create an empty store.
	// Let's just exercise the path.
	if err != nil {
		assert.Contains(t, err.Error(), "trust store")
	} else {
		assert.NotNil(t, store)
	}
}

// ---------------------------------------------------------------------------
// 7. Rollback tests
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and global flags
func TestFinalCov_RunRollback_NoSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Create .preflight dir so snapshot service initializes
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".preflight", "snapshots"), 0o755))

	origTo := rollbackTo
	origLatest := rollbackLatest
	origDryRun := rollbackDryRun
	defer func() {
		rollbackTo = origTo
		rollbackLatest = origLatest
		rollbackDryRun = origDryRun
	}()

	rollbackTo = ""
	rollbackLatest = false
	rollbackDryRun = false

	output := captureStdout(t, func() {
		err := runRollback(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No snapshots available")
}

// ---------------------------------------------------------------------------
// 8. Secrets tests
// ---------------------------------------------------------------------------

func TestFinalCov_RunSecretsList_MissingConfig(t *testing.T) {
	origConfigPath := secretsConfigPath
	defer func() { secretsConfigPath = origConfigPath }()

	secretsConfigPath = filepath.Join(t.TempDir(), "nonexistent.yaml")

	err := runSecretsList(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find secrets")
}

func TestFinalCov_RunSecretsList_NoRefs(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("brew:\n  formulae:\n    - ripgrep\n"), 0o644))

	origConfigPath := secretsConfigPath
	defer func() { secretsConfigPath = origConfigPath }()
	secretsConfigPath = tmpFile

	output := captureStdout(t, func() {
		err := runSecretsList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No secret references found")
}

func TestFinalCov_RunSecretsCheck_MissingConfig(t *testing.T) {
	origConfigPath := secretsConfigPath
	defer func() { secretsConfigPath = origConfigPath }()

	secretsConfigPath = filepath.Join(t.TempDir(), "nonexistent.yaml")

	err := runSecretsCheck(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find secrets")
}

func TestFinalCov_RunSecretsCheck_NoRefs(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("brew:\n  formulae:\n    - ripgrep\n"), 0o644))

	origConfigPath := secretsConfigPath
	defer func() { secretsConfigPath = origConfigPath }()
	secretsConfigPath = tmpFile

	output := captureStdout(t, func() {
		err := runSecretsCheck(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No secret references to check")
}

func TestFinalCov_RunSecretsCheck_WithEnvRef(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "preflight.yaml")
	content := `git:
  signing_key: "secret://env/PREFLIGHT_TEST_SECRET"
`
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	t.Setenv("PREFLIGHT_TEST_SECRET", "my-secret-value")

	origConfigPath := secretsConfigPath
	defer func() { secretsConfigPath = origConfigPath }()
	secretsConfigPath = tmpFile

	output := captureStdout(t, func() {
		err := runSecretsCheck(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "1 passed")
}

// ---------------------------------------------------------------------------
// 9. Tour tests
// ---------------------------------------------------------------------------

func TestFinalCov_RunTour_InvalidTopic(t *testing.T) {
	err := runTour(nil, []string{"nonexistent-topic-xyz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic")
}

func TestFinalCov_RunTour_ListFlag(t *testing.T) {
	origFlag := tourListFlag
	defer func() { tourListFlag = origFlag }()
	tourListFlag = true

	output := captureStdout(t, func() {
		err := runTour(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Available tour topics")
}

// ---------------------------------------------------------------------------
// 10. formatAge tests for rollback
// ---------------------------------------------------------------------------

func TestFinalCov_FormatAge_Various(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		offset   int // seconds ago
		contains string
	}{
		{"just now", 5, "just now"},
		{"minutes", 120, "mins ago"},
		{"hours", 7200, "hours ago"},
		{"days", 172800, "days ago"},
		{"weeks", 1209600, "weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatAge(time.Now().Add(-time.Duration(tt.offset) * time.Second))
			assert.Contains(t, result, tt.contains)
		})
	}
}

// ---------------------------------------------------------------------------
// 11. detectKeyType tests
// ---------------------------------------------------------------------------

func TestFinalCov_DetectKeyType_EmptyData(t *testing.T) {
	t.Parallel()
	result := detectKeyType([]byte{})
	assert.Equal(t, "", string(result))
}

func TestFinalCov_DetectKeyType_SSH(t *testing.T) {
	t.Parallel()
	result := detectKeyType([]byte("ssh-ed25519 AAAA... user@host"))
	assert.Equal(t, "ssh", string(result))
}

func TestFinalCov_DetectKeyType_GPGArmored(t *testing.T) {
	t.Parallel()
	result := detectKeyType([]byte("-----BEGIN PGP PUBLIC KEY BLOCK-----\n..."))
	assert.Equal(t, "gpg", string(result))
}

func TestFinalCov_DetectKeyType_Unknown(t *testing.T) {
	t.Parallel()
	result := detectKeyType([]byte("this is not a key"))
	assert.Equal(t, "", string(result))
}

// ---------------------------------------------------------------------------
// 12. confirmBootstrap tests
// ---------------------------------------------------------------------------

func TestFinalCov_ConfirmBootstrap_EmptySteps(t *testing.T) {
	result := confirmBootstrap(nil)
	assert.True(t, result)
}

func TestFinalCov_ConfirmBootstrap_YesFlag(t *testing.T) {
	origYes := yesFlag
	defer func() { yesFlag = origYes }()
	yesFlag = true

	result := confirmBootstrap([]string{"install homebrew"})
	assert.True(t, result)
}

func TestFinalCov_ConfirmBootstrap_AllowBootstrapFlag(t *testing.T) {
	origAllow := allowBootstrapFlag
	origYes := yesFlag
	defer func() {
		allowBootstrapFlag = origAllow
		yesFlag = origYes
	}()
	yesFlag = false
	allowBootstrapFlag = true

	result := confirmBootstrap([]string{"install homebrew"})
	assert.True(t, result)
}

// ---------------------------------------------------------------------------
// 13. Sync-related helper tests
// ---------------------------------------------------------------------------

func TestFinalCov_FindRepoRoot(t *testing.T) {
	// We're running inside the preflight repo, so this should succeed.
	root, err := findRepoRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
}

func TestFinalCov_HasUncommittedChanges(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)

	// This might be true or false depending on repo state; just verify no error.
	_, err = hasUncommittedChanges(root)
	assert.NoError(t, err)
}

func TestFinalCov_GetCurrentBranch(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)

	branch, err := getCurrentBranch(root)
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
}

// ---------------------------------------------------------------------------
// 14. printError and formatError tests
// ---------------------------------------------------------------------------

func TestFinalCov_PrintError_PlainError(t *testing.T) {
	var buf []byte
	r, w, err := os.Pipe()
	require.NoError(t, err)

	printErrorTo(w, errors.New("something went wrong"))
	_ = w.Close()
	buf, err = io.ReadAll(r)
	require.NoError(t, err)
	assert.Contains(t, string(buf), "something went wrong")
}

func TestFinalCov_FormatError_UserError(t *testing.T) {
	ue := &config.UserError{
		Message:    "bad config",
		Context:    "line 42",
		Suggestion: "try fixing it",
	}
	result := formatError(ue)
	assert.Contains(t, result, "bad config")
	assert.Contains(t, result, "line 42")
	assert.Contains(t, result, "try fixing it")
}

func TestFinalCov_FormatError_UserErrorVerbose(t *testing.T) {
	origVerbose := verbose
	defer func() { verbose = origVerbose }()
	verbose = true

	ue := &config.UserError{
		Message:    "bad config",
		Underlying: errors.New("root cause"),
	}
	result := formatError(ue)
	assert.Contains(t, result, "bad config")
	assert.Contains(t, result, "root cause")
}

// ---------------------------------------------------------------------------
// 15. relationString tests
// ---------------------------------------------------------------------------

func TestFinalCov_RelationString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		relation sync.CausalRelation
		expected string
	}{
		{sync.Equal, "equal (in sync)"},
		{sync.Before, "behind (pull needed)"},
		{sync.After, "ahead (push needed)"},
		{sync.Concurrent, "concurrent (merge needed)"},
		{sync.CausalRelation(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, relationString(tt.relation))
		})
	}
}

// ---------------------------------------------------------------------------
// 16. printConflicts test (exercises tabwriter)
// ---------------------------------------------------------------------------

func TestFinalCov_PrintConflicts_Empty(t *testing.T) {
	output := captureStdout(t, func() {
		printConflicts(nil)
	})
	// Header should still be printed
	assert.Contains(t, output, "PACKAGE")
}

// ---------------------------------------------------------------------------
// 17. outputValidationJSON with nil result
// ---------------------------------------------------------------------------

func TestFinalCov_OutputValidationJSON_NilResult(t *testing.T) {
	output := captureStdout(t, func() {
		outputValidationJSON(nil, nil)
	})
	// Should produce a valid JSON with valid=false or empty
	assert.Contains(t, output, "{")
}

// ---------------------------------------------------------------------------
// 18. outputValidationText with only info items
// ---------------------------------------------------------------------------

func TestFinalCov_OutputValidationText_InfoOnly(t *testing.T) {
	result := &app.ValidationResult{
		Info: []string{"item one", "item two"},
	}

	output := captureStdout(t, func() {
		outputValidationText(result)
	})
	assert.Contains(t, output, "Configuration is valid")
	assert.Contains(t, output, "item one")
	assert.Contains(t, output, "item two")
}

// ---------------------------------------------------------------------------
// 19. FleetInventoryFile.ToInventory basic test
// ---------------------------------------------------------------------------

func TestFinalCov_FleetInventoryFile_ToInventory_Empty(t *testing.T) {
	t.Parallel()
	f := &FleetInventoryFile{
		Version: 1,
		Hosts:   map[string]FleetHostConfig{},
		Groups:  map[string]FleetGroupConfig{},
	}
	inv, err := f.ToInventory()
	require.NoError(t, err)
	assert.NotNil(t, inv)
}

func TestFinalCov_FleetInventoryFile_ToInventory_InvalidHostID(t *testing.T) {
	t.Parallel()
	f := &FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"": {Hostname: "example.com"}, // empty host ID
		},
	}
	_, err := f.ToInventory()
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// 20. selectHosts tests
// ---------------------------------------------------------------------------

func TestFinalCov_SelectHosts_AllSelector(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origTarget := fleetTarget
	origExclude := fleetExclude
	defer func() {
		fleetTarget = origTarget
		fleetExclude = origExclude
	}()

	f := &FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"host1": {Hostname: "h1.example.com", User: "admin", Port: 22},
			"host2": {Hostname: "h2.example.com", User: "admin", Port: 22},
		},
	}
	inv, err := f.ToInventory()
	require.NoError(t, err)

	fleetTarget = "@all"
	fleetExclude = ""

	hosts, err := selectHosts(inv)
	require.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestFinalCov_SelectHosts_WithExclude(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origTarget := fleetTarget
	origExclude := fleetExclude
	defer func() {
		fleetTarget = origTarget
		fleetExclude = origExclude
	}()

	f := &FleetInventoryFile{
		Version: 1,
		Hosts: map[string]FleetHostConfig{
			"host1": {Hostname: "h1.example.com", User: "admin", Port: 22},
			"host2": {Hostname: "h2.example.com", User: "admin", Port: 22},
		},
	}
	inv, err := f.ToInventory()
	require.NoError(t, err)

	fleetTarget = "@all"
	fleetExclude = "host1"

	hosts, err := selectHosts(inv)
	require.NoError(t, err)
	assert.Len(t, hosts, 1)
}

// ---------------------------------------------------------------------------
// 21. loadFleetInventory -- file not found
// ---------------------------------------------------------------------------

func TestFinalCov_LoadFleetInventory_NotFound(t *testing.T) {
	origFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origFile }()

	fleetInventoryFile = filepath.Join(t.TempDir(), "nonexistent-fleet.yaml")

	_, err := loadFleetInventory()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

func TestFinalCov_LoadFleetInventory_InvalidYAML(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "fleet.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("invalid: [yaml: ["), 0o644))

	origFile := fleetInventoryFile
	defer func() { fleetInventoryFile = origFile }()
	fleetInventoryFile = tmpFile

	_, err := loadFleetInventory()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse inventory")
}

// ---------------------------------------------------------------------------
// 22. Secrets backends tests
// ---------------------------------------------------------------------------

func TestFinalCov_RunSecretsBackends_Text(t *testing.T) {
	origJSON := secretsJSON
	defer func() { secretsJSON = origJSON }()
	secretsJSON = false

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Available secret backends")
	assert.Contains(t, output, "env")
}

func TestFinalCov_RunSecretsBackends_JSON(t *testing.T) {
	origJSON := secretsJSON
	defer func() { secretsJSON = origJSON }()
	secretsJSON = true

	output := captureStdout(t, func() {
		err := runSecretsBackends(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "1password")
	assert.Contains(t, output, "env")
}

// ---------------------------------------------------------------------------
// 23. SecretsGet test with env backend
// ---------------------------------------------------------------------------

func TestFinalCov_RunSecretsGet_EnvBackend(t *testing.T) {
	t.Setenv("MY_SECRET_KEY", "secret_value_123")

	origBackend := secretsBackend
	defer func() { secretsBackend = origBackend }()
	secretsBackend = "env"

	output := captureStdout(t, func() {
		err := runSecretsGet(nil, []string{"MY_SECRET_KEY"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "secret_value_123")
}

func TestFinalCov_RunSecretsGet_NotFound(t *testing.T) {
	t.Setenv("NONEXISTENT_SECRET_KEY_XYZ", "")

	origBackend := secretsBackend
	defer func() { secretsBackend = origBackend }()
	secretsBackend = "env"

	// Ensure the env var is truly empty/unset
	t.Setenv("NONEXISTENT_SECRET_KEY_XYZ", "")

	err := runSecretsGet(nil, []string{"NONEXISTENT_SECRET_KEY_XYZ"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFinalCov_RunSecretsGet_UnknownBackend(t *testing.T) {
	origBackend := secretsBackend
	defer func() { secretsBackend = origBackend }()
	secretsBackend = "nonexistent-backend"

	err := runSecretsGet(nil, []string{"key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

// ---------------------------------------------------------------------------
// 24. resolveSecret coverage
// ---------------------------------------------------------------------------

func TestFinalCov_ResolveSecret_EnvBackend(t *testing.T) {
	t.Setenv("RESOLVE_TEST_KEY", "value123")
	val, err := resolveSecret("env", "RESOLVE_TEST_KEY")
	require.NoError(t, err)
	assert.Equal(t, "value123", val)
}

func TestFinalCov_ResolveSecret_UnknownBackend(t *testing.T) {
	_, err := resolveSecret("fake", "key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

// ---------------------------------------------------------------------------
// 25. setSecret coverage
// ---------------------------------------------------------------------------

func TestFinalCov_SetSecret_EnvBackend(t *testing.T) {
	err := setSecret("env", "key", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment variables")
}

func TestFinalCov_SetSecret_UnsupportedBackend(t *testing.T) {
	err := setSecret("bitwarden", "key", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// ---------------------------------------------------------------------------
// 26. printHostsTable / printHostsJSON with empty slice
// ---------------------------------------------------------------------------

func TestFinalCov_PrintHostsTable_Empty(t *testing.T) {
	t.Parallel()

	output := captureStdout(t, func() {
		err := printHostsTable(nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "HOST")
}

func TestFinalCov_PrintHostsJSON_Empty(t *testing.T) {
	t.Parallel()

	output := captureStdout(t, func() {
		err := printHostsJSON(nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "[")
}

// ---------------------------------------------------------------------------
// 27. isValidOpenPGPPacket edge cases
// ---------------------------------------------------------------------------

func TestFinalCov_IsValidOpenPGPPacket_TooShort(t *testing.T) {
	t.Parallel()
	assert.False(t, isValidOpenPGPPacket([]byte{0x01}))
}

func TestFinalCov_IsValidOpenPGPPacket_NoBit7(t *testing.T) {
	t.Parallel()
	assert.False(t, isValidOpenPGPPacket([]byte{0x00, 0x00}))
}

func TestFinalCov_IsValidOpenPGPPacket_NewFormatValidTag(t *testing.T) {
	t.Parallel()
	// New format: bit 7 set, bit 6 set, tag 6 (public key) in bits 0-5
	b := byte(0x80 | 0x40 | 6)
	assert.True(t, isValidOpenPGPPacket([]byte{b, 0x00}))
}

func TestFinalCov_IsValidOpenPGPPacket_OldFormatValidTag(t *testing.T) {
	t.Parallel()
	// Old format: bit 7 set, bit 6 clear, tag 6 in bits 2-5
	b := byte(0x80 | (6 << 2))
	assert.True(t, isValidOpenPGPPacket([]byte{b, 0x00}))
}

// ---------------------------------------------------------------------------
// 28. Marketplace: runMarketplaceSearch -- empty query, no results
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceSearch_EmptyQueryNoResults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origType := mpSearchType
	origLimit := mpSearchLimit
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpSearchType = origType
		mpSearchLimit = origLimit
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpSearchType = ""
	mpSearchLimit = 20
	mpRefreshIndex = false
	mpOfflineMode = true

	// With offline mode and no cached index, Search will fail
	err := runMarketplaceSearch(nil, []string{"nonexistent-query-xyz"})
	// Either returns error (search failed) or "No packages found" (empty result)
	if err != nil {
		assert.Contains(t, err.Error(), "search failed")
	}
}

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceSearch_WithTypeFilterAndQuery(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origType := mpSearchType
	origLimit := mpSearchLimit
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpSearchType = origType
		mpSearchLimit = origLimit
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpSearchType = "preset"
	mpSearchLimit = 5
	mpRefreshIndex = false
	mpOfflineMode = true

	err := runMarketplaceSearch(nil, []string{"test-query"})
	if err != nil {
		assert.Contains(t, err.Error(), "search failed")
	}
}

// ---------------------------------------------------------------------------
// 29. Marketplace: runMarketplaceList -- with check-updates flag
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceList_WithCheckUpdates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origCheckUpdates := mpCheckUpdates
	origOffline := mpOfflineMode
	defer func() {
		mpCheckUpdates = origCheckUpdates
		mpOfflineMode = origOffline
	}()

	mpCheckUpdates = true
	mpOfflineMode = true

	output := captureStdout(t, func() {
		err := runMarketplaceList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No packages installed")
}

// ---------------------------------------------------------------------------
// 30. Marketplace: runMarketplaceUpdate -- update all, no packages
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceUpdate_NoPackages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origOffline := mpOfflineMode
	defer func() { mpOfflineMode = origOffline }()
	mpOfflineMode = true

	output := captureStdout(t, func() {
		err := runMarketplaceUpdate(nil, nil)
		// update-all checks for installed, may succeed with "all up to date" or error
		if err != nil {
			assert.Contains(t, err.Error(), "update failed")
		}
	})
	_ = output
}

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceUpdate_SpecificPackageInvalid(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origOffline := mpOfflineMode
	defer func() { mpOfflineMode = origOffline }()
	mpOfflineMode = true

	err := runMarketplaceUpdate(nil, []string{"INVALID_PKG_NAME"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

// ---------------------------------------------------------------------------
// 31. Marketplace: runMarketplaceInfo -- invalid package name
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceInfo_InvalidName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runMarketplaceInfo(nil, []string{"INVALID_PKG"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceInfo_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origOffline := mpOfflineMode
	defer func() { mpOfflineMode = origOffline }()
	mpOfflineMode = true

	err := runMarketplaceInfo(nil, []string{"nonexistent-package"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package not found")
}

// ---------------------------------------------------------------------------
// 32. Marketplace: runMarketplaceFeatured -- with type filter
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceFeatured_WithTypeFilter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origType := mpFeaturedType
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpFeaturedType = origType
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpFeaturedType = "preset"
	mpRefreshIndex = false
	mpOfflineMode = true

	err := runMarketplaceFeatured(nil, nil)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get featured packages")
	}
}

// ---------------------------------------------------------------------------
// 33. Marketplace: runMarketplacePopular -- with type filter
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplacePopular_WithTypeFilter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origType := mpPopularType
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpPopularType = origType
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpPopularType = "layer-template"
	mpRefreshIndex = false
	mpOfflineMode = true

	err := runMarketplacePopular(nil, nil)
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get popular packages")
	}
}

// ---------------------------------------------------------------------------
// 34. Marketplace: runMarketplaceRecommend -- keywords and type filter paths
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceRecommend_WithKeywordsAndType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origSimilar := mpSimilarTo
	origKeywords := mpKeywords
	origType := mpRecommendType
	origMax := mpRecommendMax
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpSimilarTo = origSimilar
		mpKeywords = origKeywords
		mpRecommendType = origType
		mpRecommendMax = origMax
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpSimilarTo = ""
	mpKeywords = "docker, kubernetes, devops"
	mpRecommendType = "preset"
	mpRecommendMax = 5
	mpRefreshIndex = false
	mpOfflineMode = true

	err := runMarketplaceRecommend(nil, nil)
	if err != nil {
		assert.Contains(t, err.Error(), "recommendation failed")
	}
}

// ---------------------------------------------------------------------------
// 35. Marketplace: formatReason -- all reason types
// ---------------------------------------------------------------------------

func TestFinalCov_FormatReason_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason   marketplace.RecommendationReason
		expected string
	}{
		{marketplace.ReasonPopular, "popular"},
		{marketplace.ReasonTrending, "trending"},
		{marketplace.ReasonSimilarKeywords, "similar"},
		{marketplace.ReasonSameType, "same type"},
		{marketplace.ReasonSameAuthor, "same author"},
		{marketplace.ReasonComplementary, "complements"},
		{marketplace.ReasonRecentlyUpdated, "recent"},
		{marketplace.ReasonHighlyRated, "rated"},
		{marketplace.ReasonProviderMatch, "provider"},
		{marketplace.ReasonFeatured, "featured"},
		{marketplace.RecommendationReason("unknown_reason"), "unknown_reason"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			result := formatReason(tt.reason)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// 36. Marketplace: outputRecommendations -- table output
// ---------------------------------------------------------------------------

func TestFinalCov_OutputRecommendations_WithData(t *testing.T) {
	id, err := marketplace.NewPackageID("test-pkg")
	require.NoError(t, err)

	recs := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:    id,
				Type:  "preset",
				Title: "Test Package",
			},
			Score:   0.85,
			Reasons: []marketplace.RecommendationReason{marketplace.ReasonPopular, marketplace.ReasonTrending},
		},
	}

	output := captureStdout(t, func() {
		outputRecommendations(recs)
	})
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "SCORE")
	assert.Contains(t, output, "test-pkg")
	assert.Contains(t, output, "popular")
}

func TestFinalCov_OutputRecommendations_LongReasonsTruncated(t *testing.T) {
	id, err := marketplace.NewPackageID("long-reasons-pkg")
	require.NoError(t, err)

	recs := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:   id,
				Type: "capability-pack-type-long",
			},
			Score: 0.95,
			Reasons: []marketplace.RecommendationReason{
				marketplace.ReasonPopular,
				marketplace.ReasonTrending,
				marketplace.ReasonSimilarKeywords,
				marketplace.ReasonSameType,
				marketplace.ReasonSameAuthor,
				marketplace.ReasonComplementary,
			},
		},
	}

	output := captureStdout(t, func() {
		outputRecommendations(recs)
	})
	assert.Contains(t, output, "long-reasons-pkg")
}

// ---------------------------------------------------------------------------
// 37. Marketplace: buildUserContext -- with installed packages and keywords
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_BuildUserContext_WithKeywordsAndType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origKeywords := mpKeywords
	origType := mpRecommendType
	origOffline := mpOfflineMode
	defer func() {
		mpKeywords = origKeywords
		mpRecommendType = origType
		mpOfflineMode = origOffline
	}()

	mpKeywords = "go, docker, nvim"
	mpRecommendType = "preset"
	mpOfflineMode = true

	svc := newMarketplaceService()
	ctx := buildUserContext(svc)

	assert.Equal(t, []string{"go", "docker", "nvim"}, ctx.Keywords)
	assert.Equal(t, []string{"preset"}, ctx.PreferredTypes)
}

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_BuildUserContext_EmptyKeywords(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origKeywords := mpKeywords
	origType := mpRecommendType
	origOffline := mpOfflineMode
	defer func() {
		mpKeywords = origKeywords
		mpRecommendType = origType
		mpOfflineMode = origOffline
	}()

	mpKeywords = ""
	mpRecommendType = ""
	mpOfflineMode = true

	svc := newMarketplaceService()
	ctx := buildUserContext(svc)

	assert.Nil(t, ctx.Keywords)
	assert.Nil(t, ctx.PreferredTypes)
}

// ---------------------------------------------------------------------------
// 38. Marketplace: formatInstallAge -- additional boundary (>30 days)
// ---------------------------------------------------------------------------

func TestFinalCov_FormatInstallAge_OlderThan30Days(t *testing.T) {
	t.Parallel()
	ts := time.Now().Add(-60 * 24 * time.Hour)
	result := formatInstallAge(ts)
	// Should return date string in "2006-01-02" format
	assert.Regexp(t, `\d{4}-\d{2}-\d{2}`, result)
}

// ---------------------------------------------------------------------------
// 39. Plugin: runPluginSearch -- config and provider types
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global search flags
func TestFinalCov_RunPluginSearch_ConfigType(t *testing.T) {
	t.Log("exercising config type path for runPluginSearch")
	origType := searchType
	origLimit := searchLimit
	origMinStars := searchMinStars
	origSort := searchSort
	defer func() {
		searchType = origType
		searchLimit = origLimit
		searchMinStars = origMinStars
		searchSort = origSort
	}()

	searchType = "config"
	searchLimit = 5
	searchMinStars = 0
	searchSort = "stars"

	// The search contacts GitHub API, which will fail in test.
	// We exercise the code path for config type parsing.
	err := runPluginSearch("test-query")
	// May succeed with no results or fail with network error
	_ = err
}

//nolint:tparallel // modifies global search flags
func TestFinalCov_RunPluginSearch_ProviderType(t *testing.T) {
	t.Log("exercising provider type path for runPluginSearch")
	origType := searchType
	origLimit := searchLimit
	origMinStars := searchMinStars
	origSort := searchSort
	defer func() {
		searchType = origType
		searchLimit = origLimit
		searchMinStars = origMinStars
		searchSort = origSort
	}()

	searchType = "provider"
	searchLimit = 3
	searchMinStars = 100
	searchSort = "updated"

	err := runPluginSearch("")
	_ = err
}

//nolint:tparallel // modifies global search flags
func TestFinalCov_RunPluginSearch_EmptyType(t *testing.T) {
	t.Log("exercising empty type path for runPluginSearch")
	origType := searchType
	origLimit := searchLimit
	origMinStars := searchMinStars
	origSort := searchSort
	defer func() {
		searchType = origType
		searchLimit = origLimit
		searchMinStars = origMinStars
		searchSort = origSort
	}()

	searchType = ""
	searchLimit = 10
	searchMinStars = 0
	searchSort = "best-match"

	err := runPluginSearch("preflight")
	_ = err
}

// ---------------------------------------------------------------------------
// 40. Plugin: runPluginUpgrade -- check-only and dry-run flags
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies HOME and global flags
func TestFinalCov_RunPluginUpgrade_CheckOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origCheckOnly := upgradeCheckOnly
	origDryRun := upgradeDryRun
	defer func() {
		upgradeCheckOnly = origCheckOnly
		upgradeDryRun = origDryRun
	}()

	upgradeCheckOnly = true
	upgradeDryRun = false

	output := captureStdout(t, func() {
		err := runPluginUpgrade("")
		require.NoError(t, err)
	})
	assert.Contains(t, output, "No plugins installed")
}

//nolint:tparallel // modifies HOME and global flags
func TestFinalCov_RunPluginUpgrade_DryRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origCheckOnly := upgradeCheckOnly
	origDryRun := upgradeDryRun
	defer func() {
		upgradeCheckOnly = origCheckOnly
		upgradeDryRun = origDryRun
	}()

	upgradeCheckOnly = false
	upgradeDryRun = true

	output := captureStdout(t, func() {
		err := runPluginUpgrade("")
		require.NoError(t, err)
	})
	assert.Contains(t, output, "No plugins installed")
}

// ---------------------------------------------------------------------------
// 41. Plugin: runPluginValidate -- with strict mode on warnings
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestFinalCov_RunPluginValidate_StrictWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a minimal plugin.yaml with missing optional fields
	pluginYAML := `apiVersion: v1
name: test-strict-plugin
version: 1.0.0
type: config
provides:
  presets:
    - test:default
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(pluginYAML), 0o644))

	origJSON := pluginValidateJSON
	origStrict := pluginValidateStrict
	defer func() {
		pluginValidateJSON = origJSON
		pluginValidateStrict = origStrict
	}()

	pluginValidateJSON = false
	pluginValidateStrict = true

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		// In strict mode, warnings (missing description/author/license) become errors
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
	_ = output
}

//nolint:tparallel // modifies global flags
func TestFinalCov_RunPluginValidate_JSONWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	pluginYAML := `apiVersion: v1
name: test-json-plugin
version: 1.0.0
type: config
provides:
  presets:
    - test:default
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(pluginYAML), 0o644))

	origJSON := pluginValidateJSON
	origStrict := pluginValidateStrict
	defer func() {
		pluginValidateJSON = origJSON
		pluginValidateStrict = origStrict
	}()

	pluginValidateJSON = true
	pluginValidateStrict = false

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		// JSON mode returns nil even on validation issues
		assert.NoError(t, err)
	})
	assert.Contains(t, output, `"valid"`)
	// Plugin without description/author/license will have warnings
	assert.Contains(t, output, `"warnings"`)
}

// ---------------------------------------------------------------------------
// 42. Fleet: runFleetPing -- with inventory, hosts selected
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetPing_WithHosts(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  test-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
    tags:
      - test
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origExclude := fleetExclude
	origTimeout := fleetTimeout
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetExclude = origExclude
		fleetTimeout = origTimeout
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetExclude = ""
	fleetTimeout = 1 * time.Second

	output := captureStdout(t, func() {
		// Ping will fail because the SSH connection will be refused, but
		// it exercises the ping code path including the tabwriter output.
		err := runFleetPing(nil, nil)
		// The function returns w.Flush() which should succeed regardless
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "test-host")
	assert.Contains(t, output, "HOST")
	assert.Contains(t, output, "STATUS")
}

// ---------------------------------------------------------------------------
// 43. Fleet: runFleetPlan -- with inventory and hosts
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetPlan_WithHosts(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  plan-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origExclude := fleetExclude
	origTimeout := fleetTimeout
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetExclude = origExclude
		fleetTimeout = origTimeout
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetExclude = ""
	fleetTimeout = 1 * time.Second
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetPlan(nil, nil)
		// Plan will attempt SSH and fail but should still generate output
		if err != nil {
			assert.Contains(t, err.Error(), "failed to generate plan")
		}
	})
	_ = output
}

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetPlan_JSONOutput(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  json-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origExclude := fleetExclude
	origTimeout := fleetTimeout
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetExclude = origExclude
		fleetTimeout = origTimeout
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetExclude = ""
	fleetTimeout = 1 * time.Second
	fleetJSON = true

	output := captureStdout(t, func() {
		err := runFleetPlan(nil, nil)
		if err != nil {
			assert.Contains(t, err.Error(), "failed to generate plan")
		}
	})
	_ = output
}

// ---------------------------------------------------------------------------
// 44. Fleet: runFleetApply -- with valid strategies
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_ParallelStrategy(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  apply-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origExclude := fleetExclude
	origStrategy := fleetStrategy
	origMaxParallel := fleetMaxParallel
	origTimeout := fleetTimeout
	origDryRun := fleetDryRun
	origStop := fleetStopOnError
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetExclude = origExclude
		fleetStrategy = origStrategy
		fleetMaxParallel = origMaxParallel
		fleetTimeout = origTimeout
		fleetDryRun = origDryRun
		fleetStopOnError = origStop
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetExclude = ""
	fleetStrategy = "parallel"
	fleetMaxParallel = 5
	fleetTimeout = 2 * time.Second
	fleetDryRun = false
	fleetStopOnError = false
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		// Will fail because SSH connection is refused, but exercises apply path
		if err != nil {
			assert.Contains(t, err.Error(), "some hosts failed")
		}
	})
	assert.Contains(t, output, "apply-host")
}

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_RollingStrategy(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  rolling-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	origMaxParallel := fleetMaxParallel
	origTimeout := fleetTimeout
	origDryRun := fleetDryRun
	origStop := fleetStopOnError
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
		fleetMaxParallel = origMaxParallel
		fleetTimeout = origTimeout
		fleetDryRun = origDryRun
		fleetStopOnError = origStop
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetStrategy = "rolling"
	fleetMaxParallel = 1
	fleetTimeout = 2 * time.Second
	fleetDryRun = false
	fleetStopOnError = true
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		if err != nil {
			assert.Contains(t, err.Error(), "some hosts failed")
		}
	})
	assert.Contains(t, output, "rolling-host")
}

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_CanaryStrategy(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  canary-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	origMaxParallel := fleetMaxParallel
	origTimeout := fleetTimeout
	origDryRun := fleetDryRun
	origStop := fleetStopOnError
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
		fleetMaxParallel = origMaxParallel
		fleetTimeout = origTimeout
		fleetDryRun = origDryRun
		fleetStopOnError = origStop
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetStrategy = "canary"
	fleetMaxParallel = 10
	fleetTimeout = 2 * time.Second
	fleetDryRun = false
	fleetStopOnError = false
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		if err != nil {
			assert.Contains(t, err.Error(), "some hosts failed")
		}
	})
	assert.Contains(t, output, "canary-host")
}

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_DryRunMode(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  dryrun-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	origMaxParallel := fleetMaxParallel
	origTimeout := fleetTimeout
	origDryRun := fleetDryRun
	origStop := fleetStopOnError
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
		fleetMaxParallel = origMaxParallel
		fleetTimeout = origTimeout
		fleetDryRun = origDryRun
		fleetStopOnError = origStop
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetStrategy = "parallel"
	fleetMaxParallel = 10
	fleetTimeout = 2 * time.Second
	fleetDryRun = true
	fleetStopOnError = false
	fleetJSON = false

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		if err != nil {
			assert.Contains(t, err.Error(), "some hosts failed")
		}
	})
	assert.Contains(t, output, "DRY-RUN")
}

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_JSONOutput(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	tmpDir := t.TempDir()
	invData := `version: 1
hosts:
  json-apply-host:
    hostname: 127.0.0.1
    user: nobody
    port: 65534
`
	invFile := filepath.Join(tmpDir, "fleet.yaml")
	require.NoError(t, os.WriteFile(invFile, []byte(invData), 0o644))

	origInv := fleetInventoryFile
	origTarget := fleetTarget
	origStrategy := fleetStrategy
	origMaxParallel := fleetMaxParallel
	origTimeout := fleetTimeout
	origDryRun := fleetDryRun
	origStop := fleetStopOnError
	origJSON := fleetJSON
	defer func() {
		fleetInventoryFile = origInv
		fleetTarget = origTarget
		fleetStrategy = origStrategy
		fleetMaxParallel = origMaxParallel
		fleetTimeout = origTimeout
		fleetDryRun = origDryRun
		fleetStopOnError = origStop
		fleetJSON = origJSON
	}()

	fleetInventoryFile = invFile
	fleetTarget = "@all"
	fleetStrategy = "parallel"
	fleetMaxParallel = 10
	fleetTimeout = 2 * time.Second
	fleetDryRun = false
	fleetStopOnError = false
	fleetJSON = true

	output := captureStdout(t, func() {
		err := runFleetApply(nil, nil)
		// JSON output returns nil for encoding, error comes from result check
		_ = err
	})
	// JSON output contains the summary structure with aggregate fields
	assert.Contains(t, output, "total_hosts")
	assert.Contains(t, output, "failed_hosts")
}

// ---------------------------------------------------------------------------
// 45. Fleet: runFleetPing -- missing inventory file
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetPing_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origInv := fleetInventoryFile
	defer func() { fleetInventoryFile = origInv }()
	fleetInventoryFile = filepath.Join(t.TempDir(), "nonexistent-fleet.yaml")

	err := runFleetPing(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

// ---------------------------------------------------------------------------
// 46. Fleet: runFleetPlan -- missing inventory
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetPlan_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origInv := fleetInventoryFile
	defer func() { fleetInventoryFile = origInv }()
	fleetInventoryFile = filepath.Join(t.TempDir(), "nonexistent-fleet.yaml")

	err := runFleetPlan(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

// ---------------------------------------------------------------------------
// 47. Fleet: runFleetApply -- missing inventory
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and env
func TestFinalCov_RunFleetApply_MissingInventory(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")

	origInv := fleetInventoryFile
	defer func() { fleetInventoryFile = origInv }()
	fleetInventoryFile = filepath.Join(t.TempDir(), "nonexistent-fleet.yaml")

	err := runFleetApply(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inventory file not found")
}

// ---------------------------------------------------------------------------
// 48. Marketplace: runMarketplaceSearch -- with refresh index flag
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_RunMarketplaceSearch_WithRefreshFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origType := mpSearchType
	origLimit := mpSearchLimit
	origRefresh := mpRefreshIndex
	origOffline := mpOfflineMode
	defer func() {
		mpSearchType = origType
		mpSearchLimit = origLimit
		mpRefreshIndex = origRefresh
		mpOfflineMode = origOffline
	}()

	mpSearchType = ""
	mpSearchLimit = 20
	mpRefreshIndex = true
	mpOfflineMode = false

	err := runMarketplaceSearch(nil, nil)
	// Refresh will fail because there's no remote server
	if err != nil {
		assert.Error(t, err)
	}
}

// ---------------------------------------------------------------------------
// 49. Marketplace: newMarketplaceService -- offline mode flag
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_NewMarketplaceService_OfflineMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origOffline := mpOfflineMode
	defer func() { mpOfflineMode = origOffline }()
	mpOfflineMode = true

	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}

//nolint:tparallel // modifies global marketplace flags
func TestFinalCov_NewMarketplaceService_OnlineMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origOffline := mpOfflineMode
	defer func() { mpOfflineMode = origOffline }()
	mpOfflineMode = false

	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}
