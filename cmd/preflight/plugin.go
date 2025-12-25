package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/domain/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Preflight plugins",
	Long:  `Discover, install, and manage plugins that extend Preflight's capabilities.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long:  `Display all installed plugins with their version and status.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runPluginList()
	},
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a plugin",
	Long: `Install a plugin from a local path or Git repository.

Examples:
  preflight plugin install /path/to/plugin
  preflight plugin install https://github.com/example/preflight-docker.git`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runPluginInstall(args[0])
	},
}

var pluginRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove a plugin",
	Long:    `Remove an installed plugin by name.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runPluginRemove(args[0])
	},
}

var pluginInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show plugin details",
	Long:  `Display detailed information about an installed plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runPluginInfo(args[0])
	},
}

var (
	searchType     string
	searchMinStars int
	searchLimit    int
	searchSort     string

	pluginValidateJSON   bool
	pluginValidateStrict bool

	upgradeCheckOnly bool
	upgradeDryRun    bool
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for plugins on GitHub",
	Long: `Search for Preflight plugins on GitHub using topic-based discovery.

Plugins are discovered by GitHub topics:
  • preflight-plugin   - Configuration plugins (presets, capability packs)
  • preflight-provider - WASM provider plugins

Examples:
  preflight plugin search docker
  preflight plugin search --type provider kubernetes
  preflight plugin search --min-stars 10 terminal
  preflight plugin search --sort updated`,
	RunE: func(_ *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		return runPluginSearch(query)
	},
}

var pluginValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate a plugin manifest",
	Long: `Validate a plugin's manifest (plugin.yaml) for correctness.

Checks performed:
  • Required fields (apiVersion, name, version)
  • Semantic versioning format
  • WASM configuration for provider plugins
  • Capability justifications
  • Dependency syntax

Warnings (non-fatal):
  • Missing description, author, or license
  • Unsigned plugins
  • Dangerous capability requests

Examples:
  preflight plugin validate                    # Validate plugin in current directory
  preflight plugin validate ./my-plugin        # Validate plugin at path
  preflight plugin validate --json ./plugin    # Output as JSON
  preflight plugin validate --strict ./plugin  # Treat warnings as errors`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		return runPluginValidate(path)
	},
}

var pluginUpgradeCmd = &cobra.Command{
	Use:   "upgrade [name]",
	Short: "Check for and apply plugin updates",
	Long: `Check for available plugin updates and optionally apply them.

When called without arguments, checks all installed plugins.
When called with a plugin name, checks and upgrades that specific plugin.

Examples:
  preflight plugin upgrade               # Upgrade all plugins
  preflight plugin upgrade my-plugin     # Upgrade specific plugin
  preflight plugin upgrade --check       # Only check, don't upgrade
  preflight plugin upgrade --dry-run     # Show what would be upgraded`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		return runPluginUpgrade(name)
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginValidateCmd)
	pluginCmd.AddCommand(pluginUpgradeCmd)

	// Search flags
	pluginSearchCmd.Flags().StringVar(&searchType, "type", "", "Filter by plugin type: config, provider")
	pluginSearchCmd.Flags().IntVar(&searchMinStars, "min-stars", 0, "Minimum number of GitHub stars")
	pluginSearchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum number of results")
	pluginSearchCmd.Flags().StringVar(&searchSort, "sort", "stars", "Sort by: stars, updated, best-match")

	// Validate flags
	pluginValidateCmd.Flags().BoolVar(&pluginValidateJSON, "json", false, "Output validation results as JSON")
	pluginValidateCmd.Flags().BoolVar(&pluginValidateStrict, "strict", false, "Treat warnings as errors")

	// Upgrade flags
	pluginUpgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for upgrades, don't apply")
	pluginUpgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "Show what would be upgraded without making changes")
}

func runPluginList() error {
	loader := plugin.NewLoader()
	result, err := loader.Discover(context.Background())
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	if len(result.Plugins) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("")
		fmt.Println("Install plugins using:")
		fmt.Println("  preflight plugin install <path-or-url>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "────\t───────\t──────\t───────────")

	for _, p := range result.Plugins {
		status := "disabled"
		if p.Enabled {
			status = "enabled"
		}
		desc := p.Manifest.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			p.Manifest.Name,
			p.Manifest.Version,
			status,
			desc,
		)
	}
	return w.Flush()
}

