package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MaxJSONResponseSize limits the size of JSON responses to prevent DoS attacks.
// 1MB should be sufficient for any reasonable analysis response.
const MaxJSONResponseSize = 1 << 20 // 1MB

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
// Note: Error handling is done at the application layer, not in this domain model.
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
	// Check response size to prevent DoS attacks
	if len(response) > MaxJSONResponseSize {
		return nil, fmt.Errorf("response too large: %d bytes (max %d)", len(response), MaxJSONResponseSize)
	}

	jsonStr, err := extractJSON(response)
	if err != nil {
		return nil, err
	}

	var result LayerAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w", err)
	}

	return &result, nil
}

// extractJSON attempts to extract JSON from an AI response using multiple strategies.
func extractJSON(response string) (string, error) {
	// Strategy 1: Look for markdown code block with json/JSON
	jsonStr := extractFromCodeBlock(response)
	if jsonStr != "" && json.Valid([]byte(jsonStr)) {
		return jsonStr, nil
	}

	// Strategy 2: Find balanced braces starting from the first {
	jsonStr = extractBalancedJSON(response)
	if jsonStr != "" && json.Valid([]byte(jsonStr)) {
		return jsonStr, nil
	}

	// Strategy 3: Fallback to simple first-last brace extraction
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart != -1 && jsonEnd > jsonStart {
		jsonStr = response[jsonStart : jsonEnd+1]
		if json.Valid([]byte(jsonStr)) {
			return jsonStr, nil
		}
	}

	return "", fmt.Errorf("no valid JSON found in response (length: %d chars)", len(response))
}

// extractFromCodeBlock extracts JSON from markdown code blocks.
func extractFromCodeBlock(response string) string {
	// Try ```json first
	markers := []string{"```json", "```JSON", "```"}
	for _, marker := range markers {
		start := strings.Index(response, marker)
		if start == -1 {
			continue
		}
		// Find content after the marker
		contentStart := start + len(marker)
		// Skip any newline after marker
		if contentStart < len(response) && response[contentStart] == '\n' {
			contentStart++
		}

		// Find closing ```
		end := strings.Index(response[contentStart:], "```")
		if end == -1 {
			continue
		}

		content := strings.TrimSpace(response[contentStart : contentStart+end])
		if strings.HasPrefix(content, "{") {
			return content
		}
	}
	return ""
}

// extractBalancedJSON finds JSON by matching balanced braces.
func extractBalancedJSON(response string) string {
	start := strings.Index(response, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(response); i++ {
		c := response[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return response[start : i+1]
			}
		}
	}

	return ""
}
