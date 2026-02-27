package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/tui"
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
		expectedStatus advisor.AnalysisStatus
		hasRecs        bool
	}{
		{
			name: "empty layer",
			layer: advisor.LayerInfo{
				Name:     "empty",
				Path:     "layers/empty.yaml",
				Packages: []string{},
			},
			expectedStatus: advisor.StatusWarning,
			hasRecs:        true,
		},
		{
			name: "normal layer",
			layer: advisor.LayerInfo{
				Name:     "base",
				Path:     "layers/base.yaml",
				Packages: []string{"git", "curl", "wget"},
			},
			expectedStatus: advisor.StatusGood,
			hasRecs:        false,
		},
		{
			name: "large layer",
			layer: advisor.LayerInfo{
				Name:     "misc",
				Path:     "layers/misc.yaml",
				Packages: make([]string, 60), // Exceed default threshold of 50
			},
			expectedStatus: advisor.StatusWarning,
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

func TestFormatStatusIcon(t *testing.T) {
	// Tests the TUI formatting function used for analyze output
	tests := []struct {
		name     string
		status   advisor.AnalysisStatus
		expected string
	}{
		{"good", advisor.StatusGood, "✓"},
		{"warning", advisor.StatusWarning, "⚠"},
		{"needs_attention", advisor.StatusNeedsAttention, "⛔"},
		{"unknown", advisor.AnalysisStatus("unknown"), "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tui.FormatStatusIcon(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatPriorityPrefix(t *testing.T) {
	// Tests the TUI formatting function used for analyze output
	tests := []struct {
		name     string
		priority advisor.RecommendationPriority
	}{
		{"high", advisor.PriorityHigh},
		{"medium", advisor.PriorityMedium},
		{"low", advisor.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tui.FormatPriorityPrefix(tt.priority)
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

// --- Batch 3: Output and parsing tests ---

func TestFilterFindingsByType(t *testing.T) {
	findings := []security.ToolFinding{
		{Type: security.FindingDeprecated, Message: "dep1"},
		{Type: security.FindingRedundancy, Message: "red1"},
		{Type: security.FindingDeprecated, Message: "dep2"},
		{Type: security.FindingConsolidation, Message: "con1"},
		{Type: security.FindingRedundancy, Message: "red2"},
		{Type: security.FindingRedundancy, Message: "red3"},
	}

	tests := []struct {
		name        string
		findingType security.FindingType
		wantCount   int
	}{
		{"deprecated findings", security.FindingDeprecated, 2},
		{"redundancy findings", security.FindingRedundancy, 3},
		{"consolidation findings", security.FindingConsolidation, 1},
		{"unknown type returns empty", security.FindingUnknown, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFindingsByType(findings, tt.findingType)
			assert.Len(t, result, tt.wantCount)
			for _, f := range result {
				assert.Equal(t, tt.findingType, f.Type)
			}
		})
	}
}

func TestBuildToolAnalysisPrompt(t *testing.T) {
	result := &security.ToolAnalysisResult{
		Findings: []security.ToolFinding{
			{
				Type:    security.FindingDeprecated,
				Message: "golint is deprecated",
				Tools:   []string{"golint"},
			},
			{
				Type:    security.FindingRedundancy,
				Message: "grype redundant with trivy",
				Tools:   []string{"grype"},
			},
		},
		ToolsAnalyzed: 5,
		IssuesFound:   2,
	}
	toolNames := []string{"go", "golint", "trivy", "grype", "fzf"}

	prompt := buildToolAnalysisPrompt(result, toolNames)

	// Verify prompt contains all tool names
	for _, tool := range toolNames {
		assert.Contains(t, prompt, tool, "prompt should contain tool name %s", tool)
	}

	// Verify prompt contains existing finding messages
	assert.Contains(t, prompt, "golint is deprecated")
	assert.Contains(t, prompt, "grype redundant with trivy")

	// Verify prompt includes JSON format instructions
	assert.Contains(t, prompt, "insights")
	assert.Contains(t, prompt, "JSON")
}

func TestParseAIToolInsights_ValidJSON(t *testing.T) {
	content := `Some text before {"insights": [{"type": "recommendation", "severity": "warning", "tools": ["tool1"], "message": "test msg", "suggestion": "do this"}]} after`

	findings := parseAIToolInsights(content)

	require.NotNil(t, findings)
	require.Len(t, findings, 1)
	assert.Equal(t, security.FindingType("recommendation"), findings[0].Type)
	assert.Equal(t, security.SeverityWarning, findings[0].Severity)
	assert.Equal(t, []string{"tool1"}, findings[0].Tools)
	assert.Equal(t, "test msg", findings[0].Message)
	assert.Equal(t, "do this", findings[0].Suggestion)
}

func TestParseAIToolInsights_MultipleSeverities(t *testing.T) {
	content := `{"insights": [
		{"type": "deprecated", "severity": "error", "tools": ["a"], "message": "err", "suggestion": "fix"},
		{"type": "info", "severity": "info", "tools": ["b"], "message": "ok", "suggestion": "none"},
		{"type": "warn", "severity": "warning", "tools": ["c"], "message": "warn", "suggestion": "check"}
	]}`

	findings := parseAIToolInsights(content)

	require.NotNil(t, findings)
	require.Len(t, findings, 3)
	assert.Equal(t, security.SeverityError, findings[0].Severity)
	assert.Equal(t, security.SeverityInfo, findings[1].Severity)
	assert.Equal(t, security.SeverityWarning, findings[2].Severity)
}

func TestParseAIToolInsights_InvalidJSON(t *testing.T) {
	content := `{ this is not valid json }`

	findings := parseAIToolInsights(content)

	assert.Nil(t, findings)
}

func TestParseAIToolInsights_NoJSON(t *testing.T) {
	content := "This is just plain text with no JSON markers at all"

	findings := parseAIToolInsights(content)

	assert.Nil(t, findings)
}

func TestOutputAnalyzeJSON_WithReport(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "base",
				Summary:      "3 packages",
				Status:       advisor.StatusGood,
				PackageCount: 3,
				Recommendations: []advisor.AnalysisRecommendation{
					{
						Priority: advisor.PriorityMedium,
						Message:  "Consider splitting",
						Packages: []string{"git", "curl"},
					},
				},
			},
			{
				LayerName:       "dev-go",
				Summary:         "5 packages",
				Status:          advisor.StatusWarning,
				PackageCount:    5,
				Recommendations: []advisor.AnalysisRecommendation{},
			},
		},
		TotalPackages:        8,
		TotalRecommendations: 1,
		CrossLayerIssues:     []string{"git appears in multiple layers"},
	}

	output := captureStdout(t, func() {
		outputAnalyzeJSON(report, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	// Verify layers are present
	layers, ok := parsed["layers"].([]interface{})
	require.True(t, ok, "should have layers array")
	assert.Len(t, layers, 2)

	// Verify totals
	assert.Equal(t, float64(8), parsed["total_packages"])
	assert.Equal(t, float64(1), parsed["total_recommendations"])

	// Verify cross-layer issues
	issues, ok := parsed["cross_layer_issues"].([]interface{})
	require.True(t, ok, "should have cross_layer_issues array")
	assert.Len(t, issues, 1)
	assert.Equal(t, "git appears in multiple layers", issues[0])

	// Verify no error field
	_, hasError := parsed["error"]
	assert.False(t, hasError, "should not have error field")
}

func TestOutputAnalyzeJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputAnalyzeJSON(nil, fmt.Errorf("no layers found"))
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "no layers found", parsed["error"])
}

func TestOutputAnalyzeText_WithLayers(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:       "base",
				Summary:         "3 packages",
				Status:          advisor.StatusGood,
				PackageCount:    3,
				Recommendations: []advisor.AnalysisRecommendation{},
			},
			{
				LayerName:    "dev-go",
				Summary:      "5 packages",
				Status:       advisor.StatusWarning,
				PackageCount: 5,
				Recommendations: []advisor.AnalysisRecommendation{
					{
						Priority: advisor.PriorityHigh,
						Message:  "Remove deprecated tools",
						Packages: []string{"golint"},
					},
				},
			},
		},
		TotalPackages:        8,
		TotalRecommendations: 1,
	}

	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, true)
	})

	// Verify report header
	assert.Contains(t, output, "Layer Analysis Report")

	// Verify layer names appear
	assert.Contains(t, output, "base")
	assert.Contains(t, output, "dev-go")

	// Verify summary line
	assert.Contains(t, output, "2 layers analyzed")
	assert.Contains(t, output, "1 recommendations")

	// Verify total packages
	assert.Contains(t, output, "Total packages: 8")

	// Verify recommendations appear (recommend=true)
	assert.Contains(t, output, "Remove deprecated tools")

	// Verify table appears for multiple layers (quiet=false, >1 layer)
	assert.Contains(t, output, "LAYER")
	assert.Contains(t, output, "PACKAGES")
}

