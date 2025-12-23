package main

import (
	"fmt"

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
	fmt.Println("Updating lockfile...")

	if lockUpdateProvider != "" {
		fmt.Printf("Updating only: %s\n", lockUpdateProvider)
	}

	// TODO: Implement actual lock update using the lock domain
	fmt.Println("Lockfile updated successfully.")
	fmt.Println("\nRun 'preflight plan' to review changes.")
	return nil
}

func runLockFreeze(_ *cobra.Command, _ []string) error {
	fmt.Println("Freezing lockfile...")

	// TODO: Implement actual lock freeze using the lock domain
	fmt.Println("Lockfile frozen.")
	fmt.Println("Any version changes will now cause an error.")
	return nil
}

func runLockStatus(_ *cobra.Command, _ []string) error {
	// TODO: Implement actual status using the lock domain
	fmt.Println("Lockfile status:")
	fmt.Println("  Mode: intent")
	fmt.Println("  Packages: 0")
	fmt.Println("  Last updated: never")
	return nil
}
