package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient is an interface for HTTP operations (for testability).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// OIDCDiscovery represents the OIDC discovery document.
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	DeviceAuthEndpoint    string `json:"device_authorization_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// DeviceAuthResponse is the response from the device authorization endpoint.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// OIDCProvider implements Provider using OpenID Connect.
type OIDCProvider struct {
	config       ProviderConfig
	httpClient   HTTPClient
	pollInterval time.Duration // overridable for testing
}

// tokenResponse represents the OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

// userinfoResponse represents the OIDC userinfo endpoint response.
type userinfoResponse struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Groups  []string `json:"groups"`
}

// NewOIDCProvider creates a new OIDC provider.
func NewOIDCProvider(config ProviderConfig, httpClient HTTPClient) (*OIDCProvider, error) {
	if err := ValidateProviderConfig(config); err != nil {
		return nil, err
	}
	if httpClient == nil {
		return nil, fmt.Errorf("%w: HTTP client is required", ErrInvalidConfig)
	}

	return &OIDCProvider{
		config:       config.Clone(),
		httpClient:   httpClient,
		pollInterval: 5 * time.Second,
	}, nil
}

// Name returns the provider name.
func (p *OIDCProvider) Name() string {
	return p.config.Name
}

// Type returns the provider type.
func (p *OIDCProvider) Type() ProviderType {
	return ProviderTypeOIDC
}

// Discover fetches the OIDC discovery document.
func (p *OIDCProvider) Discover(ctx context.Context) (*OIDCDiscovery, error) {
	discoveryURL := strings.TrimRight(p.config.Issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDiscoveryFailed, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDiscoveryFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrDiscoveryFailed, resp.StatusCode)
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, fmt.Errorf("%w: failed to decode discovery document: %w", ErrDiscoveryFailed, err)
	}

	return &discovery, nil
}

// Authenticate performs device authorization grant flow (RFC 8628).
func (p *OIDCProvider) Authenticate(ctx context.Context) (*Token, error) {
	discovery, err := p.Discover(ctx)
	if err != nil {
		return nil, err
	}

	deviceAuth, err := p.startDeviceAuth(ctx, discovery)
	if err != nil {
		return nil, err
	}

	tokenResp, err := p.pollForToken(ctx, discovery, deviceAuth)
	if err != nil {
		return nil, err
	}

	claims, err := p.fetchUserinfo(ctx, discovery, tokenResp.AccessToken)
	if err != nil {
		// Non-fatal: proceed with empty claims.
		claims = Claims{}
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	token, err := NewToken(tokenResp.AccessToken, tokenResp.TokenType, expiresAt, claims, p.config.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
	}

	if tokenResp.RefreshToken != "" {
		token = token.WithRefreshToken(tokenResp.RefreshToken)
	}

	return &token, nil
}

// Refresh refreshes an existing token using the refresh_token grant.
func (p *OIDCProvider) Refresh(ctx context.Context, token *Token) (*Token, error) {
	if token.RefreshToken() == "" {
		return nil, fmt.Errorf("%w: no refresh token available", ErrTokenRefreshFailed)
	}

	discovery, err := p.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken()},
		"client_id":     {p.config.ClientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrTokenRefreshFailed, resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode token response: %w", ErrTokenRefreshFailed, err)
	}

	claims, err := p.fetchUserinfo(ctx, discovery, tokenResp.AccessToken)
	if err != nil {
		claims = Claims{}
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	newToken, err := NewToken(tokenResp.AccessToken, tokenResp.TokenType, expiresAt, claims, p.config.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenRefreshFailed, err)
	}

	if tokenResp.RefreshToken != "" {
		newToken = newToken.WithRefreshToken(tokenResp.RefreshToken)
	}

	return &newToken, nil
}

// Validate validates a token by checking expiry.
func (p *OIDCProvider) Validate(_ context.Context, token *Token) error {
	if token == nil {
		return fmt.Errorf("%w: token is nil", ErrInvalidToken)
	}
	if token.IsExpired() {
		return fmt.Errorf("%w: token expired at %s", ErrTokenExpired, token.ExpiresAt())
	}
	return nil
}

// startDeviceAuth initiates the device authorization flow.
func (p *OIDCProvider) startDeviceAuth(ctx context.Context, discovery *OIDCDiscovery) (*DeviceAuthResponse, error) {
	scopes := p.config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	form := url.Values{
		"client_id": {p.config.ClientID},
		"scope":     {strings.Join(scopes, " ")},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.DeviceAuthEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDeviceAuthFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDeviceAuthFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrDeviceAuthFailed, resp.StatusCode)
	}

	var deviceAuth DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceAuth); err != nil {
		return nil, fmt.Errorf("%w: failed to decode device auth response: %w", ErrDeviceAuthFailed, err)
	}

	return &deviceAuth, nil
}

// pollForToken polls the token endpoint until authorization is granted.
func (p *OIDCProvider) pollForToken(ctx context.Context, discovery *OIDCDiscovery, deviceAuth *DeviceAuthResponse) (*tokenResponse, error) {
	interval := p.pollInterval

	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceAuth.DeviceCode},
		"client_id":   {p.config.ClientID},
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, ctx.Err())
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("%w: failed to read token response: %w", ErrAuthenticationFailed, err)
		}

		var tokenResp tokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, fmt.Errorf("%w: failed to decode token response: %w", ErrAuthenticationFailed, err)
		}

		if resp.StatusCode == http.StatusOK && tokenResp.AccessToken != "" {
			return &tokenResp, nil
		}

		switch tokenResp.Error {
		case "authorization_pending":
			// Continue polling.
		case "slow_down":
			interval *= 2
		default:
			return nil, fmt.Errorf("%w: %s", ErrAuthenticationFailed, tokenResp.Error)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, ctx.Err())
		case <-time.After(interval):
		}
	}
}

// fetchUserinfo fetches user information from the userinfo endpoint.
func (p *OIDCProvider) fetchUserinfo(ctx context.Context, discovery *OIDCDiscovery, accessToken string) (Claims, error) {
	if discovery.UserinfoEndpoint == "" {
		return Claims{}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserinfoEndpoint, nil)
	if err != nil {
		return Claims{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Claims{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Claims{}, fmt.Errorf("userinfo returned status %d", resp.StatusCode)
	}

	var userinfo userinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userinfo); err != nil {
		return Claims{}, err
	}

	return NewClaims(
		userinfo.Subject,
		userinfo.Email,
		userinfo.Name,
		userinfo.Groups,
		p.config.Issuer,
		p.config.ClientID,
		nil,
	), nil
}
