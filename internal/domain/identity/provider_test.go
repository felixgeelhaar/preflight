package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProviderConfig_Valid_OIDC(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
		Scopes:   []string{"openid", "profile", "email"},
	}

	err := ValidateProviderConfig(cfg)
	require.NoError(t, err)
}

func TestValidateProviderConfig_Valid_SAML(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name:   "corporate-saml",
		Type:   ProviderTypeSAML,
		Issuer: "https://idp.example.com",
	}

	err := ValidateProviderConfig(cfg)
	require.NoError(t, err)
}

func TestValidateProviderConfig_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  ProviderConfig
		msg  string
	}{
		{
			name: "empty name",
			cfg:  ProviderConfig{Type: ProviderTypeOIDC, Issuer: "https://a.com", ClientID: "c"},
			msg:  "name is required",
		},
		{
			name: "empty type",
			cfg:  ProviderConfig{Name: "corp", Issuer: "https://a.com", ClientID: "c"},
			msg:  "type is required",
		},
		{
			name: "invalid type",
			cfg:  ProviderConfig{Name: "corp", Type: "ldap", Issuer: "https://a.com"},
			msg:  "unsupported provider type",
		},
		{
			name: "OIDC missing issuer",
			cfg:  ProviderConfig{Name: "corp", Type: ProviderTypeOIDC, ClientID: "c"},
			msg:  "issuer is required",
		},
		{
			name: "OIDC missing client ID",
			cfg:  ProviderConfig{Name: "corp", Type: ProviderTypeOIDC, Issuer: "https://a.com"},
			msg:  "client ID is required",
		},
		{
			name: "OIDC issuer not HTTPS",
			cfg:  ProviderConfig{Name: "corp", Type: ProviderTypeOIDC, Issuer: "http://a.com", ClientID: "c"},
			msg:  "issuer must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateProviderConfig(tt.cfg)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidConfig)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestProviderConfig_Clone(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
		Scopes:   []string{"openid", "profile"},
	}

	cloned := cfg.Clone()

	assert.Equal(t, cfg.Name, cloned.Name)
	assert.Equal(t, cfg.Scopes, cloned.Scopes)

	// Mutate original scopes.
	cfg.Scopes[0] = "mutated"
	assert.Equal(t, "openid", cloned.Scopes[0], "clone should not be affected by original mutation")
}

func TestProviderType_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ProviderTypeOIDC, ProviderType("oidc"))
	assert.Equal(t, ProviderTypeSAML, ProviderType("saml"))
}
