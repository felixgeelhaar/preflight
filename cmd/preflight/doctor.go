package main

import (
	"context"
	"fmt"

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
	opts := tui.NewDoctorReportOptions().
		WithAutoFix(doctorFix)

	if !doctorVerbose {
		opts.Verbose = false
	}

	ctx := context.Background()
	result, err := tui.RunDoctorReport(ctx, opts)
	if err != nil {
		return fmt.Errorf("doctor failed: %w", err)
	}

	if result.Issues == 0 {
		fmt.Println("No issues found. Your system is in sync.")
		return nil
	}

	fmt.Printf("Found %d issues.\n", result.Issues)

	if doctorFix {
		fmt.Printf("Fixed %d of %d issues.\n", result.Fixed, result.FixesFound)
	} else {
		fmt.Println("\nRun 'preflight doctor --fix' to automatically fix issues.")
	}

	return nil
}
