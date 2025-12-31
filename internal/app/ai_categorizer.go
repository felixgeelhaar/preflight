package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
)

// ProviderCategorizer uses an AIProvider to categorize packages.
type ProviderCategorizer struct {
	provider advisor.AIProvider
}

// NewProviderCategorizer creates a new AI categorizer using an AIProvider.
func NewProviderCategorizer(provider advisor.AIProvider) *ProviderCategorizer {
	return &ProviderCategorizer{provider: provider}
}

// Categorize uses AI to categorize packages into layers.
func (c *ProviderCategorizer) Categorize(ctx context.Context, req AICategorizationRequest) (*AICategorizationResult, error) {
	if len(req.Items) == 0 {
		return &AICategorizationResult{
			Categorizations: make(map[string]string),
			Reasoning:       make(map[string]string),
		}, nil
	}

	// Check if provider is available
	if !c.provider.Available() {
		return nil, fmt.Errorf("AI provider %s is not available", c.provider.Name())
	}

	// Build the prompt
	systemPrompt := "You are a package categorization assistant. Respond only with valid JSON."
	userPrompt := buildCategorizationPrompt(req)

	prompt := advisor.NewPrompt(systemPrompt, userPrompt).
		WithMaxTokens(2048).
		WithTemperature(0.3) // Lower temperature for more consistent categorization

	// Call the AI provider
	response, err := c.provider.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI completion failed: %w", err)
	}

	// Parse the response
	return parseCategorizationResponse(response.Content(), req.Items)
}

// buildCategorizationPrompt creates a prompt for package categorization.
func buildCategorizationPrompt(req AICategorizationRequest) string {
	var sb strings.Builder

	sb.WriteString("Your task is to categorize software packages into appropriate layers.\n\n")

	// Describe the strategy
	switch req.Strategy {
	case SplitByLanguage:
		sb.WriteString("Categorization Strategy: BY LANGUAGE\n")
		sb.WriteString("Group packages by their primary programming language or ecosystem.\n\n")
	case SplitByStack:
		sb.WriteString("Categorization Strategy: BY STACK\n")
		sb.WriteString("Group packages by their tech stack role (frontend, backend, devops, data, security, etc.).\n\n")
	default:
		sb.WriteString("Categorization Strategy: BY CATEGORY\n")
		sb.WriteString("Group packages by their functional category (development tools, security, containers, etc.).\n\n")
	}

	// List available layers
	sb.WriteString("Available layers:\n")
	for _, layer := range req.AvailableLayers {
		sb.WriteString(fmt.Sprintf("- %s\n", layer))
	}
	sb.WriteString("- NEW (suggest a new layer name if none fit)\n\n")

	// List packages to categorize
	sb.WriteString("Packages to categorize:\n")
	for _, item := range req.Items {
		sb.WriteString(fmt.Sprintf("- %s (provider: %s)\n", item.Name, item.Provider))
	}

	sb.WriteString("\nRespond with JSON in this exact format:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"categorizations\": {\n")
	sb.WriteString("    \"package-name\": \"layer-name\",\n")
	sb.WriteString("    ...\n")
	sb.WriteString("  },\n")
	sb.WriteString("  \"reasoning\": {\n")
	sb.WriteString("    \"package-name\": \"brief reason\",\n")
	sb.WriteString("    ...\n")
	sb.WriteString("  },\n")
	sb.WriteString("  \"new_layers\": {\n")
	sb.WriteString("    \"new-layer-name\": \"description\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n\n")
	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Use existing layers when appropriate\n")
	sb.WriteString("- For truly unique packages, use 'misc' or suggest a new layer\n")
	sb.WriteString("- Be consistent with naming conventions\n")
	sb.WriteString("- Provide brief, one-line reasoning\n")

	return sb.String()
}

// aiCategorizationResponse is the expected JSON response from AI.
type aiCategorizationResponse struct {
	Categorizations map[string]string `json:"categorizations"`
	Reasoning       map[string]string `json:"reasoning"`
	NewLayers       map[string]string `json:"new_layers,omitempty"`
}

// parseCategorizationResponse parses the AI response into categorization results.
func parseCategorizationResponse(response string, items []CapturedItem) (*AICategorizationResult, error) {
	result := &AICategorizationResult{
		Categorizations: make(map[string]string),
		Reasoning:       make(map[string]string),
	}

	// Try to extract JSON from the response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// If no JSON found, return empty result (fallback to misc)
		return result, nil
	}

	var parsed aiCategorizationResponse
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		// If parsing fails, return empty result (fallback to misc)
		return result, nil //nolint:nilerr // Intentional: graceful degradation when AI returns invalid JSON
	}

	// Build a map of item names for validation
	itemNames := make(map[string]bool)
	for _, item := range items {
		itemNames[item.Name] = true
		itemNames[strings.ToLower(item.Name)] = true
	}

	// Copy valid categorizations
	for pkg, layer := range parsed.Categorizations {
		// Validate the package exists (case-insensitive)
		if itemNames[pkg] || itemNames[strings.ToLower(pkg)] {
			result.Categorizations[pkg] = strings.ToLower(layer)
		}
	}

	// Copy reasoning
	for pkg, reason := range parsed.Reasoning {
		if itemNames[pkg] || itemNames[strings.ToLower(pkg)] {
			result.Reasoning[pkg] = reason
		}
	}

	return result, nil
}

// extractJSON extracts JSON from a response that may contain markdown or other text.
func extractJSON(response string) string {
	// Look for JSON block in markdown
	if start := strings.Index(response, "```json"); start != -1 {
		start += 7 // skip "```json"
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}

	// Look for JSON block without language specifier
	if start := strings.Index(response, "```"); start != -1 {
		start += 3 // skip "```"
		if end := strings.Index(response[start:], "```"); end != -1 {
			content := strings.TrimSpace(response[start : start+end])
			if strings.HasPrefix(content, "{") {
				return content
			}
		}
	}

	// Look for raw JSON (starts with {)
	if start := strings.Index(response, "{"); start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(response); i++ {
			switch response[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return response[start : i+1]
				}
			}
		}
	}

	return ""
}
