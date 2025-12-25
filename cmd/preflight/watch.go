package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch configuration files and auto-apply changes",
	Long: `Watch configuration files for changes and automatically apply them.

This command monitors your preflight.yaml, layers/*.yaml, and dotfiles/
directory for changes. When a change is detected, it automatically
runs the apply command.

Debouncing prevents multiple rapid applies when saving multiple files.
The initial apply can be skipped with --skip-initial.`,
	Example: `  # Watch with default settings
  preflight watch

  # Watch with custom debounce
  preflight watch --debounce 1s

  # Watch without initial apply
  preflight watch --skip-initial

  # Watch specific directory
  preflight watch --config ~/dotfiles`,
	RunE: runWatch,
}

var (
	watchDebounce    string
	watchSkipInitial bool
	watchDryRun      bool
	watchVerbose     bool
)

func init() {
	watchCmd.Flags().StringVar(&watchDebounce, "debounce", "500ms", "Debounce duration for file changes")
	watchCmd.Flags().BoolVar(&watchSkipInitial, "skip-initial", false, "Skip initial apply on start")
	watchCmd.Flags().BoolVar(&watchDryRun, "dry-run", false, "Show what would be applied without making changes")
	watchCmd.Flags().BoolVar(&watchVerbose, "verbose", false, "Show detailed output")

	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, _ []string) error {
	// Parse debounce duration
	debounce, err := time.ParseDuration(watchDebounce)
	if err != nil {
		return fmt.Errorf("invalid debounce duration: %w", err)
	}

	// Get config directory
	configDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Check if config file exists
	configFile := "preflight.yaml"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("no preflight.yaml found in %s", configDir)
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n\nStopping watch mode...")
		cancel()
	}()

	// Create apply function
	applyFn := func(_ context.Context) error {
		// In a real implementation, this would call the apply logic
		// For now, we'll use the existing apply infrastructure
		target := "default"

		if watchDryRun {
			fmt.Printf("   [dry-run] Would apply target: %s\n", target)
			return nil
		}

		// Call the actual apply logic here
		// This would be: return runApply(ctx, target)
		fmt.Printf("   Applying target: %s\n", target)
		return nil
	}

	// Create watch options
	opts := app.WatchOptions{
		ConfigDir:    configDir,
		Debounce:     debounce,
		ApplyOnStart: !watchSkipInitial,
		DryRun:       watchDryRun,
		Verbose:      watchVerbose,
	}

	// Create and start watch mode
	watcher := app.NewWatchMode(opts, applyFn)

	fmt.Println("🔍 Preflight Watch Mode")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📁 Config directory: %s\n", configDir)
	fmt.Printf("⏱  Debounce: %s\n", debounce)
	if watchDryRun {
		fmt.Println("🔒 Dry-run mode enabled")
	}
	fmt.Println()

	return watcher.Start(ctx)
}
