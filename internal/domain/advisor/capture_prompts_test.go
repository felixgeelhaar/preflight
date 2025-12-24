package advisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCaptureAnalysisPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		req             CaptureAnalysisRequest
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "basic items",
			req: CaptureAnalysisRequest{
				Items: []CapturedItem{
					{Path: ".zshrc", Type: "file", Description: "Zsh configuration"},
					{Path: ".config/nvim/init.lua", Type: "file"},
					{Path: "neovim", Type: "tool", Description: "Text editor"},
				},
			},
			wantContains: []string{
				".zshrc",
				"init.lua",
				"neovim",
				"Configuration Files",
				"Installed Tools",
			},
		},
		{
			name: "with patterns",
			req: CaptureAnalysisRequest{
				Items: []CapturedItem{
					{Path: ".zshrc", Type: "file"},
				},
				Patterns: []PatternMatch{
					{Name: "zsh", Type: "shell", Confidence: 0.9},
					{Name: "oh-my-zsh", Type: "shell", Confidence: 0.85},
				},
			},
			wantContains: []string{
				"Detected Patterns",
				"zsh (shell): 90% confidence",
				"oh-my-zsh (shell): 85% confidence",
			},
		},
		{
			name: "with user profile",
			req: CaptureAnalysisRequest{
				Items: []CapturedItem{
					{Path: ".gitconfig", Type: "file"},
				},
				UserProfile: InterviewProfile{
					ExperienceLevel: "advanced",
					Languages:       []string{"go", "python"},
					WorkContext:     WorkContextWork,
					DeviceType:      DeviceTypeLaptop,
				},
			},
			wantContains: []string{
				"User Context",
				"Experience: advanced",
				"Languages: go, python",
				"Context: work",
				"Device: laptop",
			},
		},
		{
			name: "empty request",
			req:  CaptureAnalysisRequest{},
			wantContains: []string{
				"Analyze the following",
				"Respond with a JSON object",
			},
			wantNotContains: []string{
				"Configuration Files",
				"Installed Tools",
				"Detected Patterns",
				"User Context",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prompt := BuildCaptureAnalysisPrompt(tt.req)

			for _, want := range tt.wantContains {
				assert.Contains(t, prompt.UserPrompt(), want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, prompt.UserPrompt(), notWant)
			}

			// Verify prompt configuration
			assert.Equal(t, CaptureAnalysisSystemPrompt, prompt.SystemPrompt())
			assert.Equal(t, 1024, prompt.MaxTokens())
			assert.InDelta(t, 0.3, prompt.Temperature(), 0.01)
		})
	}
}

func TestParseCaptureAnalysis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		want     *CaptureAnalysisResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			response: `Here's my analysis:
{
	"summary": "Well-organized development environment",
	"categories": {
		"shell": [".zshrc", ".oh-my-zsh/"],
		"editor": [".config/nvim/"]
	},
	"recommendations": [
		{
			"title": "Add shell completions",
			"description": "Enable completions for better productivity",
			"priority": "high",
			"category": "optimization",
			"action": "Install zsh-completions",
			"why": "Speeds up command entry"
		}
	],
	"suggested_layers": ["base", "role.go"],
	"warnings": ["No GPG signing configured"],
	"quality_score": {
		"overall": 0.8,
		"organization": 0.9,
		"security": 0.7,
		"modernness": 0.85,
		"portability": 0.75
	}
}
`,
			want: &CaptureAnalysisResponse{
				Summary: "Well-organized development environment",
				Categories: map[string][]string{
					"shell":  {".zshrc", ".oh-my-zsh/"},
					"editor": {".config/nvim/"},
				},
				Recommendations: []CaptureRecommendation{
					{
						Title:       "Add shell completions",
						Description: "Enable completions for better productivity",
						Priority:    "high",
						Category:    "optimization",
						Action:      "Install zsh-completions",
						Why:         "Speeds up command entry",
					},
				},
				SuggestedLayers: []string{"base", "role.go"},
				Warnings:        []string{"No GPG signing configured"},
				Score: QualityScore{
					Overall:      0.8,
					Organization: 0.9,
					Security:     0.7,
					Modernness:   0.85,
					Portability:  0.75,
				},
			},
			wantErr: false,
		},
		{
			name:     "no JSON in response",
			response: "I can help you with that, but I don't have enough information.",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			response: `{"summary": "test", categories: invalid}`,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseCaptureAnalysis(tt.response)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.want.Summary, result.Summary)
			assert.Equal(t, tt.want.Categories, result.Categories)
			assert.Len(t, result.Recommendations, len(tt.want.Recommendations))
			assert.Equal(t, tt.want.SuggestedLayers, result.SuggestedLayers)
		})
	}
}

func TestBuildContextualRecommendationPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		req          ContextualRecommendationRequest
		wantContains []string
	}{
		{
			name: "full context",
			req: ContextualRecommendationRequest{
				DetectedPatterns: []PatternMatch{
					{Name: "neovim", Type: "editor"},
					{Name: "zsh", Type: "shell"},
				},
				ExistingTools: []string{"ripgrep", "fzf", "git"},
				Profile: InterviewProfile{
					ExperienceLevel: "intermediate",
					Languages:       []string{"go", "rust"},
				},
				Goals: []string{"improve productivity", "learn vim"},
			},
			wantContains: []string{
				"Experience Level: intermediate",
				"neovim (editor)",
				"zsh (shell)",
				"ripgrep, fzf, git",
				"improve productivity",
				"learn vim",
				"go, rust",
			},
		},
		{
			name: "minimal context",
			req: ContextualRecommendationRequest{
				Profile: InterviewProfile{
					ExperienceLevel: "beginner",
				},
			},
			wantContains: []string{
				"Experience Level: beginner",
				"Response Format",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prompt := BuildContextualRecommendationPrompt(tt.req)

			for _, want := range tt.wantContains {
				assert.Contains(t, prompt.UserPrompt(), want)
			}

			assert.Equal(t, ContextualRecommendationSystemPrompt, prompt.SystemPrompt())
			assert.Equal(t, 1024, prompt.MaxTokens())
			assert.InDelta(t, 0.4, prompt.Temperature(), 0.01)
		})
	}
}

func TestParseContextualRecommendation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		want     *ContextualRecommendationResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			response: `Based on your setup, here are my recommendations:
{
	"summary": "Solid foundation with room for improvement",
	"recommendations": [
		{
			"tool": "lazygit",
			"category": "git",
			"title": "Terminal UI for Git",
			"description": "Interactive git client in your terminal",
			"why": "Complements your neovim workflow",
			"links": ["https://github.com/jesseduffield/lazygit"],
			"config_hint": "brew install lazygit",
			"priority": 1
		}
	],
	"gaps": ["No container tooling detected"],
	"synergies": ["neovim + fzf is a great combination"]
}
`,
			want: &ContextualRecommendationResponse{
				Summary: "Solid foundation with room for improvement",
				Recommendations: []ContextualRecommendation{
					{
						Tool:        "lazygit",
						Category:    "git",
						Title:       "Terminal UI for Git",
						Description: "Interactive git client in your terminal",
						Why:         "Complements your neovim workflow",
						Links:       []string{"https://github.com/jesseduffield/lazygit"},
						ConfigHint:  "brew install lazygit",
						Priority:    1,
					},
				},
				Gaps:      []string{"No container tooling detected"},
				Synergies: []string{"neovim + fzf is a great combination"},
			},
			wantErr: false,
		},
		{
			name:     "no JSON",
			response: "I need more information to help you.",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseContextualRecommendation(tt.response)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.want.Summary, result.Summary)
			assert.Len(t, result.Recommendations, len(tt.want.Recommendations))
			if len(tt.want.Recommendations) > 0 {
				assert.Equal(t, tt.want.Recommendations[0].Tool, result.Recommendations[0].Tool)
				assert.Equal(t, tt.want.Recommendations[0].Priority, result.Recommendations[0].Priority)
			}
			assert.Equal(t, tt.want.Gaps, result.Gaps)
			assert.Equal(t, tt.want.Synergies, result.Synergies)
		})
	}
}

func TestCapturedItem(t *testing.T) {
	t.Parallel()

	item := CapturedItem{
		Path:        ".zshrc",
		Type:        "file",
		Size:        1024,
		Description: "Zsh configuration",
	}

	assert.Equal(t, ".zshrc", item.Path)
	assert.Equal(t, "file", item.Type)
	assert.Equal(t, int64(1024), item.Size)
	assert.Equal(t, "Zsh configuration", item.Description)
}

func TestPatternMatch(t *testing.T) {
	t.Parallel()

	pattern := PatternMatch{
		Name:       "oh-my-zsh",
		Type:       "shell",
		Confidence: 0.95,
	}

	assert.Equal(t, "oh-my-zsh", pattern.Name)
	assert.Equal(t, "shell", pattern.Type)
	assert.InDelta(t, 0.95, pattern.Confidence, 0.001)
}

func TestQualityScore(t *testing.T) {
	t.Parallel()

	score := QualityScore{
		Overall:      0.85,
		Organization: 0.9,
		Security:     0.8,
		Modernness:   0.85,
		Portability:  0.75,
	}

	assert.InDelta(t, 0.85, score.Overall, 0.001)
	assert.InDelta(t, 0.9, score.Organization, 0.001)
	assert.InDelta(t, 0.8, score.Security, 0.001)
	assert.InDelta(t, 0.85, score.Modernness, 0.001)
	assert.InDelta(t, 0.75, score.Portability, 0.001)
}
