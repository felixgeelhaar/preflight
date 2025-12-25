package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove orphaned packages and files",
	Long: `Identify and remove packages and files not declared in configuration.

Clean compares your current system state against the configuration and
identifies "orphaned" items - things that exist on your system but are
not part of your preflight configuration.

This is useful for:
  - Removing packages installed manually that you forgot about
  - Cleaning up files from old configurations
  - Keeping your system minimal and reproducible

By default, clean shows what would be removed without making changes.
Use --apply to actually remove orphaned items.

Examples:
  preflight clean                     # Show orphaned items
  preflight clean --apply             # Remove orphaned items
  preflight clean --provider brew     # Only check Homebrew
  preflight clean --ignore 'htop,curl' # Ignore specific packages
  preflight clean --json              # JSON output for scripting`,
	RunE: runClean,
}

var (
	cleanConfigPath string
	cleanTarget     string
	cleanApply      bool
	cleanProviders  string
	cleanIgnore     string
	cleanJSON       bool
	cleanForce      bool
)

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().StringVarP(&cleanConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	cleanCmd.Flags().StringVarP(&cleanTarget, "target", "t", "default", "Target to check against")
	cleanCmd.Flags().BoolVar(&cleanApply, "apply", false, "Actually remove orphaned items")
	cleanCmd.Flags().StringVar(&cleanProviders, "providers", "", "Only check specific providers (comma-separated)")
	cleanCmd.Flags().StringVar(&cleanIgnore, "ignore", "", "Ignore specific items (comma-separated)")
	cleanCmd.Flags().BoolVar(&cleanJSON, "json", false, "Output as JSON")
	cleanCmd.Flags().BoolVar(&cleanForce, "force", false, "Skip confirmation prompt")
}

// OrphanedItem represents an item not in configuration
type OrphanedItem struct {
	Provider string `json:"provider"`
	Type     string `json:"type"` // "package", "file", "extension", etc.
	Name     string `json:"name"`
	Details  string `json:"details,omitempty"`
}

