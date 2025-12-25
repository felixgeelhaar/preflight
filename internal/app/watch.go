package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WatchMode represents the file watching service.
type WatchMode struct {
	configDir    string
	debounce     time.Duration
	applyFn      func(ctx context.Context) error
	stopCh       chan struct{}
	mu           sync.Mutex
	lastApply    time.Time
	pendingApply bool
}

// WatchOptions configures watch mode behavior.
type WatchOptions struct {
	ConfigDir    string
	Debounce     time.Duration
	ApplyOnStart bool
	DryRun       bool
	Verbose      bool
}

// NewWatchMode creates a new file watch service.
func NewWatchMode(opts WatchOptions, applyFn func(ctx context.Context) error) *WatchMode {
	debounce := opts.Debounce
	if debounce == 0 {
		debounce = 500 * time.Millisecond
	}

	return &WatchMode{
		configDir: opts.ConfigDir,
		debounce:  debounce,
		applyFn:   applyFn,
		stopCh:    make(chan struct{}),
	}
}

// Start begins watching for file changes.
func (w *WatchMode) Start(ctx context.Context) error {
	// Initial apply if requested
	if err := w.triggerApply(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "initial apply failed: %v\n", err)
	}

	// Start the file watcher
	return w.watch(ctx)
}

// Stop stops the file watcher.
func (w *WatchMode) Stop() {
	close(w.stopCh)
}

// watch monitors files for changes using polling.
func (w *WatchMode) watch(ctx context.Context) error {
	// Get initial state of files
	lastMod := make(map[string]time.Time)
	w.updateFileTimes(lastMod)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Printf("👁  Watching %s for changes...\n", w.configDir)
	fmt.Printf("   Press Ctrl+C to stop\n\n")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		case <-ticker.C:
			changed := w.checkForChanges(lastMod)
			if len(changed) > 0 {
				w.handleChanges(ctx, changed)
			}
		}
	}
}

// updateFileTimes scans config files and records their modification times.
func (w *WatchMode) updateFileTimes(times map[string]time.Time) {
	patterns := []string{
		"preflight.yaml",
		"layers/*.yaml",
		"dotfiles/**/*",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(w.configDir, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			times[match] = info.ModTime()
		}
	}
}

// checkForChanges compares current file times with last known times.
func (w *WatchMode) checkForChanges(lastMod map[string]time.Time) []string {
	var changed []string
	current := make(map[string]time.Time)

	w.updateFileTimes(current)

	// Check for new or modified files
	for path, modTime := range current {
		if lastTime, exists := lastMod[path]; !exists || modTime.After(lastTime) {
			changed = append(changed, path)
		}
	}

	// Check for deleted files
	for path := range lastMod {
		if _, exists := current[path]; !exists {
			changed = append(changed, path)
		}
	}

	// Update last known times
	for path, modTime := range current {
		lastMod[path] = modTime
	}
	for path := range lastMod {
		if _, exists := current[path]; !exists {
			delete(lastMod, path)
		}
	}

	return changed
}

// handleChanges processes file changes with debouncing.
func (w *WatchMode) handleChanges(ctx context.Context, changed []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Print changed files
	fmt.Printf("\n📝 Files changed:\n")
	for _, f := range changed {
		rel, _ := filepath.Rel(w.configDir, f)
		if rel == "" {
			rel = f
		}
		fmt.Printf("   - %s\n", rel)
	}

	// Check debounce
	if time.Since(w.lastApply) < w.debounce {
		if !w.pendingApply {
			w.pendingApply = true
			go func() {
				time.Sleep(w.debounce)
				w.mu.Lock()
				w.pendingApply = false
				w.mu.Unlock()
				_ = w.triggerApply(ctx) //nolint:errcheck // Async trigger, error logged internally
			}()
		}
		return
	}

	_ = w.triggerApply(ctx) //nolint:errcheck // Error logged internally
}

// triggerApply runs the apply function.
func (w *WatchMode) triggerApply(ctx context.Context) error {
	w.mu.Lock()
	w.lastApply = time.Now()
	w.mu.Unlock()

	fmt.Printf("\n🔄 Applying changes...\n\n")
	start := time.Now()

	err := w.applyFn(ctx)

	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("\n❌ Apply failed in %s: %v\n", elapsed.Round(time.Millisecond), err)
	} else {
		fmt.Printf("\n✅ Apply completed in %s\n", elapsed.Round(time.Millisecond))
	}

	fmt.Printf("\n👁  Watching for changes...\n")
	return err
}

// FileChange represents a file change event.
type FileChange struct {
	Path      string
	Operation FileOperation
	Time      time.Time
}

// FileOperation represents the type of file change.
type FileOperation string

// File operation constants for tracking changes.
const (
	FileOpCreate FileOperation = "create"
	FileOpModify FileOperation = "modify"
	FileOpDelete FileOperation = "delete"
)

// WatchResult represents the outcome of a watch cycle.
type WatchResult struct {
	Changes      []FileChange
	ApplySuccess bool
	ApplyError   error
	Duration     time.Duration
}

// FilterConfigFiles filters a list of paths to only include config files.
func FilterConfigFiles(paths []string) []string {
	var configFiles []string
	for _, p := range paths {
		base := filepath.Base(p)
		ext := strings.ToLower(filepath.Ext(p))

		// Include YAML/YML files
		if ext == ".yaml" || ext == ".yml" {
			configFiles = append(configFiles, p)
			continue
		}

		// Include specific config files
		if base == "preflight.yaml" || strings.HasPrefix(base, ".") {
			configFiles = append(configFiles, p)
		}
	}
	return configFiles
}
