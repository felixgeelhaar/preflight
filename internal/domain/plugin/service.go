// Package plugin provides plugin discovery, loading, and management.
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// DefaultService is the default implementation of Service.
type DefaultService struct {
	mu         sync.RWMutex
	registry   *Registry
	discoverer Discoverer
	searcher   Searcher
	installer  Installer
}

// ServiceOption configures a DefaultService.
type ServiceOption func(*DefaultService)

// WithDiscoverer sets the plugin discoverer.
func WithDiscoverer(d Discoverer) ServiceOption {
	return func(s *DefaultService) {
		s.discoverer = d
	}
}

// WithSearcher sets the plugin searcher.
func WithSearcher(sr Searcher) ServiceOption {
	return func(s *DefaultService) {
		s.searcher = sr
	}
}

// WithInstaller sets the plugin installer.
func WithInstaller(i Installer) ServiceOption {
	return func(s *DefaultService) {
		s.installer = i
	}
}

// NewService creates a new plugin service with the given options.
func NewService(opts ...ServiceOption) *DefaultService {
	s := &DefaultService{
		registry: NewRegistry(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set defaults if not provided
	if s.discoverer == nil {
		s.discoverer = NewLoader()
	}
	if s.searcher == nil {
		s.searcher = NewSearcher()
	}

	return s
}

// Registry returns the plugin registry.
func (s *DefaultService) Registry() *Registry {
	return s.registry
}

// Discover finds and loads all plugins from configured paths.
// The context can be used for cancellation.
func (s *DefaultService) Discover(ctx context.Context) error {
	s.mu.RLock()
	discoverer := s.discoverer
	s.mu.RUnlock()

	if discoverer == nil {
		return fmt.Errorf("no discoverer configured")
	}

	result, err := discoverer.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}

	// Register discovered plugins
	for _, p := range result.Plugins {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.registry.Register(p); err != nil {
			// Skip duplicates but log other errors
			if !IsPluginExists(err) {
				return fmt.Errorf("registering plugin %s: %w", p.ID(), err)
			}
		}
	}

	return nil
}

// Install installs a plugin from a source (path or URL).
func (s *DefaultService) Install(ctx context.Context, source string) (*Plugin, error) {
	s.mu.RLock()
	installer := s.installer
	s.mu.RUnlock()

	if installer == nil {
		return nil, fmt.Errorf("no installer configured")
	}

	plugin, err := installer.Install(ctx, source)
	if err != nil {
		return nil, err
	}

	if err := s.registry.Register(plugin); err != nil {
		return nil, fmt.Errorf("registering installed plugin: %w", err)
	}

	return plugin, nil
}

// Uninstall removes an installed plugin.
func (s *DefaultService) Uninstall(name string) error {
	s.mu.RLock()
	installer := s.installer
	s.mu.RUnlock()

	if installer == nil {
		return fmt.Errorf("no installer configured")
	}

	if err := installer.Uninstall(name); err != nil {
		return err
	}

	s.registry.Remove(name)
	return nil
}

// Search searches for plugins using the configured searcher.
func (s *DefaultService) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	searcher := s.searcher
	s.mu.RUnlock()

	if searcher == nil {
		return nil, fmt.Errorf("no searcher configured")
	}

	return searcher.Search(ctx, opts)
}

// Get retrieves a plugin by name from the registry.
func (s *DefaultService) Get(name string) (*Plugin, bool) {
	return s.registry.Get(name)
}

// List returns all registered plugins.
func (s *DefaultService) List() []*Plugin {
	return s.registry.List()
}

// Ensure DefaultService implements Service.
var _ Service = (*DefaultService)(nil)
