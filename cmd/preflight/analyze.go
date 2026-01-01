package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [layers...]",
	Short: "AI-powered layer analysis with recommendations",
	Long: `Analyze Preflight configuration layers and provide intelligent recommendations.

This command uses AI to review your layers and suggest improvements such as:
  - Misplaced packages that belong in a different layer
  - Duplicate or overlapping tools
  - Missing packages common to the layer's domain
  - Deprecated or EOL packages
  - Best practices for layer organization

Examples:
  preflight analyze                       # Analyze all layers
  preflight analyze layers/dev-go.yaml    # Analyze specific layer
  preflight analyze --recommend           # Get detailed recommendations
  preflight analyze --ai-provider gemini  # Use specific AI provider
  preflight analyze --no-ai               # Basic analysis without AI
  preflight analyze --json                # JSON output for CI`,
	RunE: runAnalyze,
}

// layerAnalyzer is the domain service for layer analysis.
// It encapsulates business logic for analyzing layer quality.
var layerAnalyzer = advisor.NewLayerAnalyzer()

var (
	analyzeRecommend bool
	analyzeJSON      bool
	analyzeQuiet     bool
	analyzeNoAI      bool
)

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().BoolVar(&analyzeRecommend, "recommend", false, "Include detailed recommendations")
	analyzeCmd.Flags().BoolVar(&analyzeJSON, "json", false, "Output results as JSON")
	analyzeCmd.Flags().BoolVarP(&analyzeQuiet, "quiet", "q", false, "Only show summary")
	analyzeCmd.Flags().BoolVar(&analyzeNoAI, "no-ai", false, "Basic analysis without AI")
}

func runAnalyze(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Collect layers to analyze
	var layerPaths []string
	if len(args) > 0 {
		layerPaths = args
	} else {
		// Find all layer files
		var err error
		layerPaths, err = findLayerFiles()
		if err != nil {
			return fmt.Errorf("failed to find layer files: %w", err)
		}
	}

	if len(layerPaths) == 0 {
		err := fmt.Errorf("no layers found")
		if analyzeJSON {
			outputAnalyzeJSON(nil, err)
		} else {
			fmt.Println("No layer files found.")
			fmt.Println("Create layers in the 'layers/' directory or provide paths as arguments.")
		}
		return err // Return error for proper exit code
	}

	// Load layer information
	layers, err := loadLayerInfos(layerPaths)
	if err != nil {
		loadErr := fmt.Errorf("failed to load layers: %w", err)
		if analyzeJSON {
			outputAnalyzeJSON(nil, loadErr)
		}
		return loadErr // Return error for proper exit code
	}

	// Get AI provider if not disabled
	var aiProvider advisor.AIProvider
	if !analyzeNoAI && !noAI {
		aiProvider = detectAIProvider()
		if aiProvider == nil && !analyzeJSON {
			fmt.Println("No AI provider configured. Running basic analysis only.")
			fmt.Println("Set ANTHROPIC_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY for AI recommendations.")
			fmt.Println()
		}
	}

	// Perform analysis
	report := analyzeLayersWithAI(ctx, layers, aiProvider)

	// Output results
	if analyzeJSON {
		outputAnalyzeJSON(report, nil)
	} else {
		outputAnalyzeText(report, analyzeQuiet, analyzeRecommend)
	}

	return nil
}

// findLayerFiles finds all layer YAML files in the standard locations.
func findLayerFiles() ([]string, error) {
	var paths []string

	// Check layers/ directory
	layersDir := "layers"
	if _, err := os.Stat(layersDir); err == nil {
		matches, err := filepath.Glob(filepath.Join(layersDir, "*.yaml"))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)

		// Also check for .yml extension
		ymlMatches, err := filepath.Glob(filepath.Join(layersDir, "*.yml"))
		if err != nil {
			return nil, err
		}
		paths = append(paths, ymlMatches...)
	}

	return paths, nil
}

// validateLayerPath validates a layer file path using the config domain service.
func validateLayerPath(path string) error {
	return config.ValidateLayerPath(path)
}

