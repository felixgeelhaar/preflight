package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CaptureAnalysisSystemPrompt is the system prompt for analyzing captured data.
const CaptureAnalysisSystemPrompt = `You are a dotfile configuration expert that helps users organize their development environment.

Your role is to:
1. Analyze captured configuration files and tools
2. Identify patterns and best practices
3. Suggest improvements and organization strategies
4. Recommend preflight configuration layers

When analyzing captured data, focus on:
- Tool usage patterns and complementary tools
- Configuration file quality and modernization opportunities
- Work/personal separation opportunities
- Cross-platform compatibility concerns
- Security best practices (never suggest exposing secrets)

Respond with structured JSON containing your analysis and recommendations.`

// CapturedItem represents a single captured configuration item.
type CapturedItem struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // file, directory, tool
	Size        int64  `json:"size,omitempty"`
	Description string `json:"description,omitempty"`
}

// CaptureAnalysisRequest contains data for AI analysis.
type CaptureAnalysisRequest struct {
	Items       []CapturedItem   `json:"items"`
	UserProfile InterviewProfile `json:"profile,omitempty"`
	Patterns    []PatternMatch   `json:"patterns,omitempty"`
}

// PatternMatch represents a detected pattern from discover analysis.
type PatternMatch struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

// CaptureAnalysisResponse contains AI's analysis of captured data.
type CaptureAnalysisResponse struct {
	Summary         string                  `json:"summary"`
	Categories      map[string][]string     `json:"categories"` // category -> items
	Recommendations []CaptureRecommendation `json:"recommendations"`
	SuggestedLayers []string                `json:"suggested_layers"`
	Warnings        []string                `json:"warnings,omitempty"`
	Score           QualityScore            `json:"quality_score"`
}

// CaptureRecommendation is a single AI recommendation for captured data.
type CaptureRecommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // high, medium, low
	Category    string `json:"category"` // organization, security, modernization, optimization
	Action      string `json:"action"`   // what to do
	Why         string `json:"why"`      // explanation
}

// QualityScore rates the overall quality of the captured configuration.
type QualityScore struct {
	Overall      float64 `json:"overall"`      // 0.0-1.0
	Organization float64 `json:"organization"` // How well organized
	Security     float64 `json:"security"`     // Security practices
	Modernness   float64 `json:"modernness"`   // Up-to-date tools
	Portability  float64 `json:"portability"`  // Cross-platform compatibility
}

// BuildCaptureAnalysisPrompt creates a prompt to analyze captured configuration.
func BuildCaptureAnalysisPrompt(req CaptureAnalysisRequest) Prompt {
	var parts []string

	parts = append(parts, "Analyze the following captured configuration and provide recommendations:")
	parts = append(parts, "")

	// Add captured items
	if len(req.Items) > 0 {
		parts = append(parts, "## Captured Items")
		parts = append(parts, "")

		// Group by type
		files := make([]CapturedItem, 0)
		tools := make([]CapturedItem, 0)

		for _, item := range req.Items {
			switch item.Type {
			case "tool":
				tools = append(tools, item)
			default:
				files = append(files, item)
			}
		}

		if len(files) > 0 {
			parts = append(parts, "### Configuration Files")
			for _, f := range files {
				line := fmt.Sprintf("- %s", f.Path)
				if f.Description != "" {
					line += fmt.Sprintf(" (%s)", f.Description)
				}
				parts = append(parts, line)
			}
			parts = append(parts, "")
		}

		if len(tools) > 0 {
			parts = append(parts, "### Installed Tools")
			for _, t := range tools {
				line := fmt.Sprintf("- %s", t.Path)
				if t.Description != "" {
					line += fmt.Sprintf(": %s", t.Description)
				}
				parts = append(parts, line)
			}
			parts = append(parts, "")
		}
	}

	// Add detected patterns
	if len(req.Patterns) > 0 {
		parts = append(parts, "## Detected Patterns")
		parts = append(parts, "")
		for _, p := range req.Patterns {
			parts = append(parts, fmt.Sprintf("- %s (%s): %.0f%% confidence", p.Name, p.Type, p.Confidence*100))
		}
		parts = append(parts, "")
	}

	// Add user profile context if available
	if req.UserProfile.ExperienceLevel != "" {
		parts = append(parts, "## User Context")
		parts = append(parts, "")
		parts = append(parts, fmt.Sprintf("Experience: %s", req.UserProfile.ExperienceLevel))
		if len(req.UserProfile.Languages) > 0 {
			parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(req.UserProfile.Languages, ", ")))
		}
		if req.UserProfile.WorkContext != "" && req.UserProfile.WorkContext != WorkContextUnknown {
			parts = append(parts, fmt.Sprintf("Context: %s", req.UserProfile.WorkContext))
		}
		if req.UserProfile.DeviceType != "" && req.UserProfile.DeviceType != DeviceTypeUnknown {
			parts = append(parts, fmt.Sprintf("Device: %s", req.UserProfile.DeviceType))
		}
		parts = append(parts, "")
	}

	parts = append(parts, "## Provide Analysis As JSON")
	parts = append(parts, "")
	parts = append(parts, "Respond with a JSON object containing:")
	parts = append(parts, "- summary: Brief overview of the configuration")
	parts = append(parts, "- categories: Group items by function (shell, editor, git, etc.)")
	parts = append(parts, "- recommendations: List of prioritized improvements")
	parts = append(parts, "- suggested_layers: Recommended preflight layers")
	parts = append(parts, "- warnings: Any security or compatibility concerns")
	parts = append(parts, "- quality_score: Rate organization, security, modernness, portability (0.0-1.0)")

	userPrompt := strings.Join(parts, "\n")

	return NewPrompt(CaptureAnalysisSystemPrompt, userPrompt).
		WithMaxTokens(1024).
		WithTemperature(0.3)
}

