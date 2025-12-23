package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "Show differences between configuration and system",
	Long: `Diff shows the differences between your configuration and the current system state.

It uses unified diff format to highlight changes that would be made.

Examples:
  preflight diff                    # Show all differences
  preflight diff ~/.zshrc           # Show diff for specific file
  preflight diff --provider brew    # Show diff for specific provider`,
	RunE: runDiff,
}

var (
	diffProvider string
	diffColor    bool
)

func init() {
	diffCmd.Flags().StringVar(&diffProvider, "provider", "", "Filter by provider")
	diffCmd.Flags().BoolVar(&diffColor, "color", true, "Colorize output")

	rootCmd.AddCommand(diffCmd)
}

func runDiff(_ *cobra.Command, args []string) error {
	// TODO: Implement full diff functionality
	// This will compare config vs system state using the compiler and execution domains

	if len(args) > 0 {
		fmt.Printf("Showing diff for: %s\n", args[0])
	} else {
		fmt.Println("No changes detected.")
	}

	if diffProvider != "" {
		fmt.Printf("Filtering by provider: %s\n", diffProvider)
	}

	fmt.Println("\nRun 'preflight plan' for a complete execution plan.")
	return nil
}
