package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore files from snapshots",
	Long: `Rollback restores files to a previous state using snapshots.

Snapshots are automatically created before apply operations. Use this command
to list available snapshots and restore files to a previous state.

Examples:
  preflight rollback                    # List available snapshots
  preflight rollback --to <id>          # Restore specific snapshot
  preflight rollback --latest           # Restore most recent snapshot
  preflight rollback --dry-run --to <id> # Preview restoration`,
	RunE: runRollback,
}

var (
	rollbackTo     string
	rollbackLatest bool
	rollbackDryRun bool
)

func init() {
	rollbackCmd.Flags().StringVar(&rollbackTo, "to", "", "Snapshot set ID to restore")
	rollbackCmd.Flags().BoolVar(&rollbackLatest, "latest", false, "Restore the most recent snapshot")
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "Preview restoration without applying")

	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create snapshot service
	snapshotSvc, err := app.DefaultSnapshotService()
	if err != nil {
		return fmt.Errorf("failed to initialize snapshot service: %w", err)
	}

	// List all snapshot sets
	sets, err := snapshotSvc.ListSnapshotSets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(sets) == 0 {
		fmt.Println("No snapshots available.")
		fmt.Println("\nSnapshots are created automatically before apply operations.")
		return nil
	}

	// Sort by creation time (newest first)
	sort.Slice(sets, func(i, j int) bool {
		return sets[i].CreatedAt.After(sets[j].CreatedAt)
	})

	// If --latest, use the most recent snapshot
	if rollbackLatest {
		rollbackTo = sets[0].ID
	}

	// If no snapshot specified, list available snapshots
	if rollbackTo == "" {
		return listSnapshots(ctx, snapshotSvc, sets)
	}

	// Find the specified snapshot set
	var targetSet *snapshot.Set
	for i := range sets {
		if sets[i].ID == rollbackTo || strings.HasPrefix(sets[i].ID, rollbackTo) {
			set, err := snapshotSvc.GetSnapshotSet(ctx, sets[i].ID)
			if err != nil {
				return fmt.Errorf("failed to get snapshot set: %w", err)
			}
			targetSet = set
			break
		}
	}

	if targetSet == nil {
		return fmt.Errorf("snapshot set not found: %s", rollbackTo)
	}

	// Show what will be restored
	fmt.Printf("Snapshot: %s\n", targetSet.ID[:8])
	fmt.Printf("Created:  %s (%s)\n", targetSet.CreatedAt.Format(time.RFC3339), targetSet.Reason)
	fmt.Printf("Files:    %d\n\n", len(targetSet.Snapshots))

	fmt.Println("Files to restore:")
	for _, snap := range targetSet.Snapshots {
		fmt.Printf("  • %s\n", snap.Path)
	}

	if rollbackDryRun {
		fmt.Println("\n--dry-run: No changes made.")
		return nil
	}

	// Confirm restoration
	fmt.Print("\nRestore these files? [y/N] ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Rollback cancelled.")
		return nil
	}

	// Perform restoration
	if err := snapshotSvc.Restore(ctx, targetSet.ID); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	fmt.Printf("\n✓ Restored %d files from snapshot %s\n", len(targetSet.Snapshots), targetSet.ID[:8])
	return nil
}

func listSnapshots(_ context.Context, _ *app.SnapshotService, sets []snapshot.Set) error {
	fmt.Println("Available Snapshots")
	fmt.Println("===================")
	fmt.Println()

	for _, set := range sets {
		age := formatAge(set.CreatedAt)
		shortID := set.ID[:8]

		fmt.Printf("  %s  %s  %-10s  %d files\n",
			shortID,
			set.CreatedAt.Format("2006-01-02 15:04"),
			fmt.Sprintf("(%s)", age),
			len(set.Snapshots),
		)

		// Show reason if available
		if set.Reason != "" {
			fmt.Printf("           Reason: %s\n", set.Reason)
		}
	}

	fmt.Println("\nUsage:")
	fmt.Println("  preflight rollback --to <id>      Restore specific snapshot")
	fmt.Println("  preflight rollback --latest       Restore most recent snapshot")
	fmt.Println("  preflight rollback --dry-run      Preview without applying")

	return nil
}

// formatAge returns a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}
