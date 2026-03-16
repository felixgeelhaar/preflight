package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	requests  []*http.Request
}

func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
	}
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	key := req.Method + " " + req.URL.String()
	if err, ok := m.errors[key]; ok {
		return nil, err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	// Fall back to URL-only match for POST requests with query params.
	key = req.Method + " " + req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
	if err, ok := m.errors[key]; ok {
		return nil, err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func (m *mockHTTPClient) addResponse(method, url string, statusCode int, body interface{}) {
	data, _ := json.Marshal(body)
	m.responses[method+" "+url] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
}

func (m *mockHTTPClient) addError(method, url string, err error) {
	m.errors[method+" "+url] = err
}

func validDiscovery() OIDCDiscovery {
	return OIDCDiscovery{
		Issuer:             "https://auth.example.com",
		TokenEndpoint:      "https://auth.example.com/oauth/token",
		UserinfoEndpoint:   "https://auth.example.com/userinfo",
		DeviceAuthEndpoint: "https://auth.example.com/oauth/device/code",
		JwksURI:            "https://auth.example.com/.well-known/jwks.json",
	}
}

func TestNewOIDCProvider_Valid(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
		Scopes:   []string{"openid", "profile"},
	}

	provider, err := NewOIDCProvider(cfg, newMockHTTPClient())
	require.NoError(t, err)
	assert.Equal(t, "corporate", provider.Name())
	assert.Equal(t, ProviderTypeOIDC, provider.Type())
}

func TestNewOIDCProvider_InvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name: "bad",
		Type: ProviderTypeOIDC,
		// Missing issuer and client ID.
	}

	_, err := NewOIDCProvider(cfg, newMockHTTPClient())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewOIDCProvider_NilHTTPClient(t *testing.T) {
	t.Parallel()

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	_, err := NewOIDCProvider(cfg, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestOIDCProvider_Discover_Success(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusOK, validDiscovery())

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	discovery, err := provider.Discover(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "https://auth.example.com/oauth/token", discovery.TokenEndpoint)
	assert.Equal(t, "https://auth.example.com/oauth/device/code", discovery.DeviceAuthEndpoint)
}

func TestOIDCProvider_Discover_HTTPError(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addError("GET", "https://auth.example.com/.well-known/openid-configuration", fmt.Errorf("connection refused"))

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	_, err = provider.Discover(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDiscoveryFailed)
}

func TestOIDCProvider_Discover_NonOKStatus(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusInternalServerError, "error")

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	_, err = provider.Discover(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDiscoveryFailed)
}

func TestOIDCProvider_Authenticate_Success(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()

	// Discovery.
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusOK, validDiscovery())

	// Device authorization.
	client.addResponse("POST", "https://auth.example.com/oauth/device/code", http.StatusOK, DeviceAuthResponse{
		DeviceCode:              "dev-code-123",
		UserCode:                "ABCD-1234",
		VerificationURI:         "https://auth.example.com/activate",
		VerificationURIComplete: "https://auth.example.com/activate?user_code=ABCD-1234",
		ExpiresIn:               600,
		Interval:                1,
	})

	// Token response (immediate success, no slow_down/pending).
	tokenResp := map[string]interface{}{
		"access_token":  "access-token-xyz",
		"refresh_token": "refresh-token-xyz",
		"token_type":    "Bearer",
		"expires_in":    3600,
	}
	client.addResponse("POST", "https://auth.example.com/oauth/token", http.StatusOK, tokenResp)

	// Userinfo.
	client.addResponse("GET", "https://auth.example.com/userinfo", http.StatusOK, map[string]interface{}{
		"sub":    "user-123",
		"email":  "user@example.com",
		"name":   "Test User",
		"groups": []string{"admin"},
	})

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
		Scopes:   []string{"openid", "profile", "email"},
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	// Override poll interval for testing.
	provider.pollInterval = 1 * time.Millisecond

	token, err := provider.Authenticate(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "access-token-xyz", token.AccessToken())
	assert.Equal(t, "refresh-token-xyz", token.RefreshToken())
	assert.Equal(t, "Bearer", token.TokenType())
	assert.Equal(t, "user-123", token.Claims().Subject())
	assert.Equal(t, "user@example.com", token.Claims().Email())
}

func TestOIDCProvider_Authenticate_DeviceAuthFailed(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusOK, validDiscovery())
	client.addResponse("POST", "https://auth.example.com/oauth/device/code", http.StatusBadRequest, map[string]string{"error": "bad_request"})

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	_, err = provider.Authenticate(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDeviceAuthFailed)
}

func TestOIDCProvider_Authenticate_ContextCanceled(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusOK, validDiscovery())
	client.addResponse("POST", "https://auth.example.com/oauth/device/code", http.StatusOK, DeviceAuthResponse{
		DeviceCode:      "dev-code-123",
		UserCode:        "ABCD-1234",
		VerificationURI: "https://auth.example.com/activate",
		ExpiresIn:       600,
		Interval:        1,
	})

	// Token endpoint returns authorization_pending.
	pendingResp := map[string]string{"error": "authorization_pending"}
	client.addResponse("POST", "https://auth.example.com/oauth/token", http.StatusBadRequest, pendingResp)

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)
	provider.pollInterval = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err = provider.Authenticate(ctx)
	require.Error(t, err)
}

func TestOIDCProvider_Refresh_Success(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	client.addResponse("GET", "https://auth.example.com/.well-known/openid-configuration", http.StatusOK, validDiscovery())

	tokenResp := map[string]interface{}{
		"access_token":  "new-access-token",
		"refresh_token": "new-refresh-token",
		"token_type":    "Bearer",
		"expires_in":    3600,
	}
	client.addResponse("POST", "https://auth.example.com/oauth/token", http.StatusOK, tokenResp)

	client.addResponse("GET", "https://auth.example.com/userinfo", http.StatusOK, map[string]interface{}{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "Test User",
	})

	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	oldToken, err := NewToken("old-access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corporate")
	require.NoError(t, err)
	oldToken = oldToken.WithRefreshToken("old-refresh")

	newToken, err := provider.Refresh(context.Background(), &oldToken)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", newToken.AccessToken())
	assert.Equal(t, "new-refresh-token", newToken.RefreshToken())
}

func TestOIDCProvider_Refresh_NoRefreshToken(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	oldToken, err := NewToken("old-access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corporate")
	require.NoError(t, err)

	_, err = provider.Refresh(context.Background(), &oldToken)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenRefreshFailed)
}

func TestOIDCProvider_Validate_Valid(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	token, err := NewToken("access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corporate")
	require.NoError(t, err)

	err = provider.Validate(context.Background(), &token)
	require.NoError(t, err)
}

func TestOIDCProvider_Validate_Expired(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	token, err := NewToken("access", "Bearer", time.Now().Add(-1*time.Hour), Claims{}, "corporate")
	require.NoError(t, err)

	err = provider.Validate(context.Background(), &token)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestOIDCProvider_Validate_NilToken(t *testing.T) {
	t.Parallel()

	client := newMockHTTPClient()
	cfg := ProviderConfig{
		Name:     "corporate",
		Type:     ProviderTypeOIDC,
		Issuer:   "https://auth.example.com",
		ClientID: "my-client-id",
	}

	provider, err := NewOIDCProvider(cfg, client)
	require.NoError(t, err)

	err = provider.Validate(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
