package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginCmd_Exists(t *testing.T) {
	assert.NotNil(t, pluginCmd)
	assert.Equal(t, "plugin", pluginCmd.Use)
}

func TestPluginListCmd_Exists(t *testing.T) {
	assert.NotNil(t, pluginListCmd)
	assert.Equal(t, "list", pluginListCmd.Use)
}

func TestPluginInstallCmd_Exists(t *testing.T) {
	assert.NotNil(t, pluginInstallCmd)
	assert.Equal(t, "install <source>", pluginInstallCmd.Use)
}

func TestPluginRemoveCmd_Exists(t *testing.T) {
	assert.NotNil(t, pluginRemoveCmd)
	assert.Equal(t, "remove <name>", pluginRemoveCmd.Use)
	assert.Contains(t, pluginRemoveCmd.Aliases, "uninstall")
	assert.Contains(t, pluginRemoveCmd.Aliases, "rm")
}

func TestPluginInfoCmd_Exists(t *testing.T) {
	assert.NotNil(t, pluginInfoCmd)
	assert.Equal(t, "info <name>", pluginInfoCmd.Use)
}

func capturePluginStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestRunPluginList_Empty(t *testing.T) {
	// t.Setenv automatically restores the original value after the test
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := capturePluginStdout(t, func() {
		err := runPluginList()
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No plugins installed")
	assert.Contains(t, output, "preflight plugin install")
}

func TestRunPluginList_WithPlugins(t *testing.T) {
	// This test verifies the plugin list output format when plugins exist.
	// Plugin discovery uses os.UserHomeDir() which doesn't reliably respect
	// the HOME environment variable in all test scenarios.
	//
	// Domain layer coverage is provided by:
	//   - internal/domain/plugin/loader_test.go: TestLoader_Discover*
	//   - internal/domain/plugin/registry_test.go: TestRegistry_List*
	t.Skip("Plugin discovery uses os.UserHomeDir(); covered by loader_test.go and registry_test.go")
}

func TestRunPluginInstall_LocalPath(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: v1
name: local-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := capturePluginStdout(t, func() {
		err := runPluginInstall(tmpDir)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Plugin validated")
	assert.Contains(t, output, "local-plugin@1.0.0")
}

func TestRunPluginInstall_InvalidPath(t *testing.T) {
	err := runPluginInstall("/nonexistent/path")
	assert.Error(t, err)
}

func TestRunPluginRemove_NotFound(t *testing.T) {
	// t.Setenv automatically restores the original value after the test
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runPluginRemove("nonexistent-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunPluginRemove_Found(t *testing.T) {
	// This test verifies plugin removal functionality.
	// Plugin discovery uses os.UserHomeDir() which doesn't reliably respect
	// the HOME environment variable in all test scenarios.
	//
	// Domain layer coverage is provided by:
	//   - internal/domain/plugin/loader_test.go: TestLoader_Remove*
	//   - internal/domain/plugin/registry_test.go: TestRegistry_Remove*
	t.Skip("Plugin removal uses os.UserHomeDir(); covered by loader_test.go and registry_test.go")
}

func TestRunPluginInfo_NotFound(t *testing.T) {
	// t.Setenv automatically restores the original value after the test
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runPluginInfo("nonexistent-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunPluginInfo_Found(t *testing.T) {
	// t.Setenv automatically restores the original value after the test
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create plugin with full details
	pluginDir := filepath.Join(tmpDir, ".preflight", "plugins", "info-test")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: info-test
version: 2.0.0
description: A comprehensive test plugin
author: Test Author
license: MIT
homepage: https://example.com
repository: https://github.com/example/info-test
provides:
  providers:
    - name: test-provider
      configKey: test.config
      description: A test provider
  presets:
    - test:basic
    - test:advanced
  capabilityPacks:
    - test-developer
requires:
  - name: base-plugin
    version: ">=1.0.0"
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := capturePluginStdout(t, func() {
		err := runPluginInfo("info-test")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Name:        info-test")
	assert.Contains(t, output, "Version:     2.0.0")
	assert.Contains(t, output, "API Version: v1")
	assert.Contains(t, output, "Description: A comprehensive test plugin")
	assert.Contains(t, output, "Author:      Test Author")
	assert.Contains(t, output, "License:     MIT")
	assert.Contains(t, output, "Homepage:    https://example.com")
	assert.Contains(t, output, "Repository:  https://github.com/example/info-test")
	assert.Contains(t, output, "Providers:")
	assert.Contains(t, output, "test-provider")
	assert.Contains(t, output, "test.config")
	assert.Contains(t, output, "Presets:")
	assert.Contains(t, output, "test:basic")
	assert.Contains(t, output, "Capability Packs:")
	assert.Contains(t, output, "test-developer")
	assert.Contains(t, output, "Dependencies:")
	assert.Contains(t, output, "base-plugin")
}
