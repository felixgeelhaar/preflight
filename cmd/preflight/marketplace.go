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

// Recommend subcommand
var marketplaceRecommendCmd = &cobra.Command{
	Use:     "recommend",
	Aliases: []string{"rec", "suggestions"},
	Short:   "Get personalized package recommendations",
	Long: `Get package recommendations based on your installed packages and preferences.

The recommendation engine considers:
  - Your installed packages and their keywords
  - Active providers in your configuration
  - Package popularity and recency
  - Complementary packages that work well together

Examples:
  preflight marketplace recommend
  preflight marketplace recommend --type preset
  preflight marketplace recommend --keywords vim,neovim
  preflight marketplace recommend --similar nvim-pro
  preflight marketplace recommend --featured`,
	RunE: runMarketplaceRecommend,
}

// Featured subcommand
var marketplaceFeaturedCmd = &cobra.Command{
	Use:   "featured",
	Short: "Show featured packages",
	Long: `Display editorially curated featured packages.

Featured packages are verified, highly-rated packages that represent
the best of the marketplace.

Examples:
  preflight marketplace featured
  preflight marketplace featured --type preset`,
	RunE: runMarketplaceFeatured,
}

// Popular subcommand
var marketplacePopularCmd = &cobra.Command{
	Use:   "popular",
	Short: "Show most popular packages",
	Long: `Display the most downloaded and starred packages.

Examples:
  preflight marketplace popular
  preflight marketplace popular --type capability-pack`,
	RunE: runMarketplacePopular,
}

// Flags
var (
	mpSearchType    string
	mpSearchLimit   int
	mpInstallVer    string
	mpOfflineMode   bool
	mpRefreshIndex  bool
	mpCheckUpdates  bool
	mpRecommendType string
	mpKeywords      string
	mpSimilarTo     string
	mpRecommendMax  int
	mpPopularType   string
	mpFeaturedType  string
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

	// Recommend flags
	marketplaceRecommendCmd.Flags().StringVar(&mpRecommendType, "type", "", "Filter by type (preset, capability-pack, layer-template)")
	marketplaceRecommendCmd.Flags().StringVar(&mpKeywords, "keywords", "", "Comma-separated keywords for recommendations")
	marketplaceRecommendCmd.Flags().StringVar(&mpSimilarTo, "similar", "", "Get recommendations similar to this package")
	marketplaceRecommendCmd.Flags().IntVar(&mpRecommendMax, "max", 10, "Maximum recommendations to show")

	// Featured flags
	marketplaceFeaturedCmd.Flags().StringVar(&mpFeaturedType, "type", "", "Filter by type")

	// Popular flags
	marketplacePopularCmd.Flags().StringVar(&mpPopularType, "type", "", "Filter by type")

	// Add subcommands
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceUninstallCmd)
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceInfoCmd)
	marketplaceCmd.AddCommand(marketplaceRecommendCmd)
	marketplaceCmd.AddCommand(marketplaceFeaturedCmd)
	marketplaceCmd.AddCommand(marketplacePopularCmd)

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

func runMarketplaceRecommend(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	if mpRefreshIndex {
		if err := svc.RefreshIndex(ctx); err != nil {
			return fmt.Errorf("failed to refresh index: %w", err)
		}
	}

	// Handle similar package mode
	if mpSimilarTo != "" {
		return runSimilarRecommendations(ctx, svc)
	}

	// Build user context from installed packages and flags
	userCtx := buildUserContext(svc)

	// Create recommender
	config := marketplace.DefaultRecommenderConfig()
	config.MaxRecommendations = mpRecommendMax
	recommender := marketplace.NewRecommender(svc, config)

	// Get recommendations
	recommendations, err := recommender.RecommendForUser(ctx, userCtx)
	if err != nil {
		return fmt.Errorf("recommendation failed: %w", err)
	}

	if len(recommendations) == 0 {
		fmt.Println("No recommendations found.")
		fmt.Println("Try installing some packages first, or use --keywords to specify interests.")
		return nil
	}

	outputRecommendations(recommendations)
	return nil
}

func runSimilarRecommendations(ctx context.Context, svc *marketplace.Service) error {
	id, err := marketplace.NewPackageID(mpSimilarTo)
	if err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	config := marketplace.DefaultRecommenderConfig()
	config.MaxRecommendations = mpRecommendMax
	recommender := marketplace.NewRecommender(svc, config)

	recommendations, err := recommender.RecommendSimilar(ctx, id)
	if err != nil {
		return fmt.Errorf("recommendation failed: %w", err)
	}

	if len(recommendations) == 0 {
		fmt.Printf("No packages similar to '%s' found.\n", mpSimilarTo)
		return nil
	}

	fmt.Printf("Packages similar to '%s':\n\n", mpSimilarTo)
	outputRecommendations(recommendations)
	return nil
}

