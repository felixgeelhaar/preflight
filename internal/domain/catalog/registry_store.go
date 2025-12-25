package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store errors.
var (
	ErrStoreCorrupted = errors.New("registry store corrupted")
)

// RegistryStoreConfig configures the registry store.
type RegistryStoreConfig struct {
	// BasePath is the directory for registry files
	BasePath string
}

// DefaultRegistryStoreConfig returns sensible defaults.
func DefaultRegistryStoreConfig() RegistryStoreConfig {
	homeDir, _ := os.UserHomeDir()
	return RegistryStoreConfig{
		BasePath: filepath.Join(homeDir, ".preflight", "catalogs"),
	}
}

// StoredSource represents a catalog source persisted to disk.
type StoredSource struct {
	Name       string     `json:"name"`
	Type       SourceType `json:"type"`
	Location   string     `json:"location"`
	AddedAt    time.Time  `json:"added_at"`
	Enabled    bool       `json:"enabled"`
	LastVerify time.Time  `json:"last_verify,omitempty"`
	Trusted    bool       `json:"trusted,omitempty"`
}

// ToSource converts a StoredSource back to a Source.
func (s StoredSource) ToSource() (Source, error) {
	switch s.Type {
	case SourceTypeURL:
		return NewURLSource(s.Name, s.Location)
	case SourceTypeLocal:
		return NewLocalSource(s.Name, s.Location)
	default:
		return Source{}, fmt.Errorf("%w: unknown source type %s", ErrInvalidSource, s.Type)
	}
}

// StoredSourceFromSource creates a StoredSource from a Source.
func StoredSourceFromSource(src Source) StoredSource {
	return StoredSource{
		Name:     src.Name(),
		Type:     src.Type(),
		Location: src.Location(),
		AddedAt:  time.Now(),
		Enabled:  true,
		Trusted:  false,
	}
}

// RegistryData is the complete registry file structure.
type RegistryData struct {
	Version int            `json:"version"`
	Sources []StoredSource `json:"sources"`
}

// RegistryStore provides persistence for external catalog sources.
type RegistryStore struct {
	config RegistryStoreConfig
	mu     sync.RWMutex
}

// NewRegistryStore creates a new registry store.
func NewRegistryStore(config RegistryStoreConfig) *RegistryStore {
	return &RegistryStore{config: config}
}

// registryPath returns the path to the registry file.
func (s *RegistryStore) registryPath() string {
	return filepath.Join(s.config.BasePath, "registry.json")
}

// EnsureDir ensures the registry directory exists.
func (s *RegistryStore) EnsureDir() error {
	return os.MkdirAll(s.config.BasePath, 0o755)
}

// Load reads all stored sources from disk.
func (s *RegistryStore) Load() ([]StoredSource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []StoredSource{}, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry RegistryData
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStoreCorrupted, err)
	}

	return registry.Sources, nil
}

// Save writes all sources to disk.
func (s *RegistryStore) Save(sources []StoredSource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create registry dir: %w", err)
	}

	registry := RegistryData{
		Version: 1,
		Sources: sources,
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(s.registryPath(), data, 0o644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// Add adds a new source to the registry.
func (s *RegistryStore) Add(src Source) error {
	sources, err := s.Load()
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, existing := range sources {
		if existing.Name == src.Name() {
			return fmt.Errorf("catalog '%s' already exists", src.Name())
		}
	}

	sources = append(sources, StoredSourceFromSource(src))
	return s.Save(sources)
}

// Remove removes a source from the registry by name.
func (s *RegistryStore) Remove(name string) error {
	sources, err := s.Load()
	if err != nil {
		return err
	}

	filtered := make([]StoredSource, 0, len(sources))
	found := false
	for _, src := range sources {
		if src.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, src)
	}

	if !found {
		return fmt.Errorf("catalog '%s' not found", name)
	}

	return s.Save(filtered)
}

// Get retrieves a source by name.
func (s *RegistryStore) Get(name string) (StoredSource, bool, error) {
	sources, err := s.Load()
	if err != nil {
		return StoredSource{}, false, err
	}

	for _, src := range sources {
		if src.Name == name {
			return src, true, nil
		}
	}

	return StoredSource{}, false, nil
}

// UpdateVerifyTime updates the last verify timestamp for a source.
func (s *RegistryStore) UpdateVerifyTime(name string) error {
	sources, err := s.Load()
	if err != nil {
		return err
	}

	for i := range sources {
		if sources[i].Name == name {
			sources[i].LastVerify = time.Now()
			return s.Save(sources)
		}
	}

	return fmt.Errorf("catalog '%s' not found", name)
}

// SetEnabled enables or disables a source.
func (s *RegistryStore) SetEnabled(name string, enabled bool) error {
	sources, err := s.Load()
	if err != nil {
		return err
	}

	for i := range sources {
		if sources[i].Name == name {
			sources[i].Enabled = enabled
			return s.Save(sources)
		}
	}

	return fmt.Errorf("catalog '%s' not found", name)
}

// SetTrusted marks a source as trusted.
func (s *RegistryStore) SetTrusted(name string, trusted bool) error {
	sources, err := s.Load()
	if err != nil {
		return err
	}

	for i := range sources {
		if sources[i].Name == name {
			sources[i].Trusted = trusted
			return s.Save(sources)
		}
	}

	return fmt.Errorf("catalog '%s' not found", name)
}
