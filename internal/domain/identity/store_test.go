package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenStore(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())
	assert.NotNil(t, store)
}

func TestTokenStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	claims := NewClaims("sub-123", "user@example.com", "Alice", []string{"admin"}, "https://issuer.com", "client-id", map[string]string{"org": "acme"})
	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second) // Truncate for JSON round-trip.
	token, err := NewToken("access-xyz", "Bearer", expiry, claims, "corporate")
	require.NoError(t, err)
	token = token.WithRefreshToken("refresh-xyz")

	err = store.Save("corporate", &token)
	require.NoError(t, err)

	loaded, err := store.Load("corporate")
	require.NoError(t, err)

	assert.Equal(t, "access-xyz", loaded.AccessToken())
	assert.Equal(t, "refresh-xyz", loaded.RefreshToken())
	assert.Equal(t, "Bearer", loaded.TokenType())
	assert.Equal(t, expiry.UTC(), loaded.ExpiresAt().UTC())
	assert.Equal(t, "corporate", loaded.ProviderName())
	assert.Equal(t, "sub-123", loaded.Claims().Subject())
	assert.Equal(t, "user@example.com", loaded.Claims().Email())
	assert.Equal(t, "Alice", loaded.Claims().Name())
	assert.Equal(t, []string{"admin"}, loaded.Claims().Groups())
	assert.Equal(t, map[string]string{"org": "acme"}, loaded.Claims().Extra())
}

func TestTokenStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	_, err := store.Load("nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestTokenStore_Delete(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	token, err := NewToken("access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corp")
	require.NoError(t, err)

	err = store.Save("corp", &token)
	require.NoError(t, err)

	err = store.Delete("corp")
	require.NoError(t, err)

	_, err = store.Load("corp")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

func TestTokenStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	err := store.Delete("nonexistent")
	require.NoError(t, err, "deleting nonexistent token should not error")
}

func TestTokenStore_List_Empty(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	providers, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, providers)
}

func TestTokenStore_List_Multiple(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	token1, err := NewToken("access1", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "alpha")
	require.NoError(t, err)

	token2, err := NewToken("access2", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "beta")
	require.NoError(t, err)

	require.NoError(t, store.Save("alpha", &token1))
	require.NoError(t, store.Save("beta", &token2))

	providers, err := store.List()
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "alpha")
	assert.Contains(t, providers, "beta")
}

func TestTokenStore_List_Sorted(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	token, err := NewToken("access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "p")
	require.NoError(t, err)

	require.NoError(t, store.Save("charlie", &token))
	require.NoError(t, store.Save("alpha", &token))
	require.NoError(t, store.Save("bravo", &token))

	providers, err := store.List()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, providers)
}

func TestTokenStore_FilePermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewTokenStore(dir)

	token, err := NewToken("access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corp")
	require.NoError(t, err)

	require.NoError(t, store.Save("corp", &token))

	tokenFile := filepath.Join(dir, "identity", "corp.json")
	info, err := os.Stat(tokenFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "token file should have 0600 permissions")
}

func TestTokenStore_Save_Overwrite(t *testing.T) {
	t.Parallel()

	store := NewTokenStore(t.TempDir())

	token1, err := NewToken("first", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "corp")
	require.NoError(t, err)

	token2, err := NewToken("second", "Bearer", time.Now().Add(2*time.Hour), Claims{}, "corp")
	require.NoError(t, err)

	require.NoError(t, store.Save("corp", &token1))
	require.NoError(t, store.Save("corp", &token2))

	loaded, err := store.Load("corp")
	require.NoError(t, err)
	assert.Equal(t, "second", loaded.AccessToken())
}
