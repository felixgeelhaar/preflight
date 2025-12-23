package main

import (
	"context"
	"fmt"

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
)

func init() {
	captureCmd.Flags().BoolVar(&captureAll, "all", false, "Accept all discovered items")
	captureCmd.Flags().StringVar(&captureProvider, "provider", "", "Only capture specific provider")

	rootCmd.AddCommand(captureCmd)
}

func runCapture(_ *cobra.Command, _ []string) error {
	opts := tui.NewCaptureReviewOptions().
		WithAcceptAll(captureAll)

	if captureAll {
		opts.Interactive = false
	}

	// TODO: Run actual capture to discover items
	// For now, create an empty list that will show "Nothing captured"
	items := []tui.CaptureItem{}

	ctx := context.Background()
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
		fmt.Println("\nRun 'preflight plan' to review the changes.")
	}

	return nil
}
