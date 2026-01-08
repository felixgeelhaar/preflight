package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	result, err := loader.Discover(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result.Plugins)
	assert.False(t, result.HasErrors())
}

func TestLoader_Discover_NonExistentPath(t *testing.T) {
	// Use a path inside a temp dir that definitely doesn't exist
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "this-path-does-not-exist")

	loader := NewLoader().WithSearchPaths(nonExistentPath)

	result, err := loader.Discover(context.Background())
	require.NoError(t, err) // Should not error on non-existent paths
	assert.Empty(t, result.Plugins)
	assert.False(t, result.HasErrors())
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
	result, err := loader.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, result.Plugins, 1)
	assert.Equal(t, "docker", result.Plugins[0].Manifest.Name)
	assert.Equal(t, "1.0.0", result.Plugins[0].Manifest.Version)
	assert.True(t, result.Plugins[0].Enabled)
	assert.False(t, result.HasErrors())
}

func TestLoader_Discover_SkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file (not directory)
	err := os.WriteFile(filepath.Join(tmpDir, "not-a-plugin.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)
	result, err := loader.Discover(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result.Plugins)
}

func TestLoader_Discover_SkipsInvalidPlugins(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid plugin directory (missing manifest)
	invalidDir := filepath.Join(tmpDir, "invalid-plugin")
	err := os.MkdirAll(invalidDir, 0755)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)
	result, err := loader.Discover(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result.Plugins)    // Should skip invalid plugins
	assert.True(t, result.HasErrors()) // But should capture the error
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

func TestLoader_LoadFromGit_InvalidRepo(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadFromGit("https://github.com/example/preflight-docker.git", "v1.0.0")
	// Now that git clone is implemented, it attempts to clone and fails for non-existent repos
	assert.Error(t, err)
	// Either a git clone error or git not found error is expected
	assert.True(t, IsGitCloneError(err) || IsGitNotFound(err), "expected GitCloneError or GitNotFoundError, got: %v", err)
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
provides:
  presets:
    - plugin-a:default
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
provides:
  capabilityPacks:
    - plugin-b-pack
`
	err = os.WriteFile(filepath.Join(pluginDir2, "plugin.yaml"), []byte(manifest2), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir1, tmpDir2)
	result, err := loader.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, result.Plugins, 2)

	// Verify both plugins were found
	names := make(map[string]bool)
	for _, p := range result.Plugins {
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

// Security Tests

func TestExtractRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "valid https url with .git",
			url:  "https://github.com/example/preflight-docker.git",
			want: "preflight-docker",
		},
		{
			name: "valid https url without .git",
			url:  "https://github.com/example/preflight-docker",
			want: "preflight-docker",
		},
		{
			name: "valid git url",
			url:  "git://github.com/example/my-plugin.git",
			want: "my-plugin",
		},
		{
			name:    "invalid scheme ftp",
			url:     "ftp://example.com/plugin.git",
			wantErr: true,
		},
		{
			name:    "empty path",
			url:     "https://github.com/",
			wantErr: true,
		},
		{
			name:    "just .git",
			url:     "https://github.com/example/.git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractRepoName(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidatePluginName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid name", input: "docker", wantErr: false},
		{name: "valid with hyphen", input: "preflight-docker", wantErr: false},
		{name: "valid with underscore", input: "my_plugin", wantErr: false},
		{name: "path traversal dots", input: "..", wantErr: true},
		{name: "path traversal in name", input: "../etc", wantErr: true},
		{name: "forward slash", input: "path/to/plugin", wantErr: true},
		{name: "backslash", input: "path\\to\\plugin", wantErr: true},
		{name: "hidden file", input: ".hidden", wantErr: true},
		{name: "empty name", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePluginName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoader_LoadFromGit_PathTraversal(t *testing.T) {
	t.Parallel()

	loader := NewLoader()

	// Test various path traversal attempts
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "path traversal in repo name",
			url:  "https://github.com/evil/..%2F..%2F..%2Fetc%2Fpasswd.git",
		},
		{
			name: "invalid scheme",
			url:  "file:///etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := loader.LoadFromGit(tt.url, "")
			assert.Error(t, err)
		})
	}
}

func TestLoader_LoadFromPath_ReturnsErrManifestNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader()

	_, err := loader.LoadFromPath(tmpDir)
	assert.ErrorIs(t, err, ErrManifestNotFound)
}

// Context cancellation tests

func TestLoader_Discover_ContextCancelled(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a valid plugin directory
	pluginDir := filepath.Join(tmpDir, "docker")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: docker
version: 1.0.0
provides:
  providers:
    - name: docker
      configKey: docker
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := loader.Discover(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}

func TestLoader_Discover_ContextCancelledDuringDiscovery(t *testing.T) {
	t.Parallel()

	// Create multiple plugin directories to increase discovery time
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	for i := 0; i < 5; i++ {
		pluginDir := filepath.Join(tmpDir1, fmt.Sprintf("plugin-%d", i))
		err := os.MkdirAll(pluginDir, 0755)
		require.NoError(t, err)

		manifest := fmt.Sprintf(`apiVersion: v1
name: plugin-%d
version: 1.0.0
provides:
  presets:
    - plugin-%d:default
`, i, i)
		err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
		require.NoError(t, err)
	}

	loader := NewLoader().WithSearchPaths(tmpDir1, tmpDir2)

	// Use a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment to timeout
	time.Sleep(10 * time.Millisecond)

	result, err := loader.Discover(ctx)
	// Either returns error from context or partial results
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	} else {
		// May have found some plugins before cancellation
		assert.NotNil(t, result)
	}
}

func TestLoader_Discover_WithValidContext(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: test-plugin
version: 1.0.0
provides:
  presets:
    - test:default
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	loader := NewLoader().WithSearchPaths(tmpDir)

	// Use a context with a long timeout (should complete)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := loader.Discover(ctx)
	require.NoError(t, err)
	assert.Len(t, result.Plugins, 1)
	assert.Equal(t, "test-plugin", result.Plugins[0].Manifest.Name)
}

// Manifest Size Enforcement Tests

func TestLoader_LoadFromPath_ManifestSizeLimit(t *testing.T) {
	t.Parallel()

	t.Run("manifest under size limit succeeds", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		manifest := `apiVersion: v1
name: small-plugin
version: 1.0.0
provides:
  presets:
    - small:default
`
		err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
		require.NoError(t, err)

		loader := NewLoader()
		plugin, err := loader.LoadFromPath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "small-plugin", plugin.Manifest.Name)
	})

	t.Run("manifest over size limit returns ManifestSizeError", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a manifest larger than 256KB (maxManifestSize)
		// The limit is 256 * 1024 = 262144 bytes
		largeData := make([]byte, 300*1024) // 300KB
		for i := range largeData {
			largeData[i] = 'x'
		}

		// Create valid YAML structure but with very long description
		manifest := fmt.Sprintf(`apiVersion: v1
name: large-plugin
version: 1.0.0
description: %s
provides:
  presets:
    - large:default
`, string(largeData))

		err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
		require.NoError(t, err)

		loader := NewLoader()
		_, err = loader.LoadFromPath(tmpDir)
		require.Error(t, err)

		// Verify it's a ManifestSizeError
		assert.True(t, IsManifestSizeError(err), "expected ManifestSizeError")
	})

	t.Run("manifest exactly at size limit succeeds", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a manifest that's just under the limit
		// maxManifestSize is 256 * 1024 = 262144 bytes
		baseYAML := `apiVersion: v1
name: edge-plugin
version: 1.0.0
provides:
  presets:
    - edge:default
description: `
		// Calculate padding needed to reach just under 256KB
		padding := make([]byte, 256*1024-len(baseYAML)-100) // Leave some room for safety
		for i := range padding {
			padding[i] = 'x'
		}

		manifest := baseYAML + string(padding)

		err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
		require.NoError(t, err)

		loader := NewLoader()
		plugin, err := loader.LoadFromPath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "edge-plugin", plugin.Manifest.Name)
	})
}

func TestManifestSizeError_Details(t *testing.T) {
	t.Parallel()

	err := &ManifestSizeError{
		Size:  300 * 1024,
		Limit: 256 * 1024,
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "307200") // 300 * 1024
	assert.Contains(t, errMsg, "262144") // 256 * 1024
}
