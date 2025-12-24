package advisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// InitInterviewSystemPrompt is the system prompt for the init interview.
const InitInterviewSystemPrompt = `You are a helpful assistant that helps users configure their development environment using preflight.

Your role is to:
1. Understand the user's development experience level
2. Learn about their primary programming languages and tools
3. Understand their development workflow and preferences
4. Recommend appropriate configuration presets and layers

Be concise and friendly. Ask follow-up questions when needed to better understand their needs.

When providing recommendations, output them as JSON in the following format:
{
  "presets": ["preset1", "preset2"],
  "layers": ["layer1", "layer2"],
  "explanation": "Brief explanation of why these were recommended"
}
`

// InterviewProfile represents information gathered during the init interview.
// This is distinct from UserProfile which is used for advisor context.
type InterviewProfile struct {
	ExperienceLevel string   // beginner, intermediate, advanced
	PrimaryLanguage string   // go, python, javascript, etc.
	Languages       []string // all languages used
	Tools           []string // IDEs, editors, etc.
	Workflows       []string // docker, kubernetes, etc.
	Goals           []string // what they want to achieve

	// Context inference fields
	WorkContext  WorkContext // inferred work/personal context
	DeviceType   DeviceType  // inferred device type
	EmailDomains []string    // detected email domains
	SSHKeyNames  []string    // detected SSH key names
}

// AIRecommendation represents a parsed configuration recommendation from AI.
// This is distinct from Recommendation which is a domain value object.
type AIRecommendation struct {
	Presets     []string `json:"presets"`
	Layers      []string `json:"layers"`
	Explanation string   `json:"explanation"`
}

// BuildInterviewPrompt creates a prompt for the interview based on interview profile.
func BuildInterviewPrompt(profile InterviewProfile) Prompt {
	var parts []string

	parts = append(parts, "Based on the following developer profile, recommend preflight configuration presets and layers:")
	parts = append(parts, "")

	if profile.ExperienceLevel != "" {
		parts = append(parts, fmt.Sprintf("Experience Level: %s", profile.ExperienceLevel))
	}
	if profile.PrimaryLanguage != "" {
		parts = append(parts, fmt.Sprintf("Primary Language: %s", profile.PrimaryLanguage))
	}
	if len(profile.Languages) > 0 {
		parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(profile.Languages, ", ")))
	}
	if len(profile.Tools) > 0 {
		parts = append(parts, fmt.Sprintf("Tools/Editors: %s", strings.Join(profile.Tools, ", ")))
	}
	if len(profile.Workflows) > 0 {
		parts = append(parts, fmt.Sprintf("Workflows: %s", strings.Join(profile.Workflows, ", ")))
	}
	if len(profile.Goals) > 0 {
		parts = append(parts, fmt.Sprintf("Goals: %s", strings.Join(profile.Goals, ", ")))
	}

	// Add inferred context information
	if profile.WorkContext != "" && profile.WorkContext != WorkContextUnknown {
		parts = append(parts, fmt.Sprintf("Detected Context: %s setup", profile.WorkContext))
	}
	if profile.DeviceType != "" && profile.DeviceType != DeviceTypeUnknown {
		parts = append(parts, fmt.Sprintf("Device Type: %s", profile.DeviceType))
	}
	if len(profile.EmailDomains) > 0 {
		parts = append(parts, fmt.Sprintf("Email Domains: %s", strings.Join(profile.EmailDomains, ", ")))
	}

	parts = append(parts, "")
	parts = append(parts, "Available presets:")
	parts = append(parts, "- minimal: Basic shell and git setup")
	parts = append(parts, "- developer: Full development environment")
	parts = append(parts, "- data-science: Python, Jupyter, data tools")
	parts = append(parts, "- devops: Docker, Kubernetes, cloud tools")
	parts = append(parts, "")
	parts = append(parts, "Available layers:")
	parts = append(parts, "- base: Core configuration (always included)")
	parts = append(parts, "- role.go: Go development")
	parts = append(parts, "- role.python: Python development")
	parts = append(parts, "- role.node: Node.js/JavaScript development")
	parts = append(parts, "- role.rust: Rust development")
	parts = append(parts, "- identity.work: Work-specific settings (use when corporate email detected)")
	parts = append(parts, "- identity.personal: Personal settings (use when personal email detected)")
	parts = append(parts, "- device.laptop: Laptop-specific settings (battery management, portability)")
	parts = append(parts, "- device.desktop: Desktop-specific settings (performance, multi-monitor)")
	parts = append(parts, "")
	parts = append(parts, "Based on the detected context, suggest appropriate identity and device layers.")
	parts = append(parts, "Respond with a JSON object containing presets, layers, and explanation.")

	userPrompt := strings.Join(parts, "\n")

	return NewPrompt(InitInterviewSystemPrompt, userPrompt).
		WithMaxTokens(512).
		WithTemperature(0.3)
}

// ParseRecommendations parses AI response into recommendations.
func ParseRecommendations(response string) (*AIRecommendation, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var rec AIRecommendation
	if err := json.Unmarshal([]byte(jsonStr), &rec); err != nil {
		return nil, fmt.Errorf("failed to parse recommendation JSON: %w", err)
	}

	return &rec, nil
}

// ExperienceQuestionPrompt generates a prompt to ask about experience level.
func ExperienceQuestionPrompt() Prompt {
	return NewPrompt(
		InitInterviewSystemPrompt,
		"Ask the user about their development experience level. Keep it brief and friendly.",
	).WithMaxTokens(256).WithTemperature(0.7)
}

// LanguageQuestionPrompt generates a prompt to ask about programming languages.
func LanguageQuestionPrompt() Prompt {
	return NewPrompt(
		InitInterviewSystemPrompt,
		"Ask the user about their primary programming languages and tools. Keep it brief and friendly.",
	).WithMaxTokens(256).WithTemperature(0.7)
}

// GoalsQuestionPrompt generates a prompt to ask about goals.
func GoalsQuestionPrompt() Prompt {
	return NewPrompt(
		InitInterviewSystemPrompt,
		"Ask the user what they want to achieve with their development environment setup. Keep it brief and friendly.",
	).WithMaxTokens(256).WithTemperature(0.7)
}
