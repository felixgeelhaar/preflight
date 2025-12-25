package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustStore_AddAndGet(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	publisher := NewPublisher("Test", "test@example.com", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)

	err := ts.Add(key)
	require.NoError(t, err)

	got, ok := ts.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "key1", got.KeyID())
	assert.Equal(t, "Test", got.Publisher().Name())
}

func TestTrustStore_AddDuplicate(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	publisher := NewPublisher("Test", "", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)

	err := ts.Add(key)
	require.NoError(t, err)

	err = ts.Add(key)
	assert.ErrorIs(t, err, ErrKeyExists)
}

func TestTrustStore_AddNil(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	err := ts.Add(nil)
	assert.ErrorIs(t, err, ErrInvalidKeyData)
}

func TestTrustStore_AddEmptyKeyID(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	key := NewTrustedKey("", SignatureTypeSSH, nil, Publisher{})
	err := ts.Add(key)
	assert.ErrorIs(t, err, ErrInvalidKeyData)
}

func TestTrustStore_Remove(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	publisher := NewPublisher("Test", "", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)

	_ = ts.Add(key)

	err := ts.Remove("key1")
	require.NoError(t, err)

	_, ok := ts.Get("key1")
	assert.False(t, ok)
}

func TestTrustStore_RemoveNotFound(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	err := ts.Remove("nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestTrustStore_List(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	// Add keys in reverse order
	for i := 3; i >= 1; i-- {
		pub := NewPublisher("Test", "", "key"+string(rune('0'+i)), SignatureTypeSSH)
		key := NewTrustedKey("key"+string(rune('0'+i)), SignatureTypeSSH, nil, pub)
		_ = ts.Add(key)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	keys := ts.List()
	assert.Len(t, keys, 3)
	// Should be sorted by added time (oldest first)
}

func TestTrustStore_ListByType(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	sshKey := NewTrustedKey("ssh1", SignatureTypeSSH, nil, Publisher{})
	gpgKey := NewTrustedKey("gpg1", SignatureTypeGPG, nil, Publisher{})

	_ = ts.Add(sshKey)
	_ = ts.Add(gpgKey)

	sshKeys := ts.ListByType(SignatureTypeSSH)
	assert.Len(t, sshKeys, 1)
	assert.Equal(t, "ssh1", sshKeys[0].KeyID())

	gpgKeys := ts.ListByType(SignatureTypeGPG)
	assert.Len(t, gpgKeys, 1)
	assert.Equal(t, "gpg1", gpgKeys[0].KeyID())
}

func TestTrustStore_ListByTrustLevel(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	key1 := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	key1.SetTrustLevel(TrustLevelVerified)

	key2 := NewTrustedKey("key2", SignatureTypeSSH, nil, Publisher{})
	key2.SetTrustLevel(TrustLevelCommunity)

	_ = ts.Add(key1)
	_ = ts.Add(key2)

	verifiedKeys := ts.ListByTrustLevel(TrustLevelVerified)
	assert.Len(t, verifiedKeys, 1)

	communityKeys := ts.ListByTrustLevel(TrustLevelCommunity)
	assert.Len(t, communityKeys, 2)
}

func TestTrustStore_Count(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	assert.Equal(t, 0, ts.Count())

	_ = ts.Add(NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{}))
	assert.Equal(t, 1, ts.Count())

	_ = ts.Add(NewTrustedKey("key2", SignatureTypeSSH, nil, Publisher{}))
	assert.Equal(t, 2, ts.Count())
}

func TestTrustStore_IsTrusted(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	key := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	_ = ts.Add(key)

	assert.True(t, ts.IsTrusted("key1"))
	assert.False(t, ts.IsTrusted("unknown"))
}

func TestTrustStore_IsTrusted_ExpiredKey(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	key := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	key.SetExpiresAt(time.Now().Add(-time.Hour)) // Expired
	_ = ts.Add(key)

	assert.False(t, ts.IsTrusted("key1"))
}

func TestTrustStore_GetTrustLevel(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	key := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	key.SetTrustLevel(TrustLevelVerified)
	_ = ts.Add(key)

	level, ok := ts.GetTrustLevel("key1")
	assert.True(t, ok)
	assert.Equal(t, TrustLevelVerified, level)

	_, ok = ts.GetTrustLevel("unknown")
	assert.False(t, ok)
}

func TestTrustStore_SetTrustLevel(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	key := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	_ = ts.Add(key)

	err := ts.SetTrustLevel("key1", TrustLevelBuiltin)
	require.NoError(t, err)

	level, _ := ts.GetTrustLevel("key1")
	assert.Equal(t, TrustLevelBuiltin, level)
}

func TestTrustStore_SetTrustLevel_NotFound(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	err := ts.SetTrustLevel("unknown", TrustLevelBuiltin)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestTrustStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "trust.json")

	// Create and save
	ts1 := NewTrustStore(storePath)

	publisher := NewPublisher("Test Author", "test@example.com", "key1", SignatureTypeSSH)
	key := NewTrustedKey("key1", SignatureTypeSSH, nil, publisher)
	key.SetTrustLevel(TrustLevelVerified)
	key.SetComment("Test key")
	key.SetFingerprint("SHA256:abc123")

	_ = ts1.Add(key)

	err := ts1.Save()
	require.NoError(t, err)

	// Load in new store
	ts2 := NewTrustStore(storePath)
	err = ts2.Load()
	require.NoError(t, err)

	got, ok := ts2.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "key1", got.KeyID())
	assert.Equal(t, SignatureTypeSSH, got.KeyType())
	assert.Equal(t, TrustLevelVerified, got.TrustLevel())
	assert.Equal(t, "Test key", got.Comment())
	assert.Equal(t, "SHA256:abc123", got.Fingerprint())
	assert.Equal(t, "Test Author", got.Publisher().Name())
	assert.Equal(t, "test@example.com", got.Publisher().Email())
}

