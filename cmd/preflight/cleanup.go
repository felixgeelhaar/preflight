package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Detect and clean up redundant packages",
	Long: `Detect and clean up redundant packages from your system.

This command analyzes installed packages for:
  - Version duplicates (e.g., go + go@1.24)
  - Overlapping tools serving the same purpose
  - Orphaned dependencies no longer needed

By default, runs in dry-run mode to show what would be removed.

Exit codes:
  0 - No redundancies found (or cleanup successful)
  1 - Redundancies found (in analysis mode)
  2 - Checker not available or operation failed

Examples:
  preflight cleanup                   # Analyze redundancies (dry-run)
  preflight cleanup --remove go@1.24  # Remove specific package
  preflight cleanup --autoremove      # Remove orphaned dependencies
  preflight cleanup --all             # Interactive cleanup of all
  preflight cleanup --json            # JSON output for CI`,
	RunE: runCleanup,
}

var (
	cleanupRemove     []string
	cleanupAutoremove bool
	cleanupAll        bool
	cleanupDryRun     bool
	cleanupJSON       bool
	cleanupQuiet      bool
	cleanupIgnore     []string
	cleanupKeep       []string
	cleanupNoOrphans  bool
	cleanupNoOverlaps bool
)

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().StringSliceVar(&cleanupRemove, "remove", nil, "Packages to remove")
	cleanupCmd.Flags().BoolVar(&cleanupAutoremove, "autoremove", false, "Remove orphaned dependencies")
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Remove all detected redundancies")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be removed without removing")
	cleanupCmd.Flags().BoolVar(&cleanupJSON, "json", false, "Output results as JSON")
	cleanupCmd.Flags().BoolVarP(&cleanupQuiet, "quiet", "q", false, "Only show summary")
	cleanupCmd.Flags().StringSliceVar(&cleanupIgnore, "ignore", nil, "Packages to ignore")
	cleanupCmd.Flags().StringSliceVar(&cleanupKeep, "keep", nil, "Packages to never remove")
	cleanupCmd.Flags().BoolVar(&cleanupNoOrphans, "no-orphans", false, "Skip orphaned dependency detection")
	cleanupCmd.Flags().BoolVar(&cleanupNoOverlaps, "no-overlaps", false, "Skip overlapping tools detection")
}

