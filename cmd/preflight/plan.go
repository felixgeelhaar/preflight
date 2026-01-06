package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show what changes preflight would make",
	Long: `Plan loads your configuration and shows what changes would be made.

This command:
1. Loads and merges configuration layers
2. Compiles config into executable steps
3. Checks current system state
4. Shows what would be changed (without making changes)`,
	RunE: runPlan,
}

var (
	planConfigPath string
	planTarget     string
)

func init() {
	rootCmd.AddCommand(planCmd)

	planCmd.Flags().StringVarP(&planConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	planCmd.Flags().StringVarP(&planTarget, "target", "t", "default", "Target to plan")
}

func runPlan(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the application
	preflight := app.New(os.Stdout)
	if modeOverride, err := resolveModeOverride(cmd); err != nil {
		return err
	} else if modeOverride != nil {
		preflight.WithMode(*modeOverride)
	}

	// Create the plan
	plan, err := preflight.Plan(ctx, planConfigPath, planTarget)
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	// Print the plan
	preflight.PrintPlan(plan)

	return nil
}
