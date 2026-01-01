package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LayerAnalysisSystemPrompt is the system prompt for layer analysis.
const LayerAnalysisSystemPrompt = `You are a Preflight configuration expert that analyzes workstation setup layers.

Your role is to:
1. Identify misplaced packages (packages that belong in a different layer)
2. Find duplicate or overlapping tools across layers
3. Suggest missing packages common to the layer's domain
4. Identify deprecated or EOL packages
5. Recommend best practices for layer organization

Be concise and actionable. Focus on practical improvements.

When providing recommendations, output them as JSON in the following format:
{
  "layer_name": "the layer being analyzed",
  "summary": "One-line summary of the layer",
  "status": "good|warning|needs_attention",
  "recommendations": [
    {
      "type": "misplacement|duplicate|missing|deprecated|best_practice",
      "priority": "high|medium|low",
      "message": "Clear description of the recommendation",
      "packages": ["affected", "packages"],
      "suggested_layer": "target layer for misplacement (optional)"
    }
  ],
  "package_count": 10,
  "well_organized": true
}
`

// LayerInfo represents information about a layer for analysis.
type LayerInfo struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Packages []string `json:"packages"`
	// Additional context
	HasGitConfig    bool `json:"has_git_config,omitempty"`
	HasSSHConfig    bool `json:"has_ssh_config,omitempty"`
	HasShellConfig  bool `json:"has_shell_config,omitempty"`
	HasEditorConfig bool `json:"has_editor_config,omitempty"`
}

// AnalysisRecommendation represents a single recommendation from analysis.
type AnalysisRecommendation struct {
	Type           string   `json:"type"`
	Priority       string   `json:"priority"`
	Message        string   `json:"message"`
	Packages       []string `json:"packages,omitempty"`
	SuggestedLayer string   `json:"suggested_layer,omitempty"`
}

// LayerAnalysisResult represents the analysis result for a single layer.
type LayerAnalysisResult struct {
	LayerName       string                   `json:"layer_name"`
	Summary         string                   `json:"summary"`
	Status          string                   `json:"status"`
	Recommendations []AnalysisRecommendation `json:"recommendations"`
	PackageCount    int                      `json:"package_count"`
	WellOrganized   bool                     `json:"well_organized"`
}

// AnalysisReport represents the complete analysis report.
type AnalysisReport struct {
	Layers               []LayerAnalysisResult `json:"layers"`
	TotalPackages        int                   `json:"total_packages"`
	TotalRecommendations int                   `json:"total_recommendations"`
	CrossLayerIssues     []string              `json:"cross_layer_issues,omitempty"`
}

// BuildLayerAnalysisPrompt creates a prompt for analyzing a single layer.
func BuildLayerAnalysisPrompt(layer LayerInfo, allLayers []LayerInfo) Prompt {
	// Pre-allocate: 5 header lines + packages + other layers info + 8 footer lines
	parts := make([]string, 0, 5+len(layer.Packages)+len(allLayers)+8)

	parts = append(parts, "Analyze this Preflight configuration layer and provide recommendations:")
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("Layer: %s", layer.Name))
	parts = append(parts, fmt.Sprintf("Path: %s", layer.Path))
	parts = append(parts, fmt.Sprintf("Packages (%d):", len(layer.Packages)))

	for _, pkg := range layer.Packages {
		parts = append(parts, fmt.Sprintf("  - %s", pkg))
	}

	// Add context about other layers for cross-reference
	if len(allLayers) > 1 {
		parts = append(parts, "")
		parts = append(parts, "Other layers in this configuration:")
		for _, other := range allLayers {
			if other.Name != layer.Name {
				parts = append(parts, fmt.Sprintf("  - %s (%d packages)", other.Name, len(other.Packages)))
			}
		}
	}

	parts = append(parts, "")
	parts = append(parts, "Consider:")
	parts = append(parts, "- Layer naming conventions (base, dev-*, role.*, identity.*, device.*)")
	parts = append(parts, "- Package grouping by purpose (media tools, security tools, dev tools)")
	parts = append(parts, "- Potential duplicates or alternatives that serve the same purpose")
	parts = append(parts, "- Missing essential packages for the layer's domain")
	parts = append(parts, "- EOL or deprecated packages that should be replaced")
	parts = append(parts, "")
	parts = append(parts, "Respond with a JSON object as specified in the system prompt.")

	userPrompt := strings.Join(parts, "\n")

	return NewPrompt(LayerAnalysisSystemPrompt, userPrompt).
		WithMaxTokens(1024).
		WithTemperature(0.3)
}

// BuildMultiLayerAnalysisPrompt creates a prompt for analyzing multiple layers together.
func BuildMultiLayerAnalysisPrompt(layers []LayerInfo) Prompt {
	// Pre-allocate: estimate 3 lines per layer + header/footer
	totalPackages := 0
	for _, layer := range layers {
		totalPackages += len(layer.Packages)
	}
	parts := make([]string, 0, 3*len(layers)+totalPackages+10)

	parts = append(parts, "Analyze all Preflight configuration layers together and identify cross-layer issues:")
	parts = append(parts, "")

	for _, layer := range layers {
		parts = append(parts, fmt.Sprintf("## %s (%d packages)", layer.Name, len(layer.Packages)))
		if len(layer.Packages) > 0 {
			for _, pkg := range layer.Packages {
				parts = append(parts, fmt.Sprintf("  - %s", pkg))
			}
		}
		parts = append(parts, "")
	}

	parts = append(parts, fmt.Sprintf("Total packages across all layers: %d", totalPackages))
	parts = append(parts, "")
	parts = append(parts, "Look for:")
	parts = append(parts, "1. Duplicate packages across layers")
	parts = append(parts, "2. Packages that should be moved to a different layer")
	parts = append(parts, "3. Missing layers that would better organize the packages")
	parts = append(parts, "4. Overlapping tools (e.g., both grype and trivy for vulnerability scanning)")
	parts = append(parts, "")
	parts = append(parts, "Respond with a JSON object containing an array of issues and suggestions.")

	userPrompt := strings.Join(parts, "\n")

	return NewPrompt(LayerAnalysisSystemPrompt, userPrompt).
		WithMaxTokens(2048).
		WithTemperature(0.3)
}

// ParseLayerAnalysisResult parses AI response into a layer analysis result.
func ParseLayerAnalysisResult(response string) (*LayerAnalysisResult, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var result LayerAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w", err)
	}

	return &result, nil
}
