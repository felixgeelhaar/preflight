package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/drift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatchGenerator(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	assert.NotNil(t, generator)
	assert.Equal(t, driftService, generator.driftService)
}

func TestPatchGenerator_GenerateFromDrifts_NoDrifts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	ctx := context.Background()
	patches, err := generator.GenerateFromDrifts(ctx, []drift.Drift{}, tmpDir)

	require.NoError(t, err)
	assert.Empty(t, patches)
}

func TestPatchGenerator_GenerateFromDrifts_NoDriftDetected(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	ctx := context.Background()

	// Drift with same hashes (no actual drift)
	drifts := []drift.Drift{
		drift.NewDrift("/path/to/file", "abc123", "abc123", time.Now(), drift.TypeNone),
	}

	patches, err := generator.GenerateFromDrifts(ctx, drifts, tmpDir)

	require.NoError(t, err)
	assert.Empty(t, patches)
}

func TestPatchGenerator_GenerateFromDrifts_WithDrift(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	ctx := context.Background()

	// Create the file first so RecordApplied can compute its hash
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	// Record the file so we have source layer info
	err = driftService.RecordApplied(ctx, testFile, "base")
	require.NoError(t, err)

	// Create drift for the tracked file
	drifts := []drift.Drift{
		drift.NewDrift(testFile, "newhash", "oldhash", time.Now(), drift.TypeManual),
	}

	patches, err := generator.GenerateFromDrifts(ctx, drifts, tmpDir)

	require.NoError(t, err)
	assert.Len(t, patches, 1)
	assert.Contains(t, patches[0].LayerPath, "base")
	assert.Equal(t, "drift:"+testFile, patches[0].Provenance)
}

func TestPatchGenerator_GenerateFromDrifts_UnknownSourceLayer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	ctx := context.Background()

	// Create drift for untracked file (no source layer)
	drifts := []drift.Drift{
		drift.NewDrift("/unknown/file", "newhash", "oldhash", time.Now(), drift.TypeManual),
	}

	patches, err := generator.GenerateFromDrifts(ctx, drifts, tmpDir)

	require.NoError(t, err)
	assert.Empty(t, patches) // No patches for untracked files
}

func TestPatchGenerator_GenerateFromIssues_Empty(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	patches := generator.GenerateFromIssues([]DoctorIssue{}, tmpDir)

	assert.Empty(t, patches)
}

func TestPatchGenerator_GenerateFromIssues_NonFixable(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	issues := []DoctorIssue{
		{
			Provider: "files",
			StepID:   "files:link:test",
			Severity: SeverityError,
			Message:  "Some error",
			Fixable:  false,
		},
	}

	patches := generator.GenerateFromIssues(issues, tmpDir)

	assert.Empty(t, patches)
}

func TestPatchGenerator_GenerateFromIssues_DriftIssue(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	issues := []DoctorIssue{
		{
			Provider: "files",
			StepID:   "files:link:zshrc",
			Severity: SeverityWarning,
			Message:  "Configuration drift detected",
			Expected: "expected state",
			Actual:   "actual state",
			Fixable:  true,
		},
	}

	patches := generator.GenerateFromIssues(issues, tmpDir)

	assert.Len(t, patches, 1)
	assert.Contains(t, patches[0].LayerPath, "captured.yaml")
	assert.Equal(t, "files:link:zshrc", patches[0].YAMLPath)
	assert.Equal(t, PatchOpModify, patches[0].Operation)
	assert.Equal(t, "drift-detected", patches[0].Provenance)
}

func TestPatchGenerator_resolveLayerPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	driftService := NewDriftService(tmpDir)
	generator := NewPatchGenerator(driftService)

	tests := []struct {
		name        string
		sourceLayer string
		wantEmpty   bool
	}{
		{
			name:        "empty source layer",
			sourceLayer: "",
			wantEmpty:   true,
		},
		{
			name:        "base layer",
			sourceLayer: "base",
			wantEmpty:   false,
		},
		{
			name:        "identity layer",
			sourceLayer: "identity.work",
			wantEmpty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := generator.resolveLayerPath(tmpDir, tt.sourceLayer)

			if tt.wantEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Contains(t, result, tt.sourceLayer)
			}
		})
	}
}

func TestPatchFromConfigDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    ConfigPatch
		wantOp   config.PatchOp
		wantPath string
	}{
		{
			name: "add operation",
			input: ConfigPatch{
				LayerPath: "/path/to/layer.yaml",
				YAMLPath:  "brew.formulae",
				Operation: PatchOpAdd,
				NewValue:  "ripgrep",
			},
			wantOp:   config.PatchOpAdd,
			wantPath: "/path/to/layer.yaml",
		},
		{
			name: "modify operation",
			input: ConfigPatch{
				LayerPath: "/path/to/layer.yaml",
				YAMLPath:  "git.user.name",
				Operation: PatchOpModify,
				OldValue:  "old",
				NewValue:  "new",
			},
			wantOp:   config.PatchOpModify,
			wantPath: "/path/to/layer.yaml",
		},
		{
			name: "remove operation",
			input: ConfigPatch{
				LayerPath: "/path/to/layer.yaml",
				YAMLPath:  "vscode.extensions[0]",
				Operation: PatchOpRemove,
				OldValue:  "old-extension",
			},
			wantOp:   config.PatchOpRemove,
			wantPath: "/path/to/layer.yaml",
		},
		{
			name: "unknown operation defaults to modify",
			input: ConfigPatch{
				LayerPath: "/path/to/layer.yaml",
				YAMLPath:  "some.path",
				Operation: "unknown",
			},
			wantOp:   config.PatchOpModify,
			wantPath: "/path/to/layer.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := PatchFromConfigDiff(tt.input)

			assert.Equal(t, tt.wantOp, result.Operation)
			assert.Equal(t, tt.wantPath, result.LayerPath)
			assert.Equal(t, tt.input.YAMLPath, result.YAMLPath)
		})
	}
}

func TestConfigPatchesToWriterPatches(t *testing.T) {
	t.Parallel()

	configPatches := []ConfigPatch{
		{
			LayerPath: "/path/to/layer1.yaml",
			YAMLPath:  "brew.formulae",
			Operation: PatchOpAdd,
			NewValue:  "ripgrep",
		},
		{
			LayerPath: "/path/to/layer2.yaml",
			YAMLPath:  "git.user.name",
			Operation: PatchOpModify,
			NewValue:  "new-name",
		},
	}

	result := ConfigPatchesToWriterPatches(configPatches)

	assert.Len(t, result, 2)
	assert.Equal(t, config.PatchOpAdd, result[0].Operation)
	assert.Equal(t, config.PatchOpModify, result[1].Operation)
}

func TestConfigPatchesToWriterPatches_Empty(t *testing.T) {
	t.Parallel()

	result := ConfigPatchesToWriterPatches([]ConfigPatch{})

	assert.Empty(t, result)
}
