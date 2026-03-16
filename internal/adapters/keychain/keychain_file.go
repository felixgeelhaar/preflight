// Package keychain provides OS keychain adapters for secure credential storage.
package keychain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// FileKeychain is a filesystem-based fallback keychain for environments
// without OS keychain access. Secrets are stored in a JSON file with
// restricted permissions (0600).
type FileKeychain struct {
	mu       sync.RWMutex
	filePath string
}

// NewFileKeychain creates a file-based keychain at the given path.
func NewFileKeychain(filePath string) *FileKeychain {
	return &FileKeychain{filePath: filePath}
}

// Get retrieves a secret by service and account.
func (k *FileKeychain) Get(service, account string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	store, err := k.load()
	if err != nil {
		if os.IsNotExist(err) {
			return "", ports.ErrKeychainItemNotFound
		}
		return "", err
	}

	key := storeKey(service, account)
	val, ok := store[key]
	if !ok {
		return "", ports.ErrKeychainItemNotFound
	}
	return val, nil
}

// Set stores a secret for the given service and account.
func (k *FileKeychain) Set(service, account, secret string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	store, err := k.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if store == nil {
		store = make(map[string]string)
	}

	key := storeKey(service, account)
	store[key] = secret
	return k.save(store)
}

// Delete removes a secret for the given service and account.
func (k *FileKeychain) Delete(service, account string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	store, err := k.load()
	if err != nil {
		if os.IsNotExist(err) {
			return ports.ErrKeychainItemNotFound
		}
		return err
	}

	key := storeKey(service, account)
	if _, ok := store[key]; !ok {
		return ports.ErrKeychainItemNotFound
	}
	delete(store, key)
	return k.save(store)
}

// Available always returns true for the file-based keychain.
func (k *FileKeychain) Available() bool {
	return true
}

func storeKey(service, account string) string {
	return service + ":" + account
}

func (k *FileKeychain) load() (map[string]string, error) {
	data, err := os.ReadFile(k.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("reading keychain file: %w", err)
	}

	var store map[string]string
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parsing keychain file: %w", err)
	}
	return store, nil
}

func (k *FileKeychain) save(store map[string]string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(k.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating keychain directory: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling keychain data: %w", err)
	}

	if err := os.WriteFile(k.filePath, data, 0600); err != nil {
		return fmt.Errorf("writing keychain file: %w", err)
	}
	return nil
}

// Compile-time interface check.
var _ ports.Keychain = (*FileKeychain)(nil)
