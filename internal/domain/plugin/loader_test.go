package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
	assert.Len(t, loader.SearchPaths, 2)
}

func TestLoader_WithSearchPaths(t *testing.T) {
	loader := NewLoader().WithSearchPaths("/custom/path1", "/custom/path2")
	assert.Len(t, loader.SearchPaths, 2)
	assert.Equal(t, "/custom/path1", loader.SearchPaths[0])
	assert.Equal(t, "/custom/path2", loader.SearchPaths[1])
}

func TestLoader_Discover_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader().WithSearchPaths(tmpDir)

	plugins, err := loader.Discover()
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestLoader_Discover_NonExistentPath(t *testing.T) {
	loader := NewLoader().WithSearchPaths("/nonexistent/path")

	plugins, err := loader.Discover()
	require.NoError(t, err) // Should not error on non-existent paths
	assert.Empty(t, plugins)
}

func TestLoader_Discover_WithPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid plugin directory
	pluginDir := filepath.Join(tmpDir, "docker")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: docker
version: 1.0.0
description: Docker provider for Preflight
provides:
  providers:
    - name: docker
      configKey: docker
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)
	plugins, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "docker", plugins[0].Manifest.Name)
	assert.Equal(t, "1.0.0", plugins[0].Manifest.Version)
	assert.True(t, plugins[0].Enabled)
}

func TestLoader_Discover_SkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file (not directory)
	err := os.WriteFile(filepath.Join(tmpDir, "not-a-plugin.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)
	plugins, err := loader.Discover()
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestLoader_Discover_SkipsInvalidPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid plugin directory (missing manifest)
	invalidDir := filepath.Join(tmpDir, "invalid-plugin")
	err := os.MkdirAll(invalidDir, 0755)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)
	plugins, err := loader.Discover()
	require.NoError(t, err)
	assert.Empty(t, plugins) // Should skip invalid plugins
}

func TestLoader_LoadFromPath_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: v1
name: kubernetes
version: 2.0.0
description: Kubernetes provider
author: K8s Team
license: Apache-2.0
provides:
  providers:
    - name: kubectl
      configKey: kubernetes.kubectl
      description: kubectl installation
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	plugin, err := loader.LoadFromPath(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "kubernetes", plugin.Manifest.Name)
	assert.Equal(t, "2.0.0", plugin.Manifest.Version)
	assert.Equal(t, "Kubernetes provider", plugin.Manifest.Description)
	assert.Equal(t, "K8s Team", plugin.Manifest.Author)
	assert.Equal(t, "Apache-2.0", plugin.Manifest.License)
	assert.Len(t, plugin.Manifest.Provides.Providers, 1)
	assert.Equal(t, tmpDir, plugin.Path)
	assert.True(t, plugin.Enabled)
	assert.False(t, plugin.LoadedAt.IsZero())
}

func TestLoader_LoadFromPath_MissingManifest(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader()
	_, err := loader.LoadFromPath(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plugin.yaml not found")
}

func TestLoader_LoadFromPath_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte("invalid: yaml: :"), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	_, err = loader.LoadFromPath(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing plugin.yaml")
}

func TestLoader_LoadFromPath_InvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing required fields
	manifest := `apiVersion: v1
name: ""
version: ""
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	_, err = loader.LoadFromPath(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest")
}

func TestLoader_LoadFromGit_NotImplemented(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadFromGit("https://github.com/example/preflight-docker.git", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git clone not implemented")
}

func TestInstallPath(t *testing.T) {
	path, err := InstallPath()
	require.NoError(t, err)
	assert.Contains(t, path, ".preflight")
	assert.Contains(t, path, "plugins")
}

func TestEnsureInstallPath(t *testing.T) {
	// Save and restore HOME
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer func() {
		_ = os.Setenv("HOME", origHome)
	}()

	path, err := EnsureInstallPath()
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLoader_Discover_MultipleSearchPaths(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create plugin in first path
	pluginDir1 := filepath.Join(tmpDir1, "plugin-a")
	err := os.MkdirAll(pluginDir1, 0755)
	require.NoError(t, err)
	manifest1 := `apiVersion: v1
name: plugin-a
version: 1.0.0
`
	err = os.WriteFile(filepath.Join(pluginDir1, "plugin.yaml"), []byte(manifest1), 0644)
	require.NoError(t, err)

	// Create plugin in second path
	pluginDir2 := filepath.Join(tmpDir2, "plugin-b")
	err = os.MkdirAll(pluginDir2, 0755)
	require.NoError(t, err)
	manifest2 := `apiVersion: v1
name: plugin-b
version: 2.0.0
`
	err = os.WriteFile(filepath.Join(pluginDir2, "plugin.yaml"), []byte(manifest2), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir1, tmpDir2)
	plugins, err := loader.Discover()
	require.NoError(t, err)
	assert.Len(t, plugins, 2)

	// Verify both plugins were found
	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Manifest.Name] = true
	}
	assert.True(t, names["plugin-a"])
	assert.True(t, names["plugin-b"])
}

func TestLoader_LoadFromPath_WithDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: v1
name: kubernetes
version: 1.0.0
provides:
  providers:
    - name: kubectl
      configKey: kubernetes
requires:
  - name: docker
    version: ">=1.0.0"
  - name: helm
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	plugin, err := loader.LoadFromPath(tmpDir)
	require.NoError(t, err)

	assert.Len(t, plugin.Manifest.Requires, 2)
	assert.Equal(t, "docker", plugin.Manifest.Requires[0].Name)
	assert.Equal(t, ">=1.0.0", plugin.Manifest.Requires[0].Version)
	assert.Equal(t, "helm", plugin.Manifest.Requires[1].Name)
	assert.Empty(t, plugin.Manifest.Requires[1].Version)
}

func TestLoader_LoadFromPath_WithPresets(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: v1
name: kubernetes
version: 1.0.0
provides:
  providers:
    - name: kubectl
      configKey: kubernetes
  presets:
    - k8s:dev
    - k8s:prod
  capabilityPacks:
    - k8s-developer
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader()
	plugin, err := loader.LoadFromPath(tmpDir)
	require.NoError(t, err)

	assert.Len(t, plugin.Manifest.Provides.Presets, 2)
	assert.Equal(t, "k8s:dev", plugin.Manifest.Provides.Presets[0])
	assert.Len(t, plugin.Manifest.Provides.CapabilityPacks, 1)
	assert.Equal(t, "k8s-developer", plugin.Manifest.Provides.CapabilityPacks[0])
}
