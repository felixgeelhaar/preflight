package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
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
	applyUpdateLock bool
	applyRollback   bool
)

type preflightClient interface {
	Plan(context.Context, string, string) (*execution.Plan, error)
	PrintPlan(*execution.Plan)
	Apply(context.Context, *execution.Plan, bool) ([]execution.StepResult, error)
	PrintResults([]execution.StepResult)
	UpdateLockFromPlan(context.Context, string, *execution.Plan) error
	WithMode(config.ReproducibilityMode) preflightClient
	WithRollbackOnFailure(bool) preflightClient
}

type preflightAdapter struct {
	*app.Preflight
}

var newPreflight = func(out io.Writer) preflightClient {
	return &preflightAdapter{app.New(out)}
}

func (p *preflightAdapter) WithMode(mode config.ReproducibilityMode) preflightClient {
	return &preflightAdapter{p.Preflight.WithMode(mode)}
}

func (p *preflightAdapter) WithRollbackOnFailure(enabled bool) preflightClient {
	return &preflightAdapter{p.Preflight.WithRollbackOnFailure(enabled)}
}

func init() {
	rootCmd.AddCommand(applyCmd)

	applyCmd.Flags().StringVarP(&applyConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	applyCmd.Flags().StringVarP(&applyTarget, "target", "t", "default", "Target to apply")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show what would be done without making changes")
	applyCmd.Flags().BoolVar(&applyUpdateLock, "update-lock", false, "Update lockfile after apply")
	applyCmd.Flags().BoolVar(&applyRollback, "rollback-on-error", false, "Attempt rollback when a step fails")
}

func runApply(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the application
	preflight := newPreflight(os.Stdout)
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight = preflight.WithMode(*modeOverride)
	}
	if applyRollback {
		preflight = preflight.WithRollbackOnFailure(true)
	}

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

	if app.RequiresBootstrapConfirmation(plan) {
		steps := app.BootstrapSteps(plan)
		if !confirmBootstrap(steps) {
			return fmt.Errorf("aborted bootstrap steps")
		}
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

	if applyUpdateLock {
		if err := preflight.UpdateLockFromPlan(ctx, applyConfigPath, plan); err != nil {
			return fmt.Errorf("update lockfile failed: %w", err)
		}
	}

	return nil
}
