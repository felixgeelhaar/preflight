// Package services provides TUI service implementations.
package services

import (
	"sort"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/felixgeelhaar/preflight/internal/tui"
)

// Ensure CatalogService implements tui.CatalogServiceInterface.
var _ tui.CatalogServiceInterface = (*CatalogService)(nil)

// PresetItem represents a preset for display in the TUI.
// This is a local type for tests; CatalogService returns tui.PresetItem.
type PresetItem = tui.PresetItem

// PackItem represents a capability pack for display in the TUI.
// This is a local type for tests; CatalogService returns tui.PackItem.
type PackItem = tui.PackItem

// CatalogService wraps the catalog domain for TUI usage.
type CatalogService struct {
	catalog *catalog.Catalog
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(cat *catalog.Catalog) *CatalogService {
	return &CatalogService{
		catalog: cat,
	}
}

// GetProviders returns a sorted list of unique provider names from all presets.
func (s *CatalogService) GetProviders() []string {
	presets := s.catalog.ListPresets()
	providerSet := make(map[string]bool)

	for _, preset := range presets {
		provider := preset.ID().Provider()
		providerSet[provider] = true
	}

	providers := make([]string, 0, len(providerSet))
	for provider := range providerSet {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

// GetPresetsForProvider returns all presets for a given provider.
func (s *CatalogService) GetPresetsForProvider(provider string) []PresetItem {
	presets := s.catalog.ListPresets()
	var items []PresetItem

	for _, preset := range presets {
		if preset.ID().Provider() == provider {
			items = append(items, PresetItem{
				ID:          preset.ID().String(),
				Title:       preset.Metadata().Title(),
				Description: preset.Metadata().Description(),
				Difficulty:  difficultyToString(preset.Difficulty()),
			})
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items
}

// GetCapabilityPacks returns all capability packs.
func (s *CatalogService) GetCapabilityPacks() []PackItem {
	packs := s.catalog.ListPacks()
	items := make([]PackItem, 0, len(packs))

	for _, pack := range packs {
		items = append(items, PackItem{
			ID:          pack.ID(),
			Title:       pack.Metadata().Title(),
			Description: pack.Metadata().Description(),
		})
	}

	// Sort by ID for consistent ordering
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items
}

// GetPreset returns a specific preset by ID.
func (s *CatalogService) GetPreset(id string) (PresetItem, bool) {
	presetID, err := catalog.ParsePresetID(id)
	if err != nil {
		return PresetItem{}, false
	}

	preset, found := s.catalog.GetPreset(presetID)
	if !found {
		return PresetItem{}, false
	}

	return PresetItem{
		ID:          preset.ID().String(),
		Title:       preset.Metadata().Title(),
		Description: preset.Metadata().Description(),
		Difficulty:  difficultyToString(preset.Difficulty()),
	}, true
}

// difficultyToString converts a DifficultyLevel to a human-readable string.
func difficultyToString(d catalog.DifficultyLevel) string {
	switch d {
	case catalog.DifficultyBeginner:
		return "Beginner"
	case catalog.DifficultyIntermediate:
		return "Intermediate"
	case catalog.DifficultyAdvanced:
		return "Advanced"
	default:
		caser := cases.Title(language.English)
		return caser.String(string(d))
	}
}
