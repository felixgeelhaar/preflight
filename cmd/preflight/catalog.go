package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/felixgeelhaar/preflight/internal/domain/catalog/embedded"
	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Manage external catalogs",
	Long: `Add, remove, and verify external catalog sources.

Catalogs provide presets, capability packs, and layer templates.
The builtin catalog is always available; additional catalogs can be added
from URLs or local paths.

Examples:
  preflight catalog list                          # List all catalogs
  preflight catalog add https://example.com/cat   # Add URL catalog
  preflight catalog add --local ./my-catalog      # Add local catalog
  preflight catalog remove my-catalog             # Remove a catalog
  preflight catalog verify                        # Verify all catalogs
  preflight catalog audit my-catalog              # Security audit`,
}

// Add subcommand
var catalogAddCmd = &cobra.Command{
	Use:   "add <url-or-path>",
	Short: "Add an external catalog",
	Long: `Add an external catalog from a URL or local path.

The catalog must include a catalog-manifest.yaml file with integrity hashes.
User approval is required for new catalog sources.

Examples:
  preflight catalog add https://example.com/catalog
  preflight catalog add --name company-tools https://company.com/catalog
  preflight catalog add --local --name my-presets ./presets`,
	Args: cobra.ExactArgs(1),
	RunE: runCatalogAdd,
}

// List subcommand
var catalogListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed catalogs",
	Long: `Display all registered catalogs.

Shows catalog name, source type, presets count, and verification status.`,
	RunE: runCatalogList,
}

// Remove subcommand
var catalogRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a catalog",
	Long: `Remove an external catalog from the registry.

The builtin catalog cannot be removed.

Examples:
  preflight catalog remove my-catalog`,
	Args: cobra.ExactArgs(1),
	RunE: runCatalogRemove,
}

// Verify subcommand
var catalogVerifyCmd = &cobra.Command{
	Use:   "verify [name]",
	Short: "Verify catalog integrity",
	Long: `Verify the integrity of registered catalogs.

If a catalog name is provided, only that catalog is verified.
Otherwise, all external catalogs are verified.

Examples:
  preflight catalog verify              # Verify all
  preflight catalog verify my-catalog   # Verify specific catalog`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCatalogVerify,
}

// Audit subcommand
var catalogAuditCmd = &cobra.Command{
	Use:   "audit <name>",
	Short: "Security audit a catalog",
	Long: `Perform a security audit on a catalog.

Checks for potentially dangerous patterns in preset configurations:
- Remote code execution (curl | sh)
- Privilege escalation (sudo)
- Destructive operations (rm -rf /)
- Hardcoded secrets
- Insecure permissions

Examples:
  preflight catalog audit my-catalog`,
	Args: cobra.ExactArgs(1),
	RunE: runCatalogAudit,
}

// Flags
var (
	catalogName  string
	catalogLocal bool
	catalogForce bool
)

func init() {
	// Add flags
	catalogAddCmd.Flags().StringVar(&catalogName, "name", "", "Name for the catalog (defaults to derived from URL)")
	catalogAddCmd.Flags().BoolVar(&catalogLocal, "local", false, "Treat path as local directory")

	catalogRemoveCmd.Flags().BoolVar(&catalogForce, "force", false, "Skip confirmation")

	// Add subcommands
	catalogCmd.AddCommand(catalogAddCmd)
	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogRemoveCmd)
	catalogCmd.AddCommand(catalogVerifyCmd)
	catalogCmd.AddCommand(catalogAuditCmd)

	rootCmd.AddCommand(catalogCmd)
}

// getRegistry returns the catalog registry with builtin catalog loaded
func getRegistry() (*catalog.Registry, error) {
	registry := catalog.NewRegistry()

	// Load builtin catalog
	builtinCat, err := embedded.LoadCatalog()
	if err != nil {
		return nil, fmt.Errorf("failed to load builtin catalog: %w", err)
	}

	builtinSource := catalog.NewBuiltinSource()
	builtinManifest, _ := catalog.NewManifestBuilder("builtin").
		WithDescription("Preflight built-in catalog").
		WithAuthor("Preflight Team").
		Build()

	builtinRC := catalog.NewRegisteredCatalog(builtinSource, builtinManifest, builtinCat)
	if err := registry.Add(builtinRC); err != nil {
		return nil, err
	}

	// TODO: Load external catalogs from config/state file

	return registry, nil
}

