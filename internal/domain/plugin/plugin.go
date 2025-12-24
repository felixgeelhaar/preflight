// Package plugin provides plugin discovery, loading, and management.
package plugin

import (
	"fmt"
	"time"
)

// Manifest describes a plugin's metadata and capabilities.
type Manifest struct {
	// APIVersion is the plugin API version (e.g., "v1")
	APIVersion string `yaml:"apiVersion"`
	// Name is the plugin identifier (e.g., "docker", "kubernetes")
	Name string `yaml:"name"`
	// Version is the semantic version (e.g., "1.0.0")
	Version string `yaml:"version"`
	// Description is a brief description of the plugin
	Description string `yaml:"description,omitempty"`
	// Author is the plugin author
	Author string `yaml:"author,omitempty"`
	// License is the plugin license (e.g., "MIT", "Apache-2.0")
	License string `yaml:"license,omitempty"`
	// Homepage is the plugin homepage URL
	Homepage string `yaml:"homepage,omitempty"`
	// Repository is the source repository URL
	Repository string `yaml:"repository,omitempty"`
	// Keywords are searchable tags
	Keywords []string `yaml:"keywords,omitempty"`
	// Provides lists the capabilities this plugin offers
	Provides Capabilities `yaml:"provides"`
	// Requires lists dependencies on other plugins
	Requires []Dependency `yaml:"requires,omitempty"`
	// MinPreflightVersion is the minimum preflight version required
	MinPreflightVersion string `yaml:"minPreflightVersion,omitempty"`
}

// Capabilities describes what a plugin provides.
type Capabilities struct {
	// Providers are custom provider implementations
	Providers []ProviderSpec `yaml:"providers,omitempty"`
	// Presets are catalog presets
	Presets []string `yaml:"presets,omitempty"`
	// CapabilityPacks are catalog capability packs
	CapabilityPacks []string `yaml:"capabilityPacks,omitempty"`
}

// ProviderSpec describes a provider implementation.
type ProviderSpec struct {
	// Name is the provider name (e.g., "docker")
	Name string `yaml:"name"`
	// ConfigKey is the config section this provider handles
	ConfigKey string `yaml:"configKey"`
	// Description describes what this provider does
	Description string `yaml:"description,omitempty"`
}

// Dependency describes a plugin dependency.
type Dependency struct {
	// Name is the required plugin name
	Name string `yaml:"name"`
	// Version is a semver constraint (e.g., ">=1.0.0")
	Version string `yaml:"version,omitempty"`
}

// Plugin represents a loaded plugin.
type Plugin struct {
	// Manifest contains the plugin metadata
	Manifest Manifest
	// Path is the plugin's installation path
	Path string
	// Enabled indicates if the plugin is enabled
	Enabled bool
	// LoadedAt is when the plugin was loaded
	LoadedAt time.Time
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return p.Manifest.Name
}

// String returns a human-readable plugin description.
func (p *Plugin) String() string {
	return fmt.Sprintf("%s@%s", p.Manifest.Name, p.Manifest.Version)
}

// Registry manages installed plugins.
type Registry struct {
	plugins map[string]*Plugin
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]*Plugin),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(plugin *Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}
	if plugin.Manifest.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if _, exists := r.plugins[plugin.Manifest.Name]; exists {
		return fmt.Errorf("plugin %q already registered", plugin.Manifest.Name)
	}
	r.plugins[plugin.Manifest.Name] = plugin
	return nil
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (*Plugin, bool) {
	plugin, ok := r.plugins[name]
	return plugin, ok
}

// List returns all registered plugins.
func (r *Registry) List() []*Plugin {
	plugins := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Enabled returns all enabled plugins.
func (r *Registry) Enabled() []*Plugin {
	plugins := make([]*Plugin, 0)
	for _, p := range r.plugins {
		if p.Enabled {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(name string) bool {
	if _, exists := r.plugins[name]; exists {
		delete(r.plugins, name)
		return true
	}
	return false
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	return len(r.plugins)
}

// ValidateManifest checks if a manifest is valid.
func ValidateManifest(m *Manifest) error {
	if m.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if m.APIVersion != "v1" {
		return fmt.Errorf("unsupported apiVersion: %s (expected v1)", m.APIVersion)
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	// Validate provider specs
	for i, p := range m.Provides.Providers {
		if p.Name == "" {
			return fmt.Errorf("provider %d: name is required", i)
		}
		if p.ConfigKey == "" {
			return fmt.Errorf("provider %q: configKey is required", p.Name)
		}
	}
	return nil
}
