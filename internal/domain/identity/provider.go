package identity

import (
	"context"
	"fmt"
	"strings"
)

// ProviderType indicates the type of identity provider.
type ProviderType string

const (
	// ProviderTypeOIDC represents an OpenID Connect provider.
	ProviderTypeOIDC ProviderType = "oidc"
	// ProviderTypeSAML represents a SAML provider.
	ProviderTypeSAML ProviderType = "saml"
)

// validProviderTypes is the set of supported provider types.
var validProviderTypes = map[ProviderType]bool{
	ProviderTypeOIDC: true,
	ProviderTypeSAML: true,
}

// ProviderConfig is the configuration for an identity provider.
type ProviderConfig struct {
	Name     string       // e.g., "corporate"
	Type     ProviderType // oidc or saml
	Issuer   string       // OIDC issuer URL
	ClientID string       // OAuth2 client ID
	Scopes   []string     // e.g., ["openid", "profile", "email", "groups"]
}

// Clone returns a deep copy of the ProviderConfig.
func (c ProviderConfig) Clone() ProviderConfig {
	out := c
	if c.Scopes != nil {
		out.Scopes = make([]string, len(c.Scopes))
		copy(out.Scopes, c.Scopes)
	}
	return out
}

// ValidateProviderConfig validates the provider configuration.
func ValidateProviderConfig(cfg ProviderConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}

	if cfg.Type == "" {
		return fmt.Errorf("%w: type is required", ErrInvalidConfig)
	}

	if !validProviderTypes[cfg.Type] {
		return fmt.Errorf("%w: unsupported provider type: %s", ErrInvalidConfig, cfg.Type)
	}

	if cfg.Type == ProviderTypeOIDC {
		if cfg.Issuer == "" {
			return fmt.Errorf("%w: issuer is required for OIDC", ErrInvalidConfig)
		}
		if !strings.HasPrefix(cfg.Issuer, "https://") {
			return fmt.Errorf("%w: issuer must use HTTPS", ErrInvalidConfig)
		}
		if cfg.ClientID == "" {
			return fmt.Errorf("%w: client ID is required for OIDC", ErrInvalidConfig)
		}
	}

	return nil
}

// Provider defines the interface for authentication providers.
type Provider interface {
	// Name returns the provider name.
	Name() string
	// Type returns the provider type.
	Type() ProviderType
	// Authenticate performs authentication and returns a token.
	Authenticate(ctx context.Context) (*Token, error)
	// Refresh refreshes an existing token.
	Refresh(ctx context.Context, token *Token) (*Token, error)
	// Validate validates a token.
	Validate(ctx context.Context, token *Token) error
}