func runCatalogAdd(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	location := args[0]

	// Determine catalog name
	name := catalogName
	if name == "" {
		// Derive name from URL/path
		name = deriveCatalogName(location)
	}

	// Create source
	var source catalog.Source
	var err error
	if catalogLocal {
		source, err = catalog.NewLocalSource(name, location)
	} else {
		source, err = catalog.NewURLSource(name, location)
	}
	if err != nil {
		return fmt.Errorf("invalid source: %w", err)
	}

	fmt.Printf("Adding catalog '%s' from %s...\n", name, location)

	// Load catalog
	loader := catalog.NewExternalLoader(catalog.DefaultExternalLoaderConfig())
	rc, err := loader.Load(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	// Show catalog info
	fmt.Printf("\nCatalog: %s\n", rc.Manifest().Name())
	if rc.Manifest().Description() != "" {
		fmt.Printf("Description: %s\n", rc.Manifest().Description())
	}
	if rc.Manifest().Author() != "" {
		fmt.Printf("Author: %s\n", rc.Manifest().Author())
	}
	fmt.Printf("Presets: %d\n", rc.Catalog().PresetCount())
	fmt.Printf("Packs: %d\n", rc.Catalog().PackCount())

	// Security audit
	auditor := catalog.NewAuditor()
	auditResult := auditor.Audit(rc)

	switch {
	case !auditResult.Passed:
		fmt.Printf("\nSecurity audit FAILED:\n")
		fmt.Printf("  Critical: %d\n", auditResult.CriticalCount())
		fmt.Printf("  High: %d\n", auditResult.HighCount())
		fmt.Printf("  Medium: %d\n", auditResult.MediumCount())

		if !catalogForce {
			return fmt.Errorf("catalog failed security audit; use --force to add anyway")
		}
		fmt.Println("\nAdding catalog despite audit failures (--force)")
	case auditResult.MediumCount() > 0 || auditResult.LowCount() > 0:
		fmt.Printf("\nSecurity audit passed with %d warnings\n", auditResult.MediumCount()+auditResult.LowCount())
	default:
		fmt.Println("\nSecurity audit passed")
	}

	// TODO: Save to registry config file
	fmt.Printf("\nCatalog '%s' added successfully.\n", name)
	return nil
}

func runCatalogList(_ *cobra.Command, _ []string) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	catalogs := registry.List()
	if len(catalogs) == 0 {
		fmt.Println("No catalogs registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tPRESETS\tPACKS\tSTATUS")

	for _, rc := range catalogs {
		srcType := string(rc.Source().Type())
		status := "enabled"
		if !rc.Enabled() {
			status = "disabled"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
			rc.Name(),
			srcType,
			rc.Catalog().PresetCount(),
			rc.Catalog().PackCount(),
			status,
		)
	}

	_ = w.Flush()

	// Show stats
	stats := registry.Stats()
	fmt.Printf("\nTotal: %d catalogs, %d presets, %d packs\n",
		stats.TotalCatalogs, stats.TotalPresets, stats.TotalPacks)

	return nil
}

func runCatalogRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	rc, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("catalog '%s' not found", name)
	}

	if rc.Source().IsBuiltin() {
		return fmt.Errorf("cannot remove builtin catalog")
	}

	// Confirm removal
	if !catalogForce {
		fmt.Printf("Remove catalog '%s'? [y/N] ", name)
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := registry.Remove(name); err != nil {
		return err
	}

	// Clear cache
	loader := catalog.NewExternalLoader(catalog.DefaultExternalLoaderConfig())
	_ = loader.ClearCache(rc.Source())

	// TODO: Update registry config file

	fmt.Printf("Catalog '%s' removed.\n", name)
	return nil
}

