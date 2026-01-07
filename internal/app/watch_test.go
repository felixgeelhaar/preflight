package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterConfigFiles(t *testing.T) {
	t.Parallel()

	input := []string{
		"preflight.yaml",
		"layers/brew.yaml",
		"layers/notes.md",
		"dotfiles/.zshrc",
		"dotfiles/starship.toml",
		"docs/config.yml",
		"/tmp/.gitconfig",
		"/tmp/ignore.txt",
		"/tmp/readme.md",
	}

	got := FilterConfigFiles(input)
	want := []string{
		"preflight.yaml",
		"layers/brew.yaml",
		"dotfiles/.zshrc",
		"docs/config.yml",
		"/tmp/.gitconfig",
	}

	assert.ElementsMatch(t, want, got)
}

func TestFilterConfigFiles_Empty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, FilterConfigFiles(nil))
}

func TestWatchMode_triggerApply(t *testing.T) {
	applied := false
	w := &WatchMode{
		configDir: t.TempDir(),
		debounce:  0,
		applyFn: func(_ context.Context) error {
			applied = true
			return nil
		},
		stopCh: make(chan struct{}),
	}

	require.NoError(t, w.triggerApply(context.Background()))
	assert.True(t, applied)
}

func TestWatchMode_handleChanges(t *testing.T) {
	temp := t.TempDir()
	file := filepath.Join(temp, "preflight.yaml")
	require.NoError(t, os.WriteFile(file, []byte("version: 1"), 0o644))

	called := make(chan struct{}, 1)
	w := &WatchMode{
		configDir: temp,
		debounce:  20 * time.Millisecond,
		applyFn: func(_ context.Context) error {
			called <- struct{}{}
			return nil
		},
		stopCh: make(chan struct{}),
	}
	w.lastApply = time.Now()

	w.handleChanges(context.Background(), []string{file})

	select {
	case <-called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("applyFn was not called")
	}
}

func TestWatchMode_checkForChanges(t *testing.T) {
	temp := t.TempDir()
	file := filepath.Join(temp, "preflight.yaml")
	require.NoError(t, os.WriteFile(file, []byte("version: 1"), 0o644))

	w := &WatchMode{
		configDir: temp,
		stopCh:    make(chan struct{}),
	}

	changed := w.checkForChanges(make(map[string]time.Time))
	require.Contains(t, changed, file)

	updated := map[string]time.Time{file: time.Now()}
	changed = w.checkForChanges(updated)
	assert.Empty(t, changed)
}

func TestNewWatchMode_defaults(t *testing.T) {
	w := NewWatchMode(WatchOptions{}, func(_ context.Context) error { return nil })
	assert.Equal(t, 500*time.Millisecond, w.debounce)
	require.NotNil(t, w.stopCh)
}

func TestWatchMode_watchContextCanceled(t *testing.T) {
	dir := t.TempDir()
	w := NewWatchMode(WatchOptions{ConfigDir: dir}, func(_ context.Context) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := w.watch(ctx)
	assert.Equal(t, context.Canceled, err)
}

func TestWatchMode_StopClosesChannel(t *testing.T) {
	w := &WatchMode{stopCh: make(chan struct{})}
	w.Stop()

	select {
	case <-w.stopCh:
	default:
		t.Fatal("stop channel should be closed")
	}
}
