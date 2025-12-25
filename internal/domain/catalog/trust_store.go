package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// TrustStore errors.
var (
	ErrKeyNotFound    = errors.New("key not found in trust store")
	ErrKeyExists      = errors.New("key already exists in trust store")
	ErrInvalidKeyData = errors.New("invalid key data")
)

// TrustStore manages trusted public keys for signature verification.
type TrustStore struct {
	mu        sync.RWMutex
	keys      map[string]*TrustedKey
	storePath string
}

// NewTrustStore creates a new trust store.
func NewTrustStore(storePath string) *TrustStore {
	return &TrustStore{
		keys:      make(map[string]*TrustedKey),
		storePath: storePath,
	}
}

// Add adds a trusted key to the store.
func (ts *TrustStore) Add(key *TrustedKey) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if key == nil {
		return fmt.Errorf("%w: key is nil", ErrInvalidKeyData)
	}

	if key.KeyID() == "" {
		return fmt.Errorf("%w: key ID is empty", ErrInvalidKeyData)
	}

	if _, exists := ts.keys[key.KeyID()]; exists {
		return fmt.Errorf("%w: %s", ErrKeyExists, key.KeyID())
	}

	ts.keys[key.KeyID()] = key
	return nil
}

// Remove removes a key from the trust store.
func (ts *TrustStore) Remove(keyID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, exists := ts.keys[keyID]; !exists {
		return fmt.Errorf("%w: %s", ErrKeyNotFound, keyID)
	}

	delete(ts.keys, keyID)
	return nil
}

// Get returns a trusted key by ID.
func (ts *TrustStore) Get(keyID string) (*TrustedKey, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	key, ok := ts.keys[keyID]
	return key, ok
}

// List returns all trusted keys, sorted by added date.
func (ts *TrustStore) List() []*TrustedKey {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]*TrustedKey, 0, len(ts.keys))
	for _, key := range ts.keys {
		result = append(result, key)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AddedAt().Before(result[j].AddedAt())
	})

	return result
}

// ListByType returns keys filtered by signature type.
func (ts *TrustStore) ListByType(sigType SignatureType) []*TrustedKey {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []*TrustedKey
	for _, key := range ts.keys {
		if key.KeyType() == sigType {
			result = append(result, key)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AddedAt().Before(result[j].AddedAt())
	})

	return result
}

// ListByTrustLevel returns keys with at least the given trust level.
func (ts *TrustStore) ListByTrustLevel(minLevel TrustLevel) []*TrustedKey {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var result []*TrustedKey
	for _, key := range ts.keys {
		if key.TrustLevel().IsAtLeast(minLevel) {
			result = append(result, key)
		}
	}

	return result
}

// Count returns the number of trusted keys.
func (ts *TrustStore) Count() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.keys)
}

// IsTrusted checks if a key ID is trusted.
func (ts *TrustStore) IsTrusted(keyID string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	key, exists := ts.keys[keyID]
	if !exists {
		return false
	}

	// Expired keys are not trusted
	return !key.IsExpired()
}

// GetTrustLevel returns the trust level for a key.
func (ts *TrustStore) GetTrustLevel(keyID string) (TrustLevel, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	key, exists := ts.keys[keyID]
	if !exists {
		return TrustLevelUntrusted, false
	}

	return key.TrustLevel(), true
}

// SetTrustLevel updates the trust level for a key.
func (ts *TrustStore) SetTrustLevel(keyID string, level TrustLevel) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	key, exists := ts.keys[keyID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrKeyNotFound, keyID)
	}

	key.SetTrustLevel(level)
	return nil
}

// StorePath returns the store path.
func (ts *TrustStore) StorePath() string {
	return ts.storePath
}

