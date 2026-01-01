package advisor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildLayerAnalysisPrompt(t *testing.T) {
	layer := LayerInfo{
		Name:     "dev-go",
		Path:     "layers/dev-go.yaml",
		Packages: []string{"go", "gopls", "golangci-lint"},
	}

	allLayers := []LayerInfo{
		layer,
		{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl"}},
	}

	prompt := BuildLayerAnalysisPrompt(layer, allLayers)

	assert.Contains(t, prompt.UserPrompt(), "dev-go")
	assert.Contains(t, prompt.UserPrompt(), "layers/dev-go.yaml")
	assert.Contains(t, prompt.UserPrompt(), "go")
	assert.Contains(t, prompt.UserPrompt(), "gopls")
	assert.Contains(t, prompt.UserPrompt(), "Other layers in this configuration")
	assert.Contains(t, prompt.UserPrompt(), "base")
	assert.NotEmpty(t, prompt.SystemPrompt())
	assert.Equal(t, 1024, prompt.MaxTokens())
}

func TestBuildLayerAnalysisPrompt_SingleLayer(t *testing.T) {
	layer := LayerInfo{
		Name:     "misc",
		Path:     "layers/misc.yaml",
		Packages: []string{"wget", "jq"},
	}

	prompt := BuildLayerAnalysisPrompt(layer, []LayerInfo{layer})

	assert.Contains(t, prompt.UserPrompt(), "misc")
	// Should not contain "Other layers" section when there's only one layer
	assert.NotContains(t, prompt.UserPrompt(), "Other layers in this configuration")
}

func TestBuildMultiLayerAnalysisPrompt(t *testing.T) {
	layers := []LayerInfo{
		{Name: "dev-go", Path: "layers/dev-go.yaml", Packages: []string{"go", "gopls"}},
		{Name: "security", Path: "layers/security.yaml", Packages: []string{"grype", "trivy"}},
	}

	prompt := BuildMultiLayerAnalysisPrompt(layers)

	assert.Contains(t, prompt.UserPrompt(), "dev-go")
	assert.Contains(t, prompt.UserPrompt(), "security")
	assert.Contains(t, prompt.UserPrompt(), "Total packages across all layers: 4")
	assert.Contains(t, prompt.UserPrompt(), "Duplicate packages across layers")
	assert.Equal(t, 2048, prompt.MaxTokens())
}

func TestParseLayerAnalysisResult_ValidJSON(t *testing.T) {
	response := `Here's my analysis:

{
  "layer_name": "dev-go",
  "summary": "Well-organized Go development layer",
  "status": "good",
  "recommendations": [
    {
      "type": "missing",
      "priority": "medium",
      "message": "Consider adding mockgen for test mocks",
      "packages": ["mockgen"]
    }
  ],
  "package_count": 5,
  "well_organized": true
}

Hope this helps!`

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "dev-go", result.LayerName)
	assert.Equal(t, "Well-organized Go development layer", result.Summary)
	assert.Equal(t, "good", result.Status)
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "missing", result.Recommendations[0].Type)
	assert.Equal(t, "medium", result.Recommendations[0].Priority)
	assert.True(t, result.WellOrganized)
}

