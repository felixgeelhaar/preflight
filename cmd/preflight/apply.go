package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration changes to your system",
	Long: `Apply executes the plan and makes changes to your system.

This command:
1. Creates an execution plan (same as 'plan' command)
2. Executes each step in dependency order
3. Reports results

Use --dry-run to see what would happen without making changes.`,
	RunE: runApply,
}

var (
	applyConfigPath string
	applyTarget     string
	applyDryRun     bool
)

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&applyConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	applyCmd.Flags().StringVarP(&applyTarget, "target", "t", "default", "Target to apply")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show what would be done without making changes")
}

func runApply(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the application
	preflight := app.New(os.Stdout)

	// Create the plan
	plan, err := preflight.Plan(ctx, applyConfigPath, applyTarget)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Show the plan first
	preflight.PrintPlan(plan)

	// If no changes needed, we're done
	if !plan.HasChanges() {
		return nil
	}

	if applyDryRun {
		fmt.Println("\n[Dry run - no changes made]")
		return nil
	}

	fmt.Println("\nApplying changes...")

	// Execute the plan
	results, err := preflight.Apply(ctx, plan, applyDryRun)
	if err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	// Print results
	preflight.PrintResults(results)

	// Check for failures
	for i := range results {
		if results[i].Error() != nil {
			return fmt.Errorf("some steps failed")
		}
	}

	return nil
}
