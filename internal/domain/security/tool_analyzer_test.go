package security

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolAnalyzer_Redundancy(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `
tools:
  trivy:
    category: security_scanner
    capabilities: [vulnerability_scanning, sbom_generation]
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
      - tool: syft
        reason: "SBOM generation"
  grype:
    category: security_scanner
    capabilities: [vulnerability_scanning]
  syft:
    category: sbom_generator
    capabilities: [sbom_generation]
`)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	t.Run("detects redundancy when superseding tool present", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"trivy", "grype"})
		require.NoError(t, err)

		// Should find redundancy: grype is redundant with trivy
		redundancies := findByType(result.Findings, FindingRedundancy)
		require.Len(t, redundancies, 1)

		finding := redundancies[0]
		assert.Contains(t, finding.Tools, "grype")
		assert.Contains(t, finding.Message, "trivy")
		assert.Equal(t, SeverityWarning, finding.Severity)
	})

	t.Run("detects multiple redundancies", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"trivy", "grype", "syft"})
		require.NoError(t, err)

		redundancies := findByType(result.Findings, FindingRedundancy)
		assert.Len(t, redundancies, 2)
	})

	t.Run("no redundancy when tools are independent", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"grype", "syft"})
		require.NoError(t, err)

		redundancies := findByType(result.Findings, FindingRedundancy)
		assert.Empty(t, redundancies)
	})

	t.Run("handles unknown tools gracefully", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"unknown-tool", "trivy"})
		require.NoError(t, err)

		// Should still work but may have unknown tool findings
		assert.NotNil(t, result)
	})
}

func TestToolAnalyzer_Deprecation(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `
tools:
  golint:
    category: go_linter
    deprecated: true
    deprecated_since: "2020-06-15"
    successor: golangci-lint
    reason: "Frozen and deprecated by Go team"
  golangci-lint:
    category: go_linter
  dep:
    category: go_dependency_manager
    deprecated: true
    successor: go_modules
    reason: "Superseded by Go modules"
`)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	t.Run("detects deprecated tool", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"golint"})
		require.NoError(t, err)

		deprecations := findByType(result.Findings, FindingDeprecated)
		require.Len(t, deprecations, 1)

		finding := deprecations[0]
		assert.Equal(t, []string{"golint"}, finding.Tools)
		assert.Contains(t, finding.Message, "deprecated")
		assert.Equal(t, "golangci-lint", finding.Replacement)
		assert.Equal(t, SeverityWarning, finding.Severity)
	})

	t.Run("detects multiple deprecated tools", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"golint", "dep", "golangci-lint"})
		require.NoError(t, err)

		deprecations := findByType(result.Findings, FindingDeprecated)
		assert.Len(t, deprecations, 2)
	})

	t.Run("no deprecation warning for current tools", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"golangci-lint"})
		require.NoError(t, err)

		deprecations := findByType(result.Findings, FindingDeprecated)
		assert.Empty(t, deprecations)
	})
}

func TestToolAnalyzer_Consolidation(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `
tools:
  trivy:
    category: security_scanner
    capabilities: [vulnerability_scanning, sbom_generation, secret_detection]
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
      - tool: syft
        reason: "SBOM generation"
      - tool: gitleaks
        reason: "secret detection"
  grype:
    category: security_scanner
    capabilities: [vulnerability_scanning]
  syft:
    category: sbom_generator
    capabilities: [sbom_generation]
  gitleaks:
    category: secret_scanner
    capabilities: [secret_detection]
`)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	t.Run("suggests consolidation when possible", func(t *testing.T) {
		t.Parallel()

		// All three tools can be consolidated to trivy
		result, err := analyzer.Analyze(ctx, []string{"grype", "syft", "gitleaks"})
		require.NoError(t, err)

		consolidations := findByType(result.Findings, FindingConsolidation)
		require.Len(t, consolidations, 1)

		finding := consolidations[0]
		assert.Equal(t, "trivy", finding.Replacement)
		assert.Len(t, finding.Tools, 3)
		assert.Equal(t, SeverityInfo, finding.Severity)
	})

	t.Run("no consolidation suggestion for single tool", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"grype"})
		require.NoError(t, err)

		consolidations := findByType(result.Findings, FindingConsolidation)
		assert.Empty(t, consolidations)
	})

	t.Run("no consolidation when consolidating tool already present", func(t *testing.T) {
		t.Parallel()

		// Don't suggest trivy if trivy is already in the list
		result, err := analyzer.Analyze(ctx, []string{"trivy", "grype", "syft"})
		require.NoError(t, err)

		consolidations := findByType(result.Findings, FindingConsolidation)
		assert.Empty(t, consolidations)
	})
}

