// Package plugin provides plugin discovery, loading, and management.
package plugin

import "context"

// Repository defines the interface for plugin storage and retrieval.
// This is a domain port that can be implemented by different adapters
// (filesystem, in-memory for testing, remote storage, etc.).
type Repository interface {
	// Load retrieves a plugin by name.
	Load(name string) (*Plugin, error)

	// Save stores a plugin.
	Save(plugin *Plugin) error

	// Delete removes a plugin by name.
	Delete(name string) error

	// List returns all available plugins.
	List() ([]*Plugin, error)

	// Exists checks if a plugin with the given name exists.
	Exists(name string) (bool, error)
}

// Discoverer defines the interface for discovering plugins.
// Implementations scan directories or other sources for plugins.
type Discoverer interface {
	// Discover finds all plugins in configured locations.
	// The context can be used for cancellation.
	Discover(ctx context.Context) (*DiscoveryResult, error)

	// LoadFromPath loads a plugin from a specific path.
	LoadFromPath(path string) (*Plugin, error)
}

// Searcher defines the interface for searching remote plugin sources.
// This is a domain port for plugin discovery services (GitHub, registry, etc.).
type Searcher interface {
	// Search finds plugins matching the given options.
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)
}

// Installer defines the interface for installing plugins from remote sources.
type Installer interface {
	// Install downloads and installs a plugin from the given source.
	Install(ctx context.Context, source string) (*Plugin, error)

	// Uninstall removes an installed plugin.
	Uninstall(name string) error
}

// Validator defines the interface for validating plugin manifests.
type Validator interface {
	// Validate checks if a manifest is valid.
	Validate(manifest *Manifest) error

	// ValidateCapabilities checks if requested WASM capabilities are allowed.
	ValidateCapabilities(caps []WASMCapability) error
}

// Service combines common plugin operations.
// This is a higher-level service that coordinates between repositories,
// discoverers, and searchers.
type Service interface {
	// Registry returns the plugin registry.
	Registry() *Registry

	// Discover finds and loads all plugins.
	// The context can be used for cancellation.
	Discover(ctx context.Context) error

	// Install installs a plugin from a source.
	Install(ctx context.Context, source string) (*Plugin, error)

	// Uninstall removes an installed plugin.
	Uninstall(name string) error

	// Search searches for plugins.
	Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error)

	// Get retrieves a plugin by name.
	Get(name string) (*Plugin, bool)

	// List returns all registered plugins.
	List() []*Plugin
}
