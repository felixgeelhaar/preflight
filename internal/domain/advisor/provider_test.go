package advisor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrompt(t *testing.T) {
	prompt := NewPrompt("system", "user")

	assert.Equal(t, "system", prompt.SystemPrompt())
	assert.Equal(t, "user", prompt.UserPrompt())
	assert.Equal(t, 1024, prompt.MaxTokens())
	assert.InDelta(t, 0.7, prompt.Temperature(), 0.001)
}

func TestPrompt_WithMaxTokens(t *testing.T) {
	prompt := NewPrompt("system", "user")
	updated := prompt.WithMaxTokens(2048)

	// Original unchanged
	assert.Equal(t, 1024, prompt.MaxTokens())
	// New prompt has updated value
	assert.Equal(t, 2048, updated.MaxTokens())
	// Other values preserved
	assert.Equal(t, "system", updated.SystemPrompt())
	assert.Equal(t, "user", updated.UserPrompt())
}

func TestPrompt_WithTemperature(t *testing.T) {
	prompt := NewPrompt("system", "user")
	updated := prompt.WithTemperature(0.3)

	// Original unchanged
	assert.InDelta(t, 0.7, prompt.Temperature(), 0.001)
	// New prompt has updated value
	assert.InDelta(t, 0.3, updated.Temperature(), 0.001)
}

func TestPrompt_Size(t *testing.T) {
	tests := []struct {
		name         string
		system       string
		user         string
		expectedSize int
	}{
		{
			name:         "empty prompts",
			system:       "",
			user:         "",
			expectedSize: 0,
		},
		{
			name:         "system only",
			system:       "hello",
			user:         "",
			expectedSize: 5,
		},
		{
			name:         "user only",
			system:       "",
			user:         "world",
			expectedSize: 5,
		},
		{
			name:         "both prompts",
			system:       "hello",
			user:         "world",
			expectedSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := NewPrompt(tt.system, tt.user)
			assert.Equal(t, tt.expectedSize, prompt.Size())
		})
	}
}

func TestPrompt_Validate(t *testing.T) {
	t.Run("valid small prompt", func(t *testing.T) {
		prompt := NewPrompt("system", "user")
		err := prompt.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid prompt at max size", func(t *testing.T) {
		// Create prompts that total exactly MaxPromptSize
		system := strings.Repeat("s", MaxPromptSize/2)
		user := strings.Repeat("u", MaxPromptSize/2)
		prompt := NewPrompt(system, user)
		err := prompt.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid prompt exceeds max size", func(t *testing.T) {
		// Create prompts that exceed MaxPromptSize
		largeContent := strings.Repeat("x", MaxPromptSize+1)
		prompt := NewPrompt(largeContent, "")
		err := prompt.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPromptTooLarge)
	})
}

func TestMaxPromptSize_Constant(t *testing.T) {
	// Verify the constant is set to 100KB
	assert.Equal(t, 100*1024, MaxPromptSize)
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse("content", 100, "gpt-4")

	assert.Equal(t, "content", resp.Content())
	assert.Equal(t, 100, resp.TokensUsed())
	assert.Equal(t, "gpt-4", resp.Model())
}

func TestResponse_EmptyValues(t *testing.T) {
	resp := NewResponse("", 0, "")

	assert.Empty(t, resp.Content())
	assert.Zero(t, resp.TokensUsed())
	assert.Empty(t, resp.Model())
}