// trustedKeyJSON is the JSON representation of a trusted key.
type trustedKeyJSON struct {
	KeyID       string        `json:"key_id"`
	KeyType     SignatureType `json:"key_type"`
	Fingerprint string        `json:"fingerprint"`
	Publisher   publisherJSON `json:"publisher"`
	TrustLevel  TrustLevel    `json:"trust_level"`
	AddedAt     time.Time     `json:"added_at"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
	Comment     string        `json:"comment,omitempty"`
}

type publisherJSON struct {
	Name    string        `json:"name"`
	Email   string        `json:"email"`
	KeyID   string        `json:"key_id"`
	KeyType SignatureType `json:"key_type"`
}

type trustStoreJSON struct {
	Version string           `json:"version"`
	Keys    []trustedKeyJSON `json:"keys"`
}

// Save persists the trust store to disk.
func (ts *TrustStore) Save() error {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.storePath == "" {
		return nil // No path configured
	}

	// Ensure directory exists
	dir := filepath.Dir(ts.storePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create trust store directory: %w", err)
	}

	// Build JSON structure
	store := trustStoreJSON{
		Version: "1.0",
		Keys:    make([]trustedKeyJSON, 0, len(ts.keys)),
	}

	for _, key := range ts.keys {
		keyJSON := trustedKeyJSON{
			KeyID:       key.KeyID(),
			KeyType:     key.KeyType(),
			Fingerprint: key.Fingerprint(),
			Publisher: publisherJSON{
				Name:    key.Publisher().Name(),
				Email:   key.Publisher().Email(),
				KeyID:   key.Publisher().KeyID(),
				KeyType: key.Publisher().KeyType(),
			},
			TrustLevel: key.TrustLevel(),
			AddedAt:    key.AddedAt(),
			Comment:    key.Comment(),
		}
		if !key.ExpiresAt().IsZero() {
			expires := key.ExpiresAt()
			keyJSON.ExpiresAt = &expires
		}
		store.Keys = append(store.Keys, keyJSON)
	}

	// Write to file
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trust store: %w", err)
	}

	if err := os.WriteFile(ts.storePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write trust store: %w", err)
	}

	return nil
}

// Load loads the trust store from disk.
func (ts *TrustStore) Load() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.storePath == "" {
		return nil
	}

	data, err := os.ReadFile(ts.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No store file yet
		}
		return fmt.Errorf("failed to read trust store: %w", err)
	}

	var store trustStoreJSON
	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("failed to unmarshal trust store: %w", err)
	}

	// Load keys
	ts.keys = make(map[string]*TrustedKey, len(store.Keys))
	for _, keyJSON := range store.Keys {
		publisher := NewPublisher(
			keyJSON.Publisher.Name,
			keyJSON.Publisher.Email,
			keyJSON.Publisher.KeyID,
			keyJSON.Publisher.KeyType,
		)

		key := &TrustedKey{
			keyID:       keyJSON.KeyID,
			keyType:     keyJSON.KeyType,
			fingerprint: keyJSON.Fingerprint,
			publisher:   publisher,
			trustLevel:  keyJSON.TrustLevel,
			addedAt:     keyJSON.AddedAt,
			comment:     keyJSON.Comment,
		}
		if keyJSON.ExpiresAt != nil {
			key.expiresAt = *keyJSON.ExpiresAt
		}

		ts.keys[key.KeyID()] = key
	}

	return nil
}

// Stats returns trust store statistics.
func (ts *TrustStore) Stats() TrustStoreStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	stats := TrustStoreStats{
		TotalKeys: len(ts.keys),
	}

	for _, key := range ts.keys {
		switch key.KeyType() {
		case SignatureTypeGPG:
			stats.GPGKeys++
		case SignatureTypeSSH:
			stats.SSHKeys++
		case SignatureTypeSigstore:
			stats.SigstoreKeys++
		}

		switch key.TrustLevel() {
		case TrustLevelBuiltin:
			stats.BuiltinLevel++
		case TrustLevelVerified:
			stats.VerifiedLevel++
		case TrustLevelCommunity:
			stats.CommunityLevel++
		case TrustLevelUntrusted:
			// Untrusted keys are not counted in any trust level bucket
		}

		if key.IsExpired() {
			stats.ExpiredKeys++
		}
	}

	return stats
}

// TrustStoreStats contains trust store statistics.
type TrustStoreStats struct {
	TotalKeys      int
	GPGKeys        int
	SSHKeys        int
	SigstoreKeys   int
	BuiltinLevel   int
	VerifiedLevel  int
	CommunityLevel int
	ExpiredKeys    int
}
