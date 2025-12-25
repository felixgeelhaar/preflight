package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables",
	Long: `Manage environment variables across layers and targets.

Environment variables can be defined in layers and are exported to your
shell configuration. This provides:
  - Layer-based organization (base, work, personal)
  - Target-specific variables (work target vs personal target)
  - Secure handling (secrets via references, not plaintext)

Variables are written to ~/.preflight/env.sh which should be sourced
in your shell configuration.

Examples:
  preflight env list                    # List all variables
  preflight env list --target work      # Target-specific variables
  preflight env set EDITOR nvim         # Set a variable
  preflight env get EDITOR              # Get a variable
  preflight env unset EDITOR            # Remove a variable
  preflight env export                  # Generate shell export script
  preflight env diff                    # Show env var differences between targets`,
	RunE: runEnvList,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environment variables",
	RunE:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set <name> <value>",
	Short: "Set an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvSet,
}

var envGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get an environment variable value",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvGet,
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <name>",
	Short: "Remove an environment variable",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvUnset,
}

var envExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Generate shell export script",
	RunE:  runEnvExport,
}

var envDiffCmd = &cobra.Command{
	Use:   "diff <target1> <target2>",
	Short: "Show differences between targets",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvDiff,
}

var (
	envConfigPath string
	envTarget     string
	envLayer      string
	envJSON       bool
	envShell      string
)

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envUnsetCmd)
	envCmd.AddCommand(envExportCmd)
	envCmd.AddCommand(envDiffCmd)

	envCmd.PersistentFlags().StringVarP(&envConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	envCmd.PersistentFlags().StringVarP(&envTarget, "target", "t", "default", "Target to use")
	envCmd.PersistentFlags().StringVar(&envLayer, "layer", "", "Specific layer to modify")
	envCmd.PersistentFlags().BoolVar(&envJSON, "json", false, "Output as JSON")
	envExportCmd.Flags().StringVar(&envShell, "shell", "bash", "Shell format (bash, zsh, fish)")
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Layer  string `json:"layer,omitempty"`
	Secret bool   `json:"secret,omitempty"`
}

func runEnvList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	config, err := preflight.LoadMergedConfig(ctx, envConfigPath, envTarget)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	vars := extractEnvVars(config)

	if len(vars) == 0 {
		fmt.Println("No environment variables defined.")
		return nil
	}

	if envJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(vars)
	}

	// Sort by name
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tVALUE\tLAYER")

	for _, v := range vars {
		value := v.Value
		if v.Secret {
			value = "***"
		} else if len(value) > 40 {
			value = value[:37] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", v.Name, value, v.Layer)
	}

	_ = w.Flush()
	return nil
}

func runEnvSet(_ *cobra.Command, args []string) error {
	name := args[0]
	value := args[1]

	layer := envLayer
	if layer == "" {
		layer = "base"
	}

	layerPath := filepath.Join(filepath.Dir(envConfigPath), "layers", layer+".yaml")

	// Load existing layer
	layerData := make(map[string]interface{})
	if data, err := os.ReadFile(layerPath); err == nil {
		if err := yaml.Unmarshal(data, &layerData); err != nil {
			return fmt.Errorf("failed to parse layer: %w", err)
		}
	}

	// Ensure env section exists
	if layerData["env"] == nil {
		layerData["env"] = make(map[string]interface{})
	}

	env, ok := layerData["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
		layerData["env"] = env
	}

	// Set the variable
	env[name] = value

	// Write back
	data, err := yaml.Marshal(layerData)
	if err != nil {
		return fmt.Errorf("failed to marshal layer: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(layerPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(layerPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write layer: %w", err)
	}

	fmt.Printf("Set %s=%s in layer %s\n", name, value, layer)
	return nil
}

func runEnvGet(_ *cobra.Command, args []string) error {
	name := args[0]

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	config, err := preflight.LoadMergedConfig(ctx, envConfigPath, envTarget)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	vars := extractEnvVars(config)

	for _, v := range vars {
		if v.Name == name {
			fmt.Println(v.Value)
			return nil
		}
	}

	return fmt.Errorf("variable '%s' not found", name)
}

