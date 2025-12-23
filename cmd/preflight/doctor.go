package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify system state and detect drift",
	Long: `Doctor checks your system against the expected configuration state.

It detects drift (changes made outside of preflight) and can suggest fixes.

Examples:
  preflight doctor              # Check for drift
  preflight doctor --fix        # Auto-fix detected issues
  preflight doctor --verbose    # Show detailed output`,
	RunE: runDoctor,
}

var (
	doctorFix     bool
	doctorVerbose bool
)

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Automatically fix detected issues")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show detailed output")

	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Resolve config path
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	// Create app instance
	preflight := app.New(os.Stdout)

	// Run doctor check
	doctorOpts := app.NewDoctorOptions(configPath, "default").
		WithVerbose(doctorVerbose)

	appReport, err := preflight.Doctor(ctx, doctorOpts)
	if err != nil {
		return fmt.Errorf("doctor check failed: %w", err)
	}

	// Convert to TUI types
	tuiReport := tui.ConvertDoctorReport(appReport)

	// Setup TUI options
	tuiOpts := tui.NewDoctorReportOptions().
		WithAutoFix(doctorFix)

	if !doctorVerbose {
		tuiOpts.Verbose = false
	}

	// Run TUI report display
	result, err := tui.RunDoctorReport(ctx, tuiReport, tuiOpts)
	if err != nil {
		return fmt.Errorf("doctor display failed: %w", err)
	}

	// Handle fix if requested
	switch {
	case doctorFix && appReport.FixableCount() > 0:
		fixResult, err := preflight.Fix(ctx, appReport)
		if err != nil {
			return fmt.Errorf("fix failed: %w", err)
		}
		fmt.Printf("Fixed %d of %d issues.\n", fixResult.FixedCount(), appReport.FixableCount())
		if fixResult.RemainingCount() > 0 {
			fmt.Printf("%d issues could not be automatically fixed.\n", fixResult.RemainingCount())
		}
	case result.Issues == 0:
		fmt.Println("No issues found. Your system is in sync.")
	case appReport.FixableCount() > 0:
		fmt.Println("\nRun 'preflight doctor --fix' to automatically fix issues.")
	}

	return nil
}
