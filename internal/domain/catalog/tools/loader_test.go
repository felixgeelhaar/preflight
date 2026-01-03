package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolParsing(t *testing.T) {
	t.Parallel()

	t.Run("parses tool with all fields", func(t *testing.T) {
		t.Parallel()

		yaml := `
tools:
  trivy:
    category: security_scanner
    capabilities:
      - vulnerability_scanning
      - sbom_generation
      - secret_detection
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
      - tool: syft
        reason: "SBOM generation"
    docs: "https://trivy.dev"
`
		kb, err := ParseKnowledgeBase([]byte(yaml))
		require.NoError(t, err)

		tool, found := kb.GetTool("trivy")
		require.True(t, found)

		assert.Equal(t, "trivy", tool.Name)
		assert.Equal(t, "security_scanner", tool.Category)
		assert.Equal(t, []string{"vulnerability_scanning", "sbom_generation", "secret_detection"}, tool.Capabilities)
		assert.Len(t, tool.Supersedes, 2)
		assert.Equal(t, "grype", tool.Supersedes[0].Tool)
		assert.Equal(t, "vulnerability scanning", tool.Supersedes[0].Reason)
		assert.Equal(t, "https://trivy.dev", tool.Docs)
	})

	t.Run("parses deprecated tool", func(t *testing.T) {
		t.Parallel()

		yaml := `
tools:
  golint:
    category: go_linter
    deprecated: true
    deprecated_since: "2020-06-15"
    successor: golangci-lint
    reason: "Frozen and deprecated by Go team"
`
		kb, err := ParseKnowledgeBase([]byte(yaml))
		require.NoError(t, err)

		tool, found := kb.GetTool("golint")
		require.True(t, found)

		assert.True(t, tool.Deprecated)
		assert.Equal(t, "2020-06-15", tool.DeprecatedSince)
		assert.Equal(t, "golangci-lint", tool.Successor)
		assert.Equal(t, "Frozen and deprecated by Go team", tool.Reason)
	})

	t.Run("returns false for unknown tool", func(t *testing.T) {
		t.Parallel()

		yaml := `tools: {}`
		kb, err := ParseKnowledgeBase([]byte(yaml))
		require.NoError(t, err)

		_, found := kb.GetTool("unknown")
		assert.False(t, found)
	})
}

func TestGetDeprecatedTools(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  golint:
    category: go_linter
    deprecated: true
    successor: golangci-lint
  golangci-lint:
    category: go_linter
  dep:
    category: go_dependency_manager
    deprecated: true
    successor: go_modules
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	deprecated := kb.GetDeprecatedTools()
	assert.Len(t, deprecated, 2)

	names := make([]string, len(deprecated))
	for i, tool := range deprecated {
		names[i] = tool.Name
	}
	assert.Contains(t, names, "golint")
	assert.Contains(t, names, "dep")
}

func TestGetToolsByCategory(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  grype:
    category: security_scanner
  trivy:
    category: security_scanner
  golangci-lint:
    category: go_linter
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	scanners := kb.GetToolsByCategory("security_scanner")
	assert.Len(t, scanners, 2)

	linters := kb.GetToolsByCategory("go_linter")
	assert.Len(t, linters, 1)

	unknown := kb.GetToolsByCategory("unknown_category")
	assert.Empty(t, unknown)
}

func TestFindSupersedes(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  trivy:
    category: security_scanner
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
      - tool: syft
        reason: "SBOM generation"
  golangci-lint:
    category: go_linter
    supersedes:
      - tool: golint
        reason: "included as linter"
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	// Find tools that trivy supersedes
	superseded := kb.FindSupersedes("trivy")
	assert.Len(t, superseded, 2)

	names := make([]string, len(superseded))
	for i, s := range superseded {
		names[i] = s.Tool
	}
	assert.Contains(t, names, "grype")
	assert.Contains(t, names, "syft")

	// Tool with no supersedes
	empty := kb.FindSupersedes("grype")
	assert.Empty(t, empty)

	// Unknown tool
	unknown := kb.FindSupersedes("unknown")
	assert.Empty(t, unknown)
}

func TestFindSupersededBy(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  trivy:
    category: security_scanner
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
  grype:
    category: security_scanner
  golangci-lint:
    category: go_linter
    supersedes:
      - tool: golint
        reason: "included"
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	// grype is superseded by trivy
	superseder, found := kb.FindSupersededBy("grype")
	require.True(t, found)
	assert.Equal(t, "trivy", superseder.Name)

	// trivy is not superseded by anything
	_, found = kb.FindSupersededBy("trivy")
	assert.False(t, found)
}

func TestFindConsolidationTarget(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  trivy:
    category: security_scanner
    capabilities:
      - vulnerability_scanning
      - sbom_generation
      - secret_detection
      - iac_scanning
    supersedes:
      - tool: grype
        reason: "vulnerability scanning"
      - tool: syft
        reason: "SBOM generation"
      - tool: gitleaks
        reason: "secret detection"
  grype:
    category: security_scanner
    capabilities:
      - vulnerability_scanning
  syft:
    category: sbom_generator
    capabilities:
      - sbom_generation
  gitleaks:
    category: secret_scanner
    capabilities:
      - secret_detection
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	// Multiple tools that can be consolidated to trivy
	suggestion, found := kb.FindConsolidationTarget([]string{"grype", "syft", "gitleaks"})
	require.True(t, found)
	assert.Equal(t, "trivy", suggestion.Target)
	assert.ElementsMatch(t, []string{"grype", "syft", "gitleaks"}, suggestion.ReplacedTools)
	assert.GreaterOrEqual(t, suggestion.CoveragePercent, 0.9)

	// Partial consolidation
	suggestion, found = kb.FindConsolidationTarget([]string{"grype", "syft"})
	require.True(t, found)
	assert.Equal(t, "trivy", suggestion.Target)
	assert.Len(t, suggestion.ReplacedTools, 2)

	// No consolidation possible
	_, found = kb.FindConsolidationTarget([]string{"unknown1", "unknown2"})
	assert.False(t, found)

	// Single tool - no consolidation needed
	_, found = kb.FindConsolidationTarget([]string{"grype"})
	assert.False(t, found)
}

func TestAllTools(t *testing.T) {
	t.Parallel()

	yaml := `
tools:
  trivy:
    category: security_scanner
  grype:
    category: security_scanner
  golangci-lint:
    category: go_linter
`
	kb, err := ParseKnowledgeBase([]byte(yaml))
	require.NoError(t, err)

	all := kb.AllTools()
	assert.Len(t, all, 3)
}

func TestLoadEmbeddedKnowledgeBase(t *testing.T) {
	t.Parallel()

	kb, err := LoadKnowledgeBase()
	require.NoError(t, err)

	// Should have loaded from embedded tools.yaml
	all := kb.AllTools()
	assert.NotEmpty(t, all, "embedded knowledge base should have tools")

	// Should have common tools defined
	_, found := kb.GetTool("trivy")
	assert.True(t, found, "trivy should be in knowledge base")

	_, found = kb.GetTool("golangci-lint")
	assert.True(t, found, "golangci-lint should be in knowledge base")
}
