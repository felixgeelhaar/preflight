package advisor

import (
	"testing"
)

// BenchmarkBuildLayerAnalysisPrompt benchmarks prompt construction.
func BenchmarkBuildLayerAnalysisPrompt(b *testing.B) {
	layer := LayerInfo{
		Name:     "dev-go",
		Path:     "layers/dev-go.yaml",
		Packages: []string{"go", "gopls", "golangci-lint", "delve", "gofumpt"},
	}
	allLayers := []LayerInfo{
		layer,
		{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl", "wget"}},
		{Name: "security", Path: "layers/security.yaml", Packages: []string{"grype", "trivy"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildLayerAnalysisPrompt(layer, allLayers)
	}
}

// BenchmarkBuildMultiLayerAnalysisPrompt benchmarks multi-layer prompt construction.
func BenchmarkBuildMultiLayerAnalysisPrompt(b *testing.B) {
	layers := []LayerInfo{
		{Name: "base", Path: "layers/base.yaml", Packages: make([]string, 20)},
		{Name: "dev-go", Path: "layers/dev-go.yaml", Packages: make([]string, 15)},
		{Name: "security", Path: "layers/security.yaml", Packages: make([]string, 10)},
		{Name: "media", Path: "layers/media.yaml", Packages: make([]string, 25)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildMultiLayerAnalysisPrompt(layers)
	}
}

// BenchmarkParseLayerAnalysisResult benchmarks JSON response parsing.
func BenchmarkParseLayerAnalysisResult(b *testing.B) {
	response := `{
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
}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseLayerAnalysisResult(response)
	}
}

// BenchmarkLayerAnalyzer_AnalyzeBasic benchmarks basic layer analysis.
func BenchmarkLayerAnalyzer_AnalyzeBasic(b *testing.B) {
	analyzer := NewLayerAnalyzer()
	layer := LayerInfo{
		Name:     "dev-go",
		Path:     "layers/dev-go.yaml",
		Packages: make([]string, 30),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.AnalyzeBasic(layer)
	}
}

// BenchmarkLayerAnalyzer_FindCrossLayerIssues benchmarks cross-layer issue detection.
func BenchmarkLayerAnalyzer_FindCrossLayerIssues(b *testing.B) {
	analyzer := NewLayerAnalyzer()

	// Create layers with some duplicates
	layers := []LayerInfo{
		{Name: "base", Packages: []string{"git", "curl", "wget", "jq"}},
		{Name: "dev-go", Packages: []string{"go", "gopls", "git", "delve"}},        // git duplicate
		{Name: "dev-python", Packages: []string{"python", "pip", "black", "curl"}}, // curl duplicate
		{Name: "security", Packages: []string{"grype", "trivy", "git"}},            // git duplicate
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.FindCrossLayerIssues(layers)
	}
}

// BenchmarkLayerAnalyzer_IsWellNamedLayer benchmarks layer name validation.
func BenchmarkLayerAnalyzer_IsWellNamedLayer(b *testing.B) {
	analyzer := NewLayerAnalyzer()
	names := []string{"base", "dev-go", "identity.work", "random-name", "device.laptop"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range names {
			_ = analyzer.IsWellNamedLayer(name)
		}
	}
}

// BenchmarkPrompt_Creation benchmarks prompt creation with various configurations.
func BenchmarkPrompt_Creation(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewPrompt("system", "user")
		}
	})

	b.Run("with_options", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewPrompt("system", "user").
				WithMaxTokens(2048).
				WithTemperature(0.7)
		}
	})
}

// BenchmarkExtractJSON benchmarks JSON extraction from AI responses.
func BenchmarkExtractJSON(b *testing.B) {
	responses := map[string]string{
		"clean_json": `{"layer_name": "test", "status": "good"}`,
		"with_text":  `Here's the analysis: {"layer_name": "test", "status": "good"} Hope this helps!`,
		"code_block": "```json\n" + `{"layer_name": "test", "status": "good"}` + "\n```",
	}

	for name, resp := range responses {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = extractJSON(resp)
			}
		})
	}
}
