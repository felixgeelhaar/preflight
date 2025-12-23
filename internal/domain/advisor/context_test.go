package advisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserProfile_Valid(t *testing.T) {
	t.Parallel()

	profile, err := NewUserProfile(ExperienceIntermediate)

	require.NoError(t, err)
	assert.Equal(t, ExperienceIntermediate, profile.Experience())
	assert.Empty(t, profile.Preferences())
	assert.Empty(t, profile.ExistingTools())
}

func TestUserProfile_WithPreferences(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceBeginner)
	prefs := map[string]string{
		"shell":  "zsh",
		"editor": "nvim",
	}

	updated := profile.WithPreferences(prefs)

	assert.Empty(t, profile.Preferences())
	assert.Equal(t, prefs, updated.Preferences())
}

func TestUserProfile_WithExistingTools(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceBeginner)
	tools := []string{"git", "docker", "node"}

	updated := profile.WithExistingTools(tools)

	assert.Empty(t, profile.ExistingTools())
	assert.Equal(t, tools, updated.ExistingTools())
}

func TestUserProfile_WithOperatingSystem(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceBeginner)

	updated := profile.WithOperatingSystem("darwin")

	assert.Empty(t, profile.OperatingSystem())
	assert.Equal(t, "darwin", updated.OperatingSystem())
}

func TestUserProfile_IsZero(t *testing.T) {
	t.Parallel()

	var zero UserProfile
	assert.True(t, zero.IsZero())

	nonZero, _ := NewUserProfile(ExperienceBeginner)
	assert.False(t, nonZero.IsZero())
}

func TestExperienceLevel_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "beginner", ExperienceBeginner.String())
	assert.Equal(t, "intermediate", ExperienceIntermediate.String())
	assert.Equal(t, "advanced", ExperienceAdvanced.String())
}

func TestExperienceLevel_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, ExperienceBeginner.IsValid())
	assert.True(t, ExperienceIntermediate.IsValid())
	assert.True(t, ExperienceAdvanced.IsValid())
	assert.False(t, ExperienceLevel("expert").IsValid())
}

func TestParseExperienceLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected ExperienceLevel
		wantErr  bool
	}{
		{"beginner", ExperienceBeginner, false},
		{"intermediate", ExperienceIntermediate, false},
		{"advanced", ExperienceAdvanced, false},
		{"BEGINNER", ExperienceBeginner, false}, // case insensitive
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			level, err := ParseExperienceLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestNewSuggestContext_Valid(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceIntermediate)
	ctx, err := NewSuggestContext("nvim", profile)

	require.NoError(t, err)
	assert.Equal(t, "nvim", ctx.Category())
	assert.Equal(t, profile, ctx.UserProfile())
}

func TestNewSuggestContext_EmptyCategory(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceIntermediate)
	_, err := NewSuggestContext("", profile)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyCategory)
}

func TestNewSuggestContext_ZeroProfile(t *testing.T) {
	t.Parallel()

	_, err := NewSuggestContext("nvim", UserProfile{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidProfile)
}

func TestSuggestContext_WithConstraints(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceIntermediate)
	ctx, _ := NewSuggestContext("nvim", profile)
	constraints := []string{"no-plugins", "minimal"}

	updated := ctx.WithConstraints(constraints)

	assert.Empty(t, ctx.Constraints())
	assert.Equal(t, constraints, updated.Constraints())
}

func TestSuggestContext_WithAdditionalContext(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceIntermediate)
	ctx, _ := NewSuggestContext("nvim", profile)

	updated := ctx.WithAdditionalContext("I primarily work on Go projects")

	assert.Empty(t, ctx.AdditionalContext())
	assert.Equal(t, "I primarily work on Go projects", updated.AdditionalContext())
}

func TestSuggestContext_WithMaxRecommendations(t *testing.T) {
	t.Parallel()

	profile, _ := NewUserProfile(ExperienceIntermediate)
	ctx, _ := NewSuggestContext("nvim", profile)

	updated := ctx.WithMaxRecommendations(5)

	assert.Equal(t, 3, ctx.MaxRecommendations()) // default
	assert.Equal(t, 5, updated.MaxRecommendations())
}

func TestSuggestContext_IsZero(t *testing.T) {
	t.Parallel()

	var zero SuggestContext
	assert.True(t, zero.IsZero())

	profile, _ := NewUserProfile(ExperienceIntermediate)
	nonZero, _ := NewSuggestContext("nvim", profile)
	assert.False(t, nonZero.IsZero())
}

func TestExplainRequest_Valid(t *testing.T) {
	t.Parallel()

	req, err := NewExplainRequest("nvim:balanced")

	require.NoError(t, err)
	assert.Equal(t, "nvim:balanced", req.PresetID())
}

func TestExplainRequest_EmptyPresetID(t *testing.T) {
	t.Parallel()

	_, err := NewExplainRequest("")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyPresetID)
}

func TestExplainRequest_WithUserExperience(t *testing.T) {
	t.Parallel()

	req, _ := NewExplainRequest("nvim:balanced")

	updated := req.WithUserExperience(ExperienceBeginner)

	assert.Equal(t, ExperienceLevel(""), req.UserExperience())
	assert.Equal(t, ExperienceBeginner, updated.UserExperience())
}

func TestExplainRequest_WithQuestions(t *testing.T) {
	t.Parallel()

	req, _ := NewExplainRequest("nvim:balanced")
	questions := []string{"What plugins are included?", "How do I customize it?"}

	updated := req.WithQuestions(questions)

	assert.Empty(t, req.Questions())
	assert.Equal(t, questions, updated.Questions())
}
