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

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Scan for security vulnerabilities",
	Long: `Scan your system for security vulnerabilities using Grype or Trivy.

This command scans installed packages and dependencies for known
vulnerabilities (CVEs) and reports them with severity levels.

Scanners (in order of preference):
  - grype: Anchore's vulnerability scanner
  - trivy: Aqua Security's vulnerability scanner

Exit codes:
  0 - No vulnerabilities found (or below --fail-on threshold)
  1 - Vulnerabilities found above --fail-on threshold
  2 - Scanner not available or scan failed

Examples:
  preflight security                      # Scan current directory
  preflight security --path /app          # Scan specific path
  preflight security --severity high      # Only show high+ severity
  preflight security --fail-on critical   # Fail only on critical
  preflight security --scanner grype      # Use specific scanner
  preflight security --json               # JSON output for CI
  preflight security --ignore CVE-2024-1234  # Ignore specific CVE`,
	RunE: runSecurity,
}

var (
	securityPath       string
	securityScanner    string
	securitySeverity   string
	securityFailOn     string
	securityIgnore     []string
	securityJSON       bool
	securityQuiet      bool
	securityListIgnore bool
)

func init() {
	rootCmd.AddCommand(securityCmd)

	securityCmd.Flags().StringVarP(&securityPath, "path", "p", ".", "Path to scan")
	securityCmd.Flags().StringVarP(&securityScanner, "scanner", "s", "auto", "Scanner to use (grype, trivy, auto)")
	securityCmd.Flags().StringVar(&securitySeverity, "severity", "medium", "Minimum severity to report (critical, high, medium, low, negligible)")
	securityCmd.Flags().StringVar(&securityFailOn, "fail-on", "high", "Fail if vulnerabilities of this severity or higher are found")
	securityCmd.Flags().StringSliceVar(&securityIgnore, "ignore", nil, "CVE IDs to ignore (can be specified multiple times)")
	securityCmd.Flags().BoolVar(&securityJSON, "json", false, "Output results as JSON")
	securityCmd.Flags().BoolVarP(&securityQuiet, "quiet", "q", false, "Only show summary")
	securityCmd.Flags().BoolVar(&securityListIgnore, "list-scanners", false, "List available scanners and exit")
}

