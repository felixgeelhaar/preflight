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
	"github.com/felixgeelhaar/preflight/internal/domain/catalog/tools"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [layers...]",
	Short: "AI-powered layer and tool analysis with recommendations",
	Long: `Analyze Preflight configuration layers and provide intelligent recommendations.

This command uses AI to review your layers and suggest improvements such as:
  - Misplaced packages that belong in a different layer
  - Duplicate or overlapping tools
  - Missing packages common to the layer's domain
  - Deprecated or EOL packages
  - Best practices for layer organization

Tool Analysis (--tools):
  Detect tool redundancy, deprecation, and consolidation opportunities:
  - Deprecated tools (golint → golangci-lint)
  - Redundant tools (grype + trivy → keep trivy)
  - Consolidation opportunities ([grype, syft, gitleaks] → trivy)

Examples:
  preflight analyze                       # Analyze all layers
  preflight analyze layers/dev-go.yaml    # Analyze specific layer
  preflight analyze --recommend           # Get detailed recommendations
  preflight analyze --tools               # Analyze tools for redundancy
  preflight analyze --tools --ai          # AI-enhanced tool analysis
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
	analyzeTools     bool
	analyzeAI        bool
	analyzeFix       bool
)

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().BoolVar(&analyzeRecommend, "recommend", false, "Include detailed recommendations")
	analyzeCmd.Flags().BoolVar(&analyzeJSON, "json", false, "Output results as JSON")
	analyzeCmd.Flags().BoolVarP(&analyzeQuiet, "quiet", "q", false, "Only show summary")
	analyzeCmd.Flags().BoolVar(&analyzeNoAI, "no-ai", false, "Basic analysis without AI")
	analyzeCmd.Flags().BoolVar(&analyzeTools, "tools", false, "Analyze tools for redundancy, deprecation, and consolidation")
	analyzeCmd.Flags().BoolVar(&analyzeAI, "ai", false, "Enable AI-enhanced analysis (for --tools mode)")
	analyzeCmd.Flags().BoolVar(&analyzeFix, "fix", false, "Generate fix suggestions (for --tools mode)")
}

func runAnalyze(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// If --tools flag is set, run tool analysis instead
	if analyzeTools {
		return runToolAnalysis(ctx, args)
	}

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

		// #nosec G304 -- layer path is validated by ValidateLayerPath.
		data, err := readLayerFile(path)
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

// runToolAnalysis runs tool analysis for redundancy, deprecation, and consolidation.
func runToolAnalysis(ctx context.Context, args []string) error {
	// Load the knowledge base
	kb, err := tools.LoadKnowledgeBase()
	if err != nil {
		if analyzeJSON {
			outputToolAnalysisJSON(nil, fmt.Errorf("failed to load knowledge base: %w", err))
		}
		return fmt.Errorf("failed to load knowledge base: %w", err)
	}

	// Extract tools to analyze
	var toolNames []string
	if len(args) > 0 {
		// If args provided, treat them as tool names
		toolNames = args
	} else {
		// Otherwise, extract tools from layers
		layerPaths, err := findLayerFiles()
		if err != nil {
			if analyzeJSON {
				outputToolAnalysisJSON(nil, fmt.Errorf("failed to find layer files: %w", err))
			}
			return fmt.Errorf("failed to find layer files: %w", err)
		}

		if len(layerPaths) == 0 {
			err := fmt.Errorf("no layers found to analyze")
			if analyzeJSON {
				outputToolAnalysisJSON(nil, err)
			} else {
				fmt.Println("No layer files found.")
				fmt.Println("Provide tool names as arguments or create layers in the 'layers/' directory.")
			}
			return err
		}

		toolNames, err = extractAllTools(layerPaths)
		if err != nil {
			if analyzeJSON {
				outputToolAnalysisJSON(nil, fmt.Errorf("failed to extract tools: %w", err))
			}
			return fmt.Errorf("failed to extract tools: %w", err)
		}
	}

	if len(toolNames) == 0 {
		if analyzeJSON {
			outputToolAnalysisJSON(&security.ToolAnalysisResult{
				Findings:      []security.ToolFinding{},
				ToolsAnalyzed: 0,
			}, nil)
		} else {
			fmt.Println("No tools found to analyze.")
		}
		return nil
	}

	// Create analyzer and run analysis
	analyzer := security.NewToolAnalyzer(kb)
	result, err := analyzer.Analyze(ctx, toolNames)
	if err != nil {
		if analyzeJSON {
			outputToolAnalysisJSON(nil, fmt.Errorf("analysis failed: %w", err))
		}
		return fmt.Errorf("analysis failed: %w", err)
	}

	// If AI enhancement requested, add AI insights
	if analyzeAI && !noAI {
		aiProvider := detectAIProvider()
		if aiProvider != nil {
			result = enhanceWithAI(ctx, result, toolNames, aiProvider)
		} else if !analyzeJSON {
			fmt.Println("No AI provider configured. Running knowledge-base analysis only.")
			fmt.Println("Set ANTHROPIC_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY for AI insights.")
			fmt.Println()
		}
	}

	// Output results
	if analyzeJSON {
		outputToolAnalysisJSON(result, nil)
	} else {
		outputToolAnalysisText(result, toolNames)
	}

	return nil
}

// extractAllTools extracts all tool names from layer files.
func extractAllTools(layerPaths []string) ([]string, error) {
	toolSet := make(map[string]bool)

	for _, path := range layerPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Extract from packages.brew.formulae
		if pkgs, ok := raw["packages"].(map[string]interface{}); ok {
			if brew, ok := pkgs["brew"].(map[string]interface{}); ok {
				if formulae, ok := brew["formulae"].([]interface{}); ok {
					for _, f := range formulae {
						if name, ok := f.(string); ok {
							toolSet[name] = true
						}
					}
				}
			}
		}

		// Extract from runtime tools (mise/asdf)
		if runtime, ok := raw["runtime"].(map[string]interface{}); ok {
			if tools, ok := runtime["tools"].(map[string]interface{}); ok {
				for tool := range tools {
					toolSet[tool] = true
				}
			}
		}

		// Extract from shell.plugins
		if shell, ok := raw["shell"].(map[string]interface{}); ok {
			if plugins, ok := shell["plugins"].([]interface{}); ok {
				for _, p := range plugins {
					if name, ok := p.(string); ok {
						toolSet[name] = true
					}
				}
			}
		}
	}

	// Convert to sorted slice
	result := make([]string, 0, len(toolSet))
	for tool := range toolSet {
		result = append(result, tool)
	}
	// Sort for consistent output
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// readLayerFile loads a layer file after path validation.
func readLayerFile(path string) ([]byte, error) {
	// #nosec G304 -- layer paths are validated before being passed to this helper.
	return os.ReadFile(path)
}

// outputToolAnalysisJSON outputs tool analysis results as JSON.
func outputToolAnalysisJSON(result *security.ToolAnalysisResult, err error) {
	output := struct {
		Findings       []security.ToolFinding `json:"findings,omitempty"`
		ToolsAnalyzed  int                    `json:"tools_analyzed"`
		IssuesFound    int                    `json:"issues_found"`
		Consolidations int                    `json:"consolidations"`
		Error          string                 `json:"error,omitempty"`
	}{}

	if err != nil {
		output.Error = err.Error()
	} else if result != nil {
		output.Findings = result.Findings
		output.ToolsAnalyzed = result.ToolsAnalyzed
		output.IssuesFound = result.IssuesFound
		output.Consolidations = result.Consolidations
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(output); encErr != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", encErr)
	}
}

// outputToolAnalysisText outputs tool analysis results as formatted text.
func outputToolAnalysisText(result *security.ToolAnalysisResult, _ []string) {
	fmt.Println("Tool Configuration Analysis")
	fmt.Println(strings.Repeat("═", 50))

	if result.ToolsAnalyzed == 0 {
		fmt.Println("No tools analyzed.")
		return
	}

	// Group findings by type
	deprecations := filterFindingsByType(result.Findings, security.FindingDeprecated)
	redundancies := filterFindingsByType(result.Findings, security.FindingRedundancy)
	consolidations := filterFindingsByType(result.Findings, security.FindingConsolidation)

	// Print deprecation warnings
	if len(deprecations) > 0 {
		fmt.Println()
		fmt.Printf("⚠️  Deprecation Warnings (%d)\n", len(deprecations))
		fmt.Println(strings.Repeat("─", 30))
		for _, f := range deprecations {
			fmt.Printf("  ! %s\n", f.Message)
			if f.Suggestion != "" {
				fmt.Printf("    → %s\n", f.Suggestion)
			}
			if f.Docs != "" {
				fmt.Printf("    📖 %s\n", f.Docs)
			}
		}
	}

	// Print redundancy issues
	if len(redundancies) > 0 {
		fmt.Println()
		fmt.Printf("🔄 Redundancy Issues (%d)\n", len(redundancies))
		fmt.Println(strings.Repeat("─", 30))
		for _, f := range redundancies {
			fmt.Printf("  ! %s\n", f.Message)
			if f.Suggestion != "" {
				fmt.Printf("    → %s\n", f.Suggestion)
			}
		}
	}

	// Print consolidation opportunities
	if len(consolidations) > 0 {
		fmt.Println()
		fmt.Printf("📦 Consolidation Opportunities (%d)\n", len(consolidations))
		fmt.Println(strings.Repeat("─", 30))
		for _, f := range consolidations {
			fmt.Printf("  ℹ %s\n", f.Message)
			if f.Suggestion != "" {
				fmt.Printf("    → %s\n", f.Suggestion)
			}
			if f.Docs != "" {
				fmt.Printf("    📖 %s\n", f.Docs)
			}
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Summary: %d tools analyzed", result.ToolsAnalyzed)
	if result.IssuesFound > 0 {
		fmt.Printf(", %d issues found", result.IssuesFound)
	}
	if result.Consolidations > 0 {
		fmt.Printf(", %d consolidation opportunities", result.Consolidations)
	}
	fmt.Println()

	// If no issues found
	if len(result.Findings) == 0 {
		fmt.Println()
		fmt.Println("✅ No issues found. Your tool configuration looks clean!")
	}
}

// filterFindingsByType returns findings of a specific type.
func filterFindingsByType(findings []security.ToolFinding, findingType security.FindingType) []security.ToolFinding {
	result := make([]security.ToolFinding, 0)
	for _, f := range findings {
		if f.Type == findingType {
			result = append(result, f)
		}
	}
	return result
}

// enhanceWithAI adds AI-enhanced insights to the analysis result.
func enhanceWithAI(ctx context.Context, result *security.ToolAnalysisResult, toolNames []string, aiProvider advisor.AIProvider) *security.ToolAnalysisResult {
	// Build a prompt for AI enhancement
	systemPrompt := "You are a development tools expert. Analyze tool configurations and provide insights about redundancy, deprecation, and optimization opportunities."
	userPrompt := buildToolAnalysisPrompt(result, toolNames)
	prompt := advisor.NewPrompt(systemPrompt, userPrompt)

	response, err := aiProvider.Complete(ctx, prompt)
	if err != nil {
		// Log error but continue with basic results
		if !analyzeQuiet && !analyzeJSON {
			fmt.Fprintf(os.Stderr, "Warning: AI enhancement failed: %v\n", err)
		}
		return result
	}

	// Parse AI response and add additional insights
	aiInsights := parseAIToolInsights(response.Content())
	if aiInsights != nil {
		// Append AI-generated findings
		result.Findings = append(result.Findings, aiInsights...)
	}

	return result
}

// buildToolAnalysisPrompt builds the prompt for AI tool analysis.
func buildToolAnalysisPrompt(result *security.ToolAnalysisResult, toolNames []string) string {
	var sb strings.Builder
	sb.WriteString("Analyze the following development tools for additional insights:\n\n")
	sb.WriteString("Tools: ")
	sb.WriteString(strings.Join(toolNames, ", "))
	sb.WriteString("\n\n")

	if len(result.Findings) > 0 {
		sb.WriteString("Existing findings:\n")
		for _, f := range result.Findings {
			fmt.Fprintf(&sb, "- %s: %s\n", f.Type, f.Message)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Please provide additional insights in JSON format:
{
  "insights": [
    {
      "type": "recommendation",
      "severity": "info",
      "tools": ["tool1"],
      "message": "insight message",
      "suggestion": "what to do"
    }
  ]
}

Focus on:
1. Security concerns with specific tool versions
2. Performance improvements from tool alternatives
3. Modern replacements for legacy tools
4. Best practices for tool combinations`)

	return sb.String()
}

// parseAIToolInsights parses AI response into tool findings.
func parseAIToolInsights(content string) []security.ToolFinding {
	// Try to extract JSON from response
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	jsonContent := content[start : end+1]

	var response struct {
		Insights []struct {
			Type       string   `json:"type"`
			Severity   string   `json:"severity"`
			Tools      []string `json:"tools"`
			Message    string   `json:"message"`
			Suggestion string   `json:"suggestion"`
		} `json:"insights"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return nil
	}

	findings := make([]security.ToolFinding, 0, len(response.Insights))
	for _, insight := range response.Insights {
		severity := security.SeverityInfo
		switch insight.Severity {
		case "warning":
			severity = security.SeverityWarning
		case "error":
			severity = security.SeverityError
		}

		findings = append(findings, security.ToolFinding{
			Type:       security.FindingType(insight.Type),
			Severity:   severity,
			Tools:      insight.Tools,
			Message:    insight.Message,
			Suggestion: insight.Suggestion,
		})
	}

	return findings
}