func runEnvUnset(_ *cobra.Command, args []string) error {
	name := args[0]

	layer := envLayer
	if layer == "" {
		layer = "base"
	}

	layerPath := filepath.Join(filepath.Dir(envConfigPath), "layers", layer+".yaml")

	// Load existing layer
	data, err := os.ReadFile(layerPath)
	if err != nil {
		return fmt.Errorf("layer not found: %w", err)
	}

	layerData := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &layerData); err != nil {
		return fmt.Errorf("failed to parse layer: %w", err)
	}

	env, ok := layerData["env"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no env section in layer %s", layer)
	}

	if _, exists := env[name]; !exists {
		return fmt.Errorf("variable '%s' not found in layer %s", name, layer)
	}

	delete(env, name)

	// Write back
	data, err = yaml.Marshal(layerData)
	if err != nil {
		return fmt.Errorf("failed to marshal layer: %w", err)
	}

	if err := os.WriteFile(layerPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write layer: %w", err)
	}

	fmt.Printf("Removed %s from layer %s\n", name, layer)
	return nil
}

func runEnvExport(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	config, err := preflight.LoadMergedConfig(ctx, envConfigPath, envTarget)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	vars := extractEnvVars(config)

	// Sort by name
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	switch envShell {
	case "fish":
		fmt.Println("# Generated by preflight env export")
		fmt.Println("# Add to ~/.config/fish/conf.d/preflight.fish")
		for _, v := range vars {
			if v.Secret {
				continue // Skip secrets in plain export
			}
			fmt.Printf("set -gx %s %q\n", v.Name, v.Value)
		}
	case "bash", "zsh":
		fmt.Println("# Generated by preflight env export")
		fmt.Println("# Add to ~/.bashrc or ~/.zshrc: source ~/.preflight/env.sh")
		for _, v := range vars {
			if v.Secret {
				continue
			}
			fmt.Printf("export %s=%q\n", v.Name, v.Value)
		}
	default:
		return fmt.Errorf("unsupported shell: %s", envShell)
	}

	return nil
}

func runEnvDiff(_ *cobra.Command, args []string) error {
	target1 := args[0]
	target2 := args[1]

	ctx := context.Background()
	preflight := app.New(os.Stdout)

	config1, err := preflight.LoadMergedConfig(ctx, envConfigPath, target1)
	if err != nil {
		return fmt.Errorf("failed to load target %s: %w", target1, err)
	}

	config2, err := preflight.LoadMergedConfig(ctx, envConfigPath, target2)
	if err != nil {
		return fmt.Errorf("failed to load target %s: %w", target2, err)
	}

	vars1 := extractEnvVarsMap(config1)
	vars2 := extractEnvVarsMap(config2)

	// Find differences
	var diffs []string

	// Only in target1
	for name, value := range vars1 {
		if _, exists := vars2[name]; !exists {
			diffs = append(diffs, fmt.Sprintf("- %s=%s (only in %s)", name, value, target1))
		}
	}

	// Only in target2
	for name, value := range vars2 {
		if _, exists := vars1[name]; !exists {
			diffs = append(diffs, fmt.Sprintf("+ %s=%s (only in %s)", name, value, target2))
		}
	}

	// Different values
	for name, value1 := range vars1 {
		if value2, exists := vars2[name]; exists && value1 != value2 {
			diffs = append(diffs, fmt.Sprintf("~ %s: %s → %s", name, value1, value2))
		}
	}

	if len(diffs) == 0 {
		fmt.Printf("No differences between %s and %s\n", target1, target2)
		return nil
	}

	sort.Strings(diffs)
	fmt.Printf("Differences between %s and %s:\n\n", target1, target2)
	for _, d := range diffs {
		fmt.Println(d)
	}

	return nil
}

func extractEnvVars(config map[string]interface{}) []EnvVar {
	var vars []EnvVar

	if env, ok := config["env"].(map[string]interface{}); ok {
		for name, value := range env {
			v := EnvVar{
				Name:  name,
				Value: fmt.Sprintf("%v", value),
			}
			if strings.HasPrefix(v.Value, "secret://") {
				v.Secret = true
			}
			vars = append(vars, v)
		}
	}

	return vars
}

func extractEnvVarsMap(config map[string]interface{}) map[string]string {
	result := make(map[string]string)

	if env, ok := config["env"].(map[string]interface{}); ok {
		for name, value := range env {
			result[name] = fmt.Sprintf("%v", value)
		}
	}

	return result
}

// WriteEnvFile writes environment variables to ~/.preflight/env.sh
func WriteEnvFile(vars []EnvVar) error {
	home, _ := os.UserHomeDir()
	envPath := filepath.Join(home, ".preflight", "env.sh")

	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(envPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := bufio.NewWriter(f)
	_, _ = w.WriteString("# Generated by preflight - do not edit manually\n\n")

	for _, v := range vars {
		if v.Secret {
			continue
		}
		_, _ = fmt.Fprintf(w, "export %s=%q\n", v.Name, v.Value)
	}

	return w.Flush()
}