func TestTrustStore_LoadNonexistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "nonexistent.json")

	ts := NewTrustStore(storePath)
	err := ts.Load()
	assert.NoError(t, err) // Should not error on missing file
	assert.Equal(t, 0, ts.Count())
}

func TestTrustStore_SaveNoPath(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	err := ts.Save()
	assert.NoError(t, err) // Should not error with no path
}

func TestTrustStore_LoadNoPath(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")
	err := ts.Load()
	assert.NoError(t, err)
}

func TestTrustStore_Stats(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("")

	sshKey1 := NewTrustedKey("ssh1", SignatureTypeSSH, nil, Publisher{})
	sshKey1.SetTrustLevel(TrustLevelVerified)

	sshKey2 := NewTrustedKey("ssh2", SignatureTypeSSH, nil, Publisher{})
	sshKey2.SetTrustLevel(TrustLevelCommunity)

	gpgKey := NewTrustedKey("gpg1", SignatureTypeGPG, nil, Publisher{})
	gpgKey.SetTrustLevel(TrustLevelBuiltin)

	expiredKey := NewTrustedKey("expired", SignatureTypeSigstore, nil, Publisher{})
	expiredKey.SetExpiresAt(time.Now().Add(-time.Hour))

	_ = ts.Add(sshKey1)
	_ = ts.Add(sshKey2)
	_ = ts.Add(gpgKey)
	_ = ts.Add(expiredKey)

	stats := ts.Stats()

	assert.Equal(t, 4, stats.TotalKeys)
	assert.Equal(t, 2, stats.SSHKeys)
	assert.Equal(t, 1, stats.GPGKeys)
	assert.Equal(t, 1, stats.SigstoreKeys)
	assert.Equal(t, 1, stats.BuiltinLevel)
	assert.Equal(t, 1, stats.VerifiedLevel)
	assert.Equal(t, 2, stats.CommunityLevel) // expiredKey defaults to community
	assert.Equal(t, 1, stats.ExpiredKeys)
}

func TestTrustStore_SaveWithExpiration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "trust.json")

	ts1 := NewTrustStore(storePath)

	key := NewTrustedKey("key1", SignatureTypeSSH, nil, Publisher{})
	expires := time.Now().Add(24 * time.Hour)
	key.SetExpiresAt(expires)

	_ = ts1.Add(key)
	_ = ts1.Save()

	ts2 := NewTrustStore(storePath)
	_ = ts2.Load()

	got, _ := ts2.Get("key1")
	assert.False(t, got.ExpiresAt().IsZero())
	assert.WithinDuration(t, expires, got.ExpiresAt(), time.Second)
}

func TestTrustStore_StorePath(t *testing.T) {
	t.Parallel()

	ts := NewTrustStore("/some/path/trust.json")
	assert.Equal(t, "/some/path/trust.json", ts.StorePath())
}

func TestTrustStore_LoadInvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "trust.json")

	err := os.WriteFile(storePath, []byte("invalid json"), 0o600)
	require.NoError(t, err)

	ts := NewTrustStore(storePath)
	err = ts.Load()
	assert.Error(t, err)
}
