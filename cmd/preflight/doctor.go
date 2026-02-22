package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify system state and detect drift",
	Long: `Doctor checks your system against the expected configuration state.

It detects drift (changes made outside of preflight) and can suggest fixes.

Examples:
  preflight doctor                    # Check for drift
  preflight doctor --fix              # Auto-fix detected issues
  preflight doctor --verbose          # Show detailed output
  preflight doctor --update-config    # Merge drift back into config
  preflight doctor --update-config --dry-run  # Preview config changes`,
	RunE: runDoctor,
}

var (
	doctorFix          bool
	doctorVerbose      bool
	doctorUpdateConfig bool
	doctorDryRun       bool
	doctorQuiet        bool
)

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Automatically fix detected issues")
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "Show detailed output")
	doctorCmd.Flags().BoolVar(&doctorUpdateConfig, "update-config", false, "Merge drift back into layer files")
	doctorCmd.Flags().BoolVar(&doctorDryRun, "dry-run", false, "Show changes without writing (use with --update-config)")
	doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "Print results without TUI (for CI/scripts)")

	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Resolve config path
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	// Create app instance
	preflight := app.New(os.Stdout)

	// Run doctor check
	doctorOpts := app.NewDoctorOptions(configPath, "default").
		WithVerbose(doctorVerbose).
		WithUpdateConfig(doctorUpdateConfig).
		WithDryRun(doctorDryRun)

	appReport, err := preflight.Doctor(ctx, doctorOpts)
	if err != nil {
		return fmt.Errorf("doctor check failed: %w", err)
	}

	// Quiet mode: print results without TUI
	if doctorQuiet {
		printDoctorQuiet(appReport)
		return nil
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

	// Handle update-config if requested
	if doctorUpdateConfig && appReport.HasPatches() {
		if doctorDryRun {
			fmt.Printf("\n--- Dry Run: Would apply %d config patches ---\n", appReport.PatchCount())
			for layer, patches := range appReport.PatchesByLayer() {
				fmt.Printf("\n%s:\n", layer)
				for _, patch := range patches {
					fmt.Printf("  %s\n", patch.Description())
				}
			}
			return nil
		}

		// Apply patches using LayerWriter
		writer := config.NewLayerWriter()
		writerPatches := app.ConfigPatchesToWriterPatches(appReport.SuggestedPatches)
		if err := writer.ApplyPatches(writerPatches); err != nil {
			return fmt.Errorf("failed to apply config patches: %w", err)
		}

		fmt.Printf("✓ Applied %d patches to config.\n", appReport.PatchCount())
		return nil
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
	case appReport.HasPatches():
		fmt.Printf("\n%d config patches suggested. Run 'preflight doctor --update-config' to apply.\n", appReport.PatchCount())
	}

	return nil
}

// printDoctorQuiet prints the doctor report without TUI.
func printDoctorQuiet(report *app.DoctorReport) {
	fmt.Println("Doctor Report")
	fmt.Println("=============")
	fmt.Println()

	if report.IssueCount() == 0 {
		fmt.Println("✓ No issues found. Your system is in sync.")
		return
	}

	fmt.Printf("Found %d issue(s):\n\n", report.IssueCount())

	for _, issue := range report.Issues {
		status := "!"
		if issue.Severity == app.SeverityError {
			status = "✗"
		}
		fmt.Printf("  %s [%s] %s\n", status, issue.Severity, issue.Message)
		if issue.Provider != "" {
			fmt.Printf("      Provider: %s\n", issue.Provider)
		}
		if issue.Expected != "" && issue.Actual != "" {
			fmt.Printf("      Expected: %s\n", issue.Expected)
			fmt.Printf("      Actual: %s\n", issue.Actual)
		}
		if issue.FixCommand != "" {
			fmt.Printf("      Fix: %s\n", issue.FixCommand)
		}
	}

	if report.FixableCount() > 0 {
		fmt.Printf("\n%d issue(s) can be auto-fixed with 'preflight doctor --fix'\n", report.FixableCount())
	}

	if report.HasPatches() {
		fmt.Printf("%d config patches suggested. Run 'preflight doctor --update-config' to apply.\n", report.PatchCount())
	}
}