func TestParseLayerAnalysisResult_NoJSON(t *testing.T) {
	response := "This is just text without any JSON"

	_, err := ParseLayerAnalysisResult(response)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestParseLayerAnalysisResult_InvalidJSON(t *testing.T) {
	response := `{ invalid json }`

	_, err := ParseLayerAnalysisResult(response)

	assert.Error(t, err)
	// Invalid JSON is now detected by json.Valid() before parsing
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestLayerAnalysisSystemPrompt(t *testing.T) {
	assert.NotEmpty(t, LayerAnalysisSystemPrompt)
	assert.Contains(t, LayerAnalysisSystemPrompt, "misplacement")
	assert.Contains(t, LayerAnalysisSystemPrompt, "duplicate")
	assert.Contains(t, LayerAnalysisSystemPrompt, "deprecated")
	assert.Contains(t, LayerAnalysisSystemPrompt, "JSON")
}

func TestLayerInfo_Fields(t *testing.T) {
	info := LayerInfo{
		Name:            "test-layer",
		Path:            "layers/test.yaml",
		Packages:        []string{"pkg1", "pkg2"},
		HasGitConfig:    true,
		HasSSHConfig:    false,
		HasShellConfig:  true,
		HasEditorConfig: false,
	}

	assert.Equal(t, "test-layer", info.Name)
	assert.Equal(t, "layers/test.yaml", info.Path)
	assert.Len(t, info.Packages, 2)
	assert.True(t, info.HasGitConfig)
	assert.False(t, info.HasSSHConfig)
	assert.True(t, info.HasShellConfig)
	assert.False(t, info.HasEditorConfig)
}

func TestAnalysisRecommendation_Fields(t *testing.T) {
	rec := AnalysisRecommendation{
		Type:           "misplacement",
		Priority:       "high",
		Message:        "Move ffmpeg to media layer",
		Packages:       []string{"ffmpeg", "ffprobe"},
		SuggestedLayer: "media",
	}

	assert.Equal(t, "misplacement", rec.Type)
	assert.Equal(t, "high", rec.Priority)
	assert.Equal(t, "Move ffmpeg to media layer", rec.Message)
	assert.Len(t, rec.Packages, 2)
	assert.Equal(t, "media", rec.SuggestedLayer)
}

func TestLayerAnalysisResult_Fields(t *testing.T) {
	result := LayerAnalysisResult{
		LayerName:     "misc",
		Summary:       "Large uncategorized layer",
		Status:        "warning",
		PackageCount:  50,
		WellOrganized: false,
		Recommendations: []AnalysisRecommendation{
			{Type: "best_practice", Priority: "medium", Message: "Consider splitting"},
		},
	}

	assert.Equal(t, "misc", result.LayerName)
	assert.Equal(t, "Large uncategorized layer", result.Summary)
	assert.Equal(t, "warning", result.Status)
	assert.Equal(t, 50, result.PackageCount)
	assert.False(t, result.WellOrganized)
	assert.Len(t, result.Recommendations, 1)
}

func TestAnalysisReport_Fields(t *testing.T) {
	report := AnalysisReport{
		Layers: []LayerAnalysisResult{
			{LayerName: "base", PackageCount: 10},
			{LayerName: "dev", PackageCount: 20},
		},
		TotalPackages:        30,
		TotalRecommendations: 5,
		CrossLayerIssues:     []string{"duplicate: git in base and dev"},
	}

	assert.Len(t, report.Layers, 2)
	assert.Equal(t, 30, report.TotalPackages)
	assert.Equal(t, 5, report.TotalRecommendations)
	assert.Len(t, report.CrossLayerIssues, 1)
}

func TestBuildLayerAnalysisPrompt_EmptyPackages(t *testing.T) {
	layer := LayerInfo{
		Name:     "empty",
		Path:     "layers/empty.yaml",
		Packages: []string{},
	}

	prompt := BuildLayerAnalysisPrompt(layer, []LayerInfo{layer})

	assert.Contains(t, prompt.UserPrompt(), "empty")
	assert.Contains(t, prompt.UserPrompt(), "Packages (0)")
}

func TestBuildLayerAnalysisPrompt_ConfigContext(t *testing.T) {
	layer := LayerInfo{
		Name:            "shell-config",
		Path:            "layers/shell.yaml",
		Packages:        []string{"zsh", "starship"},
		HasShellConfig:  true,
		HasEditorConfig: true,
	}

	prompt := BuildLayerAnalysisPrompt(layer, []LayerInfo{layer})

	// The prompt should be built without errors
	assert.NotEmpty(t, prompt.UserPrompt())
	assert.Contains(t, prompt.UserPrompt(), "zsh")
}

func TestParseLayerAnalysisResult_MultipleRecommendations(t *testing.T) {
	response := `{
  "layer_name": "misc",
  "summary": "Needs reorganization",
  "status": "needs_attention",
  "recommendations": [
    {"type": "misplacement", "priority": "high", "message": "Move media tools", "packages": ["ffmpeg"], "suggested_layer": "media"},
    {"type": "duplicate", "priority": "medium", "message": "Both grype and trivy installed", "packages": ["grype", "trivy"]},
    {"type": "deprecated", "priority": "low", "message": "Consider replacing with newer tool", "packages": ["old-tool"]}
  ],
  "package_count": 66,
  "well_organized": false
}`

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "misc", result.LayerName)
	assert.Equal(t, "needs_attention", result.Status)
	assert.Len(t, result.Recommendations, 3)
	assert.Equal(t, "misplacement", result.Recommendations[0].Type)
	assert.Equal(t, "media", result.Recommendations[0].SuggestedLayer)
	assert.Equal(t, "duplicate", result.Recommendations[1].Type)
	assert.Equal(t, "deprecated", result.Recommendations[2].Type)
	assert.False(t, result.WellOrganized)
}

func TestLayerAnalysisPrompt_ContainsGuidance(t *testing.T) {
	layer := LayerInfo{
		Name:     "test",
		Path:     "test.yaml",
		Packages: []string{"test-pkg"},
	}

	prompt := BuildLayerAnalysisPrompt(layer, []LayerInfo{layer})

	// Should contain guidance about what to check
	userPrompt := prompt.UserPrompt()
	assert.True(t, strings.Contains(userPrompt, "naming conventions") ||
		strings.Contains(userPrompt, "Package grouping") ||
		strings.Contains(userPrompt, "duplicates"))
}

func TestParseLayerAnalysisResult_FromCodeBlock(t *testing.T) {
	response := "Here's the analysis:\n```json\n" + `{
  "layer_name": "base",
  "summary": "Good configuration",
  "status": "good",
  "recommendations": [],
  "package_count": 10,
  "well_organized": true
}` + "\n```\nThat's my analysis."

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "base", result.LayerName)
	assert.Equal(t, "good", result.Status)
	assert.True(t, result.WellOrganized)
}