func runSecurity(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create scanner registry
	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	// Handle --list-scanners
	if securityListIgnore {
		return listScanners(registry)
	}

	// Get scanner
	scanner, err := getScanner(registry, securityScanner)
	if err != nil {
		if securityJSON {
			outputSecurityJSON(nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(2)
	}

	// Parse severity levels
	minSeverity, _ := security.ParseSeverity(securitySeverity)
	failOnSeverity, _ := security.ParseSeverity(securityFailOn)

	// Configure scan options
	opts := security.ScanOptions{
		MinSeverity:    minSeverity,
		FailOnSeverity: failOnSeverity,
		IgnoreIDs:      securityIgnore,
		Quiet:          securityQuiet,
		JSON:           securityJSON,
	}

	// Configure scan target
	target := security.ScanTarget{
		Type: "directory",
		Path: securityPath,
	}

	// Run scan
	result, err := scanner.Scan(ctx, target, opts)
	if err != nil {
		if securityJSON {
			outputSecurityJSON(nil, err)
		} else {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		}
		os.Exit(2)
	}

	// Filter results by severity
	result.Vulnerabilities = result.Vulnerabilities.BySeverity(minSeverity)

	// Filter out ignored CVEs
	if len(securityIgnore) > 0 {
		result.Vulnerabilities = result.Vulnerabilities.ExcludeIDs(securityIgnore)
	}

	// Output results
	if securityJSON {
		outputSecurityJSON(result, nil)
	} else {
		outputSecurityText(result, opts)
	}

	// Determine exit code
	if shouldFail(result, failOnSeverity) {
		os.Exit(1)
	}

	return nil
}

func getScanner(registry *security.ScannerRegistry, name string) (security.Scanner, error) {
	if name == "auto" || name == "" {
		scanner := registry.First()
		if scanner == nil {
			available := registry.Names()
			return nil, fmt.Errorf("no scanners available. Install one of: %s", strings.Join(available, ", "))
		}
		return scanner, nil
	}

	scanner := registry.Get(name)
	if scanner == nil {
		return nil, fmt.Errorf("scanner %q is not available", name)
	}
	return scanner, nil
}

func listScanners(registry *security.ScannerRegistry) error {
	fmt.Println("Available security scanners:")
	fmt.Println()

	for _, name := range registry.Names() {
		scanner := registry.Get(name)
		status := "not installed"
		version := ""

		if scanner != nil {
			status = "available"
			if v, err := scanner.Version(context.Background()); err == nil {
				version = v
			}
		}

		if version != "" {
			fmt.Printf("  %-10s %s (v%s)\n", name, status, version)
		} else {
			fmt.Printf("  %-10s %s\n", name, status)
		}
	}

	return nil
}

func shouldFail(result *security.ScanResult, failOn security.Severity) bool {
	for _, v := range result.Vulnerabilities {
		if v.Severity.IsAtLeast(failOn) {
			return true
		}
	}
	return false
}

func outputSecurityJSON(result *security.ScanResult, err error) {
	output := struct {
		Scanner         string                `json:"scanner,omitempty"`
		Version         string                `json:"version,omitempty"`
		Vulnerabilities []vulnerabilityJSON   `json:"vulnerabilities,omitempty"`
		Summary         *security.ScanSummary `json:"summary,omitempty"`
		Error           string                `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Error = err.Error()
	} else if result != nil {
		output.Scanner = result.Scanner
		output.Version = result.Version
		output.Vulnerabilities = toVulnerabilitiesJSON(result.Vulnerabilities)
		summary := result.Summary()
		output.Summary = &summary
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

type vulnerabilityJSON struct {
	ID        string  `json:"id"`
	Package   string  `json:"package"`
	Version   string  `json:"version"`
	Severity  string  `json:"severity"`
	CVSS      float64 `json:"cvss,omitempty"`
	FixedIn   string  `json:"fixed_in,omitempty"`
	Title     string  `json:"title,omitempty"`
	Reference string  `json:"reference,omitempty"`
}

func toVulnerabilitiesJSON(vulns security.Vulnerabilities) []vulnerabilityJSON {
	result := make([]vulnerabilityJSON, len(vulns))
	for i, v := range vulns {
		result[i] = vulnerabilityJSON{
			ID:        v.ID,
			Package:   v.Package,
			Version:   v.Version,
			Severity:  string(v.Severity),
			CVSS:      v.CVSS,
			FixedIn:   v.FixedIn,
			Title:     v.Title,
			Reference: v.Reference,
		}
	}
	return result
}

func outputSecurityText(result *security.ScanResult, opts security.ScanOptions) {
	summary := result.Summary()

	// Print header
	fmt.Printf("Security Scan Results (%s v%s)\n", result.Scanner, result.Version)
	fmt.Println(strings.Repeat("─", 50))

	if !result.HasVulnerabilities() {
		fmt.Println("✓ No vulnerabilities found")
		fmt.Printf("  Packages scanned: %d\n", summary.PackagesScanned)
		return
	}

	// Print summary
	fmt.Printf("\nSummary: %d vulnerabilities found\n", summary.TotalVulnerabilities)
	printSeverityBar(summary)
	fmt.Printf("  Packages scanned: %d\n", summary.PackagesScanned)
	if summary.FixableCount > 0 {
		fmt.Printf("  Fixable: %d\n", summary.FixableCount)
	}

	// Print vulnerabilities table (unless quiet)
	if !opts.Quiet {
		fmt.Println()
		printVulnerabilitiesTable(result.Vulnerabilities)
	}

	// Print recommendations
	if summary.Critical > 0 || summary.High > 0 {
		fmt.Println()
		fmt.Println("Recommendations:")
		if summary.Critical > 0 {
			fmt.Printf("  ⛔ %d CRITICAL vulnerabilities require immediate attention\n", summary.Critical)
		}
		if summary.High > 0 {
			fmt.Printf("  ⚠  %d HIGH severity issues should be addressed soon\n", summary.High)
		}
		if summary.FixableCount > 0 {
			fmt.Printf("  ℹ  %d vulnerabilities have fixes available\n", summary.FixableCount)
		}
	}
}

func printSeverityBar(summary security.ScanSummary) {
	parts := []struct {
		label string
		count int
		color string
	}{
		{"CRITICAL", summary.Critical, "\033[91m"},
		{"HIGH", summary.High, "\033[93m"},
		{"MEDIUM", summary.Medium, "\033[33m"},
		{"LOW", summary.Low, "\033[34m"},
	}

	fmt.Print("  ")
	for _, p := range parts {
		if p.count > 0 {
			fmt.Printf("%s%s: %d\033[0m  ", p.color, p.label, p.count)
		}
	}
	fmt.Println()
}

func printVulnerabilitiesTable(vulns security.Vulnerabilities) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SEVERITY\tID\tPACKAGE\tVERSION\tFIXED IN")
	_, _ = fmt.Fprintln(w, "────────\t──\t───────\t───────\t────────")

	for _, v := range vulns {
		fixedIn := v.FixedIn
		if fixedIn == "" {
			fixedIn = "-"
		}

		severityStr := formatSeverity(v.Severity)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			severityStr, v.ID, v.Package, v.Version, fixedIn)
	}
	_ = w.Flush()
}

func formatSeverity(s security.Severity) string {
	switch s {
	case security.SeverityCritical:
		return "\033[91mCRITICAL\033[0m"
	case security.SeverityHigh:
		return "\033[93mHIGH\033[0m"
	case security.SeverityMedium:
		return "\033[33mMEDIUM\033[0m"
	case security.SeverityLow:
		return "\033[34mLOW\033[0m"
	case security.SeverityNegligible:
		return "\033[90mNEGLIGIBLE\033[0m"
	default:
		return string(s)
	}
}
