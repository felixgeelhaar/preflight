package main

import (
	"context"
	"os"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences between configuration and system",
	Long: `Diff shows the differences between your configuration and the current system state.

It uses unified diff format to highlight changes that would be made.

Examples:
  preflight diff                    # Show all differences
  preflight diff --provider brew    # Show diff for specific provider`,
	RunE: runDiff,
}

var (
	diffProvider string
)

func init() {
	diffCmd.Flags().StringVar(&diffProvider, "provider", "", "Filter by provider")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(_ *cobra.Command, _ []string) error {
	configPath := cfgFile
	if configPath == "" {
		configPath = "preflight.yaml"
	}

	target := "default"

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	result, err := preflight.Diff(ctx, configPath, target)
	if err != nil {
		return err
	}

	preflight.PrintDiff(result)
	return nil
}