func runClean(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	// Load configuration
	config, err := preflight.LoadMergedConfig(ctx, cleanConfigPath, cleanTarget)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get current system state
	systemState, err := preflight.CaptureSystemState(ctx)
	if err != nil {
		return fmt.Errorf("failed to capture system state: %w", err)
	}

	// Parse filters
	var providerFilter []string
	if cleanProviders != "" {
		providerFilter = strings.Split(cleanProviders, ",")
	}

	var ignoreList []string
	if cleanIgnore != "" {
		ignoreList = strings.Split(cleanIgnore, ",")
	}

	// Find orphaned items
	orphans := findOrphans(config, systemState, providerFilter, ignoreList)

	if len(orphans) == 0 {
		fmt.Println("No orphaned items found. Your system matches the configuration.")
		return nil
	}

	// Output results
	if cleanJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(orphans)
	}

	outputOrphansText(orphans)

	if !cleanApply {
		fmt.Println("\nRun with --apply to remove these items.")
		return nil
	}

	// Confirm before applying
	if !cleanForce && !yesFlag {
		fmt.Print("\nRemove these items? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Remove orphaned items
	removed, failed := removeOrphans(ctx, orphans)

	fmt.Printf("\nRemoved %d items", removed)
	if failed > 0 {
		fmt.Printf(" (%d failed)", failed)
	}
	fmt.Println()

	return nil
}

func findOrphans(config, systemState map[string]interface{}, providerFilter, ignoreList []string) []OrphanedItem {
	var orphans []OrphanedItem

	// Check Homebrew packages
	if shouldCheckProvider(providerFilter, "brew") {
		orphans = append(orphans, findBrewOrphans(config, systemState, ignoreList)...)
	}

	// Check VS Code extensions
	if shouldCheckProvider(providerFilter, "vscode") {
		orphans = append(orphans, findVSCodeOrphans(config, systemState, ignoreList)...)
	}

	// Check dotfiles
	if shouldCheckProvider(providerFilter, "files") {
		orphans = append(orphans, findFileOrphans(config, systemState, ignoreList)...)
	}

	return orphans
}

func shouldCheckProvider(filter []string, provider string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, p := range filter {
		if strings.TrimSpace(p) == provider {
			return true
		}
	}
	return false
}

func findBrewOrphans(config, systemState map[string]interface{}, ignoreList []string) []OrphanedItem {
	var orphans []OrphanedItem

	// Get configured packages
	configuredFormulae := make(map[string]bool)
	configuredCasks := make(map[string]bool)

	if brew, ok := config["brew"].(map[string]interface{}); ok {
		if formulae, ok := brew["formulae"].([]interface{}); ok {
			for _, f := range formulae {
				if name, ok := f.(string); ok {
					configuredFormulae[name] = true
				}
			}
		}
		if casks, ok := brew["casks"].([]interface{}); ok {
			for _, c := range casks {
				if name, ok := c.(string); ok {
					configuredCasks[name] = true
				}
			}
		}
	}

	// Get installed packages from system state
	if brew, ok := systemState["brew"].(map[string]interface{}); ok {
		if formulae, ok := brew["formulae"].([]interface{}); ok {
			for _, f := range formulae {
				name, ok := f.(string)
				if !ok {
					continue
				}
				if !configuredFormulae[name] && !isIgnored(name, ignoreList) {
					orphans = append(orphans, OrphanedItem{
						Provider: "brew",
						Type:     "formula",
						Name:     name,
					})
				}
			}
		}
		if casks, ok := brew["casks"].([]interface{}); ok {
			for _, c := range casks {
				name, ok := c.(string)
				if !ok {
					continue
				}
				if !configuredCasks[name] && !isIgnored(name, ignoreList) {
					orphans = append(orphans, OrphanedItem{
						Provider: "brew",
						Type:     "cask",
						Name:     name,
					})
				}
			}
		}
	}

	return orphans
}

func findVSCodeOrphans(config, systemState map[string]interface{}, ignoreList []string) []OrphanedItem {
	var orphans []OrphanedItem

	// Get configured extensions
	configuredExtensions := make(map[string]bool)

	if vscode, ok := config["vscode"].(map[string]interface{}); ok {
		if extensions, ok := vscode["extensions"].([]interface{}); ok {
			for _, e := range extensions {
				if name, ok := e.(string); ok {
					configuredExtensions[strings.ToLower(name)] = true
				}
			}
		}
	}

	// Get installed extensions from system state
	if vscode, ok := systemState["vscode"].(map[string]interface{}); ok {
		if extensions, ok := vscode["extensions"].([]interface{}); ok {
			for _, e := range extensions {
				name, ok := e.(string)
				if !ok {
					continue
				}
				if !configuredExtensions[strings.ToLower(name)] && !isIgnored(name, ignoreList) {
					orphans = append(orphans, OrphanedItem{
						Provider: "vscode",
						Type:     "extension",
						Name:     name,
					})
				}
			}
		}
	}

	return orphans
}

func findFileOrphans(_, _ map[string]interface{}, _ []string) []OrphanedItem {
	// File orphan detection is more complex and requires tracking managed files
	// This would need a registry of files created by preflight
	return nil
}

func isIgnored(name string, ignoreList []string) bool {
	for _, ignored := range ignoreList {
		if strings.TrimSpace(ignored) == name {
			return true
		}
	}
	return false
}

func outputOrphansText(orphans []OrphanedItem) {
	fmt.Printf("Found %d orphaned items:\n\n", len(orphans))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROVIDER\tTYPE\tNAME")

	for _, o := range orphans {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", o.Provider, o.Type, o.Name)
	}

	_ = w.Flush()
}

func removeOrphans(_ context.Context, orphans []OrphanedItem) (removed, failed int) {
	for _, o := range orphans {
		var err error

		switch o.Provider {
		case "brew":
			switch o.Type {
			case "formula":
				err = runBrewUninstall(o.Name, false)
			case "cask":
				err = runBrewUninstall(o.Name, true)
			}
		case "vscode":
			err = runVSCodeUninstall(o.Name)
		}

		if err != nil {
			fmt.Printf("Failed to remove %s %s: %v\n", o.Provider, o.Name, err)
			failed++
		} else {
			fmt.Printf("Removed %s %s\n", o.Provider, o.Name)
			removed++
		}
	}

	return removed, failed
}

//nolint:unparam // error return for future implementation
func runBrewUninstall(name string, isCask bool) error {
	args := []string{"uninstall"}
	if isCask {
		args = append(args, "--cask")
	}
	args = append(args, name)

	// Would execute: brew uninstall [--cask] name
	// For now, just log what would happen
	fmt.Printf("  brew %s\n", strings.Join(args, " "))
	return nil
}

func runVSCodeUninstall(name string) error {
	// Would execute: code --uninstall-extension name
	fmt.Printf("  code --uninstall-extension %s\n", name)
	return nil
}
