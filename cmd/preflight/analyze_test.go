package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeCmd_Exists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "analyze [layers...]" {
			found = true
			break
		}
	}
	assert.True(t, found, "analyze command should be registered")
}

func TestAnalyzeCmd_HasFlags(t *testing.T) {
	flags := analyzeCmd.Flags()

	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{"recommend", "recommend", "false"},
		{"json", "json", "false"},
		{"quiet", "quiet", "false"},
		{"no-ai", "no-ai", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := flags.Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestExtractLayerName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		raw      map[string]interface{}
		expected string
	}{
		{
			name:     "from yaml name field",
			path:     "layers/test.yaml",
			raw:      map[string]interface{}{"name": "my-layer"},
			expected: "my-layer",
		},
		{
			name:     "from filename yaml",
			path:     "layers/dev-go.yaml",
			raw:      map[string]interface{}{},
			expected: "dev-go",
		},
		{
			name:     "from filename yml",
			path:     "layers/security.yml",
			raw:      map[string]interface{}{},
			expected: "security",
		},
		{
			name:     "empty name in yaml",
			path:     "layers/base.yaml",
			raw:      map[string]interface{}{"name": ""},
			expected: "base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLayerName(tt.path, tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPackages(t *testing.T) {
	tests := []struct {
		name     string
		raw      map[string]interface{}
		expected []string
	}{
		{
			name:     "empty packages",
			raw:      map[string]interface{}{},
			expected: nil,
		},
		{
			name: "formulae only",
			raw: map[string]interface{}{
				"packages": map[string]interface{}{
					"brew": map[string]interface{}{
						"formulae": []interface{}{"go", "git"},
					},
				},
			},
			expected: []string{"go", "git"},
		},
		{
			name: "casks only",
			raw: map[string]interface{}{
				"packages": map[string]interface{}{
					"brew": map[string]interface{}{
						"casks": []interface{}{"docker", "vscode"},
					},
				},
			},
			expected: []string{"docker (cask)", "vscode (cask)"},
		},
		{
			name: "both formulae and casks",
			raw: map[string]interface{}{
				"packages": map[string]interface{}{
					"brew": map[string]interface{}{
						"formulae": []interface{}{"go"},
						"casks":    []interface{}{"docker"},
					},
				},
			},
			expected: []string{"go", "docker (cask)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackages(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWellNamedLayer(t *testing.T) {
	// Naming convention tests are now primarily in analyzer_test.go
	// This tests the integration with the domain service
	tests := []struct {
		name     string
		expected bool
	}{
		{"base", true},
		{"dev-go", true},
		{"random-name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := layerAnalyzer.IsWellNamedLayer(tt.name)
			assert.Equal(t, tt.expected, result, "layer '%s' naming check", tt.name)
		})
	}
}

func TestAnalyzeBasic(t *testing.T) {
	// Basic analysis tests are now primarily in analyzer_test.go
	// This tests the integration with the domain service
	tests := []struct {
		name           string
		layer          advisor.LayerInfo
		expectedStatus string
		hasRecs        bool
	}{
		{
			name: "empty layer",
			layer: advisor.LayerInfo{
				Name:     "empty",
				Path:     "layers/empty.yaml",
				Packages: []string{},
			},
			expectedStatus: "warning",
			hasRecs:        true,
		},
		{
			name: "normal layer",
			layer: advisor.LayerInfo{
				Name:     "base",
				Path:     "layers/base.yaml",
				Packages: []string{"git", "curl", "wget"},
			},
			expectedStatus: "good",
			hasRecs:        false,
		},
		{
			name: "large layer",
			layer: advisor.LayerInfo{
				Name:     "misc",
				Path:     "layers/misc.yaml",
				Packages: make([]string, layerAnalyzer.LargeLayerThreshold+10), // Exceed threshold
			},
			expectedStatus: "warning",
			hasRecs:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := layerAnalyzer.AnalyzeBasic(tt.layer)

			assert.Equal(t, tt.layer.Name, result.LayerName)
			assert.Equal(t, tt.expectedStatus, result.Status)
			if tt.hasRecs {
				assert.NotEmpty(t, result.Recommendations)
			}
		})
	}
}

func TestFindCrossLayerIssues(t *testing.T) {
	// Cross-layer issue tests are now primarily in analyzer_test.go
	// This tests the integration with the domain service
	tests := []struct {
		name          string
		layers        []advisor.LayerInfo
		expectIssues  bool
		issueContains string
	}{
		{
			name: "no duplicates",
			layers: []advisor.LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"go", "gopls"}},
			},
			expectIssues: false,
		},
		{
			name: "duplicate package",
			layers: []advisor.LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"go", "git"}},
			},
			expectIssues:  true,
			issueContains: "git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := layerAnalyzer.FindCrossLayerIssues(tt.layers)

			if tt.expectIssues {
				assert.NotEmpty(t, issues)
				if tt.issueContains != "" {
					found := false
					for _, issue := range issues {
						if strings.Contains(issue, tt.issueContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "expected issue containing '%s'", tt.issueContains)
				}
			} else {
				assert.Empty(t, issues)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	// Status icon tests are now primarily in analyzer_test.go
	// This tests the integration with the domain service
	tests := []struct {
		status   string
		expected string
	}{
		{"good", "✓"},
		{"warning", "⚠"},
		{"needs_attention", "⛔"},
		{"unknown", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := advisor.GetStatusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPriorityPrefix(t *testing.T) {
	// Priority prefix tests are now primarily in analyzer_test.go
	// This tests the integration with the domain service
	tests := []struct {
		priority string
	}{
		{"high"},
		{"medium"},
		{"low"},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			result := advisor.GetPriorityPrefix(tt.priority)
			assert.NotEmpty(t, result)
		})
	}
}

func TestValidateLayerPath(t *testing.T) {
	// Create a temporary layer file for testing
	tmpDir := t.TempDir()
	validFile := tmpDir + "/test.yaml"
	if err := os.WriteFile(validFile, []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Basic validation tests - comprehensive tests are in config/layer_service_test.go
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid yaml file",
			path:    validFile,
			wantErr: false,
		},
		{
			name:    "invalid extension",
			path:    tmpDir + "/test.txt",
			wantErr: true,
		},
		{
			name:    "file not found",
			path:    tmpDir + "/nonexistent.yaml",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLayerPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadLayerInfos_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test layer files
	baseLayer := `
name: base
packages:
  brew:
    formulae:
      - git
      - curl
git:
  user:
    name: test
`
	devLayer := `
name: dev-go
packages:
  brew:
    formulae:
      - go
      - gopls
    casks:
      - goland
`
	if err := os.WriteFile(tmpDir+"/base.yaml", []byte(baseLayer), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/dev-go.yaml", []byte(devLayer), 0644); err != nil {
		t.Fatal(err)
	}

	paths := []string{tmpDir + "/base.yaml", tmpDir + "/dev-go.yaml"}
	layers, err := loadLayerInfos(paths)

	require.NoError(t, err)
	assert.Len(t, layers, 2)

	// Check base layer
	assert.Equal(t, "base", layers[0].Name)
	assert.Equal(t, []string{"git", "curl"}, layers[0].Packages)
	assert.True(t, layers[0].HasGitConfig)

	// Check dev layer
	assert.Equal(t, "dev-go", layers[1].Name)
	assert.Equal(t, []string{"go", "gopls", "goland (cask)"}, layers[1].Packages)
}

func TestLoadLayerInfos_InvalidPath(t *testing.T) {
	paths := []string{"/nonexistent/layer.yaml"}
	_, err := loadLayerInfos(paths)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "layer file not found")
}

func TestLoadLayerInfos_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := tmpDir + "/invalid.yaml"
	if err := os.WriteFile(invalidFile, []byte("{{invalid yaml}"), 0644); err != nil {
		t.Fatal(err)
	}

	paths := []string{invalidFile}
	_, err := loadLayerInfos(paths)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestFindLayerFiles(t *testing.T) {
	// Save current working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	// Create temporary directory structure
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0755))

	// Change to temp directory
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	t.Run("empty layers directory", func(t *testing.T) {
		paths, err := findLayerFiles()
		require.NoError(t, err)
		assert.Empty(t, paths)
	})

	t.Run("finds yaml files", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(layersDir, "dev.yaml"), []byte("name: dev\n"), 0644))

		paths, err := findLayerFiles()
		require.NoError(t, err)
		assert.Len(t, paths, 2)
		assert.Contains(t, paths, "layers/base.yaml")
		assert.Contains(t, paths, "layers/dev.yaml")
	})

	t.Run("finds yml files", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(layersDir, "extra.yml"), []byte("name: extra\n"), 0644))

		paths, err := findLayerFiles()
		require.NoError(t, err)
		assert.Len(t, paths, 3) // 2 yaml + 1 yml
		assert.Contains(t, paths, "layers/extra.yml")
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(layersDir, "readme.txt"), []byte("readme\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(layersDir, "config.json"), []byte("{}"), 0644))

		paths, err := findLayerFiles()
		require.NoError(t, err)
		// Should still only have the yaml/yml files
		for _, p := range paths {
			ext := filepath.Ext(p)
			assert.True(t, ext == ".yaml" || ext == ".yml", "unexpected extension: %s", ext)
		}
	})
}

func TestFindLayerFiles_NoLayersDir(t *testing.T) {
	// Save current working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	// Create temporary directory WITHOUT layers subdirectory
	tmpDir := t.TempDir()

	// Change to temp directory
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	paths, err := findLayerFiles()
	require.NoError(t, err)
	assert.Empty(t, paths)
}