func TestToolAnalyzer_Summary(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `
tools:
  trivy:
    category: security_scanner
    supersedes:
      - tool: grype
        reason: "test"
  grype:
    category: security_scanner
  golint:
    category: go_linter
    deprecated: true
    successor: golangci-lint
  golangci-lint:
    category: go_linter
`)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, []string{"trivy", "grype", "golint", "golangci-lint"})
	require.NoError(t, err)

	// Should have counts
	assert.Equal(t, 4, result.ToolsAnalyzed)
	assert.Positive(t, result.IssuesFound)

	// Summary should include deprecation + redundancy
	deprecations := findByType(result.Findings, FindingDeprecated)
	redundancies := findByType(result.Findings, FindingRedundancy)
	assert.Len(t, deprecations, 1)
	assert.Len(t, redundancies, 1)
}

func TestToolAnalyzer_EmptyInput(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `tools: {}`)
	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	result, err := analyzer.Analyze(ctx, []string{})
	require.NoError(t, err)

	assert.Equal(t, 0, result.ToolsAnalyzed)
	assert.Empty(t, result.Findings)
}

func TestToolAnalyzer_WithEmbeddedKB(t *testing.T) {
	t.Parallel()

	// Use the real embedded knowledge base
	kb, err := tools.LoadKnowledgeBase()
	require.NoError(t, err)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	t.Run("detects golint deprecation", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"golint", "golangci-lint"})
		require.NoError(t, err)

		deprecations := findByType(result.Findings, FindingDeprecated)
		require.Len(t, deprecations, 1)
		assert.Equal(t, "golangci-lint", deprecations[0].Replacement)
	})

	t.Run("detects trivy supersedes grype", func(t *testing.T) {
		t.Parallel()

		result, err := analyzer.Analyze(ctx, []string{"trivy", "grype"})
		require.NoError(t, err)

		redundancies := findByType(result.Findings, FindingRedundancy)
		require.Len(t, redundancies, 1)
		assert.Contains(t, redundancies[0].Tools, "grype")
	})
}

func TestToolAnalyzer_SortsBySeverity(t *testing.T) {
	t.Parallel()

	kb := mustParseKB(t, `
tools:
  trivy:
    category: security_scanner
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
  grype:
    category: security_scanner
  golint:
    category: go_linter
    deprecated: true
    deprecated_since: "2020-06-15"
    successor: golangci-lint
  golangci-lint:
    category: go_linter
    supersedes:
      - tool: golint
        reason: "included as linter"
`)

	analyzer := NewToolAnalyzer(kb)
	ctx := context.Background()

	// Analyze tools that will produce both deprecation and redundancy findings
	result, err := analyzer.Analyze(ctx, []string{"trivy", "grype", "golint", "golangci-lint"})
	require.NoError(t, err)

	// Verify findings are sorted by severity (warnings before info)
	// We should have at least 2 warnings (deprecation + redundancy)
	require.Greater(t, len(result.Findings), 1)

	// First findings should be warnings, not info
	for i := 0; i < len(result.Findings)-1; i++ {
		// If current is info, next cannot be warning or error
		if result.Findings[i].Severity == SeverityInfo {
			assert.NotEqual(t, SeverityWarning, result.Findings[i+1].Severity)
			assert.NotEqual(t, SeverityError, result.Findings[i+1].Severity)
		}
	}
}

func TestFindingSeverity_AllValues(t *testing.T) {
	t.Parallel()

	// Test all severity values are handled in ordering
	testCases := []struct {
		severity FindingSeverity
		order    int
	}{
		{SeverityError, 0},
		{SeverityWarning, 1},
		{SeverityInfo, 2},
		{FindingSeverity("unknown"), 3},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			t.Parallel()
			order := findingSeverityOrder(tc.severity)
			assert.Equal(t, tc.order, order)
		})
	}
}

// Helper functions

func mustParseKB(t *testing.T, yaml string) tools.KnowledgeBase {
	t.Helper()
	kb, err := tools.ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)
	return kb
}

func findByType(findings []ToolFinding, findingType FindingType) []ToolFinding {
	result := make([]ToolFinding, 0)
	for _, f := range findings {
		if f.Type == findingType {
			result = append(result, f)
		}
	}
	return result
}
