package keychain

import (
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileKeychain_Available(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))
	assert.True(t, kc.Available())
}

func TestFileKeychain_SetAndGet(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	err := kc.Set("preflight", "token", "my-secret")
	require.NoError(t, err)

	val, err := kc.Get("preflight", "token")
	require.NoError(t, err)
	assert.Equal(t, "my-secret", val)
}

func TestFileKeychain_GetNotFound(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	_, err := kc.Get("preflight", "missing")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestFileKeychain_SetOverwrite(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	err := kc.Set("preflight", "token", "first")
	require.NoError(t, err)

	err = kc.Set("preflight", "token", "second")
	require.NoError(t, err)

	val, err := kc.Get("preflight", "token")
	require.NoError(t, err)
	assert.Equal(t, "second", val)
}

func TestFileKeychain_Delete(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	err := kc.Set("preflight", "token", "value")
	require.NoError(t, err)

	err = kc.Delete("preflight", "token")
	require.NoError(t, err)

	_, err = kc.Get("preflight", "token")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestFileKeychain_DeleteNotFound(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	// File doesn't exist yet - should get not-found via the os.IsNotExist path
	err := kc.Delete("preflight", "missing")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestFileKeychain_MultipleEntries(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	err := kc.Set("service1", "account1", "secret1")
	require.NoError(t, err)

	err = kc.Set("service2", "account2", "secret2")
	require.NoError(t, err)

	val1, err := kc.Get("service1", "account1")
	require.NoError(t, err)
	assert.Equal(t, "secret1", val1)

	val2, err := kc.Get("service2", "account2")
	require.NoError(t, err)
	assert.Equal(t, "secret2", val2)
}

func TestFileKeychain_Persistence(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "keychain.json")

	// Write with one instance
	kc1 := NewFileKeychain(filePath)
	err := kc1.Set("preflight", "token", "persistent-value")
	require.NoError(t, err)

	// Read with a different instance
	kc2 := NewFileKeychain(filePath)
	val, err := kc2.Get("preflight", "token")
	require.NoError(t, err)
	assert.Equal(t, "persistent-value", val)
}

func TestFileKeychain_CreatesParentDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "dir")
	kc := NewFileKeychain(filepath.Join(dir, "keychain.json"))

	err := kc.Set("preflight", "token", "value")
	require.NoError(t, err)

	val, err := kc.Get("preflight", "token")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestFileKeychain_DeleteFromExistingStore(t *testing.T) {
	t.Parallel()

	kc := NewFileKeychain(filepath.Join(t.TempDir(), "keychain.json"))

	// Add two items
	err := kc.Set("svc", "acct1", "val1")
	require.NoError(t, err)
	err = kc.Set("svc", "acct2", "val2")
	require.NoError(t, err)

	// Delete one
	err = kc.Delete("svc", "acct1")
	require.NoError(t, err)

	// Other still exists
	val, err := kc.Get("svc", "acct2")
	require.NoError(t, err)
	assert.Equal(t, "val2", val)

	// Deleted one is gone
	_, err = kc.Get("svc", "acct1")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}