// ParseCaptureAnalysis parses AI response into a CaptureAnalysisResponse.
func ParseCaptureAnalysis(response string) (*CaptureAnalysisResponse, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var analysis CaptureAnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse capture analysis JSON: %w", err)
	}

	return &analysis, nil
}

// ContextualRecommendationSystemPrompt is the system prompt for contextual recommendations.
const ContextualRecommendationSystemPrompt = `You are a development environment optimization expert.

Given a user's detected tools and patterns, provide personalized recommendations that:
1. Build on their existing workflow
2. Fill gaps in their toolchain
3. Suggest complementary tools they might not know about
4. Respect their experience level (don't overwhelm beginners)

Focus on practical, immediately useful suggestions. Explain "why" for each recommendation.

Respond with structured JSON containing your recommendations.`

// ContextualRecommendationRequest contains context for personalized recommendations.
type ContextualRecommendationRequest struct {
	DetectedPatterns []PatternMatch   `json:"detected_patterns"`
	ExistingTools    []string         `json:"existing_tools"`
	Profile          InterviewProfile `json:"profile"`
	Goals            []string         `json:"goals,omitempty"`
}

// ContextualRecommendation is a single contextual recommendation.
type ContextualRecommendation struct {
	Tool        string   `json:"tool"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Why         string   `json:"why"`
	Links       []string `json:"links,omitempty"`
	ConfigHint  string   `json:"config_hint,omitempty"`
	Priority    int      `json:"priority"` // 1-5, 1 being highest
}

// ContextualRecommendationResponse contains contextual AI recommendations.
type ContextualRecommendationResponse struct {
	Summary         string                     `json:"summary"`
	Recommendations []ContextualRecommendation `json:"recommendations"`
	Gaps            []string                   `json:"gaps,omitempty"`      // Missing tools for their workflow
	Synergies       []string                   `json:"synergies,omitempty"` // Good tool combinations detected
}

// BuildContextualRecommendationPrompt creates a prompt for personalized recommendations.
func BuildContextualRecommendationPrompt(req ContextualRecommendationRequest) Prompt {
	var parts []string

	parts = append(parts, "Based on the following context, provide personalized tool recommendations:")
	parts = append(parts, "")

	// User profile
	if req.Profile.ExperienceLevel != "" {
		parts = append(parts, fmt.Sprintf("## Experience Level: %s", req.Profile.ExperienceLevel))
		parts = append(parts, "")
	}

	// Detected patterns
	if len(req.DetectedPatterns) > 0 {
		parts = append(parts, "## Detected Configuration Patterns")
		for _, p := range req.DetectedPatterns {
			parts = append(parts, fmt.Sprintf("- %s (%s)", p.Name, p.Type))
		}
		parts = append(parts, "")
	}

	// Existing tools
	if len(req.ExistingTools) > 0 {
		parts = append(parts, "## Existing Tools")
		parts = append(parts, strings.Join(req.ExistingTools, ", "))
		parts = append(parts, "")
	}

	// Goals
	if len(req.Goals) > 0 {
		parts = append(parts, "## User Goals")
		for _, g := range req.Goals {
			parts = append(parts, fmt.Sprintf("- %s", g))
		}
		parts = append(parts, "")
	}

	// Languages context
	if len(req.Profile.Languages) > 0 {
		parts = append(parts, "## Programming Languages")
		parts = append(parts, strings.Join(req.Profile.Languages, ", "))
		parts = append(parts, "")
	}

	parts = append(parts, "## Response Format")
	parts = append(parts, "")
	parts = append(parts, "Provide 3-5 prioritized recommendations as JSON with:")
	parts = append(parts, "- summary: Brief overview of your analysis")
	parts = append(parts, "- recommendations: List of tool/config recommendations with tool, category, title, description, why, priority (1-5)")
	parts = append(parts, "- gaps: Any important tools missing for their workflow")
	parts = append(parts, "- synergies: Good tool combinations already in use")

	userPrompt := strings.Join(parts, "\n")

	return NewPrompt(ContextualRecommendationSystemPrompt, userPrompt).
		WithMaxTokens(1024).
		WithTemperature(0.4)
}

// ParseContextualRecommendation parses AI response into recommendations.
func ParseContextualRecommendation(response string) (*ContextualRecommendationResponse, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var rec ContextualRecommendationResponse
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		return nil, fmt.Errorf("failed to parse contextual recommendation JSON: %w", err)
	}

	return &rec, nil
}
