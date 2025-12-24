package marketplace

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Index represents the marketplace registry index.
// It contains metadata about all available packages.
type Index struct {
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Packages  []Package `json:"packages"`

	// Internal lookup map (not serialized)
	byID map[string]Package `json:"-"`
}

// NewIndex creates a new empty index.
func NewIndex() *Index {
	return &Index{
		Version:  "1",
		Packages: []Package{},
		byID:     make(map[string]Package),
	}
}

// ParseIndex parses a JSON index.
func ParseIndex(data []byte) (*Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	// Build lookup map
	idx.byID = make(map[string]Package, len(idx.Packages))
	for _, pkg := range idx.Packages {
		idx.byID[pkg.ID.String()] = pkg
	}

	return &idx, nil
}

// Marshal serializes the index to JSON.
func (idx *Index) Marshal() ([]byte, error) {
	idx.UpdatedAt = time.Now()
	return json.MarshalIndent(idx, "", "  ")
}

// Get returns a package by ID.
func (idx *Index) Get(id PackageID) (Package, bool) {
	pkg, ok := idx.byID[id.String()]
	return pkg, ok
}

// Add adds a package to the index.
func (idx *Index) Add(pkg Package) error {
	if !pkg.IsValid() {
		return ErrInvalidPackage
	}
	if _, exists := idx.byID[pkg.ID.String()]; exists {
		return fmt.Errorf("package %s already exists", pkg.ID)
	}
	idx.Packages = append(idx.Packages, pkg)
	idx.byID[pkg.ID.String()] = pkg
	return nil
}

// Search finds packages matching the query.
func (idx *Index) Search(query string) []Package {
	if query == "" {
		return idx.Packages
	}

	var results []Package
	for _, pkg := range idx.Packages {
		if pkg.MatchesQuery(query) {
			results = append(results, pkg)
		}
	}
	return results
}

// SearchByType finds packages of a specific type.
func (idx *Index) SearchByType(pkgType string) []Package {
	var results []Package
	for _, pkg := range idx.Packages {
		if pkg.Type == pkgType {
			results = append(results, pkg)
		}
	}
	return results
}

// SearchByKeyword finds packages with a specific keyword.
func (idx *Index) SearchByKeyword(keyword string) []Package {
	var results []Package
	for _, pkg := range idx.Packages {
		for _, kw := range pkg.Keywords {
			if kw == keyword {
				results = append(results, pkg)
				break
			}
		}
	}
	return results
}

// ListByPopularity returns packages sorted by downloads.
func (idx *Index) ListByPopularity() []Package {
	result := make([]Package, len(idx.Packages))
	copy(result, idx.Packages)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Downloads > result[j].Downloads
	})
	return result
}

// ListByRecent returns packages sorted by update time.
func (idx *Index) ListByRecent() []Package {
	result := make([]Package, len(idx.Packages))
	copy(result, idx.Packages)
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

// Count returns the number of packages.
func (idx *Index) Count() int {
	return len(idx.Packages)
}

// Statistics returns index statistics.
func (idx *Index) Statistics() IndexStats {
	stats := IndexStats{
		TotalPackages: len(idx.Packages),
	}

	for _, pkg := range idx.Packages {
		switch pkg.Type {
		case PackageTypePreset:
			stats.Presets++
		case PackageTypeCapabilityPack:
			stats.CapabilityPacks++
		case PackageTypeLayerTemplate:
			stats.LayerTemplates++
		}
		stats.TotalDownloads += pkg.Downloads
		if pkg.Provenance.Verified {
			stats.VerifiedPackages++
		}
	}

	return stats
}

// IndexStats contains aggregate statistics about the index.
type IndexStats struct {
	TotalPackages    int `json:"total_packages"`
	Presets          int `json:"presets"`
	CapabilityPacks  int `json:"capability_packs"`
	LayerTemplates   int `json:"layer_templates"`
	TotalDownloads   int `json:"total_downloads"`
	VerifiedPackages int `json:"verified_packages"`
}
