package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerger_Merge_SingleLayer_ReturnsUnchanged(t *testing.T) {
	t.Parallel()

	layer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
`))
	require.NoError(t, err)
	layer.SetProvenance("layers/base.yaml")

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*layer})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git"}, merged.Packages.Brew.Formulae)
}

func TestMerger_Merge_TwoLayers_ListUnion(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
      - ripgrep
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    formulae:
      - docker
      - kubectl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"git", "ripgrep", "docker", "kubectl"}, merged.Packages.Brew.Formulae)
}

func TestMerger_Merge_Files_CombinesDeclarations(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
files:
  - path: ~/.gitconfig
    mode: generated
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
files:
  - path: ~/.ssh/config
    mode: template
    template: ssh/config.tmpl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	require.Len(t, merged.Files, 2)
}

func TestMerger_Merge_Files_LastWinsForSamePath(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
files:
  - path: ~/.gitconfig
    mode: generated
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
files:
  - path: ~/.gitconfig
    mode: template
    template: git/config.tmpl
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	require.Len(t, merged.Files, 1)
	assert.Equal(t, config.FileModeTemplate, merged.Files[0].Mode)
	assert.Equal(t, "git/config.tmpl", merged.Files[0].Template)
}

func TestMerger_Merge_Provenance_TracksSourceLayer(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    formulae:
      - git
`))
	require.NoError(t, err)
	baseLayer.SetProvenance("layers/base.yaml")

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    formulae:
      - docker
`))
	require.NoError(t, err)
	workLayer.SetProvenance("layers/identity.work.yaml")

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)

	// Check provenance is tracked
	gitProv := merged.GetProvenance("packages.brew.formulae", "git")
	assert.Equal(t, "layers/base.yaml", gitProv)

	dockerProv := merged.GetProvenance("packages.brew.formulae", "docker")
	assert.Equal(t, "layers/identity.work.yaml", dockerProv)
}

func TestMerger_Merge_EmptyLayers_ReturnsEmptyConfig(t *testing.T) {
	t.Parallel()

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{})

	require.NoError(t, err)
	assert.Empty(t, merged.Packages.Brew.Formulae)
	assert.Empty(t, merged.Files)
}

func TestMerger_Merge_Casks_Union(t *testing.T) {
	t.Parallel()

	baseLayer, err := config.ParseLayer([]byte(`
name: base
packages:
  brew:
    casks:
      - visual-studio-code
`))
	require.NoError(t, err)

	workLayer, err := config.ParseLayer([]byte(`
name: identity.work
packages:
  brew:
    casks:
      - docker
      - visual-studio-code
`))
	require.NoError(t, err)

	merger := config.NewMerger()
	merged, err := merger.Merge([]config.Layer{*baseLayer, *workLayer})

	require.NoError(t, err)
	// Should be union (deduplicated)
	assert.Len(t, merged.Packages.Brew.Casks, 2)
	assert.ElementsMatch(t, []string{"visual-studio-code", "docker"}, merged.Packages.Brew.Casks)
}
