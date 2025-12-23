package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture current machine configuration",
	Long: `Capture reverse-engineers your current machine setup into a preflight configuration.

It scans for installed packages, dotfiles, and settings, allowing you to
selectively import them into your configuration.

Examples:
  preflight capture                # Interactive capture
  preflight capture --all          # Accept all discovered items
  preflight capture --provider brew # Only capture Homebrew packages`,
	RunE: runCapture,
}

var (
	captureAll      bool
	captureProvider string
	captureOutput   string
	captureTarget   string
)

func init() {
	captureCmd.Flags().BoolVar(&captureAll, "all", false, "Accept all discovered items")
	captureCmd.Flags().StringVar(&captureProvider, "provider", "", "Only capture specific provider")
	captureCmd.Flags().StringVarP(&captureOutput, "output", "o", ".", "Output directory for generated config")
	captureCmd.Flags().StringVarP(&captureTarget, "target", "t", "default", "Target name for the configuration")

	rootCmd.AddCommand(captureCmd)
}

func runCapture(_ *cobra.Command, _ []string) error {
	opts := tui.NewCaptureReviewOptions().
		WithAcceptAll(captureAll)

	if captureAll {
		opts.Interactive = false
	}

	ctx := context.Background()

	// Create app instance and capture items
	preflight := app.New(os.Stdout)
	captureOpts := app.NewCaptureOptions()
	if captureProvider != "" {
		captureOpts = captureOpts.WithProviders(captureProvider)
	}

	findings, err := preflight.Capture(ctx, captureOpts)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	// Print any capture warnings
	for _, warning := range findings.Warnings {
		fmt.Printf("Warning: %s\n", warning)
	}

	// Convert app items to TUI items
	items := tui.ConvertCapturedItems(findings.Items)

	// If no items found, report and exit early
	if len(items) == 0 {
		fmt.Println("No items found to capture.")
		return nil
	}

	result, err := tui.RunCaptureReview(ctx, items, opts)
	if err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	if result.Cancelled {
		fmt.Println("Capture cancelled.")
		return nil
	}

	accepted := len(result.AcceptedItems)
	rejected := len(result.RejectedItems)

	fmt.Printf("Captured %d items (%d rejected).\n", accepted, rejected)

	if accepted > 0 {
		// Filter findings to only include accepted items
		acceptedSet := make(map[string]bool)
		for _, name := range result.AcceptedItems {
			acceptedSet[name] = true
		}

		filteredItems := make([]app.CapturedItem, 0, len(result.AcceptedItems))
		for _, item := range findings.Items {
			if acceptedSet[item.Name] {
				filteredItems = append(filteredItems, item)
			}
		}

		filteredFindings := &app.CaptureFindings{
			Items:      filteredItems,
			Providers:  findings.Providers,
			CapturedAt: findings.CapturedAt,
			HomeDir:    findings.HomeDir,
		}

		// Generate configuration from accepted items
		generator := app.NewCaptureConfigGenerator(captureOutput)
		if err := generator.GenerateFromCapture(filteredFindings, captureTarget); err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}

		fmt.Printf("\nGenerated configuration in %s/\n", captureOutput)
		fmt.Println("Run 'preflight plan' to review the changes.")
	}

	return nil
}