func runPluginInstall(source string) error {
	loader := plugin.NewLoader()

	// Determine if source is a local path or Git URL
	info, err := os.Stat(source)
	if err == nil && info.IsDir() {
		// Local path
		p, err := loader.LoadFromPath(source)
		if err != nil {
			return fmt.Errorf("loading plugin: %w", err)
		}

		// For now, just validate - actual installation would copy to install path
		fmt.Printf("✓ Plugin validated: %s@%s\n", p.Manifest.Name, p.Manifest.Version)
		fmt.Println("")
		fmt.Println("Note: Full installation (copying to ~/.preflight/plugins) not yet implemented.")
		fmt.Printf("      The plugin at %s can be used directly.\n", source)
		return nil
	}

	// Git URL
	_, err = loader.LoadFromGit(source, "latest")
	if err != nil {
		return fmt.Errorf("installing from git: %w", err)
	}

	return nil
}

func runPluginRemove(name string) error {
	loader := plugin.NewLoader()
	result, err := loader.Discover(context.Background())
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	var found *plugin.Plugin
	for _, p := range result.Plugins {
		if p.Manifest.Name == name {
			found = p
			break
		}
	}

	if found == nil {
		return fmt.Errorf("plugin %q not found", name)
	}

	fmt.Printf("Found plugin: %s@%s at %s\n", found.Manifest.Name, found.Manifest.Version, found.Path)
	fmt.Println("")
	fmt.Println("Note: Plugin removal not yet implemented.")
	fmt.Println("      To remove manually, delete the plugin directory.")

	return nil
}

func runPluginInfo(name string) error {
	loader := plugin.NewLoader()
	result, err := loader.Discover(context.Background())
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	var found *plugin.Plugin
	for _, p := range result.Plugins {
		if p.Manifest.Name == name {
			found = p
			break
		}
	}

	if found == nil {
		return fmt.Errorf("plugin %q not found", name)
	}

	m := found.Manifest
	fmt.Printf("Name:        %s\n", m.Name)
	fmt.Printf("Version:     %s\n", m.Version)
	fmt.Printf("API Version: %s\n", m.APIVersion)
	if m.Description != "" {
		fmt.Printf("Description: %s\n", m.Description)
	}
	if m.Author != "" {
		fmt.Printf("Author:      %s\n", m.Author)
	}
	if m.License != "" {
		fmt.Printf("License:     %s\n", m.License)
	}
	if m.Homepage != "" {
		fmt.Printf("Homepage:    %s\n", m.Homepage)
	}
	if m.Repository != "" {
		fmt.Printf("Repository:  %s\n", m.Repository)
	}
	fmt.Printf("Path:        %s\n", found.Path)
	fmt.Printf("Status:      %s\n", map[bool]string{true: "enabled", false: "disabled"}[found.Enabled])
	fmt.Printf("Loaded:      %s\n", found.LoadedAt.Format("2006-01-02 15:04:05"))

	if len(m.Provides.Providers) > 0 {
		fmt.Println("")
		fmt.Println("Providers:")
		for _, p := range m.Provides.Providers {
			fmt.Printf("  • %s (%s)\n", p.Name, p.ConfigKey)
			if p.Description != "" {
				fmt.Printf("    %s\n", p.Description)
			}
		}
	}

	if len(m.Provides.Presets) > 0 {
		fmt.Println("")
		fmt.Println("Presets:")
		for _, p := range m.Provides.Presets {
			fmt.Printf("  • %s\n", p)
		}
	}

	if len(m.Provides.CapabilityPacks) > 0 {
		fmt.Println("")
		fmt.Println("Capability Packs:")
		for _, p := range m.Provides.CapabilityPacks {
			fmt.Printf("  • %s\n", p)
		}
	}

	if len(m.Requires) > 0 {
		fmt.Println("")
		fmt.Println("Dependencies:")
		for _, d := range m.Requires {
			if d.Version != "" {
				fmt.Printf("  • %s %s\n", d.Name, d.Version)
			} else {
				fmt.Printf("  • %s\n", d.Name)
			}
		}
	}

	return nil
}

