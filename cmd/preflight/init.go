package main

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new preflight configuration",
	Long: `Initialize a new preflight configuration using an interactive wizard.

The wizard will guide you through:
  - Selecting providers (nvim, shell, git, brew, etc.)
  - Choosing presets for each provider
  - Configuring targets and layers

Examples:
  preflight init                    # Interactive wizard
  preflight init --provider nvim    # Start with nvim provider
  preflight init --preset balanced  # Use balanced preset
  preflight init --yes              # Accept defaults`,
	RunE: runInit,
}

var (
	initProvider    string
	initPreset      string
	initSkipWelcome bool
	initYes         bool
)

func init() {
	initCmd.Flags().StringVar(&initProvider, "provider", "", "Pre-select a provider")
	initCmd.Flags().StringVar(&initPreset, "preset", "", "Pre-select a preset")
	initCmd.Flags().BoolVar(&initSkipWelcome, "skip-welcome", false, "Skip the welcome screen")
	initCmd.Flags().BoolVarP(&initYes, "yes", "y", false, "Accept all defaults")

	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	// Check if config already exists
	if _, err := os.Stat("preflight.yaml"); err == nil {
		fmt.Println("preflight.yaml already exists.")
		fmt.Println("Use 'preflight plan' to review your configuration.")
		return nil
	}

	// Build wizard options
	opts := tui.NewInitWizardOptions()

	if initProvider != "" {
		opts = opts.WithPreselectedProvider(initProvider)
	}
	if initPreset != "" {
		opts = opts.WithPreselectedPreset(initPreset)
	}
	if initSkipWelcome || initYes {
		opts = opts.WithSkipWelcome(true)
	}

	// Run the wizard
	ctx := context.Background()
	result, err := tui.RunInitWizard(ctx, opts)
	if err != nil {
		return fmt.Errorf("init wizard failed: %w", err)
	}

	if result.Cancelled {
		fmt.Println("Initialization cancelled.")
		return nil
	}

	fmt.Printf("Configuration created: %s\n", result.ConfigPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  preflight plan   - Review the execution plan")
	fmt.Println("  preflight apply  - Apply the configuration")

	return nil
}
