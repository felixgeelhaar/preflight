package ports_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/ports"
	"github.com/stretchr/testify/assert"
)

func TestKeychainErrors(t *testing.T) {
	t.Parallel()

	assert.EqualError(t, ports.ErrKeychainItemNotFound, "keychain item not found")
	assert.EqualError(t, ports.ErrKeychainAccessDenied, "keychain access denied")
	assert.EqualError(t, ports.ErrKeychainUnavailable, "keychain service unavailable")
}

// mockKeychain verifies the Keychain interface contract.
type mockKeychain struct {
	items     map[string]string
	available bool
}

func newMockKeychain() *mockKeychain {
	return &mockKeychain{
		items:     make(map[string]string),
		available: true,
	}
}

func (m *mockKeychain) Get(service, account string) (string, error) {
	if !m.available {
		return "", ports.ErrKeychainUnavailable
	}
	key := service + ":" + account
	val, ok := m.items[key]
	if !ok {
		return "", ports.ErrKeychainItemNotFound
	}
	return val, nil
}

func (m *mockKeychain) Set(service, account, secret string) error {
	if !m.available {
		return ports.ErrKeychainUnavailable
	}
	key := service + ":" + account
	m.items[key] = secret
	return nil
}

func (m *mockKeychain) Delete(service, account string) error {
	if !m.available {
		return ports.ErrKeychainUnavailable
	}
	key := service + ":" + account
	if _, ok := m.items[key]; !ok {
		return ports.ErrKeychainItemNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *mockKeychain) Available() bool {
	return m.available
}

// Compile-time check that mockKeychain implements Keychain.
var _ ports.Keychain = (*mockKeychain)(nil)

func TestMockKeychain_SetAndGet(t *testing.T) {
	t.Parallel()

	kc := newMockKeychain()

	err := kc.Set("preflight", "token", "secret-value")
	assert.NoError(t, err)

	val, err := kc.Get("preflight", "token")
	assert.NoError(t, err)
	assert.Equal(t, "secret-value", val)
}

func TestMockKeychain_GetNotFound(t *testing.T) {
	t.Parallel()

	kc := newMockKeychain()

	_, err := kc.Get("preflight", "missing")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestMockKeychain_Delete(t *testing.T) {
	t.Parallel()

	kc := newMockKeychain()
	_ = kc.Set("preflight", "token", "value")

	err := kc.Delete("preflight", "token")
	assert.NoError(t, err)

	_, err = kc.Get("preflight", "token")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestMockKeychain_DeleteNotFound(t *testing.T) {
	t.Parallel()

	kc := newMockKeychain()

	err := kc.Delete("preflight", "missing")
	assert.ErrorIs(t, err, ports.ErrKeychainItemNotFound)
}

func TestMockKeychain_Unavailable(t *testing.T) {
	t.Parallel()

	kc := newMockKeychain()
	kc.available = false

	assert.False(t, kc.Available())

	_, err := kc.Get("preflight", "token")
	assert.ErrorIs(t, err, ports.ErrKeychainUnavailable)

	err = kc.Set("preflight", "token", "value")
	assert.ErrorIs(t, err, ports.ErrKeychainUnavailable)

	err = kc.Delete("preflight", "token")
	assert.ErrorIs(t, err, ports.ErrKeychainUnavailable)
}
