package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWatch_InvalidDebounce(t *testing.T) {

	reset := setWatchFlags("not-a-duration", false, false, false)
	defer reset()

	err := runWatch(&cobra.Command{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid debounce duration")
}

func TestRunWatch_NoConfigFile(t *testing.T) {

	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	reset := setWatchFlags("500ms", false, false, false)
	defer reset()

	command := &cobra.Command{}
	command.SetContext(context.Background())
	require.NotNil(t, command.Context())

	err = runWatch(command, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no preflight.yaml found")
}

func TestRunWatch_StartsWatcher(t *testing.T) {

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "preflight.yaml"), []byte("target: default\n"), 0o644))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(wd) }()

	reset := setWatchFlags("111ms", false, true, true)
	defer reset()

	prevApp := newWatchApp
	prevMode := newWatchMode
	fakeMode := &fakeWatchMode{}
	fakePreflight := newFakeWatchPreflight(execution.NewExecutionPlan())

	newWatchApp = func(io.Writer) watchPreflight {
		return fakePreflight
	}
	newWatchMode = func(opts app.WatchOptions, _ func(context.Context) error) watchMode {
		fakeMode.opts = opts
		fakeMode.startCalled = false
		return fakeMode
	}
	defer func() {
		newWatchApp = prevApp
		newWatchMode = prevMode
	}()

	watchCommand := &cobra.Command{}
	watchCommand.SetContext(context.Background())
	require.NotNil(t, watchCommand.Context())

	err = runWatch(watchCommand, nil)
	require.NoError(t, err)

	assert.True(t, fakeMode.startCalled)
	assert.Equal(t, mustEvalSymlinks(t, dir), mustEvalSymlinks(t, fakeMode.opts.ConfigDir))
	assert.Equal(t, 111*time.Millisecond, fakeMode.opts.Debounce)
}

func setWatchFlags(debounce string, skipInitial, dryRun, verbose bool) func() {
	prev := struct {
		debounce    string
		skipInitial bool
		dryRun      bool
		verbose     bool
	}{
		debounce:    watchDebounce,
		skipInitial: watchSkipInitial,
		dryRun:      watchDryRun,
		verbose:     watchVerbose,
	}

	watchDebounce = debounce
	watchSkipInitial = skipInitial
	watchDryRun = dryRun
	watchVerbose = verbose

	return func() {
		watchDebounce = prev.debounce
		watchSkipInitial = prev.skipInitial
		watchDryRun = prev.dryRun
		watchVerbose = prev.verbose
	}
}

func mustEvalSymlinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

type fakeWatchMode struct {
	opts        app.WatchOptions
	startCalled bool
}

func (f *fakeWatchMode) Start(context.Context) error {
	f.startCalled = true
	return nil
}

type fakeWatchPreflight struct {
	plan        *execution.Plan
	planCalled  bool
	applyCalled bool
}

func newFakeWatchPreflight(plan *execution.Plan) *fakeWatchPreflight {
	return &fakeWatchPreflight{plan: plan}
}

func (f *fakeWatchPreflight) Plan(context.Context, string, string) (*execution.Plan, error) {
	f.planCalled = true
	return f.plan, nil
}

func (f *fakeWatchPreflight) PrintPlan(*execution.Plan) {}

func (f *fakeWatchPreflight) Apply(context.Context, *execution.Plan, bool) ([]execution.StepResult, error) {
	f.applyCalled = true
	return nil, nil
}

func (f *fakeWatchPreflight) PrintResults([]execution.StepResult) {}

func (f *fakeWatchPreflight) WithMode(config.ReproducibilityMode) watchPreflight {
	return f
}