func TestOutputAnalyzeText_NoLayers(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{},
	}

	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})

	assert.Contains(t, output, "No layers to analyze")
}

func TestPrintLayerSummaryTable(t *testing.T) {
	layers := []advisor.LayerAnalysisResult{
		{
			LayerName:    "base",
			PackageCount: 3,
			Status:       advisor.StatusGood,
			Recommendations: []advisor.AnalysisRecommendation{
				{Message: "rec1"},
			},
		},
		{
			LayerName:       "dev-go",
			PackageCount:    10,
			Status:          advisor.StatusWarning,
			Recommendations: []advisor.AnalysisRecommendation{},
		},
		{
			LayerName:       "misc",
			PackageCount:    0,
			Status:          "",
			Recommendations: []advisor.AnalysisRecommendation{},
		},
	}

	output := captureStdout(t, func() {
		printLayerSummaryTable(layers)
	})

	// Verify table headers
	assert.Contains(t, output, "LAYER")
	assert.Contains(t, output, "PACKAGES")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "RECOMMENDATIONS")

	// Verify layer data appears
	assert.Contains(t, output, "base")
	assert.Contains(t, output, "dev-go")
	assert.Contains(t, output, "misc")

	// Verify empty status is replaced with "-"
	// The misc layer has empty status, should display "-"
	lines := strings.Split(output, "\n")
	foundMisc := false
	for _, line := range lines {
		if strings.Contains(line, "misc") {
			foundMisc = true
			assert.Contains(t, line, "-", "empty status should be displayed as '-'")
		}
	}
	assert.True(t, foundMisc, "misc layer should appear in table")
}

