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
  preflight capture                   # Interactive capture
  preflight capture --all             # Accept all discovered items
  preflight capture --provider brew   # Only capture Homebrew packages
  preflight capture --all --smart-split  # Organize into logical layers

The --smart-split flag automatically categorizes packages into logical layers:
  base.yaml       - Core CLI utilities (git, curl, jq, ripgrep)
  dev-go.yaml     - Go development tools (gopls, golangci-lint)
  dev-node.yaml   - Node.js ecosystem (node, pnpm, typescript)
  dev-python.yaml - Python tools (poetry, ruff, mypy)
  security.yaml   - Security scanning tools (trivy, grype, nmap)
  containers.yaml - Container/K8s tools (docker, kubectl, helm)
  And more...`,
	RunE: runCapture,
}

var (
	captureAll        bool
	captureProvider   string
	captureOutput     string
	captureTarget     string
	captureSmartSplit bool
)

func init() {
	captureCmd.Flags().BoolVar(&captureAll, "all", false, "Accept all discovered items")
	captureCmd.Flags().StringVar(&captureProvider, "provider", "", "Only capture specific provider")
	captureCmd.Flags().StringVarP(&captureOutput, "output", "o", ".", "Output directory for generated config")
	captureCmd.Flags().StringVarP(&captureTarget, "target", "t", "default", "Target name for the configuration")
	captureCmd.Flags().BoolVar(&captureSmartSplit, "smart-split", false, "Automatically organize packages into logical layer files")

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
		for _, item := range result.AcceptedItems {
			acceptedSet[item.Name] = true
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
		generator := app.NewCaptureConfigGenerator(captureOutput).
			WithSmartSplit(captureSmartSplit)
		if err := generator.GenerateFromCapture(filteredFindings, captureTarget); err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}

		if captureSmartSplit {
			fmt.Printf("\nGenerated smart-split configuration in %s/\n", captureOutput)
			fmt.Println("Packages organized into logical layer files.")
		} else {
			fmt.Printf("\nGenerated configuration in %s/\n", captureOutput)
		}
		fmt.Println("Run 'preflight plan' to review the changes.")
	}

	return nil
}
