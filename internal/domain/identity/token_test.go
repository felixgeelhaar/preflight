package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Claims tests ---

func TestNewClaims_AllFields(t *testing.T) {
	t.Parallel()

	claims := NewClaims("sub-123", "user@example.com", "Alice", []string{"admin", "dev"}, "https://issuer.example.com", "client-id", map[string]string{"org": "acme"})

	assert.Equal(t, "sub-123", claims.Subject())
	assert.Equal(t, "user@example.com", claims.Email())
	assert.Equal(t, "Alice", claims.Name())
	assert.Equal(t, []string{"admin", "dev"}, claims.Groups())
	assert.Equal(t, "https://issuer.example.com", claims.Issuer())
	assert.Equal(t, "client-id", claims.Audience())
	assert.Equal(t, map[string]string{"org": "acme"}, claims.Extra())
}

func TestClaims_IsZero(t *testing.T) {
	t.Parallel()

	var zero Claims
	assert.True(t, zero.IsZero())

	nonZero := NewClaims("sub-123", "", "", nil, "", "", nil)
	assert.False(t, nonZero.IsZero())
}

func TestClaims_HasGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		groups   []string
		group    string
		expected bool
	}{
		{"found", []string{"admin", "dev"}, "admin", true},
		{"not found", []string{"admin", "dev"}, "ops", false},
		{"empty groups", nil, "admin", false},
		{"empty search", []string{"admin"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := NewClaims("sub", "", "", tt.groups, "", "", nil)
			assert.Equal(t, tt.expected, claims.HasGroup(tt.group))
		})
	}
}

func TestClaims_GroupsImmutability(t *testing.T) {
	t.Parallel()

	groups := []string{"admin", "dev"}
	claims := NewClaims("sub", "", "", groups, "", "", nil)

	// Mutate the original slice.
	groups[0] = "mutated"
	assert.Equal(t, "admin", claims.Groups()[0], "original mutation should not affect claims")

	// Mutate the returned slice.
	returned := claims.Groups()
	returned[0] = "mutated"
	assert.Equal(t, "admin", claims.Groups()[0], "returned slice mutation should not affect claims")
}

func TestClaims_ExtraImmutability(t *testing.T) {
	t.Parallel()

	extra := map[string]string{"org": "acme"}
	claims := NewClaims("sub", "", "", nil, "", "", extra)

	// Mutate the original map.
	extra["org"] = "mutated"
	assert.Equal(t, "acme", claims.Extra()["org"], "original mutation should not affect claims")

	// Mutate the returned map.
	returned := claims.Extra()
	returned["org"] = "mutated"
	assert.Equal(t, "acme", claims.Extra()["org"], "returned map mutation should not affect claims")
}

// --- Token tests ---

func TestNewToken_Valid(t *testing.T) {
	t.Parallel()

	expiry := time.Now().Add(1 * time.Hour)
	claims := NewClaims("sub-123", "user@example.com", "Alice", nil, "", "", nil)

	token, err := NewToken("access-token-xyz", "Bearer", expiry, claims, "corporate")

	require.NoError(t, err)
	assert.Equal(t, "access-token-xyz", token.AccessToken())
	assert.Equal(t, "Bearer", token.TokenType())
	assert.Equal(t, expiry, token.ExpiresAt())
	assert.Equal(t, "sub-123", token.Claims().Subject())
	assert.Equal(t, "corporate", token.ProviderName())
	assert.Empty(t, token.RefreshToken())
}

func TestNewToken_EmptyAccessToken(t *testing.T) {
	t.Parallel()

	_, err := NewToken("", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "provider")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestNewToken_ZeroExpiry(t *testing.T) {
	t.Parallel()

	_, err := NewToken("access", "Bearer", time.Time{}, Claims{}, "provider")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestToken_IsExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expiry   time.Time
		expected bool
	}{
		{"future", time.Now().Add(1 * time.Hour), false},
		{"past", time.Now().Add(-1 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			token, err := NewToken("access", "Bearer", tt.expiry, Claims{}, "p")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, token.IsExpired())
		})
	}
}

func TestToken_NeedsRefresh(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiry    time.Time
		threshold time.Duration
		expected  bool
	}{
		{"well before threshold", time.Now().Add(2 * time.Hour), 30 * time.Minute, false},
		{"within threshold", time.Now().Add(10 * time.Minute), 30 * time.Minute, true},
		{"already expired", time.Now().Add(-1 * time.Minute), 30 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			token, err := NewToken("access", "Bearer", tt.expiry, Claims{}, "p")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, token.NeedsRefresh(tt.threshold))
		})
	}
}

func TestToken_WithRefreshToken(t *testing.T) {
	t.Parallel()

	token, err := NewToken("access", "Bearer", time.Now().Add(1*time.Hour), Claims{}, "p")
	require.NoError(t, err)
	assert.Empty(t, token.RefreshToken())

	withRefresh := token.WithRefreshToken("refresh-xyz")
	assert.Equal(t, "refresh-xyz", withRefresh.RefreshToken())
	assert.Empty(t, token.RefreshToken(), "original token should not be mutated")
	assert.Equal(t, "access", withRefresh.AccessToken())
}

func TestToken_Clone(t *testing.T) {
	t.Parallel()

	claims := NewClaims("sub", "email@test.com", "Name", []string{"g1"}, "iss", "aud", map[string]string{"k": "v"})
	expiry := time.Now().Add(1 * time.Hour)
	token, err := NewToken("access", "Bearer", expiry, claims, "provider")
	require.NoError(t, err)
	token = token.WithRefreshToken("refresh")

	cloned := token.Clone()

	assert.Equal(t, token.AccessToken(), cloned.AccessToken())
	assert.Equal(t, token.RefreshToken(), cloned.RefreshToken())
	assert.Equal(t, token.TokenType(), cloned.TokenType())
	assert.Equal(t, token.ExpiresAt(), cloned.ExpiresAt())
	assert.Equal(t, token.ProviderName(), cloned.ProviderName())
	assert.Equal(t, token.Claims().Subject(), cloned.Claims().Subject())
	assert.Equal(t, token.Claims().Groups(), cloned.Claims().Groups())
	assert.Equal(t, token.Claims().Extra(), cloned.Claims().Extra())
}
