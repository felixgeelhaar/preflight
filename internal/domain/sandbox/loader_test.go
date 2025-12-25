package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginManifest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid manifest", func(t *testing.T) {
		t.Parallel()
		m := &PluginManifest{
			ID:       "test-plugin",
			Name:     "Test Plugin",
			Module:   "plugin.wasm",
			Checksum: "abc123",
		}
		err := m.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()
		m := &PluginManifest{
			Name:     "Test Plugin",
			Module:   "plugin.wasm",
			Checksum: "abc123",
		}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
		assert.Contains(t, err.Error(), "id")
	})

	t.Run("missing name", func(t *testing.T) {
		t.Parallel()
		m := &PluginManifest{
			ID:       "test-plugin",
			Module:   "plugin.wasm",
			Checksum: "abc123",
		}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing module", func(t *testing.T) {
		t.Parallel()
		m := &PluginManifest{
			ID:       "test-plugin",
			Name:     "Test Plugin",
			Checksum: "abc123",
		}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
		assert.Contains(t, err.Error(), "module")
	})

	t.Run("missing checksum", func(t *testing.T) {
		t.Parallel()
		m := &PluginManifest{
			ID:     "test-plugin",
			Name:   "Test Plugin",
			Module: "plugin.wasm",
		}
		err := m.Validate()
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
		assert.Contains(t, err.Error(), "checksum")
	})
}

func TestNewLoader(t *testing.T) {
	t.Parallel()

	loader := NewLoader("/path/to/plugins")
	assert.NotNil(t, loader)
	assert.Equal(t, "/path/to/plugins", loader.basePath)
}

func TestLoader_LoadManifest(t *testing.T) {
	t.Parallel()

	t.Run("loads valid manifest", func(t *testing.T) {
		t.Parallel()

		// Create temp directory with plugin
		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "my-plugin")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		manifestContent := `
id: my-plugin
name: My Plugin
version: 1.0.0
description: A test plugin
author: Test Author
module: plugin.wasm
checksum: abc123def456
capabilities:
  - name: files:read
    justification: Read config files
  - name: shell:execute
    justification: Run commands
    optional: true
`
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		manifest, err := loader.LoadManifest("my-plugin")

		require.NoError(t, err)
		assert.Equal(t, "my-plugin", manifest.ID)
		assert.Equal(t, "My Plugin", manifest.Name)
		assert.Equal(t, "1.0.0", manifest.Version)
		assert.Equal(t, "A test plugin", manifest.Description)
		assert.Equal(t, "Test Author", manifest.Author)
		assert.Equal(t, "plugin.wasm", manifest.Module)
		assert.Equal(t, "abc123def456", manifest.Checksum)
		assert.Len(t, manifest.Capabilities, 2)
		assert.Equal(t, "files:read", manifest.Capabilities[0].Name)
		assert.False(t, manifest.Capabilities[0].Optional)
		assert.Equal(t, "shell:execute", manifest.Capabilities[1].Name)
		assert.True(t, manifest.Capabilities[1].Optional)
	})

	t.Run("returns error for missing manifest", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		loader := NewLoader(tmpDir)

		_, err := loader.LoadManifest("nonexistent")
		assert.ErrorIs(t, err, ErrPluginManifestNotFound)
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "bad-plugin")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte("not: valid: yaml: content"),
			0o644,
		))

		loader := NewLoader(tmpDir)
		_, err := loader.LoadManifest("bad-plugin")
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
	})

	t.Run("returns error for incomplete manifest", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "incomplete")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte("id: incomplete\n"),
			0o644,
		))

		loader := NewLoader(tmpDir)
		_, err := loader.LoadManifest("incomplete")
		assert.ErrorIs(t, err, ErrPluginManifestInvalid)
	})
}

func TestLoader_LoadPlugin(t *testing.T) {
	t.Parallel()

	t.Run("loads valid plugin", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "test-plugin")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		// Create WASM module
		wasmData := validWASMModuleForLoader()
		checksum := sha256Hex(wasmData)

		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.wasm"),
			wasmData,
			0o644,
		))

		manifestContent := "id: test-plugin\nname: Test Plugin\nmodule: plugin.wasm\nchecksum: " + checksum
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		ctx := context.Background()

		plugin, err := loader.LoadPlugin(ctx, "test-plugin")
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", plugin.ID)
		assert.Equal(t, "Test Plugin", plugin.Name)
		assert.Equal(t, wasmData, plugin.Module)
		assert.Equal(t, checksum, plugin.Checksum)
	})

	t.Run("returns error for missing module", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "missing-module")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		manifestContent := "id: missing-module\nname: Missing Module\nmodule: nonexistent.wasm\nchecksum: abc123"
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		ctx := context.Background()

		_, err := loader.LoadPlugin(ctx, "missing-module")
		assert.ErrorIs(t, err, ErrPluginModuleNotFound)
	})

	t.Run("returns error for checksum mismatch", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "bad-checksum")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		wasmData := validWASMModuleForLoader()
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.wasm"),
			wasmData,
			0o644,
		))

		manifestContent := "id: bad-checksum\nname: Bad Checksum\nmodule: plugin.wasm\nchecksum: wrongchecksum"
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		ctx := context.Background()

		_, err := loader.LoadPlugin(ctx, "bad-checksum")
		assert.ErrorIs(t, err, ErrPluginChecksumMismatch)
	})

	t.Run("loads plugin with capabilities", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "capable-plugin")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		wasmData := validWASMModuleForLoader()
		checksum := sha256Hex(wasmData)

		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.wasm"),
			wasmData,
			0o644,
		))

		manifestContent := `
id: capable-plugin
name: Capable Plugin
module: plugin.wasm
checksum: ` + checksum + `
capabilities:
  - name: files:read
    justification: Read configs
  - name: network:fetch
    justification: Download data
    optional: true
`
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		ctx := context.Background()

		plugin, err := loader.LoadPlugin(ctx, "capable-plugin")
		require.NoError(t, err)
		assert.NotNil(t, plugin.Capabilities)
		assert.Equal(t, 2, plugin.Capabilities.Count())
	})
}

