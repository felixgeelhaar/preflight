package advisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInterviewPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		profile  InterviewProfile
		contains []string
	}{
		{
			name: "full profile",
			profile: InterviewProfile{
				ExperienceLevel: "intermediate",
				PrimaryLanguage: "go",
				Languages:       []string{"go", "python", "javascript"},
				Tools:           []string{"neovim", "vscode"},
				Workflows:       []string{"docker", "kubernetes"},
				Goals:           []string{"productivity", "consistency"},
			},
			contains: []string{
				"Experience Level: intermediate",
				"Primary Language: go",
				"Languages: go, python, javascript",
				"Tools/Editors: neovim, vscode",
				"Workflows: docker, kubernetes",
				"Goals: productivity, consistency",
			},
		},
		{
			name: "minimal profile",
			profile: InterviewProfile{
				ExperienceLevel: "beginner",
			},
			contains: []string{
				"Experience Level: beginner",
				"Available presets:",
				"Available layers:",
			},
		},
		{
			name:    "empty profile",
			profile: InterviewProfile{},
			contains: []string{
				"Available presets:",
				"Available layers:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prompt := BuildInterviewPrompt(tt.profile)

			assert.NotEmpty(t, prompt.SystemPrompt())
			assert.NotEmpty(t, prompt.UserPrompt())
			assert.Equal(t, 512, prompt.MaxTokens())
			assert.Equal(t, 0.3, prompt.Temperature())

			for _, s := range tt.contains {
				assert.Contains(t, prompt.UserPrompt(), s)
			}
		})
	}
}

func TestParseRecommendations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		response   string
		wantPreset []string
		wantLayers []string
		wantErr    bool
	}{
		{
			name: "valid JSON",
			response: `Based on your profile, I recommend:
{
  "presets": ["developer"],
  "layers": ["base", "role.go"],
  "explanation": "Go developer setup"
}`,
			wantPreset: []string{"developer"},
			wantLayers: []string{"base", "role.go"},
			wantErr:    false,
		},
		{
			name:       "JSON only",
			response:   `{"presets": ["minimal"], "layers": ["base"], "explanation": "Basic setup"}`,
			wantPreset: []string{"minimal"},
			wantLayers: []string{"base"},
			wantErr:    false,
		},
		{
			name:     "no JSON",
			response: "I recommend the developer preset with go layer",
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			response: `{"presets": ["developer"`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec, err := ParseRecommendations(tt.response)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPreset, rec.Presets)
			assert.Equal(t, tt.wantLayers, rec.Layers)
			assert.NotEmpty(t, rec.Explanation)
		})
	}
}

func TestQuestionPrompts(t *testing.T) {
	t.Parallel()

	t.Run("experience question", func(t *testing.T) {
		t.Parallel()

		prompt := ExperienceQuestionPrompt()

		assert.NotEmpty(t, prompt.SystemPrompt())
		assert.Contains(t, prompt.UserPrompt(), "experience")
		assert.Equal(t, 256, prompt.MaxTokens())
		assert.Equal(t, 0.7, prompt.Temperature())
	})

	t.Run("language question", func(t *testing.T) {
		t.Parallel()

		prompt := LanguageQuestionPrompt()

		assert.NotEmpty(t, prompt.SystemPrompt())
		assert.Contains(t, prompt.UserPrompt(), "languages")
		assert.Equal(t, 256, prompt.MaxTokens())
	})

	t.Run("goals question", func(t *testing.T) {
		t.Parallel()

		prompt := GoalsQuestionPrompt()

		assert.NotEmpty(t, prompt.SystemPrompt())
		assert.Contains(t, prompt.UserPrompt(), "achieve")
		assert.Equal(t, 256, prompt.MaxTokens())
	})
}

func TestInitInterviewSystemPrompt(t *testing.T) {
	t.Parallel()

	assert.Contains(t, InitInterviewSystemPrompt, "preflight")
	assert.Contains(t, InitInterviewSystemPrompt, "JSON")
	assert.Contains(t, InitInterviewSystemPrompt, "presets")
	assert.Contains(t, InitInterviewSystemPrompt, "layers")
}
