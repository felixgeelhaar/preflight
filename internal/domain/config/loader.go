package config

import (
	"os"
	"path/filepath"
)

// Loader loads configuration from the filesystem.
type Loader struct{}

// NewLoader creates a new Loader.
func NewLoader() *Loader {
	return &Loader{}
}

// LoadManifest loads a manifest from the given path.
func (l *Loader) LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

// LoadLayer loads a layer from the given path.
func (l *Loader) LoadLayer(path string) (*Layer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	layer, err := ParseLayer(data)
	if err != nil {
		return nil, err
	}

	layer.SetProvenance(path)
	return layer, nil
}

// LoadTarget loads all layers for a target and returns a resolved Target.
func (l *Loader) LoadTarget(manifest *Manifest, target TargetName, layersDir string) (*Target, error) {
	layerNames, err := manifest.GetTarget(target)
	if err != nil {
		return nil, err
	}

	layers := make([]Layer, 0, len(layerNames))
	for _, name := range layerNames {
		path := filepath.Join(layersDir, name.String()+".yaml")
		layer, err := l.LoadLayer(path)
		if err != nil {
			return nil, err
		}
		layers = append(layers, *layer)
	}

	return &Target{
		Name:   target,
		Layers: layers,
	}, nil
}

// Load loads a manifest, resolves the target, merges layers, and returns MergedConfig.
func (l *Loader) Load(manifestPath string, target TargetName) (*MergedConfig, error) {
	// Load manifest
	manifest, err := l.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	// Determine layers directory (sibling to manifest)
	layersDir := filepath.Join(filepath.Dir(manifestPath), "layers")

	// Load target with its layers
	resolvedTarget, err := l.LoadTarget(manifest, target, layersDir)
	if err != nil {
		return nil, err
	}

	// Merge layers
	merger := NewMerger()
	merged, err := merger.Merge(resolvedTarget.Layers)
	if err != nil {
		return nil, err
	}

	return merged, nil
}
