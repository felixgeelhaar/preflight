package catalog

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Registry errors.
var (
	ErrCatalogNotFound  = errors.New("catalog not found")
	ErrCatalogExists    = errors.New("catalog already exists")
	ErrBuiltinImmutable = errors.New("builtin catalog cannot be modified")
)

// RegisteredCatalog represents a catalog that has been added to the registry.
type RegisteredCatalog struct {
	source     Source
	manifest   Manifest
	catalog    *Catalog
	addedAt    time.Time
	verifiedAt time.Time
	enabled    bool
}

// NewRegisteredCatalog creates a new registered catalog.
func NewRegisteredCatalog(source Source, manifest Manifest, catalog *Catalog) *RegisteredCatalog {
	now := time.Now()
	return &RegisteredCatalog{
		source:     source,
		manifest:   manifest,
		catalog:    catalog,
		addedAt:    now,
		verifiedAt: now,
		enabled:    true,
	}
}

// Source returns the catalog source.
func (rc *RegisteredCatalog) Source() Source {
	return rc.source
}

// Manifest returns the catalog manifest.
func (rc *RegisteredCatalog) Manifest() Manifest {
	return rc.manifest
}

// Catalog returns the catalog data.
func (rc *RegisteredCatalog) Catalog() *Catalog {
	return rc.catalog
}

// Name returns the catalog name.
func (rc *RegisteredCatalog) Name() string {
	return rc.source.Name()
}

// AddedAt returns when the catalog was added.
func (rc *RegisteredCatalog) AddedAt() time.Time {
	return rc.addedAt
}

// VerifiedAt returns when the catalog was last verified.
func (rc *RegisteredCatalog) VerifiedAt() time.Time {
	return rc.verifiedAt
}

// Enabled returns whether the catalog is enabled.
func (rc *RegisteredCatalog) Enabled() bool {
	return rc.enabled
}

// SetEnabled enables or disables the catalog.
func (rc *RegisteredCatalog) SetEnabled(enabled bool) {
	rc.enabled = enabled
}

// SetVerifiedAt updates the verification timestamp.
func (rc *RegisteredCatalog) SetVerifiedAt(t time.Time) {
	rc.verifiedAt = t
}

// Registry manages multiple catalog sources.
// It provides a unified view across all registered catalogs.
type Registry struct {
	catalogs map[string]*RegisteredCatalog
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		catalogs: make(map[string]*RegisteredCatalog),
	}
}

// Add adds a catalog to the registry.
func (r *Registry) Add(rc *RegisteredCatalog) error {
	if rc == nil {
		return fmt.Errorf("%w: catalog is nil", ErrInvalidSource)
	}

	name := rc.Name()
	if _, exists := r.catalogs[name]; exists {
		return fmt.Errorf("%w: %s", ErrCatalogExists, name)
	}

	r.catalogs[name] = rc
	return nil
}

// Remove removes a catalog from the registry.
func (r *Registry) Remove(name string) error {
	rc, exists := r.catalogs[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrCatalogNotFound, name)
	}

	if rc.Source().IsBuiltin() {
		return fmt.Errorf("%w: %s", ErrBuiltinImmutable, name)
	}

	delete(r.catalogs, name)
	return nil
}

// Get returns a registered catalog by name.
func (r *Registry) Get(name string) (*RegisteredCatalog, bool) {
	rc, ok := r.catalogs[name]
	return rc, ok
}

// List returns all registered catalogs, sorted by name.
func (r *Registry) List() []*RegisteredCatalog {
	result := make([]*RegisteredCatalog, 0, len(r.catalogs))
	for _, rc := range r.catalogs {
		result = append(result, rc)
	}
	sort.Slice(result, func(i, j int) bool {
		// Builtin first, then alphabetically
		if result[i].Source().IsBuiltin() != result[j].Source().IsBuiltin() {
			return result[i].Source().IsBuiltin()
		}
		return result[i].Name() < result[j].Name()
	})
	return result
}

// ListEnabled returns all enabled catalogs.
func (r *Registry) ListEnabled() []*RegisteredCatalog {
	var result []*RegisteredCatalog
	for _, rc := range r.List() {
		if rc.Enabled() {
			result = append(result, rc)
		}
	}
	return result
}

// Count returns the number of registered catalogs.
func (r *Registry) Count() int {
	return len(r.catalogs)
}

// FindPreset searches for a preset across all enabled catalogs.
// Returns the preset and the catalog it was found in.
func (r *Registry) FindPreset(id PresetID) (Preset, *RegisteredCatalog, bool) {
	for _, rc := range r.ListEnabled() {
		if preset, ok := rc.Catalog().GetPreset(id); ok {
			return preset, rc, true
		}
	}
	return Preset{}, nil, false
}

// FindPack searches for a capability pack across all enabled catalogs.
func (r *Registry) FindPack(id string) (CapabilityPack, *RegisteredCatalog, bool) {
	for _, rc := range r.ListEnabled() {
		if pack, ok := rc.Catalog().GetPack(id); ok {
			return pack, rc, true
		}
	}
	return CapabilityPack{}, nil, false
}

// AllPresets returns all presets from all enabled catalogs.
func (r *Registry) AllPresets() []Preset {
	seen := make(map[string]bool)
	var result []Preset

	for _, rc := range r.ListEnabled() {
		for _, preset := range rc.Catalog().ListPresets() {
			key := preset.ID().String()
			if !seen[key] {
				seen[key] = true
				result = append(result, preset)
			}
		}
	}

	return result
}

// AllPacks returns all capability packs from all enabled catalogs.
func (r *Registry) AllPacks() []CapabilityPack {
	seen := make(map[string]bool)
	var result []CapabilityPack

	for _, rc := range r.ListEnabled() {
		for _, pack := range rc.Catalog().ListPacks() {
			if !seen[pack.ID()] {
				seen[pack.ID()] = true
				result = append(result, pack)
			}
		}
	}

	return result
}

// PresetCount returns the total number of unique presets.
func (r *Registry) PresetCount() int {
	return len(r.AllPresets())
}

// PackCount returns the total number of unique capability packs.
func (r *Registry) PackCount() int {
	return len(r.AllPacks())
}

// Stats returns registry statistics.
func (r *Registry) Stats() RegistryStats {
	var stats RegistryStats
	stats.TotalCatalogs = r.Count()

	for _, rc := range r.catalogs {
		if rc.Enabled() {
			stats.EnabledCatalogs++
		}
		switch rc.Source().Type() {
		case SourceTypeBuiltin:
			stats.BuiltinCatalogs++
		case SourceTypeURL:
			stats.URLCatalogs++
		case SourceTypeLocal:
			stats.LocalCatalogs++
		}
	}

	stats.TotalPresets = r.PresetCount()
	stats.TotalPacks = r.PackCount()

	return stats
}

// RegistryStats contains registry statistics.
type RegistryStats struct {
	TotalCatalogs   int
	EnabledCatalogs int
	BuiltinCatalogs int
	URLCatalogs     int
	LocalCatalogs   int
	TotalPresets    int
	TotalPacks      int
}
