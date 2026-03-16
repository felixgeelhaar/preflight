// Package ports defines interfaces for external dependencies.
package ports

import "errors"

// Keychain errors.
var (
	ErrKeychainItemNotFound = errors.New("keychain item not found")
	ErrKeychainAccessDenied = errors.New("keychain access denied")
	ErrKeychainUnavailable  = errors.New("keychain service unavailable")
)

// Keychain provides secure credential storage using the OS keychain.
// Implementations exist for macOS Keychain, Linux secret-service (D-Bus),
// and Windows Credential Manager.
type Keychain interface {
	// Get retrieves a secret by service and account.
	Get(service, account string) (string, error)

	// Set stores a secret for the given service and account.
	Set(service, account, secret string) error

	// Delete removes a secret for the given service and account.
	Delete(service, account string) error

	// Available returns true if the keychain backend is accessible.
	Available() bool
}