func runPluginSearch(query string) error {
	ctx := context.Background()

	// Build search options
	opts := plugin.DefaultSearchOptions()
	opts.Query = query
	opts.Limit = searchLimit
	opts.MinStars = searchMinStars
	opts.SortBy = searchSort

	// Parse type filter
	switch searchType {
	case "config":
		opts.Type = plugin.TypeConfig
	case "provider":
		opts.Type = plugin.TypeProvider
	case "":
		// No filter
	default:
		return fmt.Errorf("invalid type %q: must be 'config' or 'provider'", searchType)
	}

	// Create searcher and search
	searcher := plugin.NewSearcher()
	results, err := searcher.Search(ctx, opts)
	if err != nil {
		return fmt.Errorf("searching plugins: %w", err)
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("No plugins found matching your criteria.")
		fmt.Println("")
		fmt.Println("Tips:")
		fmt.Println("  • Try a different search query")
		fmt.Println("  • Remove the --type filter")
		fmt.Println("  • Lower the --min-stars threshold")
		fmt.Println("")
		fmt.Println("To create a plugin, see: https://felixgeelhaar.github.io/preflight/guides/plugins/")
		return nil
	}

	// Print header
	fmt.Printf("Found %d plugin(s)", len(results))
	if query != "" {
		fmt.Printf(" matching %q", query)
	}
	fmt.Println(":")
	fmt.Println("")

	// Print results in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPOSITORY\tTYPE\tSTARS\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "──────────\t────\t─────\t───────────")

	for _, r := range results {
		typeLabel := "config"
		if r.PluginType == plugin.TypeProvider {
			typeLabel = "provider"
		}

		desc := r.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			r.FullName,
			typeLabel,
			r.Stars,
			desc,
		)
	}
	_ = w.Flush()

	fmt.Println("")
	fmt.Println("Install a plugin with:")
	fmt.Println("  preflight plugin install https://github.com/<repository>")

	return nil
}

// ValidationResult contains the result of plugin validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Plugin   string   `json:"plugin,omitempty"`
	Version  string   `json:"version,omitempty"`
	Path     string   `json:"path"`
}

func runPluginValidate(path string) error {
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Path:     path,
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("path does not exist: %s", path))
		return outputValidationResult(result)
	}

	if !info.IsDir() {
		result.Valid = false
		result.Errors = append(result.Errors, "path must be a directory containing plugin.yaml")
		return outputValidationResult(result)
	}

	// Try to load the plugin
	loader := plugin.NewLoader()
	p, err := loader.LoadFromPath(path)
	if err != nil {
		result.Valid = false
		// Parse the error to provide better feedback
		if plugin.IsManifestSizeError(err) {
			result.Errors = append(result.Errors, "manifest file exceeds size limit (256KB)")
		} else if plugin.IsValidationError(err) {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("loading plugin: %v", err))
		}
		return outputValidationResult(result)
	}

	result.Plugin = p.Manifest.Name
	result.Version = p.Manifest.Version

	// Additional validation checks

	// Check semantic versioning
	v := p.Manifest.Version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		result.Errors = append(result.Errors, fmt.Sprintf("version %q is not valid semver (e.g., 1.0.0)", p.Manifest.Version))
		result.Valid = false
	}

	// Check for recommended fields (warnings)
	if p.Manifest.Description == "" {
		result.Warnings = append(result.Warnings, "missing description field")
	}
	if p.Manifest.Author == "" {
		result.Warnings = append(result.Warnings, "missing author field")
	}
	if p.Manifest.License == "" {
		result.Warnings = append(result.Warnings, "missing license field")
	}

	// Check for signature
	if p.Manifest.Signature == nil {
		result.Warnings = append(result.Warnings, "plugin is not signed")
	}

	// Check dependencies have version constraints
	for _, dep := range p.Manifest.Requires {
		if dep.Version == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("dependency %q has no version constraint", dep.Name))
		}
	}

	// Check WASM config for provider plugins
	if p.Manifest.Type == plugin.TypeProvider {
		if p.Manifest.WASM == nil {
			result.Errors = append(result.Errors, "provider plugin requires wasm configuration")
			result.Valid = false
		} else {
			// Check for dangerous capabilities
			dangerousCaps := []string{"fs:write", "net:raw", "exec"}
			for _, cap := range p.Manifest.WASM.Capabilities {
				for _, dangerous := range dangerousCaps {
					if cap.Name == dangerous {
						if cap.Justification == "" {
							result.Warnings = append(result.Warnings,
								fmt.Sprintf("dangerous capability %q should have justification", cap.Name))
						}
					}
				}
			}
		}
	}

	// In strict mode, warnings become errors
	if pluginValidateStrict && len(result.Warnings) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, result.Warnings...)
		result.Warnings = nil
	}

	return outputValidationResult(result)
}

