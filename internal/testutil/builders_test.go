package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestBuilder(t *testing.T) {
	t.Parallel()

	manifest := NewManifestBuilder().
		WithVersion(1).
		WithTarget("default", "base", "work").
		Build()

	assert.Equal(t, 1, manifest.Version)
	assert.Len(t, manifest.Targets, 1)
	assert.Equal(t, "default", manifest.Targets[0].Name)
	assert.Equal(t, []string{"base", "work"}, manifest.Targets[0].Layers)
}

func TestManifestBuilder_MultipleTargets(t *testing.T) {
	t.Parallel()

	manifest := NewManifestBuilder().
		WithTarget("work", "base", "work").
		WithTarget("personal", "base", "personal").
		Build()

	assert.Len(t, manifest.Targets, 2)
}

func TestManifestBuilder_Defaults(t *testing.T) {
	t.Parallel()

	manifest := NewManifestBuilder().
		WithDefaults(map[string]interface{}{
			"shell": map[string]interface{}{
				"preferred": "zsh",
			},
		}).
		Build()

	assert.NotNil(t, manifest.Defaults)
	assert.Equal(t, "zsh", manifest.Defaults["shell"].(map[string]interface{})["preferred"])
}

func TestLayerBuilder(t *testing.T) {
	t.Parallel()

	layer := NewLayerBuilder("base").
		WithBrew("git", "curl", "wget").
		Build()

	assert.Equal(t, "base", layer.Name)
	assert.Equal(t, []string{"git", "curl", "wget"}, layer.Brew.Formulae)
}

func TestLayerBuilder_WithGit(t *testing.T) {
	t.Parallel()

	layer := NewLayerBuilder("identity").
		WithGit("user.name", "John Doe").
		WithGit("user.email", "john@example.com").
		Build()

	assert.Equal(t, "John Doe", layer.Git.Config["user.name"])
	assert.Equal(t, "john@example.com", layer.Git.Config["user.email"])
}

func TestLayerBuilder_ToYAML(t *testing.T) {
	t.Parallel()

	layer := NewLayerBuilder("base").
		WithBrew("git").
		Build()

	yaml := layer.ToYAML()

	assert.Contains(t, yaml, "name: base")
	assert.Contains(t, yaml, "git")
}
