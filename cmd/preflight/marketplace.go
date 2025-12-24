package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"mp", "market"},
	Short:   "Discover and install community packages",
	Long: `Browse, search, and install community presets, capability packs, and layer templates.

The marketplace provides a registry of curated configurations that can be installed
and used in your preflight setup.

Examples:
  preflight marketplace search nvim          # Search for packages
  preflight marketplace install nvim-pro     # Install a package
  preflight marketplace list                 # List installed packages
  preflight marketplace update               # Update all packages`,
}

// Search subcommand
var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for packages",
	Long: `Search the marketplace registry for packages matching your query.

Searches package names, descriptions, keywords, and authors.

Examples:
  preflight marketplace search nvim
  preflight marketplace search --type preset
  preflight marketplace search --type capability-pack go`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMarketplaceSearch,
}

// Install subcommand
var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <package> [version]",
	Short: "Install a package",
	Long: `Download and install a package from the marketplace.

If no version is specified, the latest version is installed.
Packages are verified using SHA256 checksums before installation.

Examples:
  preflight marketplace install nvim-pro
  preflight marketplace install nvim-pro@1.2.0
  preflight marketplace install nvim-pro --version 1.2.0`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runMarketplaceInstall,
}

// Uninstall subcommand
var marketplaceUninstallCmd = &cobra.Command{
	Use:     "uninstall <package>",
	Aliases: []string{"remove", "rm"},
	Short:   "Uninstall a package",
	Long: `Remove an installed package from your system.

This removes the package files but does not undo any configuration changes
that may have been applied.

Examples:
  preflight marketplace uninstall nvim-pro`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceUninstall,
}

// Update subcommand
var marketplaceUpdateCmd = &cobra.Command{
	Use:   "update [package]",
	Short: "Update packages",
	Long: `Update installed packages to their latest versions.

If a package name is provided, only that package is updated.
Otherwise, all installed packages are checked for updates.

Examples:
  preflight marketplace update              # Update all
  preflight marketplace update nvim-pro     # Update specific package`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMarketplaceUpdate,
}

// List subcommand
var marketplaceListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed packages",
	Long: `Display all installed marketplace packages.

Shows package name, version, installation date, and update status.

Examples:
  preflight marketplace list`,
	RunE: runMarketplaceList,
}

// Info subcommand
var marketplaceInfoCmd = &cobra.Command{
	Use:   "info <package>",
	Short: "Show package details",
	Long: `Display detailed information about a package.

Shows title, description, author, license, versions, and more.

Examples:
  preflight marketplace info nvim-pro`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceInfo,
}

// Flags
var (
	mpSearchType   string
	mpSearchLimit  int
	mpInstallVer   string
	mpOfflineMode  bool
	mpRefreshIndex bool
	mpCheckUpdates bool
)

func init() {
	// Search flags
	marketplaceSearchCmd.Flags().StringVar(&mpSearchType, "type", "", "Filter by type (preset, capability-pack, layer-template)")
	marketplaceSearchCmd.Flags().IntVar(&mpSearchLimit, "limit", 20, "Maximum results to show")

	// Install flags
	marketplaceInstallCmd.Flags().StringVar(&mpInstallVer, "version", "", "Version to install")

	// Global marketplace flags
	marketplaceCmd.PersistentFlags().BoolVar(&mpOfflineMode, "offline", false, "Use cached data only")
	marketplaceCmd.PersistentFlags().BoolVar(&mpRefreshIndex, "refresh", false, "Force refresh of package index")

	// List flags
	marketplaceListCmd.Flags().BoolVar(&mpCheckUpdates, "check-updates", false, "Check for available updates")

	// Add subcommands
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceUninstallCmd)
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceInfoCmd)

	rootCmd.AddCommand(marketplaceCmd)
}

func newMarketplaceService() *marketplace.Service {
	config := marketplace.DefaultServiceConfig()
	config.OfflineMode = mpOfflineMode
	return marketplace.NewService(config)
}

func runMarketplaceSearch(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	if mpRefreshIndex {
		if err := svc.RefreshIndex(ctx); err != nil {
			return fmt.Errorf("failed to refresh index: %w", err)
		}
	}

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	var results []marketplace.Package
	var err error

	if mpSearchType != "" {
		results, err = svc.SearchByType(ctx, mpSearchType)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		// Filter by query if provided
		if query != "" {
			var filtered []marketplace.Package
			for _, pkg := range results {
				if pkg.MatchesQuery(query) {
					filtered = append(filtered, pkg)
				}
			}
			results = filtered
		}
	} else {
		results, err = svc.Search(ctx, query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
	}

	if len(results) == 0 {
		fmt.Println("No packages found.")
		return nil
	}

	// Limit results
	if mpSearchLimit > 0 && len(results) > mpSearchLimit {
		results = results[:mpSearchLimit]
	}

	// Display results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tTITLE\tVERSION\tDOWNLOADS")

	for _, pkg := range results {
		version := ""
		if v, ok := pkg.LatestVersion(); ok {
			version = v.Version
		}

		pkgType := pkg.Type
		if len(pkgType) > 12 {
			pkgType = pkgType[:12]
		}

		title := pkg.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			pkg.ID.String(), pkgType, title, version, pkg.Downloads)
	}

	_ = w.Flush()

	if len(results) == mpSearchLimit {
		fmt.Printf("\nShowing first %d results. Use --limit to see more.\n", mpSearchLimit)
	}

	return nil
}

