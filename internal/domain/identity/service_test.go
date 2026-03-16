package identity

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name         string
	providerType ProviderType
	authToken    *Token
	authErr      error
	refreshToken *Token
	refreshErr   error
	validateErr  error
}

func (m *mockProvider) Name() string       { return m.name }
func (m *mockProvider) Type() ProviderType { return m.providerType }

func (m *mockProvider) Authenticate(_ context.Context) (*Token, error) {
	if m.authErr != nil {
		return nil, m.authErr
	}
	return m.authToken, nil
}

func (m *mockProvider) Refresh(_ context.Context, _ *Token) (*Token, error) {
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	return m.refreshToken, nil
}

func (m *mockProvider) Validate(_ context.Context, _ *Token) error {
	return m.validateErr
}

func newTestToken(t *testing.T, providerName string) Token {
	t.Helper()
	claims := NewClaims("sub-123", "user@example.com", "Alice", []string{"admin"}, "https://issuer.com", "client-id", nil)
	token, err := NewToken("access-token", "Bearer", time.Now().Add(1*time.Hour), claims, providerName)
	require.NoError(t, err)
	return token
}

func TestNewService(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)
	assert.NotNil(t, svc)
}

func TestService_RegisterProvider(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	provider := &mockProvider{name: "corporate", providerType: ProviderTypeOIDC}

	err := svc.RegisterProvider(provider)
	require.NoError(t, err)

	providers := svc.ListProviders()
	assert.Equal(t, []string{"corporate"}, providers)
}

func TestService_RegisterProvider_Duplicate(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	provider := &mockProvider{name: "corporate", providerType: ProviderTypeOIDC}

	require.NoError(t, svc.RegisterProvider(provider))

	err := svc.RegisterProvider(provider)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderExists)
}

func TestService_ListProviders_Sorted(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	require.NoError(t, svc.RegisterProvider(&mockProvider{name: "charlie"}))
	require.NoError(t, svc.RegisterProvider(&mockProvider{name: "alpha"}))
	require.NoError(t, svc.RegisterProvider(&mockProvider{name: "bravo"}))

	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, svc.ListProviders())
}

func TestService_ListProviders_Empty(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	providers := svc.ListProviders()
	assert.Empty(t, providers)
}

func TestService_Login_Success(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	token := newTestToken(t, "corporate")
	provider := &mockProvider{
		name:         "corporate",
		providerType: ProviderTypeOIDC,
		authToken:    &token,
	}

	require.NoError(t, svc.RegisterProvider(provider))

	result, err := svc.Login(context.Background(), "corporate")
	require.NoError(t, err)
	assert.Equal(t, "access-token", result.AccessToken())

	// Token should be persisted.
	loaded, err := store.Load("corporate")
	require.NoError(t, err)
	assert.Equal(t, "access-token", loaded.AccessToken())
}

func TestService_Login_ProviderNotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	_, err := svc.Login(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestService_Login_AuthFailed(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	provider := &mockProvider{
		name:    "corporate",
		authErr: fmt.Errorf("%w: access denied", ErrAuthenticationFailed),
	}

	require.NoError(t, svc.RegisterProvider(provider))

	_, err := svc.Login(context.Background(), "corporate")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAuthenticationFailed)
}

func TestService_Logout(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	token := newTestToken(t, "corporate")
	provider := &mockProvider{name: "corporate", authToken: &token}

	require.NoError(t, svc.RegisterProvider(provider))
	_, err := svc.Login(context.Background(), "corporate")
	require.NoError(t, err)

	err = svc.Logout("corporate")
	require.NoError(t, err)

	// Token should be removed.
	_, err = store.Load("corporate")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestService_Logout_ProviderNotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	err := svc.Logout("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestService_Status_Authenticated(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	token := newTestToken(t, "corporate")
	provider := &mockProvider{name: "corporate", authToken: &token}

	require.NoError(t, svc.RegisterProvider(provider))
	_, err := svc.Login(context.Background(), "corporate")
	require.NoError(t, err)

	status, err := svc.Status("corporate")
	require.NoError(t, err)
	assert.Equal(t, "access-token", status.AccessToken())
}

func TestService_Status_NotAuthenticated(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	require.NoError(t, svc.RegisterProvider(&mockProvider{name: "corporate"}))

	_, err := svc.Status("corporate")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestService_Status_ProviderNotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	_, err := svc.Status("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestService_WhoAmI(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	token := newTestToken(t, "corporate")
	provider := &mockProvider{name: "corporate", authToken: &token}

	require.NoError(t, svc.RegisterProvider(provider))
	_, err := svc.Login(context.Background(), "corporate")
	require.NoError(t, err)

	claims, err := svc.WhoAmI("corporate")
	require.NoError(t, err)
	assert.Equal(t, "sub-123", claims.Subject())
	assert.Equal(t, "user@example.com", claims.Email())
}

func TestService_WhoAmI_NotAuthenticated(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	require.NoError(t, svc.RegisterProvider(&mockProvider{name: "corporate"}))

	_, err := svc.WhoAmI("corporate")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestService_CurrentToken(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	token := newTestToken(t, "corporate")
	provider := &mockProvider{name: "corporate", authToken: &token}

	require.NoError(t, svc.RegisterProvider(provider))
	_, err := svc.Login(context.Background(), "corporate")
	require.NoError(t, err)

	current, err := svc.CurrentToken("corporate")
	require.NoError(t, err)
	assert.Equal(t, "access-token", current.AccessToken())
}

func TestService_CurrentToken_ProviderNotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	_, err := svc.CurrentToken("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	svc := NewService(store)

	// Create tokens and providers before spawning goroutines to avoid
	// calling require (which uses t.FailNow) from non-test goroutines.
	mockProviders := make([]*mockProvider, 10)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("provider-%d", i)
		token := newTestToken(t, name)
		mockProviders[i] = &mockProvider{name: name, authToken: &token}
	}

	// Register providers concurrently.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = svc.RegisterProvider(mockProviders[idx])
		}(i)
	}
	wg.Wait()

	providers := svc.ListProviders()
	assert.Len(t, providers, 10)

	// Verify sorted.
	assert.True(t, sort.StringsAreSorted(providers))
}
