package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
)

var (
	tourListFlag bool
)

var tourCmd = &cobra.Command{
	Use:   "tour [topic]",
	Short: "Interactive guided walkthroughs",
	Long: `Tour provides interactive guided walkthroughs of preflight features.

Available topics:
  basics      - Preflight fundamentals
  config      - Configuration deep-dive
  layers      - Layer composition
  providers   - Provider overview
  presets     - Using presets
  workflow    - Daily workflow

Examples:
  preflight tour            # Open topic menu
  preflight tour basics     # Start the basics tour
  preflight tour --list     # List available topics`,
	RunE: runTour,
}

func init() {
	tourCmd.Flags().BoolVar(&tourListFlag, "list", false, "List available topics")
	rootCmd.AddCommand(tourCmd)
}

func runTour(_ *cobra.Command, args []string) error {
	// Handle --list flag
	if tourListFlag {
		printTourTopics()
		return nil
	}

	// Get initial topic if provided
	var initialTopic string
	if len(args) > 0 {
		initialTopic = args[0]
		// Validate topic exists
		if _, found := tui.GetTopic(initialTopic); !found {
			validTopics := tui.GetTopicIDs()
			return fmt.Errorf("unknown topic: %s\nAvailable topics: %s",
				initialTopic, strings.Join(validTopics, ", "))
		}
	}

	// Initialize progress store for tracking
	progressStore, err := tui.NewTourProgressStore()
	if err != nil {
		// Non-fatal: continue without progress tracking
		progressStore = nil
	}

	// Build tour options
	opts := tui.NewTourOptions()
	if initialTopic != "" {
		opts = opts.WithInitialTopic(initialTopic)
	}
	if progressStore != nil {
		opts = opts.WithProgressStore(progressStore)
	}

	// Run the interactive tour
	ctx := context.Background()
	result, err := tui.RunTour(ctx, opts)
	if err != nil {
		return fmt.Errorf("tour failed: %w", err)
	}

	if result.Cancelled {
		return nil
	}

	// Show progress and next steps after completing tour
	fmt.Println()
	if result.TopicsCompleted > 0 {
		fmt.Printf("Progress: %d/%d topics completed\n", result.TopicsCompleted, result.TotalTopics)
		fmt.Println()
	}

	if result.TopicsCompleted == result.TotalTopics && result.TotalTopics > 0 {
		fmt.Println("🎉 Congratulations! You've completed all tour topics!")
		fmt.Println()
	}

	fmt.Println("Ready to get started?")
	fmt.Println()
	fmt.Println("  preflight init      Create new configuration")
	fmt.Println("  preflight capture   Capture current machine")
	fmt.Println("  preflight --help    See all commands")

	return nil
}

func printTourTopics() {
	topics := tui.GetAllTopics()

	fmt.Println("Available tour topics:")
	fmt.Println()
	for _, topic := range topics {
		fmt.Printf("  %-12s %s\n", topic.ID, topic.Description)
	}
	fmt.Println()
	fmt.Println("Run 'preflight tour <topic>' to start a tour.")
}