func TestOutputToolAnalysisJSON_WithResult(t *testing.T) {
	result := &security.ToolAnalysisResult{
		Findings: []security.ToolFinding{
			{
				Type:       security.FindingDeprecated,
				Severity:   security.SeverityWarning,
				Tools:      []string{"golint"},
				Message:    "golint is deprecated",
				Suggestion: "Use golangci-lint",
			},
		},
		ToolsAnalyzed:  5,
		IssuesFound:    1,
		Consolidations: 0,
	}

	output := captureStdout(t, func() {
		outputToolAnalysisJSON(result, nil)
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, float64(5), parsed["tools_analyzed"])
	assert.Equal(t, float64(1), parsed["issues_found"])
	assert.Equal(t, float64(0), parsed["consolidations"])

	findings, ok := parsed["findings"].([]interface{})
	require.True(t, ok, "should have findings array")
	assert.Len(t, findings, 1)

	finding := findings[0].(map[string]interface{})
	assert.Equal(t, "deprecated", finding["type"])
	assert.Equal(t, "warning", finding["severity"])
	assert.Equal(t, "golint is deprecated", finding["message"])
}

func TestOutputToolAnalysisJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputToolAnalysisJSON(nil, fmt.Errorf("analysis failed: missing knowledge base"))
	})

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "analysis failed: missing knowledge base", parsed["error"])
	assert.Equal(t, float64(0), parsed["tools_analyzed"])
}

func TestOutputToolAnalysisText_WithFindings(t *testing.T) {
	result := &security.ToolAnalysisResult{
		Findings: []security.ToolFinding{
			{
				Type:       security.FindingDeprecated,
				Severity:   security.SeverityWarning,
				Tools:      []string{"golint"},
				Message:    "golint is deprecated",
				Suggestion: "Use golangci-lint instead",
				Docs:       "https://example.com/golangci-lint",
			},
			{
				Type:       security.FindingRedundancy,
				Severity:   security.SeverityWarning,
				Tools:      []string{"grype"},
				Message:    "grype is redundant with trivy",
				Suggestion: "Remove grype, keep trivy",
			},
			{
				Type:       security.FindingConsolidation,
				Severity:   security.SeverityInfo,
				Tools:      []string{"syft", "gitleaks"},
				Message:    "syft and gitleaks can be consolidated",
				Suggestion: "Replace with trivy for simplified toolchain",
				Docs:       "https://example.com/trivy",
			},
		},
		ToolsAnalyzed:  6,
		IssuesFound:    2,
		Consolidations: 1,
	}

	output := captureStdout(t, func() {
		outputToolAnalysisText(result, []string{"golint", "grype", "trivy", "syft", "gitleaks", "fzf"})
	})

	// Verify header
	assert.Contains(t, output, "Tool Configuration Analysis")

	// Verify deprecation section
	assert.Contains(t, output, "Deprecation Warnings")
	assert.Contains(t, output, "golint is deprecated")
	assert.Contains(t, output, "Use golangci-lint instead")
	assert.Contains(t, output, "https://example.com/golangci-lint")

	// Verify redundancy section
	assert.Contains(t, output, "Redundancy Issues")
	assert.Contains(t, output, "grype is redundant with trivy")
	assert.Contains(t, output, "Remove grype, keep trivy")

	// Verify consolidation section
	assert.Contains(t, output, "Consolidation Opportunities")
	assert.Contains(t, output, "syft and gitleaks can be consolidated")

	// Verify summary
	assert.Contains(t, output, "6 tools analyzed")
	assert.Contains(t, output, "2 issues found")
	assert.Contains(t, output, "1 consolidation opportunities")
}

func TestOutputToolAnalysisText_NoFindings(t *testing.T) {
	result := &security.ToolAnalysisResult{
		Findings:       []security.ToolFinding{},
		ToolsAnalyzed:  4,
		IssuesFound:    0,
		Consolidations: 0,
	}

	output := captureStdout(t, func() {
		outputToolAnalysisText(result, []string{"go", "git", "fzf", "ripgrep"})
	})

	assert.Contains(t, output, "Tool Configuration Analysis")
	assert.Contains(t, output, "No issues found")
	assert.Contains(t, output, "4 tools analyzed")
}
