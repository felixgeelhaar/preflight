// Package embedded provides the embedded catalog data and loader.
package embedded

import (
	_ "embed"
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"gopkg.in/yaml.v3"
)

//go:embed catalog.yaml
var catalogYAML []byte

// catalogDTO is the data transfer object for parsing the YAML file.
type catalogDTO struct {
	Presets []presetDTO         `yaml:"presets"`
	Packs   []capabilityPackDTO `yaml:"capability_packs"`
}

type presetDTO struct {
	ID         string                 `yaml:"id"`
	Metadata   metadataDTO            `yaml:"metadata"`
	Difficulty string                 `yaml:"difficulty"`
	Config     map[string]interface{} `yaml:"config"`
	Requires   []string               `yaml:"requires"`
	Conflicts  []string               `yaml:"conflicts"`
}

type metadataDTO struct {
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Tradeoffs   []string          `yaml:"tradeoffs"`
	DocLinks    map[string]string `yaml:"doc_links"`
	Tags        []string          `yaml:"tags"`
}

type capabilityPackDTO struct {
	ID       string      `yaml:"id"`
	Metadata metadataDTO `yaml:"metadata"`
	Presets  []string    `yaml:"presets"`
	Tools    []string    `yaml:"tools"`
}

// LoadCatalog loads the embedded catalog from the YAML file.
func LoadCatalog() (*catalog.Catalog, error) {
	var dto catalogDTO
	if err := yaml.Unmarshal(catalogYAML, &dto); err != nil {
		return nil, fmt.Errorf("failed to parse catalog YAML: %w", err)
	}

	cat := catalog.NewCatalog()

	// Load presets
	for _, p := range dto.Presets {
		preset, err := parsePreset(p)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preset %s: %w", p.ID, err)
		}
		if err := cat.AddPreset(preset); err != nil {
			return nil, fmt.Errorf("failed to add preset %s: %w", p.ID, err)
		}
	}

	// Load capability packs
	for _, cp := range dto.Packs {
		pack, err := parseCapabilityPack(cp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse capability pack %s: %w", cp.ID, err)
		}
		if err := cat.AddPack(pack); err != nil {
			return nil, fmt.Errorf("failed to add capability pack %s: %w", cp.ID, err)
		}
	}

	return cat, nil
}

func parsePreset(dto presetDTO) (catalog.Preset, error) {
	id, err := catalog.ParsePresetID(dto.ID)
	if err != nil {
		return catalog.Preset{}, err
	}

	meta, err := parseMetadata(dto.Metadata)
	if err != nil {
		return catalog.Preset{}, err
	}

	difficulty, err := catalog.ParseDifficultyLevel(dto.Difficulty)
	if err != nil {
		return catalog.Preset{}, err
	}

	preset, err := catalog.NewPreset(id, meta, difficulty, dto.Config)
	if err != nil {
		return catalog.Preset{}, err
	}

	// Parse requires
	if len(dto.Requires) > 0 {
		requires := make([]catalog.PresetID, 0, len(dto.Requires))
		for _, r := range dto.Requires {
			reqID, err := catalog.ParsePresetID(r)
			if err != nil {
				return catalog.Preset{}, fmt.Errorf("invalid requires %s: %w", r, err)
			}
			requires = append(requires, reqID)
		}
		preset = preset.WithRequires(requires)
	}

	// Parse conflicts
	if len(dto.Conflicts) > 0 {
		conflicts := make([]catalog.PresetID, 0, len(dto.Conflicts))
		for _, c := range dto.Conflicts {
			conflictID, err := catalog.ParsePresetID(c)
			if err != nil {
				return catalog.Preset{}, fmt.Errorf("invalid conflict %s: %w", c, err)
			}
			conflicts = append(conflicts, conflictID)
		}
		preset = preset.WithConflicts(conflicts)
	}

	return preset, nil
}

func parseMetadata(dto metadataDTO) (catalog.Metadata, error) {
	meta, err := catalog.NewMetadata(dto.Title, dto.Description)
	if err != nil {
		return catalog.Metadata{}, err
	}

	if len(dto.Tradeoffs) > 0 {
		meta = meta.WithTradeoffs(dto.Tradeoffs)
	}

	if len(dto.DocLinks) > 0 {
		meta = meta.WithDocLinks(dto.DocLinks)
	}

	if len(dto.Tags) > 0 {
		meta = meta.WithTags(dto.Tags)
	}

	return meta, nil
}

func parseCapabilityPack(dto capabilityPackDTO) (catalog.CapabilityPack, error) {
	meta, err := parseMetadata(dto.Metadata)
	if err != nil {
		return catalog.CapabilityPack{}, err
	}

	pack, err := catalog.NewCapabilityPack(dto.ID, meta)
	if err != nil {
		return catalog.CapabilityPack{}, err
	}

	// Parse preset IDs
	if len(dto.Presets) > 0 {
		presets := make([]catalog.PresetID, 0, len(dto.Presets))
		for _, p := range dto.Presets {
			presetID, err := catalog.ParsePresetID(p)
			if err != nil {
				return catalog.CapabilityPack{}, fmt.Errorf("invalid preset %s: %w", p, err)
			}
			presets = append(presets, presetID)
		}
		pack = pack.WithPresets(presets)
	}

	if len(dto.Tools) > 0 {
		pack = pack.WithTools(dto.Tools)
	}

	return pack, nil
}