func TestParseLayerAnalysisResult_FromGenericCodeBlock(t *testing.T) {
	response := "Analysis result:\n```\n" + `{"layer_name": "dev", "summary": "OK", "status": "good", "recommendations": [], "package_count": 5, "well_organized": true}` + "\n```"

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "dev", result.LayerName)
}

func TestParseLayerAnalysisResult_WithPrecedingText(t *testing.T) {
	response := `Here's what I found after analyzing the layer:

The layer contains useful development tools. Here's the structured analysis:

{
  "layer_name": "dev-tools",
  "summary": "Well organized development layer",
  "status": "good",
  "recommendations": [],
  "package_count": 15,
  "well_organized": true
}

I hope this helps!`

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "dev-tools", result.LayerName)
	assert.Equal(t, "good", result.Status)
}

func TestParseLayerAnalysisResult_NestedBraces(t *testing.T) {
	// JSON with nested objects should parse correctly
	response := `{
  "layer_name": "test",
  "summary": "Test with nested",
  "status": "good",
  "recommendations": [
    {
      "type": "info",
      "priority": "low",
      "message": "Example with {braces} in text"
    }
  ],
  "package_count": 1,
  "well_organized": true
}`

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "test", result.LayerName)
	assert.Len(t, result.Recommendations, 1)
}

func TestParseLayerAnalysisResult_EscapedQuotes(t *testing.T) {
	response := `{
  "layer_name": "test",
  "summary": "Has \"quoted\" text",
  "status": "good",
  "recommendations": [],
  "package_count": 1,
  "well_organized": true
}`

	result, err := ParseLayerAnalysisResult(response)

	require.NoError(t, err)
	assert.Equal(t, "test", result.LayerName)
	assert.Contains(t, result.Summary, "quoted")
}

func TestLayerAnalysisResult_StatusField(t *testing.T) {
	// Error handling is done at the application layer, not in the domain model.
	// The Status field indicates analysis outcome, with fallback to basic analysis.
	result := LayerAnalysisResult{
		LayerName: "test",
		Status:    "warning",
		Summary:   "AI unavailable - 10 packages",
	}

	assert.Equal(t, "warning", result.Status)
	assert.Contains(t, result.Summary, "AI unavailable")
}

func TestParseLayerAnalysisResult_ResponseTooLarge(t *testing.T) {
	// Create a response larger than MaxJSONResponseSize
	largeResponse := strings.Repeat("x", MaxJSONResponseSize+1)

	_, err := ParseLayerAnalysisResult(largeResponse)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "response too large")
}

func TestMaxJSONResponseSize(t *testing.T) {
	// Verify the constant is set to 1MB
	assert.Equal(t, 1<<20, MaxJSONResponseSize)
}
