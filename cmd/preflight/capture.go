package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/felixgeelhaar/preflight/internal/adapters/filesystem"
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
  preflight capture                           # Interactive capture
  preflight capture --all                     # Accept all discovered items
  preflight capture --yes                     # Accept all (alias for --all)
  preflight capture --provider brew           # Only capture Homebrew packages
  preflight capture --all --smart-split       # Organize into logical layers (category-based)
  preflight capture --all --split-by language # Organize by programming language
  preflight capture --all --split-by stack    # Organize by tech stack (frontend, backend, devops)
  preflight capture --all --split-by provider # Organize by provider (brew, git, vscode)
  preflight capture --include-configs         # Copy config files to dotfiles/
  preflight capture --include-configs -t work # Copy configs to dotfiles.work/

Split strategies:
  category (default) - Fine-grained categories (base, dev-go, security, containers)
  language           - By programming language (go, node, python, rust, java)
  stack              - By tech stack role (frontend, backend, devops, data, security)
  provider           - By provider name (brew, git, shell, vscode)

The --smart-split flag is equivalent to --split-by category.`,
	RunE: runCapture,
}

var (
	captureAll            bool
	captureProvider       string
	captureOutput         string
	captureTarget         string
	captureSmartSplit     bool
	captureSplitBy        string
	captureIncludeConfigs bool
)

func init() {
	captureCmd.Flags().BoolVar(&captureAll, "all", false, "Accept all discovered items")
	captureCmd.Flags().StringVar(&captureProvider, "provider", "", "Only capture specific provider")
	captureCmd.Flags().StringVarP(&captureOutput, "output", "o", ".", "Output directory for generated config")
	captureCmd.Flags().StringVarP(&captureTarget, "target", "t", "default", "Target name for the configuration")
	captureCmd.Flags().BoolVar(&captureSmartSplit, "smart-split", false, "Automatically organize packages into logical layer files (equivalent to --split-by category)")
	captureCmd.Flags().StringVar(&captureSplitBy, "split-by", "", fmt.Sprintf("Split strategy for layer organization (%s)", strings.Join(app.ValidSplitStrategies(), ", ")))
	captureCmd.Flags().BoolVar(&captureIncludeConfigs, "include-configs", false, "Copy config files to dotfiles/ directory for full reproducibility")

	rootCmd.AddCommand(captureCmd)
}

func runCapture(_ *cobra.Command, _ []string) error {
	// Respect both --all flag and global --yes flag
	acceptAll := captureAll || yesFlag

	opts := tui.NewCaptureReviewOptions().
		WithAcceptAll(acceptAll)

	if acceptAll {
		opts.Interactive = false
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

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

		// Capture config files BEFORE config generation if requested
		// This allows the generator to populate config_source fields
		var dotfilesResult *app.DotfilesCaptureResult
		if captureIncludeConfigs {
			var err error
			dotfilesResult, err = captureDotfiles()
			if err != nil {
				return fmt.Errorf("dotfiles capture failed: %w", err)
			}
		}

		// Generate configuration from accepted items
		generator := app.NewCaptureConfigGenerator(captureOutput)

		// Pass dotfiles result to generator for config_source population
		if dotfilesResult != nil {
			generator.WithDotfiles(dotfilesResult)
		}

		// Handle split strategy
		var strategy app.SplitStrategy
		var usingSplit bool

		if captureSplitBy != "" {
			var err error
			strategy, err = app.ParseSplitStrategy(captureSplitBy)
			if err != nil {
				return err
			}
			generator.WithSplitStrategy(strategy)
			usingSplit = true
		} else if captureSmartSplit {
			generator.WithSmartSplit(true)
			strategy = app.SplitByCategory
			usingSplit = true
		}

		if err := generator.GenerateFromCapture(filteredFindings, captureTarget); err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}

		if usingSplit {
			fmt.Printf("\nGenerated configuration in %s/ using '%s' split strategy\n", captureOutput, strategy)
			fmt.Println("Packages organized into logical layer files.")
		} else {
			fmt.Printf("\nGenerated configuration in %s/\n", captureOutput)
		}

		if dotfilesResult != nil && dotfilesResult.FileCount() > 0 {
			fmt.Println("Config sources automatically linked to layer configuration.")
		}

		fmt.Println("Run 'preflight plan' to review the changes.")
	}

	return nil
}

// captureDotfiles copies config files to the dotfiles/ directory.
// Returns the capture result so it can be passed to the config generator.
func captureDotfiles() (*app.DotfilesCaptureResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	fs := filesystem.NewRealFileSystem()
	capturer := app.NewDotfilesCapturer(fs, homeDir, captureOutput)

	// Use per-target directory if target is specified and not "default"
	if captureTarget != "" && captureTarget != "default" {
		capturer = capturer.WithTarget(captureTarget)
	}

	result, err := capturer.Capture()
	if err != nil {
		return nil, err
	}

	// Print warnings
	for _, warning := range result.Warnings {
		fmt.Printf("⚠ Warning: %s\n", warning)
	}

	// Print broken symlinks
	if len(result.BrokenSymlinks) > 0 {
		fmt.Printf("\n⚠ Skipped %d broken symlink(s):\n", len(result.BrokenSymlinks))
		for _, bs := range result.BrokenSymlinks {
			if bs.Target != "" {
				fmt.Printf("  %s -> %s (target does not exist)\n", bs.Path, bs.Target)
			} else {
				fmt.Printf("  %s (broken symlink)\n", bs.Path)
			}
		}
	}

	if result.FileCount() > 0 {
		fmt.Printf("\nCaptured %d config files to %s/\n", result.FileCount(), result.TargetDir)

		// Show summary by provider
		byProvider := result.DotfilesByProvider()
		for provider, files := range byProvider {
			fileCount := 0
			for _, f := range files {
				if !f.IsDirectory {
					fileCount++
				}
			}
			if fileCount > 0 {
				fmt.Printf("  %s: %d files\n", provider, fileCount)
			}
		}
	} else {
		fmt.Println("\nNo config files found to capture.")
	}

	return result, nil
}
