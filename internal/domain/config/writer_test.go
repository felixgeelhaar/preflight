package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLayerWriter(t *testing.T) {
	t.Parallel()

	writer := NewLayerWriter()
	assert.NotNil(t, writer)
}

func TestLayerWriter_ApplyPatch_Add(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `steps:
  - id: brew.git
    provider: brew
    formula: git
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "steps[1]",
		Operation: PatchOpAdd,
		NewValue: map[string]interface{}{
			"id":       "brew.curl",
			"provider": "brew",
			"formula":  "curl",
		},
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify patch was applied
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "brew.curl")
	assert.Contains(t, string(content), "curl")
}

func TestLayerWriter_ApplyPatch_Modify(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `files:
  links:
    - src: ~/.bashrc
      dest: ~/dotfiles/bashrc
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "files.links[0].src",
		Operation: PatchOpModify,
		OldValue:  "~/.bashrc",
		NewValue:  "~/.zshrc",
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify patch was applied
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "~/.zshrc")
	assert.NotContains(t, string(content), "~/.bashrc")
}

func TestLayerWriter_ApplyPatch_Remove(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `brew:
  formulae:
    - git
    - curl
    - wget
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "brew.formulae[1]",
		Operation: PatchOpRemove,
		OldValue:  "curl",
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify item was removed
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "git")
	assert.Contains(t, string(content), "wget")
	assert.NotContains(t, string(content), "curl")
}

func TestLayerWriter_ApplyPatch_FileNotFound(t *testing.T) {
	t.Parallel()

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: "/nonexistent/path/layer.yaml",
		YAMLPath:  "key",
		Operation: PatchOpModify,
		NewValue:  "value",
	}

	err := writer.ApplyPatch(patch)
	assert.Error(t, err)
}

func TestLayerWriter_ApplyPatches(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `name: test-layer
enabled: false
count: 1
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patches := []Patch{
		{
			LayerPath: layerPath,
			YAMLPath:  "enabled",
			Operation: PatchOpModify,
			OldValue:  false,
			NewValue:  true,
		},
		{
			LayerPath: layerPath,
			YAMLPath:  "count",
			Operation: PatchOpModify,
			OldValue:  1,
			NewValue:  5,
		},
	}

	err = writer.ApplyPatches(patches)
	require.NoError(t, err)

	// Verify all patches were applied
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "enabled: true")
	assert.Contains(t, string(content), "count: 5")
}

func TestLayerWriter_ApplyPatches_StopsOnError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `key: value
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patches := []Patch{
		{
			LayerPath: layerPath,
			YAMLPath:  "key",
			Operation: PatchOpModify,
			NewValue:  "new-value",
		},
		{
			LayerPath: "/nonexistent/path.yaml",
			YAMLPath:  "missing",
			Operation: PatchOpAdd,
			NewValue:  "data",
		},
	}

	err = writer.ApplyPatches(patches)
	assert.Error(t, err)
}

func TestLayerWriter_ApplyPatch_PreservesComments(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create layer file with comments
	initial := `# This is a config file
name: test
# This is important
value: 10
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "value",
		Operation: PatchOpModify,
		OldValue:  10,
		NewValue:  20,
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify comments are preserved
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# This is a config file")
	assert.Contains(t, string(content), "# This is important")
	assert.Contains(t, string(content), "value: 20")
}

func TestLayerWriter_ApplyPatch_AddTopLevelKey(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file
	initial := `existing: value
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "newkey",
		Operation: PatchOpAdd,
		NewValue:  "newvalue",
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify new key was added
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "existing: value")
	assert.Contains(t, string(content), "newkey: newvalue")
}

func TestLayerWriter_ApplyPatch_NestedPath(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layerPath := filepath.Join(tmpDir, "layer.yaml")

	// Create initial layer file with nested structure
	initial := `parent:
  child:
    grandchild: oldvalue
`
	err := os.WriteFile(layerPath, []byte(initial), 0644)
	require.NoError(t, err)

	writer := NewLayerWriter()
	patch := Patch{
		LayerPath: layerPath,
		YAMLPath:  "parent.child.grandchild",
		Operation: PatchOpModify,
		OldValue:  "oldvalue",
		NewValue:  "newvalue",
	}

	err = writer.ApplyPatch(patch)
	require.NoError(t, err)

	// Verify nested value was modified
	content, err := os.ReadFile(layerPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "grandchild: newvalue")
}
