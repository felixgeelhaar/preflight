package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExtractEnvVars_ValuesAndSecrets(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"env": map[string]interface{}{
			"EDITOR":    "nvim",
			"GOPATH":    "/home/user/go",
			"API_TOKEN": "secret://vault/api-token",
			"DB_PASS":   "secret://vault/db-pass",
		},
	}

	vars := extractEnvVars(config)
	assert.Len(t, vars, 4)

	varMap := make(map[string]EnvVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}

	// Non-secret values
	assert.Equal(t, "nvim", varMap["EDITOR"].Value)
	assert.False(t, varMap["EDITOR"].Secret)

	assert.Equal(t, "/home/user/go", varMap["GOPATH"].Value)
	assert.False(t, varMap["GOPATH"].Secret)

	// Secret values
	assert.True(t, varMap["API_TOKEN"].Secret)
	assert.True(t, varMap["DB_PASS"].Secret)
}

func TestExtractEnvVars_EmptyConfig(t *testing.T) {
	t.Parallel()

	vars := extractEnvVars(map[string]interface{}{})
	assert.Empty(t, vars)
}

func TestExtractEnvVars_NoEnvSection(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []string{"git"},
		},
	}

	vars := extractEnvVars(config)
	assert.Empty(t, vars)
}

func TestExtractEnvVars_NonMapEnvSection(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"env": "not a map",
	}

	vars := extractEnvVars(config)
	assert.Empty(t, vars)
}

func TestExtractEnvVars_IntegerValue(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"env": map[string]interface{}{
			"PORT": 8080,
		},
	}

	vars := extractEnvVars(config)
	require.Len(t, vars, 1)
	assert.Equal(t, "8080", vars[0].Value)
	assert.False(t, vars[0].Secret)
}

func TestExtractEnvVarsMap_ReturnsStringMap(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"env": map[string]interface{}{
			"SHELL":  "/bin/zsh",
			"TERM":   "xterm-256color",
			"DEBUG":  true,
			"GOMAXP": 4,
		},
	}

	result := extractEnvVarsMap(config)
	assert.Len(t, result, 4)
	assert.Equal(t, "/bin/zsh", result["SHELL"])
	assert.Equal(t, "xterm-256color", result["TERM"])
	assert.Equal(t, "true", result["DEBUG"])
	assert.Equal(t, "4", result["GOMAXP"])
}

func TestExtractEnvVarsMap_EmptyConfig(t *testing.T) {
	t.Parallel()

	result := extractEnvVarsMap(map[string]interface{}{})
	assert.Empty(t, result)
	assert.NotNil(t, result) // Should return empty map, not nil
}

func TestExtractEnvVarsMap_NoEnvSection(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"other": "value",
	}

	result := extractEnvVarsMap(config)
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestRunEnvSet_NewLayerFile(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "test-layer"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	output := captureStdout(t, func() {
		err := runEnvSet(&cobra.Command{}, []string{"MY_VAR", "my_value"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Set MY_VAR=my_value in layer test-layer")

	// Verify layer file was created
	layerPath := filepath.Join(tmpDir, "layers", "test-layer.yaml")
	data, err := os.ReadFile(layerPath)
	require.NoError(t, err)

	var layerData map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &layerData))

	env, ok := layerData["env"].(map[string]interface{})
	require.True(t, ok, "env section should exist")
	assert.Equal(t, "my_value", env["MY_VAR"])
}

func TestRunEnvSet_DefaultLayerIsBase(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "" // empty defaults to "base"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	output := captureStdout(t, func() {
		err := runEnvSet(&cobra.Command{}, []string{"EDITOR", "vim"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "in layer base")

	// Verify base.yaml was created
	layerPath := filepath.Join(tmpDir, "layers", "base.yaml")
	_, err := os.Stat(layerPath)
	require.NoError(t, err)
}

func TestRunEnvSet_ExistingLayerPreservesOtherVars(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "existing"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	// Create an existing layer with some vars
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"env": map[string]interface{}{
			"EXISTING_VAR": "keep_me",
		},
		"other_section": "preserved",
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "existing.yaml"), data, 0o644))

	// Add a new var
	captureStdout(t, func() {
		err := runEnvSet(&cobra.Command{}, []string{"NEW_VAR", "new_value"})
		require.NoError(t, err)
	})

	// Read back and verify both vars exist
	readData, err := os.ReadFile(filepath.Join(layersDir, "existing.yaml"))
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, yaml.Unmarshal(readData, &result))

	env, ok := result["env"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "keep_me", env["EXISTING_VAR"])
	assert.Equal(t, "new_value", env["NEW_VAR"])

	// Other sections should be preserved
	assert.Equal(t, "preserved", result["other_section"])
}

