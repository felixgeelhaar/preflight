// Package advisor provides AI-powered analysis and recommendations for Preflight configurations.
package advisor

import (
	"fmt"
	"strings"
)

// AnalysisStatus represents the status of a layer analysis.
type AnalysisStatus string

const (
	// StatusGood indicates the layer is well-organized.
	StatusGood AnalysisStatus = "good"
	// StatusWarning indicates the layer has minor issues.
	StatusWarning AnalysisStatus = "warning"
	// StatusNeedsAttention indicates the layer requires attention.
	StatusNeedsAttention AnalysisStatus = "needs_attention"
)

// RecommendationPriority represents the priority level of a recommendation.
type RecommendationPriority string

const (
	// PriorityHigh indicates an urgent recommendation.
	PriorityHigh RecommendationPriority = "high"
	// PriorityMedium indicates a moderate recommendation.
	PriorityMedium RecommendationPriority = "medium"
	// PriorityLow indicates a minor recommendation.
	PriorityLow RecommendationPriority = "low"
)

// RecommendationType categorizes the type of recommendation.
type RecommendationType string

const (
	// TypeBestPractice indicates a best practice recommendation.
	TypeBestPractice RecommendationType = "best_practice"
	// TypeMisplaced indicates a package in the wrong layer.
	TypeMisplaced RecommendationType = "misplaced"
	// TypeMissing indicates a missing recommended package.
	TypeMissing RecommendationType = "missing"
	// TypeDeprecated indicates a deprecated package.
	TypeDeprecated RecommendationType = "deprecated"
)

// LayerAnalyzer provides heuristic-based analysis of configuration layers.
// This is a domain service that encapsulates business rules about layer quality.
type LayerAnalyzer struct {
	// LargeLayerThreshold is the number of packages above which a layer is considered large.
	// Layers with more than this many packages should be considered for splitting.
	LargeLayerThreshold int

	// WellNamedPrefixes are the recognized naming convention prefixes for layers.
	WellNamedPrefixes []string
}

// NewLayerAnalyzer creates a new LayerAnalyzer with default configuration.
func NewLayerAnalyzer() *LayerAnalyzer {
	return &LayerAnalyzer{
		LargeLayerThreshold: 50,
		WellNamedPrefixes: []string{
			"base", "dev-", "role.", "identity.", "device.", "misc", "security", "media",
		},
	}
}

// AnalyzeBasic performs heuristic-based analysis of a single layer without AI.
// This provides basic recommendations based on layer size and naming conventions.
func (a *LayerAnalyzer) AnalyzeBasic(layer LayerInfo) LayerAnalysisResult {
	result := LayerAnalysisResult{
		LayerName:       layer.Name,
		PackageCount:    len(layer.Packages),
		Recommendations: []AnalysisRecommendation{},
		WellOrganized:   true,
	}

	// Determine status based on package count
	switch {
	case len(layer.Packages) == 0:
		result.Summary = "Empty layer"
		result.Status = string(StatusWarning)
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     string(TypeBestPractice),
			Priority: string(PriorityLow),
			Message:  "Layer has no packages defined",
		})
	case len(layer.Packages) > a.LargeLayerThreshold:
		result.Summary = fmt.Sprintf("Large layer with %d packages", len(layer.Packages))
		result.Status = string(StatusWarning)
		result.WellOrganized = false
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     string(TypeBestPractice),
			Priority: string(PriorityMedium),
			Message:  fmt.Sprintf("Consider splitting into smaller, focused layers (threshold: %d packages)", a.LargeLayerThreshold),
		})
	default:
		result.Summary = fmt.Sprintf("%d packages", len(layer.Packages))
		result.Status = string(StatusGood)
	}

	// Check layer naming conventions
	if !a.IsWellNamedLayer(layer.Name) {
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     string(TypeBestPractice),
			Priority: string(PriorityLow),
			Message:  "Consider using naming convention: base, dev-*, role.*, identity.*, device.*",
		})
	}

	return result
}

// IsWellNamedLayer checks if a layer follows recognized naming conventions.
func (a *LayerAnalyzer) IsWellNamedLayer(name string) bool {
	for _, prefix := range a.WellNamedPrefixes {
		if name == prefix || strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// FindCrossLayerIssues analyzes multiple layers for cross-cutting concerns.
// It detects issues like duplicate packages across layers.
func (a *LayerAnalyzer) FindCrossLayerIssues(layers []LayerInfo) []string {
	var issues []string

	// Check for duplicate packages across layers
	packageLayers := make(map[string][]string)
	for _, layer := range layers {
		for _, pkg := range layer.Packages {
			// Normalize package name (remove " (cask)" suffix for comparison)
			normalizedPkg := strings.TrimSuffix(pkg, " (cask)")
			packageLayers[normalizedPkg] = append(packageLayers[normalizedPkg], layer.Name)
		}
	}

	for pkg, layerNames := range packageLayers {
		if len(layerNames) > 1 {
			issues = append(issues, fmt.Sprintf("Package '%s' appears in multiple layers: %s",
				pkg, strings.Join(layerNames, ", ")))
		}
	}

	return issues
}

// GetStatusIcon returns a display icon for the given status.
func GetStatusIcon(status string) string {
	switch AnalysisStatus(status) {
	case StatusGood:
		return "✓"
	case StatusWarning:
		return "⚠"
	case StatusNeedsAttention:
		return "⛔"
	default:
		return "○"
	}
}

// GetPriorityPrefix returns a colored prefix for the given priority.
func GetPriorityPrefix(priority string) string {
	switch RecommendationPriority(priority) {
	case PriorityHigh:
		return "\033[91m•\033[0m"
	case PriorityMedium:
		return "\033[93m•\033[0m"
	case PriorityLow:
		return "\033[32m•\033[0m"
	default:
		return "•"
	}
}
