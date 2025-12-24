package app

import (
	"context"
	"path/filepath"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/drift"
)

// PatchGenerator generates config patches from detected drift.
type PatchGenerator struct {
	driftService *DriftService
}

// NewPatchGenerator creates a new PatchGenerator.
func NewPatchGenerator(driftService *DriftService) *PatchGenerator {
	return &PatchGenerator{
		driftService: driftService,
	}
}

// GenerateFromDrifts generates patches from detected drifts.
// It uses the source layer information to determine which config file to patch.
func (g *PatchGenerator) GenerateFromDrifts(ctx context.Context, drifts []drift.Drift, configDir string) ([]ConfigPatch, error) {
	var patches []ConfigPatch

	for _, d := range drifts {
		if !d.HasDrift() {
			continue
		}

		// Get the file state to determine source layer
		files, err := g.driftService.ListTrackedFiles(ctx)
		if err != nil {
			return nil, err
		}

		var sourceLayer string
		for _, f := range files {
			if f.Path == d.Path {
				sourceLayer = f.SourceLayer
				break
			}
		}

		if sourceLayer == "" {
			// Can't generate patch without knowing the source layer
			continue
		}

		// Determine the layer file path
		layerPath := g.resolveLayerPath(configDir, sourceLayer)
		if layerPath == "" {
			continue
		}

		// Generate patch based on drift type
		patch := g.createFilePatch(d, layerPath)
		if patch != nil {
			patches = append(patches, *patch)
		}
	}

	return patches, nil
}

// GenerateFromIssues generates patches from doctor issues.
func (g *PatchGenerator) GenerateFromIssues(issues []DoctorIssue, configDir string) []ConfigPatch {
	var patches []ConfigPatch

	for _, issue := range issues {
		if !issue.Fixable {
			continue
		}

		// For now, we can suggest patches for drift issues
		if issue.Severity == SeverityWarning && issue.Message == "Configuration drift detected" {
			// This is a drift issue - we could generate a patch to update config
			// to match current state, but we need more context about what changed
			patch := ConfigPatch{
				LayerPath:  filepath.Join(configDir, "layers", "captured.yaml"),
				YAMLPath:   issue.StepID,
				Operation:  PatchOpModify,
				OldValue:   issue.Expected,
				NewValue:   issue.Actual,
				Provenance: "drift-detected",
			}
			patches = append(patches, patch)
		}
	}

	return patches
}

// resolveLayerPath determines the full path to a layer file.
func (g *PatchGenerator) resolveLayerPath(configDir, sourceLayer string) string {
	if sourceLayer == "" {
		return ""
	}

	// Check common patterns
	candidates := []string{
		filepath.Join(configDir, "layers", sourceLayer+".yaml"),
		filepath.Join(configDir, "layers", sourceLayer+".yml"),
		filepath.Join(configDir, sourceLayer+".yaml"),
		filepath.Join(configDir, sourceLayer+".yml"),
	}

	for _, candidate := range candidates {
		// In production, we'd check if the file exists
		// For now, return the first candidate
		return candidate
	}

	return candidates[0]
}

// createFilePatch creates a patch for a file drift.
func (g *PatchGenerator) createFilePatch(d drift.Drift, layerPath string) *ConfigPatch {
	// For file drifts, we need to determine the YAML path based on the file path
	// This is provider-specific logic

	// Extract the relative path for YAML path construction
	yamlPath := "files.links" // Default assumption

	return &ConfigPatch{
		LayerPath:  layerPath,
		YAMLPath:   yamlPath,
		Operation:  PatchOpModify,
		OldValue:   d.ExpectedHash,
		NewValue:   d.CurrentHash,
		Provenance: "drift:" + d.Path,
	}
}

// PatchFromConfigDiff creates a patch from a ConfigPatch (domain type) to config.Patch (writer type).
func PatchFromConfigDiff(cp ConfigPatch) config.Patch {
	var op config.PatchOp
	switch cp.Operation {
	case PatchOpAdd:
		op = config.PatchOpAdd
	case PatchOpModify:
		op = config.PatchOpModify
	case PatchOpRemove:
		op = config.PatchOpRemove
	default:
		op = config.PatchOpModify
	}

	return config.Patch{
		LayerPath:  cp.LayerPath,
		YAMLPath:   cp.YAMLPath,
		Operation:  op,
		OldValue:   cp.OldValue,
		NewValue:   cp.NewValue,
		Provenance: cp.Provenance,
	}
}

// ConfigPatchesToWriterPatches converts app ConfigPatches to config.Patches.
func ConfigPatchesToWriterPatches(patches []ConfigPatch) []config.Patch {
	result := make([]config.Patch, len(patches))
	for i, p := range patches {
		result[i] = PatchFromConfigDiff(p)
	}
	return result
}
