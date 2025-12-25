package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare <source> <target>",
	Short: "Compare two configurations or targets",
	Long: `Compare configurations side-by-side to identify differences.

Comparison modes:
  - Two targets in the same config file
  - Two different config files
  - Local config vs remote URL
  - Current state vs config (use 'diff' command instead)

The output shows additions (+), removals (-), and changes (~) between
the source and target configurations.

Examples:
  preflight compare work personal                    # Compare targets
  preflight compare -c config1.yaml -C config2.yaml  # Compare files
  preflight compare work --remote github:user/repo   # Compare with remote
  preflight compare --providers brew,git             # Filter by providers`,
	Args: cobra.MaximumNArgs(2),
	RunE: runCompare,
}

var (
	compareConfigPath       string
	compareSecondConfigPath string
	compareRemote           string
	compareProviders        string
	compareJSON             bool
	compareVerbose          bool
)

func init() {
	rootCmd.AddCommand(compareCmd)

	compareCmd.Flags().StringVarP(&compareConfigPath, "config", "c", "preflight.yaml", "First config file")
	compareCmd.Flags().StringVarP(&compareSecondConfigPath, "config2", "C", "", "Second config file (for file comparison)")
	compareCmd.Flags().StringVar(&compareRemote, "remote", "", "Remote config URL (github:user/repo)")
	compareCmd.Flags().StringVar(&compareProviders, "providers", "", "Filter by providers (comma-separated)")
	compareCmd.Flags().BoolVar(&compareJSON, "json", false, "Output as JSON")
	compareCmd.Flags().BoolVarP(&compareVerbose, "verbose", "v", false, "Show detailed differences")
}

func runCompare(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	preflight := app.New(os.Stdout)

	var sourceTarget, destTarget string

	// Determine comparison mode
	switch {
	case len(args) == 2:
		// Compare two targets
		sourceTarget = args[0]
		destTarget = args[1]
	case len(args) == 1 && compareSecondConfigPath != "":
		// Compare same target across two files
		sourceTarget = args[0]
		destTarget = args[0]
	case compareSecondConfigPath != "":
		// Compare default targets across two files
		sourceTarget = "default"
		destTarget = "default"
	default:
		return fmt.Errorf("usage: preflight compare <source-target> <dest-target> or use --config2")
	}

	// Load source configuration
	sourceConfig, err := preflight.LoadMergedConfig(ctx, compareConfigPath, sourceTarget)
	if err != nil {
		return fmt.Errorf("failed to load source config: %w", err)
	}

	// Load destination configuration
	destConfigPath := compareConfigPath
	if compareSecondConfigPath != "" {
		destConfigPath = compareSecondConfigPath
	}

	destConfig, err := preflight.LoadMergedConfig(ctx, destConfigPath, destTarget)
	if err != nil {
		return fmt.Errorf("failed to load destination config: %w", err)
	}

	// Filter by providers if specified
	var providerFilter []string
	if compareProviders != "" {
		providerFilter = strings.Split(compareProviders, ",")
	}

	// Compare configurations
	diffs := compareConfigs(sourceConfig, destConfig, providerFilter)

	// Output results
	if compareJSON {
		return outputCompareJSON(diffs)
	}

	outputCompareText(sourceTarget, destTarget, diffs)
	return nil
}

type configDiff struct {
	Provider string
	Key      string
	Type     string // "added", "removed", "changed"
	Source   interface{}
	Dest     interface{}
}

func compareConfigs(source, dest map[string]interface{}, providerFilter []string) []configDiff {
	var diffs []configDiff

	// Check all keys in source
	for provider, sourceVal := range source {
		if len(providerFilter) > 0 && !containsProvider(providerFilter, provider) {
			continue
		}

		destVal, exists := dest[provider]
		if !exists {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      "",
				Type:     "removed",
				Source:   sourceVal,
				Dest:     nil,
			})
			continue
		}

		// Compare provider contents
		providerDiffs := compareProviderConfig(provider, sourceVal, destVal)
		diffs = append(diffs, providerDiffs...)
	}

	// Check for keys only in dest
	for provider, destVal := range dest {
		if len(providerFilter) > 0 && !containsProvider(providerFilter, provider) {
			continue
		}

		if _, exists := source[provider]; !exists {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      "",
				Type:     "added",
				Source:   nil,
				Dest:     destVal,
			})
		}
	}

	return diffs
}

func compareProviderConfig(provider string, source, dest interface{}) []configDiff {
	var diffs []configDiff

	sourceMap, sourceOK := source.(map[string]interface{})
	destMap, destOK := dest.(map[string]interface{})

	if !sourceOK || !destOK {
		// Not maps, compare directly
		if !equalValues(source, dest) {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      "",
				Type:     "changed",
				Source:   source,
				Dest:     dest,
			})
		}
		return diffs
	}

	// Compare map keys
	for key, sourceVal := range sourceMap {
		destVal, exists := destMap[key]
		if !exists {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      key,
				Type:     "removed",
				Source:   sourceVal,
				Dest:     nil,
			})
			continue
		}

		if !equalValues(sourceVal, destVal) {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      key,
				Type:     "changed",
				Source:   sourceVal,
				Dest:     destVal,
			})
		}
	}

	for key, destVal := range destMap {
		if _, exists := sourceMap[key]; !exists {
			diffs = append(diffs, configDiff{
				Provider: provider,
				Key:      key,
				Type:     "added",
				Source:   nil,
				Dest:     destVal,
			})
		}
	}

	return diffs
}

func equalValues(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func containsProvider(providers []string, provider string) bool {
	for _, p := range providers {
		if strings.TrimSpace(p) == provider {
			return true
		}
	}
	return false
}

func outputCompareText(source, dest string, diffs []configDiff) {
	if len(diffs) == 0 {
		fmt.Printf("No differences between %s and %s\n", source, dest)
		return
	}

	fmt.Printf("Comparing %s → %s\n\n", source, dest)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CHANGE\tPROVIDER\tKEY\tDETAILS")

	for _, d := range diffs {
		var symbol, details string
		switch d.Type {
		case "added":
			symbol = "+ added"
			details = formatValue(d.Dest)
		case "removed":
			symbol = "- removed"
			details = formatValue(d.Source)
		case "changed":
			symbol = "~ changed"
			details = fmt.Sprintf("%v → %v", formatValue(d.Source), formatValue(d.Dest))
		}

		key := d.Key
		if key == "" {
			key = "(entire section)"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", symbol, d.Provider, key, truncate(details, 50))
	}

	_ = w.Flush()

	fmt.Printf("\nTotal: %d difference(s)\n", len(diffs))
}

func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	switch val := v.(type) {
	case []interface{}:
		if len(val) <= 3 {
			return fmt.Sprintf("%v", val)
		}
		return fmt.Sprintf("[%d items]", len(val))
	case map[string]interface{}:
		return fmt.Sprintf("{%d keys}", len(val))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func outputCompareJSON(diffs []configDiff) error {
	type jsonDiff struct {
		Provider string      `json:"provider"`
		Key      string      `json:"key,omitempty"`
		Type     string      `json:"type"`
		Source   interface{} `json:"source,omitempty"`
		Dest     interface{} `json:"dest,omitempty"`
	}

	output := make([]jsonDiff, len(diffs))
	for i, d := range diffs {
		output[i] = jsonDiff(d) //nolint:staticcheck // S1016: intentional field-by-field copy for type conversion
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
