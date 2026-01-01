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
// Use NewLayerAnalyzer with options to configure.
type LayerAnalyzer struct {
	largeLayerThreshold int
	wellNamedPrefixes   []string
}

// LayerAnalyzerOption configures a LayerAnalyzer.
type LayerAnalyzerOption func(*LayerAnalyzer)

// WithLargeLayerThreshold sets the threshold above which a layer is considered large.
// Layers with more packages than this should be considered for splitting.
// Default is 50.
func WithLargeLayerThreshold(threshold int) LayerAnalyzerOption {
	return func(a *LayerAnalyzer) {
		a.largeLayerThreshold = threshold
	}
}

// WithWellNamedPrefixes sets the recognized naming convention prefixes for layers.
// Default is: base, dev-, role., identity., device., misc, security, media.
func WithWellNamedPrefixes(prefixes []string) LayerAnalyzerOption {
	return func(a *LayerAnalyzer) {
		a.wellNamedPrefixes = prefixes
	}
}

// NewLayerAnalyzer creates a new LayerAnalyzer with default configuration.
// Use functional options to customize behavior.
func NewLayerAnalyzer(opts ...LayerAnalyzerOption) *LayerAnalyzer {
	a := &LayerAnalyzer{
		largeLayerThreshold: 50,
		wellNamedPrefixes: []string{
			"base", "dev-", "role.", "identity.", "device.", "misc", "security", "media",
		},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// LargeLayerThreshold returns the current large layer threshold.
//
// Deprecated: Access to this field is for backwards compatibility.
// Use WithLargeLayerThreshold option when creating the analyzer.
func (a *LayerAnalyzer) LargeLayerThreshold() int {
	return a.largeLayerThreshold
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
		result.Status = StatusWarning
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     TypeBestPractice,
			Priority: PriorityLow,
			Message:  "Layer has no packages defined",
		})
	case len(layer.Packages) > a.largeLayerThreshold:
		result.Summary = fmt.Sprintf("Large layer with %d packages", len(layer.Packages))
		result.Status = StatusWarning
		result.WellOrganized = false
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     TypeBestPractice,
			Priority: PriorityMedium,
			Message:  fmt.Sprintf("Consider splitting into smaller, focused layers (threshold: %d packages)", a.largeLayerThreshold),
		})
	default:
		result.Summary = fmt.Sprintf("%d packages", len(layer.Packages))
		result.Status = StatusGood
	}

	// Check layer naming conventions
	if !a.IsWellNamedLayer(layer.Name) {
		result.Recommendations = append(result.Recommendations, AnalysisRecommendation{
			Type:     TypeBestPractice,
			Priority: PriorityLow,
			Message:  "Consider using naming convention: base, dev-*, role.*, identity.*, device.*",
		})
	}

	return result
}

// IsWellNamedLayer checks if a layer follows recognized naming conventions.
func (a *LayerAnalyzer) IsWellNamedLayer(name string) bool {
	for _, prefix := range a.wellNamedPrefixes {
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

	// Pre-calculate total packages for map pre-allocation
	totalPackages := 0
	for _, layer := range layers {
		totalPackages += len(layer.Packages)
	}

	// Check for duplicate packages across layers
	// Pre-allocate map to avoid rehashing during growth
	packageLayers := make(map[string][]string, totalPackages)
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
//
// Deprecated: Use tui.FormatStatusIcon instead. This function will be removed in a future version.
// Presentation logic should live in the TUI layer, not the domain.
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
//
// Deprecated: Use tui.FormatPriorityPrefix instead. This function will be removed in a future version.
// Presentation logic should live in the TUI layer, not the domain.
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
