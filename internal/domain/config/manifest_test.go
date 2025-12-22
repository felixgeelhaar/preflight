package config_test

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest_MinimalManifest_ReturnsManifest(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  default:
    - base
`

	manifest, err := config.ParseManifest([]byte(yaml))

	require.NoError(t, err)
	require.Len(t, manifest.Targets, 1)

	defaultTarget, ok := manifest.Targets["default"]
	require.True(t, ok)
	require.Len(t, defaultTarget, 1)
	assert.Equal(t, "base", defaultTarget[0].String())
}

func TestParseManifest_WithMultipleTargets_ParsesAllTargets(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  work:
    - base
    - identity.work
    - role.go-developer
  personal:
    - base
    - identity.personal
`

	manifest, err := config.ParseManifest([]byte(yaml))

	require.NoError(t, err)
	require.Len(t, manifest.Targets, 2)

	workTarget := manifest.Targets["work"]
	require.Len(t, workTarget, 3)
	assert.Equal(t, "base", workTarget[0].String())
	assert.Equal(t, "identity.work", workTarget[1].String())
	assert.Equal(t, "role.go-developer", workTarget[2].String())

	personalTarget := manifest.Targets["personal"]
	require.Len(t, personalTarget, 2)
	assert.Equal(t, "base", personalTarget[0].String())
	assert.Equal(t, "identity.personal", personalTarget[1].String())
}

func TestParseManifest_WithDefaults_ParsesDefaults(t *testing.T) {
	t.Parallel()

	yaml := `
defaults:
  mode: locked
  editor: nvim

targets:
  work:
    - base
`

	manifest, err := config.ParseManifest([]byte(yaml))

	require.NoError(t, err)
	assert.Equal(t, config.ModeLocked, manifest.Defaults.Mode)
	assert.Equal(t, "nvim", manifest.Defaults.Editor)
}

func TestParseManifest_MissingTargets_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
defaults:
  mode: locked
`

	_, err := config.ParseManifest([]byte(yaml))

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrNoTargets)
}

func TestParseManifest_InvalidYAML_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  - invalid: yaml: structure
`

	_, err := config.ParseManifest([]byte(yaml))

	require.Error(t, err)
}

func TestParseManifest_InvalidLayerName_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  work:
    - "invalid/layer/name"
`

	_, err := config.ParseManifest([]byte(yaml))

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrInvalidLayerName)
}

func TestManifest_GetTarget_ReturnsLayerNames(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  work:
    - base
    - identity.work
`

	manifest, err := config.ParseManifest([]byte(yaml))
	require.NoError(t, err)

	targetName, _ := config.NewTargetName("work")
	layers, err := manifest.GetTarget(targetName)

	require.NoError(t, err)
	require.Len(t, layers, 2)
	assert.Equal(t, "base", layers[0].String())
	assert.Equal(t, "identity.work", layers[1].String())
}

func TestManifest_GetTarget_NotFound_ReturnsError(t *testing.T) {
	t.Parallel()

	yaml := `
targets:
  work:
    - base
`

	manifest, err := config.ParseManifest([]byte(yaml))
	require.NoError(t, err)

	targetName, _ := config.NewTargetName("nonexistent")
	_, err = manifest.GetTarget(targetName)

	require.Error(t, err)
	require.ErrorIs(t, err, config.ErrTargetNotFound)
}
