package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/spf13/cobra"
)

var deprecatedCmd = &cobra.Command{
	Use:   "deprecated",
	Short: "Check for deprecated packages",
	Long: `Check for packages that are deprecated or disabled.

This command checks installed packages for deprecation warnings and
reports them with their reason and any available alternatives.

Checkers (in order of preference):
  - brew: Homebrew formulae deprecation and disable status

Exit codes:
  0 - No deprecated packages found
  1 - Deprecated or disabled packages found
  2 - Checker not available or check failed

Examples:
  preflight deprecated              # Check for deprecated packages
  preflight deprecated --json       # JSON output for CI
  preflight deprecated --ignore pkg # Ignore specific packages`,
	RunE: runDeprecated,
}

var (
	deprecatedIgnore []string
	deprecatedJSON   bool
	deprecatedQuiet  bool
)

func init() {
	rootCmd.AddCommand(deprecatedCmd)

	deprecatedCmd.Flags().StringSliceVar(&deprecatedIgnore, "ignore", nil, "Package names to ignore (can be specified multiple times)")
	deprecatedCmd.Flags().BoolVar(&deprecatedJSON, "json", false, "Output results as JSON")
	deprecatedCmd.Flags().BoolVarP(&deprecatedQuiet, "quiet", "q", false, "Only show summary")
}

func runDeprecated(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create checker registry
	registry := security.NewDeprecationCheckerRegistry()
	registry.Register(security.NewBrewDeprecationChecker())

	// Get first available checker
	checkers := registry.All()
	if len(checkers) == 0 {
		if deprecatedJSON {
			outputDeprecatedJSON(nil, fmt.Errorf("no checkers available"))
		} else {
			fmt.Fprintf(os.Stderr, "Error: no deprecation checkers available. Install Homebrew.\n")
		}
		os.Exit(2)
	}

	// Configure options
	opts := security.DeprecationOptions{
		IgnorePackages: deprecatedIgnore,
	}

	// Run checks with all available checkers
	var allPackages security.DeprecatedPackages
	var checkerName string

	for _, checker := range checkers {
		result, err := checker.Check(ctx, opts)
		if err != nil {
			if deprecatedJSON {
				outputDeprecatedJSON(nil, err)
			} else {
				fmt.Fprintf(os.Stderr, "Check failed (%s): %v\n", checker.Name(), err)
			}
			os.Exit(2)
		}

		checkerName = checker.Name()
		allPackages = append(allPackages, result.Packages...)
	}

	// Build combined result
	result := &security.DeprecatedResult{
		Checker:  checkerName,
		Packages: allPackages,
	}

	// Output results
	if deprecatedJSON {
		outputDeprecatedJSON(result, nil)
	} else {
		outputDeprecatedText(result, deprecatedQuiet)
	}

	// Determine exit code
	if len(result.Packages) > 0 {
		os.Exit(1)
	}

	return nil
}

func outputDeprecatedJSON(result *security.DeprecatedResult, err error) {
	output := struct {
		Checker  string                      `json:"checker,omitempty"`
		Packages []deprecatedPackageJSON     `json:"packages,omitempty"`
		Summary  *security.DeprecatedSummary `json:"summary,omitempty"`
		Error    string                      `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Error = err.Error()
	} else if result != nil {
		output.Checker = result.Checker
		output.Packages = toDeprecatedPackagesJSON(result.Packages)
		summary := result.Summary()
		output.Summary = &summary
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

type deprecatedPackageJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Provider    string `json:"provider"`
	Reason      string `json:"reason"`
	Date        string `json:"date,omitempty"`
	Alternative string `json:"alternative,omitempty"`
	Message     string `json:"message,omitempty"`
}

func toDeprecatedPackagesJSON(packages security.DeprecatedPackages) []deprecatedPackageJSON {
	result := make([]deprecatedPackageJSON, len(packages))
	for i, pkg := range packages {
		dateStr := ""
		if pkg.Date != nil {
			dateStr = pkg.Date.Format("2006-01-02")
		}

		result[i] = deprecatedPackageJSON{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Provider:    pkg.Provider,
			Reason:      pkg.Reason.String(),
			Date:        dateStr,
			Alternative: pkg.Alternative,
			Message:     pkg.Message,
		}
	}
	return result
}

func outputDeprecatedText(result *security.DeprecatedResult, quiet bool) {
	summary := result.Summary()

	// Print header
	fmt.Printf("Deprecated Packages Check (%s)\n", result.Checker)
	fmt.Println(strings.Repeat("─", 50))

	if len(result.Packages) == 0 {
		fmt.Println("✓ No deprecated packages found")
		return
	}

	// Print summary
	fmt.Printf("\nSummary: %d packages require attention\n", summary.Total)
	printDeprecationSummaryBar(summary)

	// Print packages table (unless quiet)
	if !quiet {
		fmt.Println()
		printDeprecatedTable(result.Packages)
	}

	// Print recommendations
	fmt.Println()
	fmt.Println("Recommendations:")
	if summary.Disabled > 0 {
		fmt.Printf("  ⛔ %d packages are DISABLED and may stop working\n", summary.Disabled)
	}
	if summary.Deprecated > 0 {
		fmt.Printf("  ⚠  %d packages are DEPRECATED and should be replaced\n", summary.Deprecated)
	}
	fmt.Println("     Run 'brew info <package>' for details and alternatives")
}

func printDeprecationSummaryBar(summary security.DeprecatedSummary) {
	parts := []struct {
		label string
		count int
		color string
	}{
		{"DISABLED", summary.Disabled, "\033[91m"},
		{"DEPRECATED", summary.Deprecated, "\033[93m"},
		{"EOL", summary.EOL, "\033[33m"},
		{"UNMAINTAINED", summary.Unmaintained, "\033[34m"},
	}

	fmt.Print("  ")
	for _, p := range parts {
		if p.count > 0 {
			fmt.Printf("%s%s: %d\033[0m  ", p.color, p.label, p.count)
		}
	}
	fmt.Println()
}

func printDeprecatedTable(packages security.DeprecatedPackages) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "STATUS\tPACKAGE\tVERSION\tMESSAGE")
	_, _ = fmt.Fprintln(w, "──────\t───────\t───────\t───────")

	for _, pkg := range packages {
		statusStr := formatDeprecationStatus(pkg.Reason)
		message := pkg.Message
		if message == "" {
			message = "-"
		}
		// Truncate long messages
		if len(message) > 50 {
			message = message[:47] + "..."
		}

		version := pkg.Version
		if version == "" {
			version = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			statusStr, pkg.Name, version, message)
	}
	_ = w.Flush()
}

func formatDeprecationStatus(r security.DeprecationReason) string {
	switch r {
	case security.ReasonDisabled:
		return "\033[91mDISABLED\033[0m"
	case security.ReasonDeprecated:
		return "\033[93mDEPRECATED\033[0m"
	case security.ReasonEOL:
		return "\033[33mEOL\033[0m"
	case security.ReasonUnmaintained:
		return "\033[34mUNMAINTAINED\033[0m"
	default:
		return string(r)
	}
}
