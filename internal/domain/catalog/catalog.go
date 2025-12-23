package catalog

import (
	"errors"
	"fmt"
)

// Catalog errors.
var (
	ErrInvalidPreset   = errors.New("preset is invalid")
	ErrDuplicatePreset = errors.New("preset already exists")
	ErrInvalidPack     = errors.New("capability pack is invalid")
	ErrDuplicatePack   = errors.New("capability pack already exists")
)

// Catalog is the aggregate root for presets and capability packs.
// It provides a unified view of all available configuration bundles.
type Catalog struct {
	presets map[string]Preset         // key: "provider:name"
	packs   map[string]CapabilityPack // key: pack ID
}

// NewCatalog creates a new empty Catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		presets: make(map[string]Preset),
		packs:   make(map[string]CapabilityPack),
	}
}

// AddPreset adds a preset to the catalog.
func (c *Catalog) AddPreset(preset Preset) error {
	if preset.IsZero() {
		return ErrInvalidPreset
	}

	key := preset.ID().String()
	if _, exists := c.presets[key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicatePreset, key)
	}

	c.presets[key] = preset
	return nil
}

// GetPreset retrieves a preset by its ID.
func (c *Catalog) GetPreset(id PresetID) (Preset, bool) {
	preset, ok := c.presets[id.String()]
	return preset, ok
}

// FindPresetsByProvider returns all presets for a given provider.
func (c *Catalog) FindPresetsByProvider(provider string) []Preset {
	var result []Preset
	for _, preset := range c.presets {
		if preset.ID().MatchesProvider(provider) {
			result = append(result, preset)
		}
	}
	return result
}

// FindPresetsByDifficulty returns all presets at a given difficulty level.
func (c *Catalog) FindPresetsByDifficulty(difficulty DifficultyLevel) []Preset {
	var result []Preset
	for _, preset := range c.presets {
		if preset.Difficulty() == difficulty {
			result = append(result, preset)
		}
	}
	return result
}

// ListPresets returns all presets in the catalog.
func (c *Catalog) ListPresets() []Preset {
	result := make([]Preset, 0, len(c.presets))
	for _, preset := range c.presets {
		result = append(result, preset)
	}
	return result
}

// PresetCount returns the number of presets in the catalog.
func (c *Catalog) PresetCount() int {
	return len(c.presets)
}

// AddPack adds a capability pack to the catalog.
func (c *Catalog) AddPack(pack CapabilityPack) error {
	if pack.IsZero() {
		return ErrInvalidPack
	}

	if _, exists := c.packs[pack.ID()]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicatePack, pack.ID())
	}

	c.packs[pack.ID()] = pack
	return nil
}

// GetPack retrieves a capability pack by its ID.
func (c *Catalog) GetPack(id string) (CapabilityPack, bool) {
	pack, ok := c.packs[id]
	return pack, ok
}

// ListPacks returns all capability packs in the catalog.
func (c *Catalog) ListPacks() []CapabilityPack {
	result := make([]CapabilityPack, 0, len(c.packs))
	for _, pack := range c.packs {
		result = append(result, pack)
	}
	return result
}

// PackCount returns the number of capability packs in the catalog.
func (c *Catalog) PackCount() int {
	return len(c.packs)
}

// String returns a summary string.
func (c *Catalog) String() string {
	return fmt.Sprintf("Catalog (%d presets, %d packs)", len(c.presets), len(c.packs))
}
