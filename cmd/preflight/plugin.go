package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/domain/plugin"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
}

func runPluginList() error {
	loader := plugin.NewLoader()
	plugins, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		fmt.Println("")
		fmt.Println("Install plugins using:")
		fmt.Println("  preflight plugin install <path-or-url>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tDESCRIPTION")
	_, _ = fmt.Fprintln(w, "────\t───────\t──────\t───────────")

	for _, p := range plugins {
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
	plugins, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	var found *plugin.Plugin
	for _, p := range plugins {
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
	plugins, err := loader.Discover()
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	var found *plugin.Plugin
	for _, p := range plugins {
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
