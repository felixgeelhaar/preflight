package identity

import (
	"fmt"
	"time"
)

// Claims represents OIDC/SAML identity claims (immutable value object).
type Claims struct {
	subject  string
	email    string
	name     string
	groups   []string
	issuer   string
	audience string
	extra    map[string]string
}

// NewClaims creates a new Claims value object with defensive copies.
func NewClaims(subject, email, name string, groups []string, issuer, audience string, extra map[string]string) Claims {
	c := Claims{
		subject:  subject,
		email:    email,
		name:     name,
		issuer:   issuer,
		audience: audience,
	}

	if groups != nil {
		c.groups = make([]string, len(groups))
		copy(c.groups, groups)
	}

	if extra != nil {
		c.extra = make(map[string]string, len(extra))
		for k, v := range extra {
			c.extra[k] = v
		}
	}

	return c
}

// Subject returns the sub claim.
func (c Claims) Subject() string { return c.subject }

// Email returns the email claim.
func (c Claims) Email() string { return c.email }

// Name returns the name claim.
func (c Claims) Name() string { return c.name }

// Groups returns a defensive copy of the groups claim.
func (c Claims) Groups() []string {
	if c.groups == nil {
		return nil
	}
	out := make([]string, len(c.groups))
	copy(out, c.groups)
	return out
}

// Issuer returns the iss claim.
func (c Claims) Issuer() string { return c.issuer }

// Audience returns the aud claim.
func (c Claims) Audience() string { return c.audience }

// Extra returns a defensive copy of the additional claims.
func (c Claims) Extra() map[string]string {
	if c.extra == nil {
		return nil
	}
	out := make(map[string]string, len(c.extra))
	for k, v := range c.extra {
		out[k] = v
	}
	return out
}

// HasGroup returns true if the claims contain the given group.
func (c Claims) HasGroup(group string) bool {
	if group == "" {
		return false
	}
	for _, g := range c.groups {
		if g == group {
			return true
		}
	}
	return false
}

// IsZero returns true if this is a zero-value Claims.
func (c Claims) IsZero() bool {
	return c.subject == "" && c.email == "" && c.name == "" &&
		c.groups == nil && c.issuer == "" && c.audience == "" && c.extra == nil
}

// Token represents an authentication token (immutable value object).
type Token struct {
	accessToken  string
	refreshToken string
	tokenType    string
	expiresAt    time.Time
	claims       Claims
	providerName string
}

// NewToken creates a new Token value object.
// Access token and expiry are required.
func NewToken(accessToken, tokenType string, expiresAt time.Time, claims Claims, providerName string) (Token, error) {
	if accessToken == "" {
		return Token{}, fmt.Errorf("%w: access token is required", ErrInvalidToken)
	}
	if expiresAt.IsZero() {
		return Token{}, fmt.Errorf("%w: expiry is required", ErrInvalidToken)
	}

	return Token{
		accessToken:  accessToken,
		tokenType:    tokenType,
		expiresAt:    expiresAt,
		claims:       claims,
		providerName: providerName,
	}, nil
}

// AccessToken returns the access token string.
func (t Token) AccessToken() string { return t.accessToken }

// RefreshToken returns the refresh token string.
func (t Token) RefreshToken() string { return t.refreshToken }

// TokenType returns the token type (e.g. "Bearer").
func (t Token) TokenType() string { return t.tokenType }

// ExpiresAt returns the token expiry time.
func (t Token) ExpiresAt() time.Time { return t.expiresAt }

// Claims returns the token claims.
func (t Token) Claims() Claims { return t.claims }

// ProviderName returns the name of the provider that issued this token.
func (t Token) ProviderName() string { return t.providerName }

// IsExpired returns true if the token has expired.
func (t Token) IsExpired() bool {
	return time.Now().After(t.expiresAt)
}

// NeedsRefresh returns true if the token expires within the given threshold.
func (t Token) NeedsRefresh(threshold time.Duration) bool {
	return time.Now().Add(threshold).After(t.expiresAt)
}

// WithRefreshToken returns a new Token with the refresh token set.
func (t Token) WithRefreshToken(refreshToken string) Token {
	clone := t.Clone()
	clone.refreshToken = refreshToken
	return clone
}

// Clone returns a deep copy of the Token.
func (t Token) Clone() Token {
	return Token{
		accessToken:  t.accessToken,
		refreshToken: t.refreshToken,
		tokenType:    t.tokenType,
		expiresAt:    t.expiresAt,
		claims: NewClaims(
			t.claims.subject,
			t.claims.email,
			t.claims.name,
			t.claims.groups,
			t.claims.issuer,
			t.claims.audience,
			t.claims.extra,
		),
		providerName: t.providerName,
	}
}
