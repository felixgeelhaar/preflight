package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
	"github.com/felixgeelhaar/preflight/internal/telemetry"
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
	applyCmd.Flags().BoolVar(&applyRollback, "rollback-on-error", true, "Attempt rollback when a step fails (disable with --rollback-on-error=false)")
}

func runApply(cmd *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create the application
	preflight := newPreflight(os.Stdout)
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight = preflight.WithMode(*modeOverride)
	}
	preflight = preflight.WithRollbackOnFailure(applyRollback)

	// Create the plan
	plan, err := preflight.Plan(ctx, applyConfigPath, applyTarget)
	if err != nil {
		return (&config.UserError{
			Code:       config.ErrCodeValidationFailed,
			Message:    "could not generate plan from your configuration",
			Suggestion: "Run 'preflight validate' to check your config for issues, or 'preflight diff' to see what differs.",
			Underlying: err,
		})
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
			return &config.UserError{
				Code:       "BOOTSTRAP_DECLINED",
				Message:    "bootstrap steps declined; nothing was applied",
				Suggestion: "Re-run 'preflight apply' and confirm the bootstrap prompt, or pass --allow-bootstrap to skip the prompt.",
			}
		}
	}

	fmt.Println("\nApplying changes...")

	// Execute the plan
	results, err := preflight.Apply(ctx, plan, applyDryRun)
	// Print results before deciding what to return so the user always sees
	// per-step status, even on partial failure.
	preflight.PrintResults(results)

	// Collect per-step failures regardless of whether Apply itself returned an
	// error — Execute now joins step errors but legacy callers / fakes may
	// return (results, nil) with errored results inside.
	failedIDs := make([]string, 0, len(results))
	for i := range results {
		if results[i].Error() != nil {
			failedIDs = append(failedIDs, results[i].StepID().String())
		}
	}

	if err != nil || len(failedIDs) > 0 {
		suggestion := "Inspect the failing step's output above. Run 'preflight rollback' to restore the previous state, or 'preflight doctor' to diagnose the underlying issue."
		if len(failedIDs) == 1 {
			suggestion = fmt.Sprintf("Step %q failed. Run 'preflight rollback' to restore previous state, or 'preflight doctor --verbose' to diagnose.", failedIDs[0])
		}
		msg := "apply failed: some steps failed"
		if len(failedIDs) > 0 {
			msg = fmt.Sprintf("apply failed: %d step(s) did not complete", len(failedIDs))
		}
		return &config.UserError{
			Code:       "APPLY_FAILED",
			Message:    msg,
			Suggestion: suggestion,
			Underlying: err,
		}
	}

	// First successful apply — record activation event for the North Star
	// metric (Time-to-First-Successful-Apply). RecordOnce fires only on the
	// first successful apply per machine; subsequent applies are no-ops.
	// Recorder is opt-in and a no-op until the user has granted consent.
	recordOnce(telemetry.EventApplyFirstOK)

	if applyUpdateLock {
		if err := preflight.UpdateLockFromPlan(ctx, applyConfigPath, plan); err != nil {
			return &config.UserError{
				Code:       "LOCK_UPDATE_FAILED",
				Message:    "could not update preflight.lock after apply",
				Suggestion: "Apply succeeded; only the lockfile update failed. Re-run 'preflight lock --update' once write access to the file is restored.",
				Underlying: err,
			}
		}
	}

	return nil
}
