package main

import (
	"os"
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
	tests := []struct {
		name     string
		expected bool
	}{
		{"base", true},
		{"dev-go", true},
		{"dev-python", true},
		{"role.developer", true},
		{"identity.work", true},
		{"device.laptop", true},
		{"misc", true},
		{"security", true},
		{"media", true},
		{"random-name", false},
		{"my-layer", false},
		{"tools", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWellNamedLayer(tt.name)
			assert.Equal(t, tt.expected, result, "layer '%s' naming check", tt.name)
		})
	}
}

func TestPerformBasicAnalysis(t *testing.T) {
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
				Packages: make([]string, 60),
			},
			expectedStatus: "warning",
			hasRecs:        true,
		},
		{
			name: "poorly named layer",
			layer: advisor.LayerInfo{
				Name:     "my-stuff",
				Path:     "layers/my-stuff.yaml",
				Packages: []string{"git"},
			},
			expectedStatus: "good",
			hasRecs:        true, // Should have naming convention recommendation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := performBasicAnalysis(tt.layer)

			assert.Equal(t, tt.layer.Name, result.LayerName)
			assert.Equal(t, tt.expectedStatus, result.Status)
			if tt.hasRecs {
				assert.NotEmpty(t, result.Recommendations)
			}
		})
	}
}

func TestFindCrossLayerIssues(t *testing.T) {
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
		{
			name: "cask duplicate normalized",
			layers: []advisor.LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"docker"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"docker (cask)"}},
			},
			expectIssues:  true,
			issueContains: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := findCrossLayerIssues(tt.layers)

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
	tests := []struct {
		status   string
		expected string
	}{
		{"good", "✓"},
		{"warning", "⚠"},
		{"needs_attention", "⛔"},
		{"unknown", "○"},
		{"", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPriorityPrefix(t *testing.T) {
	tests := []struct {
		priority string
	}{
		{"high"},
		{"medium"},
		{"low"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			result := getPriorityPrefix(tt.priority)
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

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
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
			errMsg:  "invalid layer file extension",
		},
		{
			name:    "file not found",
			path:    tmpDir + "/nonexistent.yaml",
			wantErr: true,
			errMsg:  "layer file not found",
		},
		{
			name:    "directory instead of file",
			path:    tmpDir,
			wantErr: true,
			errMsg:  "invalid layer file extension",
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
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
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
