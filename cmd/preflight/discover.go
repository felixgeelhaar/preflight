package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/adapters/command"
	"github.com/felixgeelhaar/preflight/internal/adapters/github"
	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover popular dotfile configurations",
	Long: `Discover analyzes popular dotfile repositories to suggest configurations.

It searches GitHub for highly-starred dotfile repositories, analyzes their
structure, and provides suggestions for your preflight configuration.

Examples:
  preflight discover                     # Analyze top dotfile repos
  preflight discover --max-repos 100     # Analyze more repositories
  preflight discover --min-stars 50      # Only repos with 50+ stars
  preflight discover --language shell    # Filter by language`,
	RunE: runDiscover,
}

var (
	discoverMaxRepos int
	discoverMinStars int
	discoverLanguage string
	discoverShowAll  bool
)

func init() {
	discoverCmd.Flags().IntVar(&discoverMaxRepos, "max-repos", 50, "Maximum repositories to analyze")
	discoverCmd.Flags().IntVar(&discoverMinStars, "min-stars", 10, "Minimum star count")
	discoverCmd.Flags().StringVar(&discoverLanguage, "language", "", "Filter by language (e.g., shell, vim)")
	discoverCmd.Flags().BoolVar(&discoverShowAll, "all", false, "Show all detected patterns")

	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create the GitHub source
	runner := command.NewRealRunner()
	source := github.NewDiscoverSource(runner)

	// Create the analyzer
	analyzer := discover.NewAnalyzer(source)

	// Build discovery options
	opts := discover.DiscoveryOptions{
		Source:   "github",
		MinStars: discoverMinStars,
		MaxRepos: discoverMaxRepos,
		Language: discoverLanguage,
	}

	fmt.Println("Analyzing popular dotfile repositories...")
	fmt.Println()

	// Run analysis
	result, err := analyzer.Analyze(ctx, opts)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	if result.ReposAnalyzed == 0 {
		fmt.Println("No repositories found matching your criteria.")
		fmt.Println("\nTry adjusting --min-stars or --language filters.")
		return nil
	}

	fmt.Printf("Analyzed %d repositories from %s\n\n", result.ReposAnalyzed, result.Source)

	// Show detected patterns
	if len(result.Patterns) == 0 {
		fmt.Println("No configuration patterns detected.")
		return nil
	}

	// Generate suggestions
	generator := discover.NewSuggestionGenerator()
	suggestions := generator.Generate(result.Patterns, result.ReposAnalyzed)

	// Display patterns summary
	fmt.Println("Detected Patterns")
	fmt.Println("=================")
	fmt.Println()

	maxPatterns := 10
	if discoverShowAll {
		maxPatterns = len(result.Patterns)
	}

	for i, p := range result.Patterns {
		if i >= maxPatterns {
			remaining := len(result.Patterns) - maxPatterns
			fmt.Printf("\n  ... and %d more patterns (use --all to show all)\n", remaining)
			break
		}

		percentage := (p.Occurrences * 100) / result.ReposAnalyzed
		typeIcon := getPatternIcon(p.Type)
		fmt.Printf("  %s %-20s %3d repos (%d%%)\n", typeIcon, p.Name, p.Occurrences, percentage)
	}

	// Show top suggestions
	fmt.Println()
	fmt.Println("Suggested Configuration")
	fmt.Println("=======================")
	fmt.Println()

	var topSuggestions []discover.Suggestion
	if len(suggestions) < 5 {
		topSuggestions = suggestions
	} else {
		topSuggestions = suggestions[:5]
	}

	for i, s := range topSuggestions {
		fmt.Printf("%d. %s\n", i+1, s.Title)
		fmt.Printf("   %s\n", s.Description)
		if len(s.Reasons) > 0 {
			fmt.Printf("   → %s\n", s.Reasons[0])
		}
		fmt.Println()

		// Show config snippet
		if verbose {
			fmt.Println("   Config snippet:")
			for _, line := range strings.Split(s.ConfigSnippet, "\n") {
				fmt.Printf("     %s\n", line)
			}
			fmt.Println()
		}

		// Show doc links
		if len(s.Links) > 0 && verbose {
			fmt.Printf("   Docs: %s\n", s.Links[0])
			fmt.Println()
		}
	}

	if !verbose {
		fmt.Println("Tip: Use --verbose to see configuration snippets and documentation links.")
	}

	return nil
}

// getPatternIcon returns an icon for the pattern type.
func getPatternIcon(t discover.PatternType) string {
	switch t {
	case discover.PatternTypeShell:
		return "🐚"
	case discover.PatternTypeEditor:
		return "📝"
	case discover.PatternTypeGit:
		return "📦"
	case discover.PatternTypeSSH:
		return "🔐"
	case discover.PatternTypeTmux:
		return "🖥️"
	case discover.PatternTypePackageManager:
		return "📦"
	default:
		return "•"
	}
}