func TestLoader_ListPlugins(t *testing.T) {
	t.Parallel()

	t.Run("lists plugin directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		// Create plugin directories
		for _, name := range []string{"plugin-a", "plugin-b"} {
			dir := filepath.Join(tmpDir, name)
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, "plugin.yaml"),
				[]byte("id: "+name+"\nname: "+name+"\nmodule: p.wasm\nchecksum: abc"),
				0o644,
			))
		}

		// Create non-plugin directory
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "not-a-plugin"), 0o755))

		// Create regular file
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello"), 0o644))

		loader := NewLoader(tmpDir)
		plugins, err := loader.ListPlugins()

		require.NoError(t, err)
		assert.Len(t, plugins, 2)
		assert.Contains(t, plugins, "plugin-a")
		assert.Contains(t, plugins, "plugin-b")
	})

	t.Run("returns empty for nonexistent directory", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader("/nonexistent/path")
		plugins, err := loader.ListPlugins()

		assert.NoError(t, err)
		assert.Empty(t, plugins)
	})
}

func TestCalculateChecksum(t *testing.T) {
	t.Parallel()

	t.Run("calculates correct checksum", func(t *testing.T) {
		t.Parallel()

		tmpFile := filepath.Join(t.TempDir(), "test.bin")
		data := []byte("hello world")
		require.NoError(t, os.WriteFile(tmpFile, data, 0o644))

		checksum, err := CalculateChecksum(tmpFile)
		require.NoError(t, err)

		// SHA256 of "hello world"
		expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		assert.Equal(t, expected, checksum)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		t.Parallel()

		_, err := CalculateChecksum("/nonexistent/file")
		assert.Error(t, err)
	})
}

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	runtime, err := NewWazeroRuntime(ctx)
	require.NoError(t, err)
	defer func() { _ = runtime.Close() }()

	loader := NewLoader("/plugins")
	executor := NewExecutor(runtime, loader)

	assert.NotNil(t, executor)
	assert.Equal(t, runtime, executor.runtime)
	assert.Equal(t, loader, executor.loader)
}

func TestExecutor_Run(t *testing.T) {
	t.Parallel()

	t.Run("runs valid plugin", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		runtime, err := NewWazeroRuntime(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = runtime.Close() })

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "run-test")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		wasmData := validWASMModuleForLoader()
		checksum := sha256Hex(wasmData)

		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.wasm"),
			wasmData,
			0o644,
		))

		manifestContent := "id: run-test\nname: Run Test\nmodule: plugin.wasm\nchecksum: " + checksum
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		executor := NewExecutor(runtime, loader)

		result, err := executor.Run(ctx, "run-test", DefaultConfig(), nil)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("returns error for missing plugin", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		runtime, err := NewWazeroRuntime(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = runtime.Close() })

		loader := NewLoader(t.TempDir())
		executor := NewExecutor(runtime, loader)

		_, err = executor.Run(ctx, "nonexistent", DefaultConfig(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "load plugin")
	})
}

func TestExecutor_ValidatePlugin(t *testing.T) {
	t.Parallel()

	t.Run("validates valid plugin", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		runtime, err := NewWazeroRuntime(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = runtime.Close() })

		tmpDir := t.TempDir()
		pluginDir := filepath.Join(tmpDir, "validate-test")
		require.NoError(t, os.MkdirAll(pluginDir, 0o755))

		wasmData := validWASMModuleForLoader()
		checksum := sha256Hex(wasmData)

		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.wasm"),
			wasmData,
			0o644,
		))

		manifestContent := "id: validate-test\nname: Validate Test\nmodule: plugin.wasm\nchecksum: " + checksum
		require.NoError(t, os.WriteFile(
			filepath.Join(pluginDir, "plugin.yaml"),
			[]byte(manifestContent),
			0o644,
		))

		loader := NewLoader(tmpDir)
		executor := NewExecutor(runtime, loader)

		err = executor.ValidatePlugin(ctx, "validate-test", DefaultConfig())
		assert.NoError(t, err)
	})
}

// validWASMModuleForLoader returns a minimal valid WASM module for loader tests.
func validWASMModuleForLoader() []byte {
	return []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // WASM version 1
		// Type section
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		// Function section
		0x03, 0x02, 0x01, 0x00,
		// Export section
		0x07, 0x08, 0x01, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x00, 0x00,
		// Code section
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}
}
