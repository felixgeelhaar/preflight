package identity

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Service is the aggregate root for identity management.
// It is safe for concurrent use.
type Service struct {
	mu        sync.RWMutex
	providers map[string]Provider
	store     *TokenStore
}

// NewService creates a new identity Service.
func NewService(store *TokenStore) *Service {
	return &Service{
		providers: make(map[string]Provider),
		store:     store,
	}
}

// RegisterProvider registers an identity provider with the service.
func (s *Service) RegisterProvider(provider Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := provider.Name()
	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("%w: %s", ErrProviderExists, name)
	}

	s.providers[name] = provider
	return nil
}

// Login authenticates with the named provider and persists the token.
func (s *Service) Login(ctx context.Context, providerName string) (*Token, error) {
	provider, err := s.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	token, err := provider.Authenticate(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.store.Save(providerName, token); err != nil {
		return nil, fmt.Errorf("failed to persist token: %w", err)
	}

	return token, nil
}

// Logout removes the stored token for the named provider.
func (s *Service) Logout(providerName string) error {
	if _, err := s.getProvider(providerName); err != nil {
		return err
	}

	return s.store.Delete(providerName)
}

// Status returns the current token for the named provider.
func (s *Service) Status(providerName string) (*Token, error) {
	if _, err := s.getProvider(providerName); err != nil {
		return nil, err
	}

	return s.store.Load(providerName)
}

// WhoAmI returns the claims for the current token of the named provider.
func (s *Service) WhoAmI(providerName string) (*Claims, error) {
	if _, err := s.getProvider(providerName); err != nil {
		return nil, err
	}

	token, err := s.store.Load(providerName)
	if err != nil {
		return nil, err
	}

	claims := token.Claims()
	return &claims, nil
}

// CurrentToken returns the current token for the named provider.
func (s *Service) CurrentToken(providerName string) (*Token, error) {
	if _, err := s.getProvider(providerName); err != nil {
		return nil, err
	}

	return s.store.Load(providerName)
}

// ListProviders returns the names of all registered providers, sorted alphabetically.
func (s *Service) ListProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// getProvider retrieves a provider by name, returning ErrProviderNotFound if not registered.
func (s *Service) getProvider(name string) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}