// loadLayerInfos loads layer information from YAML files.
func loadLayerInfos(paths []string) ([]advisor.LayerInfo, error) {
	layers := make([]advisor.LayerInfo, 0, len(paths))

	for _, path := range paths {
		// Validate path before reading
		if err := validateLayerPath(path); err != nil {
			return nil, err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Parse YAML to extract packages
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		info := advisor.LayerInfo{
			Name:     extractLayerName(path, raw),
			Path:     path,
			Packages: extractPackages(raw),
		}

		// Check for config sections
		if _, ok := raw["git"]; ok {
			info.HasGitConfig = true
		}
		if _, ok := raw["ssh"]; ok {
			info.HasSSHConfig = true
		}
		if _, ok := raw["shell"]; ok {
			info.HasShellConfig = true
		}
		if _, ok := raw["nvim"]; ok {
			info.HasEditorConfig = true
		}
		if _, ok := raw["vscode"]; ok {
			info.HasEditorConfig = true
		}

		layers = append(layers, info)
	}

	return layers, nil
}

// extractLayerName gets the layer name from YAML or filename.
func extractLayerName(path string, raw map[string]interface{}) string {
	if name, ok := raw["name"].(string); ok && name != "" {
		return name
	}
	// Use filename without extension
	base := filepath.Base(path)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
}

// extractPackages extracts all packages from a layer.
func extractPackages(raw map[string]interface{}) []string {
	var packages []string

	// Extract from packages.brew.formulae
	if pkgs, ok := raw["packages"].(map[string]interface{}); ok {
		if brew, ok := pkgs["brew"].(map[string]interface{}); ok {
			if formulae, ok := brew["formulae"].([]interface{}); ok {
				for _, f := range formulae {
					if name, ok := f.(string); ok {
						packages = append(packages, name)
					}
				}
			}
			if casks, ok := brew["casks"].([]interface{}); ok {
				for _, c := range casks {
					if name, ok := c.(string); ok {
						packages = append(packages, name+" (cask)")
					}
				}
			}
		}
	}

	return packages
}

// analyzeLayersWithAI performs analysis using AI if available.
func analyzeLayersWithAI(ctx context.Context, layers []advisor.LayerInfo, aiProvider advisor.AIProvider) *advisor.AnalysisReport {
	report := &advisor.AnalysisReport{
		Layers: make([]advisor.LayerAnalysisResult, 0, len(layers)),
	}

	for _, layer := range layers {
		report.TotalPackages += len(layer.Packages)

		result := advisor.LayerAnalysisResult{
			LayerName:    layer.Name,
			PackageCount: len(layer.Packages),
		}

		// If AI is available, get recommendations
		if aiProvider != nil {
			prompt := advisor.BuildLayerAnalysisPrompt(layer, layers)
			response, err := aiProvider.Complete(ctx, prompt)
			if err != nil {
				// Log error details and continue with basic analysis
				if !analyzeQuiet && !analyzeJSON {
					fmt.Fprintf(os.Stderr, "Warning: AI analysis failed for layer %s: %v\n", layer.Name, err)
				}
				// Fall back to basic analysis when AI fails
				result = layerAnalyzer.AnalyzeBasic(layer)
				result.Summary = fmt.Sprintf("AI unavailable - %s", result.Summary)
			} else {
				// Parse AI response
				aiResult, parseErr := advisor.ParseLayerAnalysisResult(response.Content())
				if parseErr != nil {
					// Log parse error and fall back to basic analysis
					if !analyzeQuiet && !analyzeJSON {
						fmt.Fprintf(os.Stderr, "Warning: Failed to parse AI response for layer %s: %v\n", layer.Name, parseErr)
					}
					result = layerAnalyzer.AnalyzeBasic(layer)
				} else {
					result = *aiResult
					result.LayerName = layer.Name
					result.PackageCount = len(layer.Packages)
				}
			}
		} else {
			// Basic analysis without AI
			result = layerAnalyzer.AnalyzeBasic(layer)
		}

		report.TotalRecommendations += len(result.Recommendations)
		report.Layers = append(report.Layers, result)
	}

	// Check for cross-layer issues using domain service
	report.CrossLayerIssues = layerAnalyzer.FindCrossLayerIssues(layers)

	return report
}

func outputAnalyzeJSON(report *advisor.AnalysisReport, err error) {
	output := struct {
		Layers               []advisor.LayerAnalysisResult `json:"layers,omitempty"`
		TotalPackages        int                           `json:"total_packages,omitempty"`
		TotalRecommendations int                           `json:"total_recommendations,omitempty"`
		CrossLayerIssues     []string                      `json:"cross_layer_issues,omitempty"`
		Error                string                        `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Error = err.Error()
	} else if report != nil {
		output.Layers = report.Layers
		output.TotalPackages = report.TotalPackages
		output.TotalRecommendations = report.TotalRecommendations
		output.CrossLayerIssues = report.CrossLayerIssues
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(output); encErr != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", encErr)
	}
}

func outputAnalyzeText(report *advisor.AnalysisReport, quiet bool, recommend bool) {
	fmt.Println("Layer Analysis Report")
	fmt.Println(strings.Repeat("─", 50))

	if len(report.Layers) == 0 {
		fmt.Println("No layers to analyze.")
		return
	}

	// Print each layer
	for _, layer := range report.Layers {
		statusIcon := tui.FormatStatusIcon(layer.Status)
		fmt.Printf("\n📦 %s (%d packages)\n", layer.LayerName, layer.PackageCount)
		fmt.Printf("  %s %s\n", statusIcon, layer.Summary)

		// Print recommendations if enabled
		if recommend && len(layer.Recommendations) > 0 {
			fmt.Println("  💡 Recommendations:")
			for _, rec := range layer.Recommendations {
				priority := tui.FormatPriorityPrefix(rec.Priority)
				fmt.Printf("    %s %s\n", priority, rec.Message)
				if len(rec.Packages) > 0 && !quiet {
					fmt.Printf("       Packages: %s\n", strings.Join(rec.Packages, ", "))
				}
			}
		}
	}

	// Print cross-layer issues
	if len(report.CrossLayerIssues) > 0 {
		fmt.Println()
		fmt.Println("⚠  Cross-Layer Issues:")
		for _, issue := range report.CrossLayerIssues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Summary: %d layers analyzed, %d recommendations\n",
		len(report.Layers), report.TotalRecommendations)
	if report.TotalPackages > 0 {
		fmt.Printf("Total packages: %d\n", report.TotalPackages)
	}

	// Print detailed table if not quiet
	if !quiet && len(report.Layers) > 1 {
		fmt.Println()
		printLayerSummaryTable(report.Layers)
	}
}

func printLayerSummaryTable(layers []advisor.LayerAnalysisResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "LAYER\tPACKAGES\tSTATUS\tRECOMMENDATIONS")
	_, _ = fmt.Fprintln(w, "─────\t────────\t──────\t───────────────")

	for _, layer := range layers {
		status := layer.Status
		if status == "" {
			status = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%d\n",
			layer.LayerName, layer.PackageCount, status, len(layer.Recommendations))
	}
	_ = w.Flush()
}
