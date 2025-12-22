package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLayer_MinimalLayer_ReturnsLayer(t *testing.T) {
	t.Parallel()

	yaml := `name: base`

	layer, err := config.ParseLayer([]byte(yaml))

	require.NoError(t, err)
	assert.Equal(t, "base", layer.Name.String())
}

func TestParseLayer_WithPackages_ParsesBrewFormulae(t *testing.T) {
	t.Parallel()

	yaml := `
name: base
packages:
  brew:
    formulae:
      - git
      - ripgrep
      - fd
    casks:
      - visual-studio-code
      - docker
`

	layer, err := config.ParseLayer([]byte(yaml))

	require.NoError(t, err)
	assert.Equal(t, "base", layer.Name.String())
	assert.ElementsMatch(t, []string{"git", "ripgrep", "fd"}, layer.Packages.Brew.Formulae)
	assert.ElementsMatch(t, []string{"visual-studio-code", "docker"}, layer.Packages.Brew.Casks)
}

func TestParseLayer_WithFiles_ParsesFileDeclarations(t *testing.T) {
	t.Parallel()

	yaml := `
name: identity.work
files:
  - path: ~/.gitconfig
    mode: generated
  - path: ~/.ssh/config
    mode: template
    template: ssh/config.tmpl
`

	layer, err := config.ParseLayer([]byte(yaml))

	require.NoError(t, err)
	assert.Equal(t, "identity.work", layer.Name.String())
	require.Len(t, layer.Files, 2)

	assert.Equal(t, "~/.gitconfig", layer.Files[0].Path)
	assert.Equal(t, config.FileModeGenerated, layer.Files[0].Mode)

	assert.Equal(t, "~/.ssh/config", layer.Files[1].Path)
	assert.Equal(t, config.FileModeTemplate, layer.Files[1].Mode)
	assert.Equal(t, "ssh/config.tmpl", layer.Files[1].Template)
}

func TestParseLayer_InvalidYAML_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
name: base
packages:
  - invalid: yaml: structure
`

	_, err := config.ParseLayer([]byte(yaml))

	require.Error(t, err)
}

func TestParseLayer_EmptyName_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `name: ""`

	_, err := config.ParseLayer([]byte(yaml))

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyLayerName)
}

func TestParseLayer_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
packages:
  brew:
    formulae:
      - git
`

	_, err := config.ParseLayer([]byte(yaml))

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrEmptyLayerName)
}

func TestLayer_SetProvenance_SetsProvenanceField(t *testing.T) {
	t.Parallel()

	yaml := `name: base`
	layer, err := config.ParseLayer([]byte(yaml))
	require.NoError(t, err)

	layer.SetProvenance("/path/to/layers/base.yaml")

	assert.Equal(t, "/path/to/layers/base.yaml", layer.Provenance)
}
