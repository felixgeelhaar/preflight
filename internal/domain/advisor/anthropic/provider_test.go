package anthropic

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

	p := NewProvider("sk-ant-test-key")

	assert.NotNil(t, p)
	assert.True(t, p.Available())
}

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")

	assert.Equal(t, "anthropic", p.Name())
}

func TestProvider_Model(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")

	assert.Equal(t, "claude-3-5-sonnet-20241022", p.Model())
}

func TestProvider_WithModel(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key").WithModel("claude-3-opus-20240229")

	assert.Equal(t, "claude-3-opus-20240229", p.Model())
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
				APIKey: "sk-ant-test-key",
				Model:  "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "claude-3-5-sonnet-20241022",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "sk-ant-test-key",
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

func TestNewProviderWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		wantErrType error
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "sk-ant-test-key",
				Model:  "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom endpoint",
			config: Config{
				APIKey:   "sk-ant-test-key",
				Model:    "claude-3-5-sonnet-20241022",
				Endpoint: "https://custom.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "claude-3-5-sonnet-20241022",
			},
			wantErr:     true,
			wantErrType: ErrEmptyAPIKey,
		},
		{
			name: "missing model",
			config: Config{
				APIKey: "sk-ant-test-key",
			},
			wantErr:     true,
			wantErrType: ErrEmptyModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := NewProviderWithConfig(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p)
				assert.Equal(t, tt.config.APIKey, p.apiKey)
				assert.Equal(t, tt.config.Model, p.model)
			}
		})
	}
}

func TestProvider_Complete_Available(t *testing.T) {
	t.Parallel()

	p := NewProvider("sk-ant-test-key")
	prompt := advisor.NewPrompt("system", "user")

	_, err := p.Complete(context.Background(), prompt)

	// The API is not actually integrated, so it returns ErrNotConfigured
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotConfigured)
}