func runMarketplaceInstall(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	pkgName := args[0]
	version := mpInstallVer

	// Handle package@version syntax
	if strings.Contains(pkgName, "@") {
		parts := strings.SplitN(pkgName, "@", 2)
		pkgName = parts[0]
		if version == "" {
			version = parts[1]
		}
	}

	// Handle version from second arg
	if len(args) > 1 && version == "" {
		version = args[1]
	}

	if version == "" {
		version = "latest"
	}

	id, err := marketplace.NewPackageID(pkgName)
	if err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	fmt.Printf("Installing %s@%s...\n", pkgName, version)

	installed, err := svc.Install(ctx, id, version)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("Installed %s@%s to %s\n", installed.Package.Title, installed.Version, installed.Path)

	if installed.Package.Provenance.Verified {
		fmt.Println("  Package is verified.")
	}

	return nil
}

func runMarketplaceUninstall(_ *cobra.Command, args []string) error {
	svc := newMarketplaceService()

	pkgName := args[0]
	id, err := marketplace.NewPackageID(pkgName)
	if err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	fmt.Printf("Uninstalling %s...\n", pkgName)

	if err := svc.Uninstall(id); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	fmt.Printf("Uninstalled %s\n", pkgName)
	return nil
}

func runMarketplaceUpdate(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	if len(args) > 0 {
		// Update specific package
		pkgName := args[0]
		id, err := marketplace.NewPackageID(pkgName)
		if err != nil {
			return fmt.Errorf("invalid package name: %w", err)
		}

		fmt.Printf("Updating %s...\n", pkgName)

		installed, err := svc.Update(ctx, id)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}

		fmt.Printf("Updated to %s@%s\n", installed.Package.Title, installed.Version)
		return nil
	}

	// Update all packages
	fmt.Println("Checking for updates...")

	updated, err := svc.UpdateAll(ctx)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	if len(updated) == 0 {
		fmt.Println("All packages are up to date.")
		return nil
	}

	fmt.Printf("Updated %d packages:\n", len(updated))
	for _, pkg := range updated {
		fmt.Printf("  %s -> %s\n", pkg.Package.ID.String(), pkg.Version)
	}

	return nil
}

func runMarketplaceList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	installed, err := svc.List()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	if len(installed) == 0 {
		fmt.Println("No packages installed.")
		fmt.Println("Use 'preflight marketplace search' to find packages.")
		return nil
	}

	// Check for updates if requested
	var updates []marketplace.UpdateInfo
	if mpCheckUpdates {
		updates, err = svc.CheckUpdates(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not check for updates: %v\n", err)
		}
	}

	updateMap := make(map[string]string)
	for _, u := range updates {
		updateMap[u.Package.ID.String()] = u.LatestVersion
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED\tUPDATE")

	for _, pkg := range installed {
		updateCol := "-"
		if latest, ok := updateMap[pkg.Package.ID.String()]; ok {
			updateCol = latest + " available"
		}

		age := formatInstallAge(pkg.InstalledAt)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			pkg.Package.ID.String(), pkg.Version, age, updateCol)
	}

	_ = w.Flush()

	if len(updates) > 0 {
		fmt.Printf("\n%d updates available. Run 'preflight marketplace update' to install.\n", len(updates))
	}

	return nil
}

func runMarketplaceInfo(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	pkgName := args[0]
	id, err := marketplace.NewPackageID(pkgName)
	if err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	pkg, err := svc.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("package not found: %w", err)
	}

	fmt.Printf("Name:        %s\n", pkg.ID.String())
	fmt.Printf("Title:       %s\n", pkg.Title)
	fmt.Printf("Type:        %s\n", pkg.Type)

	if pkg.Description != "" {
		fmt.Printf("Description: %s\n", pkg.Description)
	}

	if len(pkg.Keywords) > 0 {
		fmt.Printf("Keywords:    %s\n", strings.Join(pkg.Keywords, ", "))
	}

	fmt.Println()

	// Provenance
	if !pkg.Provenance.IsZero() {
		fmt.Println("Provenance:")
		if pkg.Provenance.Author != "" {
			fmt.Printf("  Author:     %s\n", pkg.Provenance.Author)
		}
		if pkg.Provenance.Repository != "" {
			fmt.Printf("  Repository: %s\n", pkg.Provenance.Repository)
		}
		if pkg.Provenance.License != "" {
			fmt.Printf("  License:    %s\n", pkg.Provenance.License)
		}
		if pkg.Provenance.Verified {
			fmt.Printf("  Verified:   Yes\n")
		}
		fmt.Println()
	}

	// Versions
	fmt.Println("Versions:")
	for i, v := range pkg.Versions {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(pkg.Versions)-5)
			break
		}
		date := ""
		if !v.ReleasedAt.IsZero() {
			date = " (" + v.ReleasedAt.Format("2006-01-02") + ")"
		}
		fmt.Printf("  %s%s\n", v.Version, date)
	}

	fmt.Println()
	fmt.Printf("Downloads:   %d\n", pkg.Downloads)
	fmt.Printf("Stars:       %d\n", pkg.Stars)

	return nil
}

func formatInstallAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	default:
		return t.Format("2006-01-02")
	}
}
