package config

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// ReproducibilityMode controls version resolution behavior.
type ReproducibilityMode string

const (
	// ModeIntent installs latest compatible versions.
	ModeIntent ReproducibilityMode = "intent"
	// ModeLocked prefers lockfile, updates intentionally.
	ModeLocked ReproducibilityMode = "locked"
	// ModeFrozen fails if resolution differs from lock.
	ModeFrozen ReproducibilityMode = "frozen"
)

// DefaultConfig holds manifest-level defaults.
type DefaultConfig struct {
	Mode   ReproducibilityMode `yaml:"mode,omitempty"`
	Editor string              `yaml:"editor,omitempty"`
}

// Manifest is the root configuration (preflight.yaml).
type Manifest struct {
	Defaults DefaultConfig
	Targets  map[string][]LayerName
}

// Errors for Manifest validation.
var (
	ErrNoTargets      = errors.New("manifest must define at least one target")
	ErrTargetNotFound = errors.New("target not found")
)

// manifestYAML is the YAML representation for unmarshaling.
type manifestYAML struct {
	Defaults DefaultConfig       `yaml:"defaults,omitempty"`
	Targets  map[string][]string `yaml:"targets"`
}

// ParseManifest parses a Manifest from YAML bytes.
func ParseManifest(data []byte) (*Manifest, error) {
	var raw manifestYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if len(raw.Targets) == 0 {
		return nil, ErrNoTargets
	}

	targets := make(map[string][]LayerName)
	for targetName, layerNames := range raw.Targets {
		layers := make([]LayerName, 0, len(layerNames))
		for _, name := range layerNames {
			ln, err := NewLayerName(name)
			if err != nil {
				return nil, err
			}
			layers = append(layers, ln)
		}
		targets[targetName] = layers
	}

	return &Manifest{
		Defaults: raw.Defaults,
		Targets:  targets,
	}, nil
}

// GetTarget returns the layer names for a given target.
func (m *Manifest) GetTarget(name TargetName) ([]LayerName, error) {
	layers, ok := m.Targets[name.String()]
	if !ok {
		return nil, ErrTargetNotFound
	}
	return layers, nil
}
