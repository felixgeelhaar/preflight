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

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "Check for outdated packages",
	Long: `Check for packages that have available updates.

This command checks installed packages for available updates and reports
them with their update type (major, minor, patch).

Checkers (in order of preference):
  - brew: Homebrew formulae and casks

Exit codes:
  0 - No outdated packages found (or below threshold)
  1 - Outdated packages found above threshold
  2 - Checker not available or check failed

Examples:
  preflight outdated                  # Check for outdated packages
  preflight outdated --all            # Include patch updates
  preflight outdated --json           # JSON output for CI
  preflight outdated --fail-on major  # Fail only on major updates
  preflight outdated --ignore go      # Ignore specific packages`,
	RunE: runOutdated,
}

var (
	outdatedIncludeAll bool
	outdatedFailOn     string
	outdatedIgnore     []string
	outdatedJSON       bool
	outdatedQuiet      bool
)

func init() {
	rootCmd.AddCommand(outdatedCmd)

	outdatedCmd.Flags().BoolVar(&outdatedIncludeAll, "all", false, "Include patch updates (default: minor and above)")
	outdatedCmd.Flags().StringVar(&outdatedFailOn, "fail-on", "minor", "Fail if updates of this type or higher are found (major, minor, patch)")
	outdatedCmd.Flags().StringSliceVar(&outdatedIgnore, "ignore", nil, "Package names to ignore (can be specified multiple times)")
	outdatedCmd.Flags().BoolVar(&outdatedJSON, "json", false, "Output results as JSON")
	outdatedCmd.Flags().BoolVarP(&outdatedQuiet, "quiet", "q", false, "Only show summary")
}

func runOutdated(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create checker registry
	registry := security.NewOutdatedCheckerRegistry()
	registry.Register(security.NewBrewOutdatedChecker())

	// Get first available checker
	checkers := registry.All()
	if len(checkers) == 0 {
		if outdatedJSON {
			outputOutdatedJSON(nil, fmt.Errorf("no checkers available"))
		} else {
			fmt.Fprintf(os.Stderr, "Error: no outdated checkers available. Install Homebrew.\n")
		}
		os.Exit(2)
	}

	// Parse fail-on threshold
	failOnType := parseUpdateType(outdatedFailOn)

	// Configure options
	opts := security.OutdatedOptions{
		IncludePatch:   outdatedIncludeAll,
		IncludePinned:  false,
		IgnorePackages: outdatedIgnore,
	}

	// Run checks with all available checkers
	var allPackages security.OutdatedPackages
	var checkerName string

	for _, checker := range checkers {
		result, err := checker.Check(ctx, opts)
		if err != nil {
			if outdatedJSON {
				outputOutdatedJSON(nil, err)
			} else {
				fmt.Fprintf(os.Stderr, "Check failed (%s): %v\n", checker.Name(), err)
			}
			os.Exit(2)
		}

		checkerName = checker.Name()
		allPackages = append(allPackages, result.Packages...)
	}

	// Build combined result
	result := &security.OutdatedResult{
		Checker:  checkerName,
		Packages: allPackages,
	}

	// Output results
	if outdatedJSON {
		outputOutdatedJSON(result, nil)
	} else {
		outputOutdatedText(result, outdatedQuiet)
	}

	// Determine exit code
	if shouldFailOutdated(result, failOnType) {
		os.Exit(1)
	}

	return nil
}

func parseUpdateType(s string) security.UpdateType {
	switch strings.ToLower(s) {
	case "major":
		return security.UpdateMajor
	case "minor":
		return security.UpdateMinor
	case "patch":
		return security.UpdatePatch
	default:
		return security.UpdateMinor
	}
}

func shouldFailOutdated(result *security.OutdatedResult, failOn security.UpdateType) bool {
	for _, pkg := range result.Packages {
		if pkg.UpdateType.IsAtLeast(failOn) {
			return true
		}
	}
	return false
}

func outputOutdatedJSON(result *security.OutdatedResult, err error) {
	output := struct {
		Checker  string                    `json:"checker,omitempty"`
		Packages []outdatedPackageJSON     `json:"packages,omitempty"`
		Summary  *security.OutdatedSummary `json:"summary,omitempty"`
		Error    string                    `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Error = err.Error()
	} else if result != nil {
		output.Checker = result.Checker
		output.Packages = toOutdatedPackagesJSON(result.Packages)
		summary := result.Summary()
		output.Summary = &summary
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

type outdatedPackageJSON struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateType     string `json:"update_type"`
	Provider       string `json:"provider"`
	Pinned         bool   `json:"pinned,omitempty"`
}

func toOutdatedPackagesJSON(packages security.OutdatedPackages) []outdatedPackageJSON {
	result := make([]outdatedPackageJSON, len(packages))
	for i, pkg := range packages {
		result[i] = outdatedPackageJSON{
			Name:           pkg.Name,
			CurrentVersion: pkg.CurrentVersion,
			LatestVersion:  pkg.LatestVersion,
			UpdateType:     pkg.UpdateType.String(),
			Provider:       pkg.Provider,
			Pinned:         pkg.Pinned,
		}
	}
	return result
}

func outputOutdatedText(result *security.OutdatedResult, quiet bool) {
	summary := result.Summary()

	// Print header
	fmt.Printf("Outdated Packages Check (%s)\n", result.Checker)
	fmt.Println(strings.Repeat("─", 50))

	if len(result.Packages) == 0 {
		fmt.Println("✓ All packages are up to date")
		return
	}

	// Print summary
	fmt.Printf("\nSummary: %d packages have updates available\n", summary.Total)
	printUpdateTypeBar(summary)

	if summary.Pinned > 0 {
		fmt.Printf("  Pinned (excluded): %d\n", summary.Pinned)
	}

	// Print packages table (unless quiet)
	if !quiet {
		fmt.Println()
		printOutdatedTable(result.Packages)
	}

	// Print recommendations
	if summary.Major > 0 {
		fmt.Println()
		fmt.Println("Recommendations:")
		fmt.Printf("  ⚠  %d packages have MAJOR updates (may include breaking changes)\n", summary.Major)
		fmt.Println("     Review changelogs before updating: brew upgrade <package>")
	}
}

func printUpdateTypeBar(summary security.OutdatedSummary) {
	parts := []struct {
		label string
		count int
		color string
	}{
		{"MAJOR", summary.Major, "\033[91m"},
		{"MINOR", summary.Minor, "\033[93m"},
		{"PATCH", summary.Patch, "\033[32m"},
	}

	fmt.Print("  ")
	for _, p := range parts {
		if p.count > 0 {
			fmt.Printf("%s%s: %d\033[0m  ", p.color, p.label, p.count)
		}
	}
	fmt.Println()
}

func printOutdatedTable(packages security.OutdatedPackages) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TYPE\tPACKAGE\tCURRENT\tLATEST\tPROVIDER")
	_, _ = fmt.Fprintln(w, "────\t───────\t───────\t──────\t────────")

	for _, pkg := range packages {
		typeStr := formatUpdateType(pkg.UpdateType)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			typeStr, pkg.Name, pkg.CurrentVersion, pkg.LatestVersion, pkg.Provider)
	}
	_ = w.Flush()
}

func formatUpdateType(t security.UpdateType) string {
	switch t {
	case security.UpdateMajor:
		return "\033[91mMAJOR\033[0m"
	case security.UpdateMinor:
		return "\033[93mMINOR\033[0m"
	case security.UpdatePatch:
		return "\033[32mPATCH\033[0m"
	default:
		return string(t)
	}
}