func outputValidationResult(result ValidationResult) error {
	if pluginValidateJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	if result.Valid {
		fmt.Printf("✓ Plugin validated: %s@%s\n", result.Plugin, result.Version)
	} else {
		fmt.Println("✗ Validation failed")
	}

	if len(result.Errors) > 0 {
		fmt.Println("")
		fmt.Println("Errors:")
		for _, e := range result.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("")
		fmt.Println("Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
	}

	if result.Valid {
		fmt.Println("")
		fmt.Println("Path:", result.Path)
		return nil
	}

	return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
}

func runPluginUpgrade(name string) error {
	ctx := context.Background()

	// Discover plugins and build registry
	loader := plugin.NewLoader()
	discoverResult, err := loader.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	if len(discoverResult.Plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	// Build registry from discovered plugins
	registry := plugin.NewRegistry()
	for _, p := range discoverResult.Plugins {
		if err := registry.Register(p); err != nil {
			// Skip plugins that can't be registered
			continue
		}
	}

	checker := plugin.NewUpgradeChecker(registry)

	// Check or upgrade specific plugin
	if name != "" {
		info, err := checker.CheckUpgrade(ctx, name)
		if err != nil {
			return fmt.Errorf("checking upgrade: %w", err)
		}

		if !info.UpgradeAvailable {
			fmt.Printf("✓ %s@%s is up to date\n", info.Name, info.CurrentVersion)
			return nil
		}

		fmt.Printf("Upgrade available: %s@%s → %s\n", info.Name, info.CurrentVersion, info.LatestVersion)
		if info.ChangelogURL != "" {
			fmt.Printf("  Changelog: %s\n", info.ChangelogURL)
		}

		if upgradeCheckOnly {
			return nil
		}

		if upgradeDryRun {
			fmt.Println("")
			fmt.Println("Dry run: No changes made.")
			return nil
		}

		// Perform upgrade
		result, err := checker.Upgrade(ctx, name, false)
		if err != nil {
			return fmt.Errorf("upgrading plugin: %w", err)
		}

		fmt.Println("")
		fmt.Printf("✓ Upgraded %s to %s\n", result.Name, result.CurrentVersion)
		return nil
	}

	// Check all plugins
	infos, err := checker.CheckAllUpgrades(ctx)
	if err != nil {
		return fmt.Errorf("checking upgrades: %w", err)
	}

	// Count available upgrades
	var available int
	for _, info := range infos {
		if info.UpgradeAvailable {
			available++
		}
	}

	if available == 0 {
		fmt.Println("✓ All plugins are up to date.")
		return nil
	}

	fmt.Printf("Found %d upgrade(s) available:\n\n", available)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PLUGIN\tCURRENT\tLATEST")
	_, _ = fmt.Fprintln(w, "──────\t───────\t──────")

	for _, info := range infos {
		if info.UpgradeAvailable {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
				info.Name,
				info.CurrentVersion,
				info.LatestVersion,
			)
		}
	}
	_ = w.Flush()

	if upgradeCheckOnly {
		return nil
	}

	if upgradeDryRun {
		fmt.Println("")
		fmt.Println("Dry run: No changes made.")
		return nil
	}

	// Perform upgrades
	fmt.Println("")
	fmt.Println("Upgrading plugins...")

	results, err := checker.UpgradeAll(ctx, false)
	if err != nil {
		return fmt.Errorf("upgrading plugins: %w", err)
	}

	fmt.Println("")
	for _, result := range results {
		if result.UpgradeAvailable {
			fmt.Printf("✗ Failed to upgrade %s\n", result.Name)
		} else {
			fmt.Printf("✓ Upgraded %s to %s\n", result.Name, result.CurrentVersion)
		}
	}

	return nil
}
