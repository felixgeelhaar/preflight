package openai

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider_NoAPIKey(t *testing.T) {
	t.Parallel()

	p := NewProvider("")

	assert.NotNil(t, p)
	assert.False(t, p.Available())
}

func TestNewProvider_WithAPIKey(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-test-key")

	assert.NotNil(t, p)
	assert.True(t, p.Available())
}

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-test-key")

	assert.Equal(t, "openai", p.Name())
}

func TestProvider_Model(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-test-key")

	assert.Equal(t, "gpt-4o", p.Model())
}

func TestProvider_WithModel(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-test-key").WithModel("gpt-3.5-turbo")

	assert.Equal(t, "gpt-3.5-turbo", p.Model())
}

func TestProvider_Complete_NotAvailable(t *testing.T) {
	t.Parallel()

	p := NewProvider("")
	prompt := advisor.NewPrompt("system", "user")

	_, err := p.Complete(context.Background(), prompt)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestProvider_ImplementsAIProvider(t *testing.T) {
	t.Parallel()

	var _ advisor.AIProvider = (*Provider)(nil)
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid",
			config: Config{
				APIKey: "sk-test-key",
				Model:  "gpt-4o",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "gpt-4o",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "sk-test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