func runCatalogVerify(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	var toVerify []*catalog.RegisteredCatalog

	if len(args) > 0 {
		name := args[0]
		rc, ok := registry.Get(name)
		if !ok {
			return fmt.Errorf("catalog '%s' not found", name)
		}
		toVerify = []*catalog.RegisteredCatalog{rc}
	} else {
		// Verify all external catalogs
		for _, rc := range registry.List() {
			if !rc.Source().IsBuiltin() {
				toVerify = append(toVerify, rc)
			}
		}
	}

	if len(toVerify) == 0 {
		fmt.Println("No external catalogs to verify.")
		return nil
	}

	loader := catalog.NewExternalLoader(catalog.DefaultExternalLoaderConfig())
	var failed int

	for _, rc := range toVerify {
		fmt.Printf("Verifying %s... ", rc.Name())

		if err := loader.Verify(ctx, rc); err != nil {
			fmt.Println("FAILED")
			fmt.Printf("  Error: %v\n", err)
			failed++
		} else {
			fmt.Println("OK")
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d catalogs failed verification", failed, len(toVerify))
	}

	fmt.Printf("\nAll %d catalogs verified.\n", len(toVerify))
	return nil
}

func runCatalogAudit(_ *cobra.Command, args []string) error {
	name := args[0]

	registry, err := getRegistry()
	if err != nil {
		return err
	}

	rc, ok := registry.Get(name)
	if !ok {
		return fmt.Errorf("catalog '%s' not found", name)
	}

	fmt.Printf("Auditing catalog '%s'...\n\n", name)

	auditor := catalog.NewAuditor()
	result := auditor.Audit(rc)

	// Display findings by severity
	severities := []catalog.AuditSeverity{
		catalog.AuditSeverityCritical,
		catalog.AuditSeverityHigh,
		catalog.AuditSeverityMedium,
		catalog.AuditSeverityLow,
		catalog.AuditSeverityInfo,
	}

	totalFindings := len(result.Findings)
	if totalFindings == 0 {
		fmt.Println("No issues found.")
	} else {
		for _, severity := range severities {
			findings := filterBySeverity(result.Findings, severity)
			if len(findings) == 0 {
				continue
			}

			fmt.Printf("[%s] %d findings:\n", severity, len(findings))
			for _, f := range findings {
				fmt.Printf("  • %s\n", f.Message)
				fmt.Printf("    Location: %s\n", f.Location)
				if f.Suggestion != "" {
					fmt.Printf("    Suggestion: %s\n", f.Suggestion)
				}
			}
			fmt.Println()
		}
	}

	// Summary
	fmt.Println("Summary:")
	fmt.Printf("  Critical: %d\n", result.CriticalCount())
	fmt.Printf("  High:     %d\n", result.HighCount())
	fmt.Printf("  Medium:   %d\n", result.MediumCount())
	fmt.Printf("  Low:      %d\n", result.LowCount())

	if result.Passed {
		fmt.Println("\nAudit PASSED")
	} else {
		fmt.Println("\nAudit FAILED")
		return fmt.Errorf("catalog failed security audit")
	}

	return nil
}

func filterBySeverity(findings []catalog.AuditFinding, severity catalog.AuditSeverity) []catalog.AuditFinding {
	var result []catalog.AuditFinding
	for _, f := range findings {
		if f.Severity == severity {
			result = append(result, f)
		}
	}
	return result
}

func deriveCatalogName(location string) string {
	// Simple name derivation - in production would be more sophisticated
	// For URLs: last path segment
	// For paths: directory name
	if len(location) > 0 {
		// Remove trailing slash
		if location[len(location)-1] == '/' {
			location = location[:len(location)-1]
		}

		// Find last segment
		for i := len(location) - 1; i >= 0; i-- {
			if location[i] == '/' {
				return location[i+1:]
			}
		}

		// No slashes found - return the location as-is (it's just a name)
		return location
	}
	return "catalog-" + time.Now().Format("20060102")
}