func TestRunEnvSet_OverwritesExistingVar(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "overwrite"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"env": map[string]interface{}{
			"MY_VAR": "old_value",
		},
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "overwrite.yaml"), data, 0o644))

	captureStdout(t, func() {
		err := runEnvSet(&cobra.Command{}, []string{"MY_VAR", "new_value"})
		require.NoError(t, err)
	})

	readData, err := os.ReadFile(filepath.Join(layersDir, "overwrite.yaml"))
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, yaml.Unmarshal(readData, &result))

	env := result["env"].(map[string]interface{})
	assert.Equal(t, "new_value", env["MY_VAR"])
}

func TestRunEnvUnset_RemovesVar(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "unset-layer"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"env": map[string]interface{}{
			"REMOVE_ME": "goodbye",
			"KEEP_ME":   "hello",
		},
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "unset-layer.yaml"), data, 0o644))

	output := captureStdout(t, func() {
		err := runEnvUnset(&cobra.Command{}, []string{"REMOVE_ME"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Removed REMOVE_ME from layer unset-layer")

	// Verify the var was removed but the other remains
	readData, err := os.ReadFile(filepath.Join(layersDir, "unset-layer.yaml"))
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, yaml.Unmarshal(readData, &result))

	env := result["env"].(map[string]interface{})
	assert.Nil(t, env["REMOVE_ME"])
	assert.Equal(t, "hello", env["KEEP_ME"])
}

func TestRunEnvUnset_VarNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "unset-missing"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"env": map[string]interface{}{
			"OTHER_VAR": "value",
		},
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "unset-missing.yaml"), data, 0o644))

	err = runEnvUnset(&cobra.Command{}, []string{"NONEXISTENT"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "NONEXISTENT")
}

func TestRunEnvUnset_NoEnvSection(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "no-env"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []string{"git"},
		},
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "no-env.yaml"), data, 0o644))

	err = runEnvUnset(&cobra.Command{}, []string{"ANYTHING"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no env section")
}

func TestRunEnvUnset_LayerFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "nonexistent-layer"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	err := runEnvUnset(&cobra.Command{}, []string{"VAR"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "layer not found")
}

func TestRunEnvUnset_DefaultLayerIsBase(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = ""
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	existingData := map[string]interface{}{
		"env": map[string]interface{}{
			"REMOVE_ME": "value",
		},
	}
	data, err := yaml.Marshal(existingData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), data, 0o644))

	output := captureStdout(t, func() {
		err := runEnvUnset(&cobra.Command{}, []string{"REMOVE_ME"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "from layer base")
}

func TestWriteEnvFile_ContentFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "EDITOR", Value: "nvim"},
		{Name: "GOPATH", Value: "/home/user/go"},
		{Name: "API_KEY", Value: "secret://vault/key", Secret: true},
		{Name: "SHELL", Value: "/bin/zsh"},
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	content := string(data)

	// Header present
	assert.Contains(t, content, "# Generated by preflight")
	assert.Contains(t, content, "do not edit manually")

	// Non-secret vars exported
	assert.Contains(t, content, `export EDITOR="nvim"`)
	assert.Contains(t, content, `export GOPATH="/home/user/go"`)
	assert.Contains(t, content, `export SHELL="/bin/zsh"`)

	// Secrets excluded
	assert.NotContains(t, content, "API_KEY")
	assert.NotContains(t, content, "secret://")
}

