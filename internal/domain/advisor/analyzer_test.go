package advisor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLayerAnalyzer(t *testing.T) {
	analyzer := NewLayerAnalyzer()

	assert.NotNil(t, analyzer)
	assert.Equal(t, 50, analyzer.LargeLayerThreshold())
}

func TestNewLayerAnalyzer_WithOptions(t *testing.T) {
	analyzer := NewLayerAnalyzer(
		WithLargeLayerThreshold(100),
		WithWellNamedPrefixes([]string{"custom-"}),
	)

	assert.NotNil(t, analyzer)
	assert.Equal(t, 100, analyzer.LargeLayerThreshold())
	assert.True(t, analyzer.IsWellNamedLayer("custom-layer"))
	assert.False(t, analyzer.IsWellNamedLayer("base"))
}

func TestLayerAnalyzer_AnalyzeBasic(t *testing.T) {
	analyzer := NewLayerAnalyzer()

	tests := []struct {
		name           string
		layer          LayerInfo
		expectedStatus AnalysisStatus
		hasRecs        bool
	}{
		{
			name: "empty layer",
			layer: LayerInfo{
				Name:     "empty",
				Path:     "layers/empty.yaml",
				Packages: []string{},
			},
			expectedStatus: StatusWarning,
			hasRecs:        true,
		},
		{
			name: "normal layer",
			layer: LayerInfo{
				Name:     "base",
				Path:     "layers/base.yaml",
				Packages: []string{"git", "curl", "wget"},
			},
			expectedStatus: StatusGood,
			hasRecs:        false,
		},
		{
			name: "large layer",
			layer: LayerInfo{
				Name:     "misc",
				Path:     "layers/misc.yaml",
				Packages: make([]string, 60), // Exceed threshold
			},
			expectedStatus: StatusWarning,
			hasRecs:        true,
		},
		{
			name: "poorly named layer",
			layer: LayerInfo{
				Name:     "my-stuff",
				Path:     "layers/my-stuff.yaml",
				Packages: []string{"git"},
			},
			expectedStatus: StatusGood,
			hasRecs:        true, // Should have naming convention recommendation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.AnalyzeBasic(tt.layer)

			assert.Equal(t, tt.layer.Name, result.LayerName)
			assert.Equal(t, tt.expectedStatus, result.Status)
			if tt.hasRecs {
				assert.NotEmpty(t, result.Recommendations)
			}
		})
	}
}

func TestLayerAnalyzer_IsWellNamedLayer(t *testing.T) {
	analyzer := NewLayerAnalyzer()

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
			result := analyzer.IsWellNamedLayer(tt.name)
			assert.Equal(t, tt.expected, result, "layer '%s' naming check", tt.name)
		})
	}
}

func TestLayerAnalyzer_FindCrossLayerIssues(t *testing.T) {
	analyzer := NewLayerAnalyzer()

	tests := []struct {
		name          string
		layers        []LayerInfo
		expectIssues  bool
		issueContains string
	}{
		{
			name: "no duplicates",
			layers: []LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"go", "gopls"}},
			},
			expectIssues: false,
		},
		{
			name: "duplicate package",
			layers: []LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"git", "curl"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"go", "git"}},
			},
			expectIssues:  true,
			issueContains: "git",
		},
		{
			name: "cask duplicate normalized",
			layers: []LayerInfo{
				{Name: "base", Path: "layers/base.yaml", Packages: []string{"docker"}},
				{Name: "dev", Path: "layers/dev.yaml", Packages: []string{"docker (cask)"}},
			},
			expectIssues:  true,
			issueContains: "docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := analyzer.FindCrossLayerIssues(tt.layers)

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

func TestLayerAnalyzer_CustomThreshold(t *testing.T) {
	analyzer := NewLayerAnalyzer(
		WithLargeLayerThreshold(10),
		WithWellNamedPrefixes([]string{"base"}),
	)

	layer := LayerInfo{
		Name:     "base",
		Path:     "layers/base.yaml",
		Packages: make([]string, 15),
	}

	result := analyzer.AnalyzeBasic(layer)
	assert.Equal(t, StatusWarning, result.Status)
	assert.False(t, result.WellOrganized)
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
			result := GetStatusIcon(tt.status)
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
			result := GetPriorityPrefix(tt.priority)
			assert.NotEmpty(t, result)
		})
	}
}

func TestAnalysisStatus_Values(t *testing.T) {
	// Ensure the typed constants have the expected string values
	assert.Equal(t, "good", string(StatusGood))
	assert.Equal(t, "warning", string(StatusWarning))
	assert.Equal(t, "needs_attention", string(StatusNeedsAttention))
}

func TestRecommendationPriority_Values(t *testing.T) {
	assert.Equal(t, "high", string(PriorityHigh))
	assert.Equal(t, "medium", string(PriorityMedium))
	assert.Equal(t, "low", string(PriorityLow))
}

func TestRecommendationType_Values(t *testing.T) {
	assert.Equal(t, "best_practice", string(TypeBestPractice))
	assert.Equal(t, "misplaced", string(TypeMisplaced))
	assert.Equal(t, "missing", string(TypeMissing))
	assert.Equal(t, "deprecated", string(TypeDeprecated))
}
