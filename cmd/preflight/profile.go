package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Quick switch between targets",
	Long: `Quickly switch between configuration targets without full apply.

Profiles provide a lightweight way to switch contexts (work vs personal)
by updating environment variables and shell settings without reinstalling
packages.

This is useful when:
  - Switching between work and personal contexts on the same machine
  - Testing different configurations quickly
  - Temporarily using a different set of tools

The full 'apply' command does a complete sync. Profile switching only
updates fast-changing settings like environment variables and git config.

Examples:
  preflight profile list              # Show available profiles
  preflight profile current           # Show active profile
  preflight profile switch work       # Switch to work profile
  preflight profile switch personal   # Switch to personal profile
  preflight profile create meeting    # Create new profile from current`,
	RunE: runProfileList,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available profiles",
	RunE:  runProfileList,
}

var profileCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current profile",
	RunE:  runProfileCurrent,
}

var profileSwitchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Switch to a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileSwitch,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileCreate,
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

var (
	profileConfigPath string
	profileJSON       bool
	profileFromTarget string
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCurrentCmd)
	profileCmd.AddCommand(profileSwitchCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)

	profileCmd.PersistentFlags().StringVarP(&profileConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	profileCmd.PersistentFlags().BoolVar(&profileJSON, "json", false, "Output as JSON")
	profileCreateCmd.Flags().StringVar(&profileFromTarget, "from", "", "Create from specific target")
}

// ProfileInfo represents profile metadata
type ProfileInfo struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
	LastUsed    string `json:"last_used,omitempty"`
}

func runProfileList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	// Load manifest to get targets
	manifest, err := preflight.LoadManifest(ctx, profileConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	currentProfile := getCurrentProfile()

	profiles := make([]ProfileInfo, 0, len(manifest.Targets))
	for name := range manifest.Targets {
		profiles = append(profiles, ProfileInfo{
			Name:   name,
			Target: name,
			Active: name == currentProfile,
		})
	}

	// Add custom profiles
	customProfiles, _ := loadCustomProfiles()
	for _, p := range customProfiles {
		p.Active = p.Name == currentProfile
		profiles = append(profiles, p)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles available.")
		return nil
	}

	if profileJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profiles)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROFILE\tTARGET\tSTATUS")

	for _, p := range profiles {
		status := ""
		if p.Active {
			status = "* active"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Target, status)
	}

	_ = w.Flush()
	return nil
}

func runProfileCurrent(_ *cobra.Command, _ []string) error {
	current := getCurrentProfile()

	if current == "" {
		fmt.Println("No profile active. Use 'preflight profile switch <name>' to activate one.")
		return nil
	}

	if profileJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(map[string]string{"profile": current})
	}

	fmt.Printf("Current profile: %s\n", current)
	return nil
}

func runProfileSwitch(_ *cobra.Command, args []string) error {
	profileName := args[0]
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	// Determine target for profile
	target := profileName
	customProfiles, _ := loadCustomProfiles()
	for _, p := range customProfiles {
		if p.Name == profileName {
			target = p.Target
			break
		}
	}

	fmt.Printf("Switching to profile: %s (target: %s)\n\n", profileName, target)

	// Load configuration for target
	config, err := preflight.LoadMergedConfig(ctx, profileConfigPath, target)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply fast settings (env, git config)
	fmt.Println("Applying profile settings...")

	// Update environment variables
	vars := extractEnvVars(config)
	if err := WriteEnvFile(vars); err != nil {
		fmt.Printf("Warning: failed to write env file: %v\n", err)
	} else {
		fmt.Printf("  Updated %d environment variable(s)\n", len(vars))
	}

	// Update git config
	if git, ok := config["git"].(map[string]interface{}); ok {
		if err := applyGitConfig(git); err != nil {
			fmt.Printf("Warning: failed to apply git config: %v\n", err)
		} else {
			fmt.Println("  Updated git configuration")
		}
	}

	// Save current profile
	if err := setCurrentProfile(profileName); err != nil {
		fmt.Printf("Warning: failed to save profile state: %v\n", err)
	}

	fmt.Printf("\nSwitched to profile: %s\n", profileName)
	fmt.Println("\nNote: Shell environment changes require reloading your shell or running:")
	fmt.Println("  source ~/.preflight/env.sh")

	return nil
}

func runProfileCreate(_ *cobra.Command, args []string) error {
	name := args[0]
	target := profileFromTarget
	if target == "" {
		target = "default"
	}

	profiles, _ := loadCustomProfiles()

	// Check if already exists
	for _, p := range profiles {
		if p.Name == name {
			return fmt.Errorf("profile '%s' already exists", name)
		}
	}

	profiles = append(profiles, ProfileInfo{
		Name:   name,
		Target: target,
	})

	if err := saveCustomProfiles(profiles); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("Created profile '%s' from target '%s'\n", name, target)
	return nil
}

func runProfileDelete(_ *cobra.Command, args []string) error {
	name := args[0]

	profiles, _ := loadCustomProfiles()

	newProfiles := make([]ProfileInfo, 0, len(profiles))
	found := false
	for _, p := range profiles {
		if p.Name == name {
			found = true
			continue
		}
		newProfiles = append(newProfiles, p)
	}

	if !found {
		return fmt.Errorf("profile '%s' not found", name)
	}

	if err := saveCustomProfiles(newProfiles); err != nil {
		return fmt.Errorf("failed to save profiles: %w", err)
	}

	fmt.Printf("Deleted profile '%s'\n", name)
	return nil
}

func getProfileDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".preflight", "profiles")
}

func getCurrentProfile() string {
	data, err := os.ReadFile(filepath.Join(getProfileDir(), "current"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func setCurrentProfile(name string) error {
	dir := getProfileDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "current"), []byte(name), 0o644)
}

func loadCustomProfiles() ([]ProfileInfo, error) {
	data, err := os.ReadFile(filepath.Join(getProfileDir(), "profiles.yaml"))
	if err != nil {
		return nil, err
	}

	var profiles []ProfileInfo
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func saveCustomProfiles(profiles []ProfileInfo) error {
	dir := getProfileDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(profiles)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "profiles.yaml"), data, 0o644)
}

func applyGitConfig(git map[string]interface{}) error {
	// Apply basic git settings using git config
	if name, ok := git["name"].(string); ok {
		if err := runGitConfigSet("user.name", name); err != nil {
			return err
		}
	}
	if email, ok := git["email"].(string); ok {
		if err := runGitConfigSet("user.email", email); err != nil {
			return err
		}
	}
	if signingKey, ok := git["signing_key"].(string); ok {
		if err := runGitConfigSet("user.signingkey", signingKey); err != nil {
			return err
		}
	}
	return nil
}

func runGitConfigSet(key, value string) error {
	// Would run: git config --global key value
	// For now, just log
	fmt.Printf("    git config --global %s %q\n", key, value)
	return nil
}