func TestWriteEnvFile_EmptyVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := WriteEnvFile([]EnvVar{})
	require.NoError(t, err)

	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	content := string(data)

	// Should still have the header
	assert.Contains(t, content, "# Generated by preflight")
	// But no export lines
	assert.NotContains(t, content, "export")
}

func TestWriteEnvFile_AllSecretsExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "SECRET1", Value: "secret://a", Secret: true},
		{Name: "SECRET2", Value: "secret://b", Secret: true},
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)
	content := string(data)

	assert.NotContains(t, content, "export")
}

func TestWriteEnvFile_CreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "TEST", Value: "value"},
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	// Verify directory was created
	preflightDir := filepath.Join(tmpDir, ".preflight")
	info, err := os.Stat(preflightDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestEnvCmd_AllSubcommandsRegistered(t *testing.T) {
	t.Parallel()

	expectedSubs := []string{"list", "set", "get", "unset", "export", "diff"}
	subNames := make(map[string]bool)
	for _, cmd := range envCmd.Commands() {
		subNames[cmd.Name()] = true
	}

	for _, expected := range expectedSubs {
		assert.True(t, subNames[expected], "env should have subcommand: %s", expected)
	}
}

func TestEnvCmd_PersistentFlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{"config", "config", "preflight.yaml"},
		{"target", "target", "default"},
		{"layer", "layer", ""},
		{"json", "json", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := envCmd.PersistentFlags().Lookup(tt.flag)
			require.NotNil(t, f, "persistent flag %s should exist", tt.flag)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

func TestEnvExportCmd_HasShellFlag(t *testing.T) {
	t.Parallel()

	f := envExportCmd.Flags().Lookup("shell")
	require.NotNil(t, f, "export command should have --shell flag")
	assert.Equal(t, "bash", f.DefValue)
}

func TestEnvSetCmd_RequiresExactlyTwoArgs(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, envSetCmd.Args)
}

func TestEnvGetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, envGetCmd.Args)
}

func TestEnvUnsetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, envUnsetCmd.Args)
}

func TestEnvDiffCmd_RequiresExactlyTwoArgs(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, envDiffCmd.Args)
}

func TestEnvVar_StructJSON(t *testing.T) {
	t.Parallel()

	v := EnvVar{
		Name:   "TEST",
		Value:  "value",
		Layer:  "base",
		Secret: false,
	}

	assert.Equal(t, "TEST", v.Name)
	assert.Equal(t, "value", v.Value)
	assert.Equal(t, "base", v.Layer)
	assert.False(t, v.Secret)
}

func TestRunEnvSet_InvalidYAMLInExistingLayer(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "bad-yaml"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	// Write invalid YAML
	err := os.WriteFile(filepath.Join(layersDir, "bad-yaml.yaml"), []byte("{{invalid"), 0o644)
	require.NoError(t, err)

	err = runEnvSet(&cobra.Command{}, []string{"VAR", "value"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse layer")
}

func TestRunEnvUnset_InvalidYAMLInExistingLayer(t *testing.T) {
	tmpDir := t.TempDir()

	savedConfigPath := envConfigPath
	savedLayer := envLayer
	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "bad-yaml-unset"
	defer func() {
		envConfigPath = savedConfigPath
		envLayer = savedLayer
	}()

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	err := os.WriteFile(filepath.Join(layersDir, "bad-yaml-unset.yaml"), []byte("{{invalid"), 0o644)
	require.NoError(t, err)

	err = runEnvUnset(&cobra.Command{}, []string{"VAR"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse layer")
}
