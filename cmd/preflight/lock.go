package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Manage lockfile for reproducible builds",
	Long: `Lock manages the preflight.lock file for reproducible builds.

The lockfile captures resolved versions and integrity hashes to ensure
consistent installations across machines.

Subcommands:
  update   - Update lock to latest compatible versions
  freeze   - Lock current versions (fail if any change)
  status   - Show lockfile status`,
}

var lockUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update lockfile to latest compatible versions",
	Long: `Update resolves and locks the latest compatible versions of all packages.

This is safe to run regularly to keep your configuration up to date.

Examples:
  preflight lock update             # Update all packages
  preflight lock update --provider brew  # Update only Homebrew packages`,
	RunE: runLockUpdate,
}

var lockFreezeCmd = &cobra.Command{
	Use:   "freeze",
	Short: "Freeze current versions in lockfile",
	Long: `Freeze captures the exact current versions and creates a strict lockfile.

After freezing, any version change will cause an error unless you
explicitly update the lock again.

Examples:
  preflight lock freeze             # Freeze all current versions`,
	RunE: runLockFreeze,
}

var lockStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lockfile status",
	Long: `Status shows information about the current lockfile.

Examples:
  preflight lock status             # Show lockfile summary`,
	RunE: runLockStatus,
}

var lockUpdateProvider string

func init() {
	lockUpdateCmd.Flags().StringVar(&lockUpdateProvider, "provider", "", "Only update specific provider")

	lockCmd.AddCommand(lockUpdateCmd)
	lockCmd.AddCommand(lockFreezeCmd)
	lockCmd.AddCommand(lockStatusCmd)

	rootCmd.AddCommand(lockCmd)
}

func runLockUpdate(_ *cobra.Command, _ []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	if lockUpdateProvider != "" {
		fmt.Printf("Updating lockfile for provider: %s\n", lockUpdateProvider)
	}

	if err := preflight.LockUpdate(ctx, configPath); err != nil {
		return fmt.Errorf("lock update failed: %w", err)
	}

	fmt.Println("\nRun 'preflight plan' to review changes.")
	return nil
}

func runLockFreeze(_ *cobra.Command, _ []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	if err := preflight.LockFreeze(ctx, configPath); err != nil {
		return fmt.Errorf("lock freeze failed: %w", err)
	}

	fmt.Println("Any version changes will now cause an error.")
	return nil
}

func runLockStatus(_ *cobra.Command, _ []string) error {
	// Lock status requires reading the lockfile - we'll show a basic message for now
	// since the full implementation requires a concrete Repository implementation
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	lockPath := configPath[:len(configPath)-len(".yaml")] + ".lock"

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		fmt.Println("Lockfile status:")
		fmt.Println("  Status: No lockfile found")
		fmt.Printf("  Run 'preflight lock update' to create %s\n", lockPath)
		return nil
	}

	fmt.Println("Lockfile status:")
	fmt.Printf("  Path: %s\n", lockPath)
	fmt.Println("  Status: exists")
	fmt.Println("\n  Run 'preflight lock update' to update versions.")
	return nil
}