func runCleanup(_ *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Create checker
	checker := security.NewBrewRedundancyChecker()
	if !checker.Available() {
		if cleanupJSON {
			outputCleanupJSON(nil, nil, fmt.Errorf("brew not available"))
		} else {
			fmt.Fprintf(os.Stderr, "Error: Homebrew is not installed\n")
		}
		cancel()
		os.Exit(2)
	}

	// Handle specific remove requests
	if len(cleanupRemove) > 0 {
		cancel()
		return handleRemove(ctx, checker, cleanupRemove)
	}

	// Handle autoremove
	if cleanupAutoremove {
		cancel()
		return handleAutoremove(ctx, checker)
	}

	// Run analysis
	opts := security.RedundancyOptions{
		IgnorePackages:  cleanupIgnore,
		KeepPackages:    cleanupKeep,
		IncludeOrphans:  !cleanupNoOrphans,
		IncludeOverlaps: !cleanupNoOverlaps,
	}

	result, err := checker.Check(ctx, opts)
	if err != nil {
		if cleanupJSON {
			outputCleanupJSON(nil, nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		}
		cancel()
		os.Exit(2)
	}

	// Handle --all flag
	if cleanupAll && len(result.Redundancies) > 0 {
		cancel()
		return handleCleanupAll(ctx, checker, result)
	}

	// Output analysis results
	if cleanupJSON {
		outputCleanupJSON(result, nil, nil)
	} else {
		outputCleanupText(result, cleanupQuiet)
	}

	// Exit with code 1 if redundancies found
	if len(result.Redundancies) > 0 {
		cancel()
		os.Exit(1)
	}

	cancel()
	return nil
}

func handleRemove(ctx context.Context, checker *security.BrewRedundancyChecker, packages []string) error {
	if cleanupDryRun {
		if cleanupJSON {
			outputCleanupJSON(nil, &security.CleanupResult{
				Removed: packages,
				DryRun:  true,
			}, nil)
		} else {
			fmt.Println("Would remove:")
			for _, pkg := range packages {
				fmt.Printf("  - %s\n", pkg)
			}
		}
		return nil
	}

	// Confirm unless --yes flag
	if !yesFlag {
		fmt.Printf("Remove %d package(s)? [y/N] ", len(packages))
		var response string
		_, _ = fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	err := checker.Cleanup(ctx, packages, false)
	if err != nil {
		if cleanupJSON {
			outputCleanupJSON(nil, nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
		}
		os.Exit(2)
	}

	if cleanupJSON {
		outputCleanupJSON(nil, &security.CleanupResult{
			Removed: packages,
			DryRun:  false,
		}, nil)
	} else {
		fmt.Printf("✓ Removed %d package(s)\n", len(packages))
	}

	return nil
}

func handleAutoremove(ctx context.Context, checker *security.BrewRedundancyChecker) error {
	removed, err := checker.Autoremove(ctx, cleanupDryRun)
	if err != nil {
		if cleanupJSON {
			outputCleanupJSON(nil, nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Autoremove failed: %v\n", err)
		}
		os.Exit(2)
	}

	if cleanupJSON {
		outputCleanupJSON(nil, &security.CleanupResult{
			Removed: removed,
			DryRun:  cleanupDryRun,
		}, nil)
	} else {
		switch {
		case len(removed) == 0:
			fmt.Println("✓ No orphaned dependencies to remove")
		case cleanupDryRun:
			fmt.Printf("Would remove %d orphaned dependencies:\n", len(removed))
			for _, pkg := range removed {
				fmt.Printf("  - %s\n", pkg)
			}
		default:
			fmt.Printf("✓ Removed %d orphaned dependencies\n", len(removed))
		}
	}

	return nil
}

func handleCleanupAll(ctx context.Context, checker *security.BrewRedundancyChecker, result *security.RedundancyResult) error {
	// Collect all packages to remove
	toRemove := make([]string, 0, len(result.Redundancies))
	for _, red := range result.Redundancies {
		toRemove = append(toRemove, red.Remove...)
	}

	if len(toRemove) == 0 {
		if cleanupJSON {
			outputCleanupJSON(result, &security.CleanupResult{DryRun: cleanupDryRun}, nil)
		} else {
			fmt.Println("✓ Nothing to clean up")
		}
		return nil
	}

	if cleanupDryRun {
		if cleanupJSON {
			outputCleanupJSON(result, &security.CleanupResult{
				Removed: toRemove,
				DryRun:  true,
			}, nil)
		} else {
			fmt.Printf("Would remove %d package(s):\n", len(toRemove))
			for _, pkg := range toRemove {
				fmt.Printf("  - %s\n", pkg)
			}
		}
		return nil
	}

	// Confirm unless --yes flag
	if !yesFlag {
		fmt.Printf("Remove %d package(s)? [y/N] ", len(toRemove))
		var response string
		_, _ = fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	err := checker.Cleanup(ctx, toRemove, false)
	if err != nil {
		if cleanupJSON {
			outputCleanupJSON(result, nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
		}
		os.Exit(2)
	}

	if cleanupJSON {
		outputCleanupJSON(result, &security.CleanupResult{
			Removed: toRemove,
			DryRun:  false,
		}, nil)
	} else {
		fmt.Printf("✓ Removed %d package(s)\n", len(toRemove))
	}

	return nil
}

func outputCleanupJSON(result *security.RedundancyResult, cleanup *security.CleanupResult, err error) {
	output := security.CleanupResultJSON{}

	if err != nil {
		output.Error = err.Error()
	} else {
		if result != nil {
			output.Redundancies = result.Redundancies
			summary := result.Summary()
			output.Summary = &summary
		}
		if cleanup != nil {
			output.Cleanup = cleanup
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

func outputCleanupText(result *security.RedundancyResult, quiet bool) {
	summary := result.Summary()

	// Print header
	fmt.Printf("Redundancy Analysis (%s)\n", result.Checker)
	fmt.Println(strings.Repeat("─", 50))

	if len(result.Redundancies) == 0 {
		fmt.Println("✓ No redundancies detected")
		return
	}

	// Print summary
	fmt.Printf("\nSummary: %d redundancies found (%d packages removable)\n", summary.Total, summary.Removable)
	printRedundancySummaryBar(summary)

	if quiet {
		fmt.Println()
		fmt.Println("Run 'preflight cleanup --all' to clean up")
		return
	}

	// Print by type
	duplicates := result.Redundancies.ByType(security.RedundancyDuplicate)
	if len(duplicates) > 0 {
		fmt.Println()
		fmt.Printf("Version Duplicates (%d)\n", len(duplicates))
		printRedundancyTable(duplicates)
	}

	overlaps := result.Redundancies.ByType(security.RedundancyOverlap)
	if len(overlaps) > 0 {
		fmt.Println()
		fmt.Printf("Overlapping Tools (%d)\n", len(overlaps))
		printOverlapTable(overlaps)
	}

	orphans := result.Redundancies.ByType(security.RedundancyOrphan)
	if len(orphans) > 0 {
		fmt.Println()
		fmt.Printf("Orphaned Dependencies (%d packages)\n", orphans.TotalRemovable())
		for _, o := range orphans {
			fmt.Printf("  %s\n", strings.Join(o.Packages, ", "))
			fmt.Printf("  → Run: %s\n", o.Action)
		}
	}

	// Print actions
	fmt.Println()
	fmt.Println("Actions:")
	if len(duplicates) > 0 {
		var removable []string
		for _, d := range duplicates {
			removable = append(removable, d.Remove...)
		}
		if len(removable) > 0 {
			fmt.Printf("  preflight cleanup --remove %s\n", strings.Join(removable, " "))
		}
	}
	if len(orphans) > 0 {
		fmt.Println("  preflight cleanup --autoremove")
	}
	fmt.Println("  preflight cleanup --all    # Remove all")
}

func printRedundancySummaryBar(summary security.RedundancySummary) {
	parts := []struct {
		label string
		count int
		color string
	}{
		{"DUPLICATES", summary.Duplicates, "\033[93m"},
		{"OVERLAPS", summary.Overlaps, "\033[33m"},
		{"ORPHANS", summary.Orphans, "\033[34m"},
	}

	fmt.Print("  ")
	for _, p := range parts {
		if p.count > 0 {
			fmt.Printf("%s%s: %d\033[0m  ", p.color, p.label, p.count)
		}
	}
	fmt.Println()
}

func printRedundancyTable(redundancies security.Redundancies) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, r := range redundancies {
		pkgsStr := strings.Join(r.Packages, " + ")
		fmt.Printf("  %s\n", pkgsStr)
		fmt.Printf("    → %s\n", r.Recommendation)
		if len(r.Remove) > 0 {
			fmt.Printf("    → Remove: %s\n", strings.Join(r.Remove, ", "))
		}
	}

	_ = w.Flush()
}

func printOverlapTable(redundancies security.Redundancies) {
	for _, r := range redundancies {
		pkgsStr := strings.Join(r.Packages, ", ")
		fmt.Printf("  %s: %s\n", formatCategory(r.Category), pkgsStr)
		fmt.Printf("    → %s\n", r.Recommendation)
		if len(r.Remove) > 0 && len(r.Keep) > 0 {
			fmt.Printf("    → Keep: %s, Remove: %s\n", strings.Join(r.Keep, ", "), strings.Join(r.Remove, ", "))
		}
	}
}

func formatCategory(category string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(category, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
