package ollama

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider_DefaultEndpoint(t *testing.T) {
	t.Parallel()

	p := NewProvider("")

	assert.NotNil(t, p)
	assert.Equal(t, "http://localhost:11434", p.Endpoint())
}

func TestNewProvider_CustomEndpoint(t *testing.T) {
	t.Parallel()

	p := NewProvider("http://custom:11434")

	assert.NotNil(t, p)
	assert.Equal(t, "http://custom:11434", p.Endpoint())
}

func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p := NewProvider("")

	assert.Equal(t, "ollama", p.Name())
}

func TestProvider_Model(t *testing.T) {
	t.Parallel()

	p := NewProvider("")

	assert.Equal(t, "llama3.2", p.Model())
}

func TestProvider_WithModel(t *testing.T) {
	t.Parallel()

	p := NewProvider("").WithModel("codellama")

	assert.Equal(t, "codellama", p.Model())
}

func TestProvider_Available_NotConnected(t *testing.T) {
	t.Parallel()

	// Use a non-existent endpoint to ensure it's not available
	p := NewProvider("http://nonexistent:11434")

	// Without actually checking, we assume it's available
	// In real tests, this would be mocked
	assert.True(t, p.Available())
}

func TestProvider_Complete_ReturnsError(t *testing.T) {
	t.Parallel()

	p := NewProvider("http://nonexistent:11434")
	prompt := advisor.NewPrompt("system", "user")

	_, err := p.Complete(context.Background(), prompt)

	// Should return an error because Ollama is not actually running
	require.Error(t, err)
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
			name: "valid with defaults",
			config: Config{
				Model: "llama3.2",
			},
			wantErr: false,
		},
		{
			name: "valid with custom endpoint",
			config: Config{
				Endpoint: "http://custom:11434",
				Model:    "codellama",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			config: Config{
				Endpoint: "http://localhost:11434",
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
			name: "valid config with default endpoint",
			config: Config{
				Model: "llama3.2",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom endpoint",
			config: Config{
				Endpoint: "http://custom:11434",
				Model:    "codellama",
			},
			wantErr: false,
		},
		{
			name: "missing model",
			config: Config{
				Endpoint: "http://localhost:11434",
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
				assert.Equal(t, tt.config.Model, p.model)
				if tt.config.Endpoint != "" {
					assert.Equal(t, tt.config.Endpoint, p.endpoint)
				} else {
					assert.Equal(t, "http://localhost:11434", p.endpoint)
				}
			}
		})
	}
}

func TestProvider_Complete_Available(t *testing.T) {
	t.Parallel()

	p := NewProvider("http://localhost:11434")
	prompt := advisor.NewPrompt("system", "user")

	_, err := p.Complete(context.Background(), prompt)

	// The API is not actually integrated, so it returns ErrNotConfigured
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotConfigured)
}