func buildUserContext(svc *marketplace.Service) marketplace.UserContext {
	var userCtx marketplace.UserContext

	// Get installed packages
	installed, err := svc.List()
	if err == nil {
		for _, pkg := range installed {
			userCtx.InstalledPackages = append(userCtx.InstalledPackages, pkg.Package.ID)
		}
	}

	// Parse keywords from flag
	if mpKeywords != "" {
		for _, kw := range strings.Split(mpKeywords, ",") {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				userCtx.Keywords = append(userCtx.Keywords, kw)
			}
		}
	}

	// Filter by type if specified
	if mpRecommendType != "" {
		userCtx.PreferredTypes = []string{mpRecommendType}
	}

	return userCtx
}

func outputRecommendations(recommendations []marketplace.Recommendation) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tSCORE\tREASONS")

	for _, rec := range recommendations {
		pkgType := rec.Package.Type
		if len(pkgType) > 12 {
			pkgType = pkgType[:12]
		}

		// Format reasons
		var reasons []string
		for _, r := range rec.Reasons {
			reasons = append(reasons, formatReason(r))
		}
		reasonStr := strings.Join(reasons, ", ")
		if len(reasonStr) > 40 {
			reasonStr = reasonStr[:37] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%.1f%%\t%s\n",
			rec.Package.ID.String(), pkgType, rec.Score*100, reasonStr)
	}

	_ = w.Flush()
}

func formatReason(r marketplace.RecommendationReason) string {
	switch r {
	case marketplace.ReasonPopular:
		return "popular"
	case marketplace.ReasonTrending:
		return "trending"
	case marketplace.ReasonSimilarKeywords:
		return "similar"
	case marketplace.ReasonSameType:
		return "same type"
	case marketplace.ReasonSameAuthor:
		return "same author"
	case marketplace.ReasonComplementary:
		return "complements"
	case marketplace.ReasonRecentlyUpdated:
		return "recent"
	case marketplace.ReasonHighlyRated:
		return "rated"
	case marketplace.ReasonProviderMatch:
		return "provider"
	case marketplace.ReasonFeatured:
		return "featured"
	default:
		return string(r)
	}
}

func runMarketplaceFeatured(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	if mpRefreshIndex {
		if err := svc.RefreshIndex(ctx); err != nil {
			return fmt.Errorf("failed to refresh index: %w", err)
		}
	}

	config := marketplace.DefaultRecommenderConfig()
	config.MaxRecommendations = 10
	recommender := marketplace.NewRecommender(svc, config)

	recommendations, err := recommender.FeaturedPackages(ctx)
	if err != nil {
		return fmt.Errorf("failed to get featured packages: %w", err)
	}

	if len(recommendations) == 0 {
		fmt.Println("No featured packages available.")
		return nil
	}

	// Filter by type if specified
	if mpFeaturedType != "" {
		var filtered []marketplace.Recommendation
		for _, rec := range recommendations {
			if rec.Package.Type == mpFeaturedType {
				filtered = append(filtered, rec)
			}
		}
		recommendations = filtered
	}

	fmt.Println("Featured Packages:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tTITLE\tAUTHOR\tSTARS")

	for _, rec := range recommendations {
		pkgType := rec.Package.Type
		if len(pkgType) > 12 {
			pkgType = pkgType[:12]
		}

		title := rec.Package.Title
		if len(title) > 25 {
			title = title[:22] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			rec.Package.ID.String(), pkgType, title,
			rec.Package.Provenance.Author, rec.Package.Stars)
	}

	_ = w.Flush()
	return nil
}

func runMarketplacePopular(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	svc := newMarketplaceService()

	if mpRefreshIndex {
		if err := svc.RefreshIndex(ctx); err != nil {
			return fmt.Errorf("failed to refresh index: %w", err)
		}
	}

	config := marketplace.DefaultRecommenderConfig()
	config.MaxRecommendations = 20
	recommender := marketplace.NewRecommender(svc, config)

	recommendations, err := recommender.PopularPackages(ctx, mpPopularType)
	if err != nil {
		return fmt.Errorf("failed to get popular packages: %w", err)
	}

	if len(recommendations) == 0 {
		fmt.Println("No packages found.")
		return nil
	}

	fmt.Println("Most Popular Packages:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "RANK\tNAME\tTYPE\tDOWNLOADS\tSTARS")

	for i, rec := range recommendations {
		pkgType := rec.Package.Type
		if len(pkgType) > 12 {
			pkgType = pkgType[:12]
		}

		_, _ = fmt.Fprintf(w, "#%d\t%s\t%s\t%d\t%d\n",
			i+1, rec.Package.ID.String(), pkgType,
			rec.Package.Downloads, rec.Package.Stars)
	}

	_ = w.Flush()
	return nil
}
