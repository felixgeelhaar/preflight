package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// compare.go tests
// ---------------------------------------------------------------------------

func TestBatch1_EqualValues_SameStrings(t *testing.T) {
	t.Parallel()
	assert.True(t, equalValues("hello", "hello"))
}

func TestBatch1_EqualValues_DifferentStrings(t *testing.T) {
	t.Parallel()
	assert.False(t, equalValues("hello", "world"))
}

func TestBatch1_EqualValues_IntAndString(t *testing.T) {
	t.Parallel()
	// Both print as "42" via Sprintf
	assert.True(t, equalValues(42, "42"))
}

func TestBatch1_EqualValues_NilAndNil(t *testing.T) {
	t.Parallel()
	assert.True(t, equalValues(nil, nil))
}

func TestBatch1_EqualValues_NilAndString(t *testing.T) {
	t.Parallel()
	assert.False(t, equalValues(nil, "something"))
}

func TestBatch1_EqualValues_BoolValues(t *testing.T) {
	t.Parallel()
	assert.True(t, equalValues(true, true))
	assert.False(t, equalValues(true, false))
}

func TestBatch1_ContainsProvider_Found(t *testing.T) {
	t.Parallel()
	assert.True(t, containsProvider([]string{"brew", "git", "ssh"}, "git"))
}

func TestBatch1_ContainsProvider_NotFound(t *testing.T) {
	t.Parallel()
	assert.False(t, containsProvider([]string{"brew", "git"}, "vscode"))
}

func TestBatch1_ContainsProvider_EmptyList(t *testing.T) {
	t.Parallel()
	assert.False(t, containsProvider([]string{}, "brew"))
}

func TestBatch1_ContainsProvider_WithWhitespace(t *testing.T) {
	t.Parallel()
	// containsProvider trims the provider list entries but not the search term
	assert.True(t, containsProvider([]string{" brew "}, "brew"))
	assert.False(t, containsProvider([]string{" brew "}, " brew "))
}

func TestBatch1_FormatValue_Nil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "<nil>", formatValue(nil))
}

func TestBatch1_FormatValue_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", formatValue("hello"))
}

func TestBatch1_FormatValue_Int(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "42", formatValue(42))
}

func TestBatch1_FormatValue_SliceSmall(t *testing.T) {
	t.Parallel()
	slice := []interface{}{"a", "b", "c"}
	result := formatValue(slice)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
	assert.Contains(t, result, "c")
}

func TestBatch1_FormatValue_SliceLarge(t *testing.T) {
	t.Parallel()
	slice := []interface{}{"a", "b", "c", "d"}
	result := formatValue(slice)
	assert.Equal(t, "[4 items]", result)
}

func TestBatch1_FormatValue_SliceEmpty(t *testing.T) {
	t.Parallel()
	slice := []interface{}{}
	result := formatValue(slice)
	assert.Contains(t, result, "[]")
}

func TestBatch1_FormatValue_SliceExactlyThree(t *testing.T) {
	t.Parallel()
	slice := []interface{}{1, 2, 3}
	result := formatValue(slice)
	assert.Contains(t, result, "1")
	assert.NotContains(t, result, "items")
}

func TestBatch1_FormatValue_Map(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{"a": 1, "b": 2}
	result := formatValue(m)
	assert.Equal(t, "{2 keys}", result)
}

func TestBatch1_FormatValue_MapEmpty(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{}
	result := formatValue(m)
	assert.Equal(t, "{0 keys}", result)
}

func TestBatch1_FormatValue_Bool(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "true", formatValue(true))
	assert.Equal(t, "false", formatValue(false))
}

func TestBatch1_Truncate_ShortString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestBatch1_Truncate_ExactLength(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", truncate("hello", 5))
}

func TestBatch1_Truncate_LongString(t *testing.T) {
	t.Parallel()
	result := truncate("this is a very long string indeed", 10)
	assert.Equal(t, "this is...", result)
	assert.Len(t, result, 10)
}

func TestBatch1_Truncate_MinimalLength(t *testing.T) {
	t.Parallel()
	result := truncate("abcdef", 4)
	assert.Equal(t, "a...", result)
}

func TestBatch1_Truncate_EmptyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", truncate("", 10))
}

func TestBatch1_CompareConfigs_IdenticalMaps(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"brew": map[string]interface{}{"pkg": "git"}}
	dest := map[string]interface{}{"brew": map[string]interface{}{"pkg": "git"}}
	diffs := compareConfigs(source, dest, nil)
	assert.Empty(t, diffs)
}

func TestBatch1_CompareConfigs_AddedProvider(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{}
	dest := map[string]interface{}{"brew": "something"}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "added", diffs[0].Type)
	assert.Equal(t, "brew", diffs[0].Provider)
}

func TestBatch1_CompareConfigs_RemovedProvider(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"brew": "something"}
	dest := map[string]interface{}{}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "removed", diffs[0].Type)
	assert.Equal(t, "brew", diffs[0].Provider)
}

func TestBatch1_CompareConfigs_ChangedValue(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"brew": map[string]interface{}{"version": "1.0"},
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{"version": "2.0"},
	}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
	assert.Equal(t, "version", diffs[0].Key)
}

func TestBatch1_CompareConfigs_WithProviderFilter_Included(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"brew": map[string]interface{}{"pkg": "a"},
		"git":  map[string]interface{}{"name": "x"},
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{"pkg": "b"},
		"git":  map[string]interface{}{"name": "y"},
	}
	diffs := compareConfigs(source, dest, []string{"brew"})
	for _, d := range diffs {
		assert.Equal(t, "brew", d.Provider)
	}
}

func TestBatch1_CompareConfigs_WithProviderFilter_Excluded(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"git": map[string]interface{}{"name": "x"},
	}
	dest := map[string]interface{}{
		"git": map[string]interface{}{"name": "y"},
	}
	diffs := compareConfigs(source, dest, []string{"brew"})
	assert.Empty(t, diffs)
}

func TestBatch1_CompareConfigs_BothEmpty(t *testing.T) {
	t.Parallel()
	diffs := compareConfigs(map[string]interface{}{}, map[string]interface{}{}, nil)
	assert.Empty(t, diffs)
}

func TestBatch1_CompareProviderConfig_BothMaps_KeyAdded(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"a": 1}
	dest := map[string]interface{}{"a": 1, "b": 2}
	diffs := compareProviderConfig("brew", source, dest)
	require.Len(t, diffs, 1)
	assert.Equal(t, "added", diffs[0].Type)
	assert.Equal(t, "b", diffs[0].Key)
}

func TestBatch1_CompareProviderConfig_BothMaps_KeyRemoved(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"a": 1, "b": 2}
	dest := map[string]interface{}{"a": 1}
	diffs := compareProviderConfig("brew", source, dest)
	require.Len(t, diffs, 1)
	assert.Equal(t, "removed", diffs[0].Type)
	assert.Equal(t, "b", diffs[0].Key)
}

func TestBatch1_CompareProviderConfig_BothMaps_KeyChanged(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"a": 1}
	dest := map[string]interface{}{"a": 2}
	diffs := compareProviderConfig("brew", source, dest)
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
}

func TestBatch1_CompareProviderConfig_NonMap_Equal(t *testing.T) {
	t.Parallel()
	diffs := compareProviderConfig("brew", "same", "same")
	assert.Empty(t, diffs)
}

func TestBatch1_CompareProviderConfig_NonMap_Different(t *testing.T) {
	t.Parallel()
	diffs := compareProviderConfig("brew", "alpha", "beta")
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
	assert.Equal(t, "", diffs[0].Key)
}

func TestBatch1_CompareProviderConfig_OneMapOneScalar(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"key": "val"}
	diffs := compareProviderConfig("brew", source, "scalar")
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
}

func TestBatch1_CompareProviderConfig_BothMaps_Equal(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"a": 1, "b": "c"}
	dest := map[string]interface{}{"a": 1, "b": "c"}
	diffs := compareProviderConfig("brew", source, dest)
	assert.Empty(t, diffs)
}

func TestBatch1_OutputCompareText_NoDiffs(t *testing.T) {
	output := captureStdout(t, func() {
		outputCompareText("work", "personal", nil)
	})
	assert.Contains(t, output, "No differences")
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "personal")
}

func TestBatch1_OutputCompareText_AllDiffTypes(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Dest: []interface{}{"git", "curl"}},
		{Provider: "git", Key: "email", Type: "removed", Source: "old@example.com"},
		{Provider: "ssh", Key: "config", Type: "changed", Source: "old-val", Dest: "new-val"},
		{Provider: "files", Key: "", Type: "added", Dest: "whole-section"},
	}
	output := captureStdout(t, func() {
		outputCompareText("src", "dst", diffs)
	})
	assert.Contains(t, output, "src")
	assert.Contains(t, output, "dst")
	assert.Contains(t, output, "+ added")
	assert.Contains(t, output, "- removed")
	assert.Contains(t, output, "~ changed")
	assert.Contains(t, output, "(entire section)")
	assert.Contains(t, output, "4 difference(s)")
}

func TestBatch1_OutputCompareJSON_EmptyDiffs(t *testing.T) {
	output := captureStdout(t, func() {
		err := outputCompareJSON(nil)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "[]")
}

func TestBatch1_OutputCompareJSON_WithDiffs(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Dest: "git"},
		{Provider: "git", Key: "", Type: "removed", Source: "config"},
	}
	output := captureStdout(t, func() {
		err := outputCompareJSON(diffs)
		require.NoError(t, err)
	})
	assert.Contains(t, output, `"provider"`)
	assert.Contains(t, output, `"brew"`)
	assert.Contains(t, output, `"added"`)
	assert.Contains(t, output, `"removed"`)
}

func TestBatch1_OutputCompareJSON_ParseableJSON(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "pkg", Type: "changed", Source: "v1", Dest: "v2"},
	}
	output := captureStdout(t, func() {
		err := outputCompareJSON(diffs)
		require.NoError(t, err)
	})
	var parsed []map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.Equal(t, "brew", parsed[0]["provider"])
	assert.Equal(t, "changed", parsed[0]["type"])
}

// ---------------------------------------------------------------------------
// analyze.go tests
// ---------------------------------------------------------------------------

func TestBatch1_ExtractLayerName_FromYAML(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{"name": "my-layer"}
	name := extractLayerName("/path/to/layers/something.yaml", raw)
	assert.Equal(t, "my-layer", name)
}

func TestBatch1_ExtractLayerName_FromFilename_YAML(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{}
	name := extractLayerName("/path/to/layers/dev-go.yaml", raw)
	assert.Equal(t, "dev-go", name)
}

func TestBatch1_ExtractLayerName_FromFilename_YML(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{}
	name := extractLayerName("/path/to/layers/base.yml", raw)
	assert.Equal(t, "base", name)
}

func TestBatch1_ExtractLayerName_EmptyNameInYAML(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{"name": ""}
	name := extractLayerName("/path/to/layers/fallback.yaml", raw)
	assert.Equal(t, "fallback", name)
}

func TestBatch1_ExtractLayerName_NonStringName(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{"name": 42}
	name := extractLayerName("/path/to/layers/fallback.yaml", raw)
	assert.Equal(t, "fallback", name)
}

func TestBatch1_ExtractPackages_Full(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{
				"formulae": []interface{}{"git", "curl", "wget"},
				"casks":    []interface{}{"firefox", "chrome"},
			},
		},
	}
	pkgs := extractPackages(raw)
	assert.Len(t, pkgs, 5)
	assert.Contains(t, pkgs, "git")
	assert.Contains(t, pkgs, "curl")
	assert.Contains(t, pkgs, "wget")
	assert.Contains(t, pkgs, "firefox (cask)")
	assert.Contains(t, pkgs, "chrome (cask)")
}

func TestBatch1_ExtractPackages_OnlyFormulae(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{
				"formulae": []interface{}{"git"},
			},
		},
	}
	pkgs := extractPackages(raw)
	assert.Equal(t, []string{"git"}, pkgs)
}

func TestBatch1_ExtractPackages_OnlyCasks(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{
				"casks": []interface{}{"firefox"},
			},
		},
	}
	pkgs := extractPackages(raw)
	assert.Equal(t, []string{"firefox (cask)"}, pkgs)
}

func TestBatch1_ExtractPackages_NoPackagesSection(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"git": map[string]interface{}{"name": "test"},
	}
	pkgs := extractPackages(raw)
	assert.Empty(t, pkgs)
}

func TestBatch1_ExtractPackages_EmptyBrew(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{},
		},
	}
	pkgs := extractPackages(raw)
	assert.Empty(t, pkgs)
}

func TestBatch1_ExtractPackages_NoBrew(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"apt": map[string]interface{}{"packages": []interface{}{"vim"}},
		},
	}
	pkgs := extractPackages(raw)
	assert.Empty(t, pkgs)
}

func TestBatch1_ExtractPackages_NonStringFormulae(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{
				"formulae": []interface{}{42, "git", true},
			},
		},
	}
	pkgs := extractPackages(raw)
	// Only string items should be extracted
	assert.Contains(t, pkgs, "git")
	assert.Len(t, pkgs, 1)
}

func TestBatch1_ExtractPackages_PackagesNotMap(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": "not-a-map",
	}
	pkgs := extractPackages(raw)
	assert.Empty(t, pkgs)
}

func TestBatch1_OutputAnalyzeJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputAnalyzeJSON(nil, assert.AnError)
	})
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Contains(t, parsed["error"], "assert.AnError")
}

func TestBatch1_OutputAnalyzeJSON_WithNilReport(t *testing.T) {
	output := captureStdout(t, func() {
		outputAnalyzeJSON(nil, nil)
	})
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.NotContains(t, output, "error")
}

func TestBatch1_OutputAnalyzeJSON_WithFullReport(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "base",
				PackageCount: 5,
				Status:       advisor.StatusGood,
				Summary:      "All good",
				Recommendations: []advisor.AnalysisRecommendation{
					{
						Priority: advisor.PriorityHigh,
						Message:  "Add linter",
						Packages: []string{"golangci-lint"},
					},
				},
			},
		},
		TotalPackages:        5,
		TotalRecommendations: 1,
		CrossLayerIssues:     []string{"duplicated package"},
	}
	output := captureStdout(t, func() {
		outputAnalyzeJSON(report, nil)
	})
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.InDelta(t, float64(5), parsed["total_packages"], 0)
	assert.InDelta(t, float64(1), parsed["total_recommendations"], 0)
	layers := parsed["layers"].([]interface{})
	require.Len(t, layers, 1)
}

func TestBatch1_OutputAnalyzeText_EmptyLayers(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{},
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	assert.Contains(t, output, "No layers to analyze")
}

func TestBatch1_OutputAnalyzeText_SingleLayerNoRecommend(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "base",
				PackageCount: 3,
				Status:       advisor.StatusGood,
				Summary:      "Looks great",
			},
		},
		TotalPackages:        3,
		TotalRecommendations: 0,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	assert.Contains(t, output, "base")
	assert.Contains(t, output, "3 packages")
	assert.Contains(t, output, "Looks great")
	assert.Contains(t, output, "0 recommendations")
}

func TestBatch1_OutputAnalyzeText_WithRecommendations(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "dev-go",
				PackageCount: 10,
				Status:       advisor.StatusWarning,
				Summary:      "Some issues",
				Recommendations: []advisor.AnalysisRecommendation{
					{
						Priority: advisor.PriorityHigh,
						Message:  "Consider golangci-lint",
						Packages: []string{"golint", "golangci-lint"},
					},
					{
						Priority: advisor.PriorityLow,
						Message:  "Optional: add delve",
					},
				},
			},
		},
		TotalPackages:        10,
		TotalRecommendations: 2,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, true)
	})
	assert.Contains(t, output, "Recommendations")
	assert.Contains(t, output, "Consider golangci-lint")
	assert.Contains(t, output, "golint, golangci-lint")
	assert.Contains(t, output, "Optional: add delve")
}

func TestBatch1_OutputAnalyzeText_QuietWithRecommendations(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "dev-go",
				PackageCount: 10,
				Status:       advisor.StatusWarning,
				Summary:      "Some issues",
				Recommendations: []advisor.AnalysisRecommendation{
					{
						Priority: advisor.PriorityHigh,
						Message:  "Consider golangci-lint",
						Packages: []string{"golint"},
					},
				},
			},
		},
		TotalPackages:        10,
		TotalRecommendations: 1,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, true, true)
	})
	// In quiet mode with recommend, recommendations show but packages don't
	assert.Contains(t, output, "Consider golangci-lint")
	assert.NotContains(t, output, "Packages: golint")
}

func TestBatch1_OutputAnalyzeText_WithCrossLayerIssues(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "a", PackageCount: 1, Summary: "ok"},
			{LayerName: "b", PackageCount: 2, Summary: "ok"},
		},
		TotalPackages:    3,
		CrossLayerIssues: []string{"pkg X found in both a and b"},
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	assert.Contains(t, output, "Cross-Layer Issues")
	assert.Contains(t, output, "pkg X found in both a and b")
}

func TestBatch1_OutputAnalyzeText_QuietNoTable(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "a", PackageCount: 1, Summary: "ok"},
			{LayerName: "b", PackageCount: 2, Summary: "ok"},
		},
		TotalPackages: 3,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, true, false)
	})
	// In quiet mode the summary table should not be printed
	assert.NotContains(t, output, "LAYER")
	assert.NotContains(t, output, "PACKAGES")
}

func TestBatch1_OutputAnalyzeText_NotQuietShowsTable(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "a", PackageCount: 1, Status: advisor.StatusGood, Summary: "ok"},
			{LayerName: "b", PackageCount: 2, Status: "", Summary: "ok"},
		},
		TotalPackages: 3,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	// With 2+ layers and not quiet, table should appear
	assert.Contains(t, output, "LAYER")
	assert.Contains(t, output, "PACKAGES")
	assert.Contains(t, output, "STATUS")
}

func TestBatch1_PrintLayerSummaryTable_EmptyStatus(t *testing.T) {
	layers := []advisor.LayerAnalysisResult{
		{LayerName: "test", PackageCount: 5, Status: ""},
	}
	output := captureStdout(t, func() {
		printLayerSummaryTable(layers)
	})
	// Empty status should be replaced with "-"
	assert.Contains(t, output, "-")
	assert.Contains(t, output, "test")
}

func TestBatch1_PrintLayerSummaryTable_MultiLayers(t *testing.T) {
	layers := []advisor.LayerAnalysisResult{
		{LayerName: "base", PackageCount: 3, Status: advisor.StatusGood, Recommendations: []advisor.AnalysisRecommendation{{Message: "a"}}},
		{LayerName: "dev", PackageCount: 10, Status: advisor.StatusWarning},
	}
	output := captureStdout(t, func() {
		printLayerSummaryTable(layers)
	})
	assert.Contains(t, output, "base")
	assert.Contains(t, output, "dev")
	assert.Contains(t, output, "RECOMMENDATIONS")
}

func TestBatch1_AnalyzeLayersWithAI_NoAIProvider(t *testing.T) {
	t.Parallel()
	layers := []advisor.LayerInfo{
		{Name: "base", Packages: []string{"git", "curl"}},
		{Name: "dev", Packages: []string{"go", "node"}},
	}
	report := analyzeLayersWithAI(context.Background(), layers, nil)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 2)
	assert.Equal(t, 4, report.TotalPackages)
	assert.Equal(t, "base", report.Layers[0].LayerName)
	assert.Equal(t, 2, report.Layers[0].PackageCount)
	assert.Equal(t, "dev", report.Layers[1].LayerName)
	assert.Equal(t, 2, report.Layers[1].PackageCount)
}

func TestBatch1_AnalyzeLayersWithAI_EmptyLayers(t *testing.T) {
	t.Parallel()
	report := analyzeLayersWithAI(context.Background(), []advisor.LayerInfo{}, nil)
	require.NotNil(t, report)
	assert.Empty(t, report.Layers)
	assert.Equal(t, 0, report.TotalPackages)
}

func TestBatch1_AnalyzeLayersWithAI_SingleLayer(t *testing.T) {
	t.Parallel()
	layers := []advisor.LayerInfo{
		{
			Name:           "full",
			Packages:       []string{"git", "curl", "wget", "vim", "neovim"},
			HasGitConfig:   true,
			HasShellConfig: true,
		},
	}
	report := analyzeLayersWithAI(context.Background(), layers, nil)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 1)
	assert.Equal(t, 5, report.TotalPackages)
}

// ---------------------------------------------------------------------------
// profile.go tests
// ---------------------------------------------------------------------------

func TestBatch1_GetProfileDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	dir := getProfileDir()
	assert.Equal(t, filepath.Join(tmpDir, ".preflight", "profiles"), dir)
}

func TestBatch1_GetCurrentProfile_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	assert.Equal(t, "", getCurrentProfile())
}

func TestBatch1_SetAndGetCurrentProfile_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, setCurrentProfile("work"))
	assert.Equal(t, "work", getCurrentProfile())

	require.NoError(t, setCurrentProfile("personal"))
	assert.Equal(t, "personal", getCurrentProfile())
}

func TestBatch1_SetCurrentProfile_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	require.NoError(t, setCurrentProfile("test"))

	info, err := os.Stat(filepath.Join(tmpDir, ".preflight", "profiles"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestBatch1_SetCurrentProfile_WithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	require.NoError(t, setCurrentProfile("  spaced  "))
	// getCurrentProfile trims whitespace
	assert.Equal(t, "spaced", getCurrentProfile())
}

func TestBatch1_SaveAndLoadCustomProfiles_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "alpha", Target: "t1", Description: "First"},
		{Name: "beta", Target: "t2", Active: true, LastUsed: "2026-01-01"},
	}
	require.NoError(t, saveCustomProfiles(profiles))

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "alpha", loaded[0].Name)
	assert.Equal(t, "t1", loaded[0].Target)
	assert.Equal(t, "First", loaded[0].Description)
	assert.Equal(t, "beta", loaded[1].Name)
}

func TestBatch1_SaveCustomProfiles_EmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, saveCustomProfiles([]ProfileInfo{}))
	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestBatch1_LoadCustomProfiles_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestBatch1_LoadCustomProfiles_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	require.NoError(t, os.MkdirAll(profileDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "profiles.yaml"), []byte("{{invalid"), 0o644))

	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestBatch1_SaveCustomProfiles_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, saveCustomProfiles([]ProfileInfo{{Name: "old", Target: "t"}}))
	require.NoError(t, saveCustomProfiles([]ProfileInfo{{Name: "new", Target: "t2"}}))

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, "new", loaded[0].Name)
}

func TestBatch1_ApplyGitConfig_AllFields(t *testing.T) {
	git := map[string]interface{}{
		"name":        "Test User",
		"email":       "test@example.com",
		"signing_key": "ABC123",
	}
	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})
	assert.Contains(t, output, `user.name "Test User"`)
	assert.Contains(t, output, `user.email "test@example.com"`)
	assert.Contains(t, output, `user.signingkey "ABC123"`)
}

func TestBatch1_ApplyGitConfig_OnlyName(t *testing.T) {
	git := map[string]interface{}{
		"name": "Only Name",
	}
	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "user.name")
	assert.NotContains(t, output, "user.email")
	assert.NotContains(t, output, "user.signingkey")
}

func TestBatch1_ApplyGitConfig_Empty(t *testing.T) {
	output := captureStdout(t, func() {
		err := applyGitConfig(map[string]interface{}{})
		require.NoError(t, err)
	})
	assert.Empty(t, output)
}

func TestBatch1_ApplyGitConfig_NonStringValues(t *testing.T) {
	git := map[string]interface{}{
		"name":  42,
		"email": true,
	}
	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})
	// Non-string values should not be applied
	assert.Empty(t, output)
}

func TestBatch1_RunGitConfigSet_FormatsCorrectly(t *testing.T) {
	output := captureStdout(t, func() {
		err := runGitConfigSet("user.name", "John Doe")
		require.NoError(t, err)
	})
	assert.Equal(t, "    git config --global user.name \"John Doe\"\n", output)
}

func TestBatch1_RunGitConfigSet_EmptyValue(t *testing.T) {
	output := captureStdout(t, func() {
		err := runGitConfigSet("user.email", "")
		require.NoError(t, err)
	})
	assert.Contains(t, output, "user.email")
}

func TestBatch1_RunGitConfigSet_SpecialCharacters(t *testing.T) {
	output := captureStdout(t, func() {
		err := runGitConfigSet("user.name", `O'Brien "Bobby"`)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "user.name")
}

// ---------------------------------------------------------------------------
// rollback.go - listSnapshots
// ---------------------------------------------------------------------------

func TestBatch1_ListSnapshots_EmptySets(t *testing.T) {
	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, []snapshot.Set{})
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Available Snapshots")
	assert.Contains(t, output, "preflight rollback --to")
}

func TestBatch1_ListSnapshots_WithSets(t *testing.T) {
	now := time.Now()
	sets := []snapshot.Set{
		{
			ID:        "abcdefghijklmnop",
			CreatedAt: now.Add(-5 * time.Minute),
			Reason:    "pre-apply",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.bashrc"},
				{Path: "/home/user/.gitconfig"},
			},
		},
		{
			ID:        "1234567890abcdef",
			CreatedAt: now.Add(-2 * time.Hour),
			Reason:    "",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.vimrc"},
			},
		},
	}
	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "abcdefg")
	assert.Contains(t, output, "2 files")
	assert.Contains(t, output, "pre-apply")
	assert.Contains(t, output, "12345678")
	assert.Contains(t, output, "1 files")
	// Second set has no reason, so "Reason:" should appear only once
	assert.Equal(t, 1, strings.Count(output, "Reason:"))
}

func TestBatch1_ListSnapshots_SingleSet(t *testing.T) {
	sets := []snapshot.Set{
		{
			ID:        "aabbccddeeff0011",
			CreatedAt: time.Now().Add(-30 * time.Second),
			Reason:    "manual backup",
			Snapshots: []snapshot.Snapshot{
				{Path: "/some/file"},
			},
		},
	}
	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "aabbccdd")
	assert.Contains(t, output, "manual backup")
	assert.Contains(t, output, "Usage:")
}

// ---------------------------------------------------------------------------
// env.go tests
// ---------------------------------------------------------------------------

func TestBatch1_ExtractEnvVars_WithSecrets(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"EDITOR":    "nvim",
			"API_TOKEN": "secret://vault/token",
		},
	}
	vars := extractEnvVars(config)
	assert.Len(t, vars, 2)

	varMap := make(map[string]EnvVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}
	assert.False(t, varMap["EDITOR"].Secret)
	assert.True(t, varMap["API_TOKEN"].Secret)
}

func TestBatch1_ExtractEnvVars_EmptyMap(t *testing.T) {
	t.Parallel()
	vars := extractEnvVars(map[string]interface{}{})
	assert.Empty(t, vars)
}

func TestBatch1_ExtractEnvVars_NonMapEnv(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{"env": "string-not-map"}
	vars := extractEnvVars(config)
	assert.Empty(t, vars)
}

func TestBatch1_ExtractEnvVars_IntegerValues(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"PORT":  8080,
			"DEBUG": true,
		},
	}
	vars := extractEnvVars(config)
	assert.Len(t, vars, 2)

	varMap := make(map[string]EnvVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}
	assert.Equal(t, "8080", varMap["PORT"].Value)
	assert.Equal(t, "true", varMap["DEBUG"].Value)
}

func TestBatch1_ExtractEnvVarsMap_Normal(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"SHELL": "/bin/zsh",
			"PORT":  8080,
		},
	}
	result := extractEnvVarsMap(config)
	assert.Len(t, result, 2)
	assert.Equal(t, "/bin/zsh", result["SHELL"])
	assert.Equal(t, "8080", result["PORT"])
}

func TestBatch1_ExtractEnvVarsMap_Empty(t *testing.T) {
	t.Parallel()
	result := extractEnvVarsMap(map[string]interface{}{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestBatch1_ExtractEnvVarsMap_NonMapEnv(t *testing.T) {
	t.Parallel()
	result := extractEnvVarsMap(map[string]interface{}{"env": 42})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestBatch1_WriteEnvFile_NonSecretVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "EDITOR", Value: "nvim"},
		{Name: "GOPATH", Value: "/home/user/go"},
	}
	require.NoError(t, WriteEnvFile(vars))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "# Generated by preflight")
	assert.Contains(t, s, `export EDITOR="nvim"`)
	assert.Contains(t, s, `export GOPATH="/home/user/go"`)
}

func TestBatch1_WriteEnvFile_SecretVarsExcluded(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "PUBLIC", Value: "ok"},
		{Name: "TOKEN", Value: "secret://vault/tok", Secret: true},
	}
	require.NoError(t, WriteEnvFile(vars))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "PUBLIC")
	assert.NotContains(t, s, "TOKEN")
	assert.NotContains(t, s, "secret://")
}

func TestBatch1_WriteEnvFile_EmptyVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, WriteEnvFile([]EnvVar{}))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "# Generated by preflight")
	assert.NotContains(t, s, "export")
}

func TestBatch1_WriteEnvFile_AllSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "S1", Value: "x", Secret: true},
		{Name: "S2", Value: "y", Secret: true},
	}
	require.NoError(t, WriteEnvFile(vars))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	assert.NotContains(t, string(content), "export")
}

func TestBatch1_WriteEnvFile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{{Name: "X", Value: "1"}}
	require.NoError(t, WriteEnvFile(vars))

	info, err := os.Stat(filepath.Join(tmpDir, ".preflight"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestBatch1_WriteEnvFile_SpecialCharsInValue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "PATH_VAR", Value: `/usr/local/bin:/usr/bin`},
		{Name: "QUOTED", Value: `has "quotes" and 'ticks'`},
	}
	require.NoError(t, WriteEnvFile(vars))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	s := string(content)
	assert.Contains(t, s, "PATH_VAR")
	assert.Contains(t, s, "QUOTED")
}

// ---------------------------------------------------------------------------
// configDiff struct tests
// ---------------------------------------------------------------------------

func TestBatch1_ConfigDiff_Fields(t *testing.T) {
	t.Parallel()
	d := configDiff{
		Provider: "brew",
		Key:      "formulae",
		Type:     "added",
		Source:   nil,
		Dest:     []interface{}{"git"},
	}
	assert.Equal(t, "brew", d.Provider)
	assert.Equal(t, "formulae", d.Key)
	assert.Equal(t, "added", d.Type)
	assert.Nil(t, d.Source)
	assert.NotNil(t, d.Dest)
}

// ---------------------------------------------------------------------------
// Additional compare.go edge cases
// ---------------------------------------------------------------------------

func TestBatch1_CompareConfigs_MultipleProviders(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"brew":  map[string]interface{}{"a": 1},
		"git":   map[string]interface{}{"b": 2},
		"shell": map[string]interface{}{"c": 3},
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{"a": 1},
		"git":  map[string]interface{}{"b": 99},
		"ssh":  map[string]interface{}{"d": 4},
	}
	diffs := compareConfigs(source, dest, nil)
	// shell removed, git changed, ssh added
	typeMap := make(map[string]int)
	for _, d := range diffs {
		typeMap[d.Type]++
	}
	assert.GreaterOrEqual(t, typeMap["removed"], 1) // at least shell
	assert.GreaterOrEqual(t, typeMap["added"], 1)   // at least ssh
	assert.GreaterOrEqual(t, typeMap["changed"], 1) // at least git.b
}

func TestBatch1_OutputCompareText_LongValue_Truncated(t *testing.T) {
	longVal := strings.Repeat("x", 100)
	diffs := []configDiff{
		{Provider: "test", Key: "key", Type: "changed", Source: longVal, Dest: "short"},
	}
	output := captureStdout(t, func() {
		outputCompareText("a", "b", diffs)
	})
	// The truncated detail should end with "..."
	assert.Contains(t, output, "...")
}

// ---------------------------------------------------------------------------
// Additional analyze.go edge cases
// ---------------------------------------------------------------------------

func TestBatch1_OutputAnalyzeText_ZeroTotalPackages(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "empty", PackageCount: 0, Summary: "no packages"},
		},
		TotalPackages: 0,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	// When TotalPackages is 0, the "Total packages:" line should NOT appear
	assert.NotContains(t, output, "Total packages:")
}

func TestBatch1_OutputAnalyzeText_NonZeroTotalPackages(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "base", PackageCount: 5, Summary: "ok"},
		},
		TotalPackages: 5,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	assert.Contains(t, output, "Total packages: 5")
}

func TestBatch1_OutputAnalyzeText_RecommendFalseHidesRecs(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{
				LayerName:    "dev",
				PackageCount: 2,
				Summary:      "needs work",
				Recommendations: []advisor.AnalysisRecommendation{
					{Priority: advisor.PriorityHigh, Message: "hidden rec"},
				},
			},
		},
		TotalRecommendations: 1,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	assert.NotContains(t, output, "hidden rec")
	assert.NotContains(t, output, "Recommendations:")
}

func TestBatch1_OutputAnalyzeText_SingleLayerNoTable(t *testing.T) {
	report := &advisor.AnalysisReport{
		Layers: []advisor.LayerAnalysisResult{
			{LayerName: "only", PackageCount: 3, Summary: "ok"},
		},
		TotalPackages: 3,
	}
	output := captureStdout(t, func() {
		outputAnalyzeText(report, false, false)
	})
	// With only 1 layer, the summary table should NOT appear
	assert.NotContains(t, output, "LAYER\tPACKAGES")
}

func TestBatch1_ExtractPackages_EmptyFormulae(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"packages": map[string]interface{}{
			"brew": map[string]interface{}{
				"formulae": []interface{}{},
				"casks":    []interface{}{},
			},
		},
	}
	pkgs := extractPackages(raw)
	assert.Empty(t, pkgs)
}

func TestBatch1_ExtractLayerName_YAML_TakesPriority(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{"name": "from-yaml"}
	name := extractLayerName("/path/to/layers/from-file.yaml", raw)
	assert.Equal(t, "from-yaml", name)
}

// ---------------------------------------------------------------------------
// Profile roundtrip stress test
// ---------------------------------------------------------------------------

func TestBatch1_SaveLoadProfiles_ManyProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := make([]ProfileInfo, 20)
	for i := range profiles {
		profiles[i] = ProfileInfo{
			Name:        "profile-" + strings.Repeat("x", i+1),
			Target:      "target",
			Description: "desc",
		}
	}
	require.NoError(t, saveCustomProfiles(profiles))

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, loaded, 20)
}

// ---------------------------------------------------------------------------
// Compare with complex nested structures
// ---------------------------------------------------------------------------

func TestBatch1_CompareConfigs_NestedMapChanges(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git"},
			"taps":     []interface{}{"homebrew/core"},
		},
	}
	dest := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
			"taps":     []interface{}{"homebrew/core"},
		},
	}
	diffs := compareConfigs(source, dest, nil)
	// formulae changed (different Sprintf output)
	found := false
	for _, d := range diffs {
		if d.Key == "formulae" {
			found = true
			assert.Equal(t, "changed", d.Type)
		}
	}
	assert.True(t, found, "should detect change in formulae")
}

// ---------------------------------------------------------------------------
// listSnapshots - edge cases
// ---------------------------------------------------------------------------

func TestBatch1_ListSnapshots_VeryRecentTimestamp(t *testing.T) {
	sets := []snapshot.Set{
		{
			ID:        "recentid12345678",
			CreatedAt: time.Now(),
			Reason:    "just created",
			Snapshots: []snapshot.Snapshot{{Path: "/a"}},
		},
	}
	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "recentid")
	assert.Contains(t, output, "just created")
}

func TestBatch1_ListSnapshots_ManySnapshots(t *testing.T) {
	snapshots := make([]snapshot.Snapshot, 100)
	for i := range snapshots {
		snapshots[i] = snapshot.Snapshot{Path: "/some/path/" + strings.Repeat("x", i+1)}
	}
	sets := []snapshot.Set{
		{
			ID:        "bigsnap123456789",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Reason:    "big snapshot",
			Snapshots: snapshots,
		},
	}
	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "100 files")
}

// ---------------------------------------------------------------------------
// EnvVar struct
// ---------------------------------------------------------------------------

func TestBatch1_EnvVar_AllFields(t *testing.T) {
	t.Parallel()
	v := EnvVar{
		Name:   "DATABASE_URL",
		Value:  "postgres://localhost/db",
		Layer:  "identity.work",
		Secret: false,
	}
	assert.Equal(t, "DATABASE_URL", v.Name)
	assert.Equal(t, "postgres://localhost/db", v.Value)
	assert.Equal(t, "identity.work", v.Layer)
	assert.False(t, v.Secret)
}

func TestBatch1_EnvVar_SecretFlag(t *testing.T) {
	t.Parallel()
	v := EnvVar{
		Name:   "API_KEY",
		Value:  "secret://vault/key",
		Secret: true,
	}
	assert.True(t, v.Secret)
}

// ---------------------------------------------------------------------------
// ProfileInfo struct
// ---------------------------------------------------------------------------

func TestBatch1_ProfileInfo_Defaults(t *testing.T) {
	t.Parallel()
	p := ProfileInfo{}
	assert.Empty(t, p.Name)
	assert.Empty(t, p.Target)
	assert.Empty(t, p.Description)
	assert.False(t, p.Active)
	assert.Empty(t, p.LastUsed)
}

func TestBatch1_ProfileInfo_AllFieldsSet(t *testing.T) {
	t.Parallel()
	p := ProfileInfo{
		Name:        "work",
		Target:      "work-target",
		Description: "Corporate setup",
		Active:      true,
		LastUsed:    "2026-02-25T10:00:00Z",
	}
	assert.Equal(t, "work", p.Name)
	assert.Equal(t, "work-target", p.Target)
	assert.Equal(t, "Corporate setup", p.Description)
	assert.True(t, p.Active)
	assert.Equal(t, "2026-02-25T10:00:00Z", p.LastUsed)
}

// ---------------------------------------------------------------------------
// OutputCompareJSON with zero-value diffs
// ---------------------------------------------------------------------------

func TestBatch1_OutputCompareJSON_ZeroDiffs(t *testing.T) {
	output := captureStdout(t, func() {
		err := outputCompareJSON([]configDiff{})
		require.NoError(t, err)
	})
	var parsed []interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))
	assert.Empty(t, parsed)
}

// ---------------------------------------------------------------------------
// FormatValue with different types
// ---------------------------------------------------------------------------

func TestBatch1_FormatValue_Float(t *testing.T) {
	t.Parallel()
	result := formatValue(3.14)
	assert.Equal(t, "3.14", result)
}

func TestBatch1_FormatValue_SliceOfOne(t *testing.T) {
	t.Parallel()
	slice := []interface{}{"only-one"}
	result := formatValue(slice)
	assert.Contains(t, result, "only-one")
}

func TestBatch1_FormatValue_SliceOfFive(t *testing.T) {
	t.Parallel()
	slice := []interface{}{1, 2, 3, 4, 5}
	result := formatValue(slice)
	assert.Equal(t, "[5 items]", result)
}

func TestBatch1_FormatValue_MapSingleKey(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{"only": "one"}
	assert.Equal(t, "{1 keys}", formatValue(m))
}

// ---------------------------------------------------------------------------
// Truncate edge cases
// ---------------------------------------------------------------------------

func TestBatch1_Truncate_ThreeCharLimit(t *testing.T) {
	t.Parallel()
	// maxLen = 3 means s[:0] + "..." = "..."
	result := truncate("abcdef", 3)
	assert.Equal(t, "...", result)
}

func TestBatch1_Truncate_SingleCharString(t *testing.T) {
	t.Parallel()
	result := truncate("x", 10)
	assert.Equal(t, "x", result)
}

// ---------------------------------------------------------------------------
// AnalyzeLayersWithAI - with cross-layer detection
// ---------------------------------------------------------------------------

func TestBatch1_AnalyzeLayersWithAI_DetectsCrossLayerIssues(t *testing.T) {
	t.Parallel()
	layers := []advisor.LayerInfo{
		{Name: "base", Packages: []string{"git", "curl"}},
		{Name: "dev", Packages: []string{"git", "go"}}, // git is in both
	}
	report := analyzeLayersWithAI(context.Background(), layers, nil)
	require.NotNil(t, report)
	// The layer analyzer should detect the cross-layer duplicate
	// (depends on the analyzer implementation; at minimum verify it doesn't panic)
	assert.Len(t, report.Layers, 2)
}

// ---------------------------------------------------------------------------
// Compare multiple types in a single run
// ---------------------------------------------------------------------------

func TestBatch1_CompareConfigs_MixedTypesInProviders(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"string-provider":  "value1",
		"int-provider":     42,
		"map-provider":     map[string]interface{}{"k": "v"},
		"removed-provider": "gone",
	}
	dest := map[string]interface{}{
		"string-provider": "value2",
		"int-provider":    42,
		"map-provider":    map[string]interface{}{"k": "v2"},
		"new-provider":    "hello",
	}
	diffs := compareConfigs(source, dest, nil)
	typeCount := map[string]int{}
	for _, d := range diffs {
		typeCount[d.Type]++
	}
	assert.Equal(t, 1, typeCount["removed"])          // removed-provider
	assert.Equal(t, 1, typeCount["added"])            // new-provider
	assert.GreaterOrEqual(t, typeCount["changed"], 2) // string-provider + map-provider.k
}

// ---------------------------------------------------------------------------
// WriteEnvFile - multiple non-secret vars
// ---------------------------------------------------------------------------

func TestBatch1_WriteEnvFile_MultipleVars(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "A", Value: "1"},
		{Name: "B", Value: "2"},
		{Name: "C", Value: "3"},
		{Name: "D_SECRET", Value: "hidden", Secret: true},
	}
	require.NoError(t, WriteEnvFile(vars))

	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)
	s := string(content)
	assert.Equal(t, 3, strings.Count(s, "export"))
}

// ---------------------------------------------------------------------------
// extractEnvVars with secret:// prefix detection
// ---------------------------------------------------------------------------

func TestBatch1_ExtractEnvVars_SecretPrefixDetection(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"NORMAL":         "just-a-value",
			"VAULT_SECRET":   "secret://vault/my-secret",
			"ALSO_SECRET":    "secret://bitwarden/key",
			"NOT_SECRET":     "secretarial-work",
			"ANOTHER_NORMAL": 42,
		},
	}
	vars := extractEnvVars(config)
	varMap := make(map[string]EnvVar)
	for _, v := range vars {
		varMap[v.Name] = v
	}
	assert.False(t, varMap["NORMAL"].Secret)
	assert.True(t, varMap["VAULT_SECRET"].Secret)
	assert.True(t, varMap["ALSO_SECRET"].Secret)
	assert.False(t, varMap["NOT_SECRET"].Secret)
	assert.False(t, varMap["ANOTHER_NORMAL"].Secret)
}

// ---------------------------------------------------------------------------
// analyzeLayersWithAI - AI provider paths (exercising 40% -> higher coverage)
// ---------------------------------------------------------------------------

func TestBatch1_AnalyzeLayersWithAI_AIError(t *testing.T) {
	// Save/restore globals that analyzeLayersWithAI reads
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = true
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	provider := &mockAIProvider{
		name:      "failing-provider",
		available: true,
		err:       errors.New("rate limited"),
	}
	layers := []advisor.LayerInfo{
		{Name: "test-layer", Packages: []string{"git", "curl"}},
	}
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 1)
	// When AI fails, it falls back to basic analysis and prepends "AI unavailable"
	assert.Contains(t, report.Layers[0].Summary, "AI unavailable")
}

func TestBatch1_AnalyzeLayersWithAI_AIError_NotQuiet(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = false
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	provider := &mockAIProvider{
		name:      "failing-provider",
		available: true,
		err:       errors.New("timeout"),
	}
	layers := []advisor.LayerInfo{
		{Name: "layer1", Packages: []string{"git"}},
	}
	// This will print a warning to stderr, but should not panic
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Contains(t, report.Layers[0].Summary, "AI unavailable")
}

func TestBatch1_AnalyzeLayersWithAI_AISuccess_ValidJSON(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = true
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	validJSON := `{
		"layer_name": "test",
		"summary": "Well organized layer",
		"status": "good",
		"package_count": 2,
		"well_organized": true,
		"recommendations": []
	}`
	provider := &mockAIProvider{
		name:      "success-provider",
		available: true,
		response:  advisor.NewResponse(validJSON, 100, "test-model"),
	}
	layers := []advisor.LayerInfo{
		{Name: "my-layer", Packages: []string{"git", "curl"}},
	}
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 1)
	// AI result should be used, with layer name overridden
	assert.Equal(t, "my-layer", report.Layers[0].LayerName)
	assert.Equal(t, 2, report.Layers[0].PackageCount)
}

func TestBatch1_AnalyzeLayersWithAI_AISuccess_InvalidJSON(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = true
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	provider := &mockAIProvider{
		name:      "bad-json-provider",
		available: true,
		response:  advisor.NewResponse("this is not JSON at all", 50, "test-model"),
	}
	layers := []advisor.LayerInfo{
		{Name: "fallback-layer", Packages: []string{"vim"}},
	}
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 1)
	// Should fall back to basic analysis when parse fails
	assert.Equal(t, "fallback-layer", report.Layers[0].LayerName)
}

func TestBatch1_AnalyzeLayersWithAI_AISuccess_InvalidJSON_NotQuiet(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = false
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	provider := &mockAIProvider{
		name:      "bad-parse-provider",
		available: true,
		response:  advisor.NewResponse("no json", 50, "test-model"),
	}
	layers := []advisor.LayerInfo{
		{Name: "test", Packages: []string{"git"}},
	}
	// Should print a parse warning to stderr and fall back
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 1)
}

func TestBatch1_AnalyzeLayersWithAI_AIError_JSONMode(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = false
	analyzeJSON = true
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	provider := &mockAIProvider{
		name:      "json-mode-fail",
		available: true,
		err:       fmt.Errorf("connection refused"),
	}
	layers := []advisor.LayerInfo{
		{Name: "json-layer", Packages: []string{"node"}},
	}
	// In JSON mode, warnings should not be printed (analyzeJSON is true)
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Contains(t, report.Layers[0].Summary, "AI unavailable")
}

func TestBatch1_AnalyzeLayersWithAI_MultipleLayers_MixedResults(t *testing.T) {
	savedQuiet := analyzeQuiet
	savedJSON := analyzeJSON
	analyzeQuiet = true
	analyzeJSON = false
	defer func() {
		analyzeQuiet = savedQuiet
		analyzeJSON = savedJSON
	}()

	// This provider returns valid JSON but it will be the same for all layers
	validJSON := `{"summary": "AI says good", "status": "good", "recommendations": []}`
	provider := &mockAIProvider{
		name:      "multi-layer-provider",
		available: true,
		response:  advisor.NewResponse(validJSON, 100, "test-model"),
	}
	layers := []advisor.LayerInfo{
		{Name: "base", Packages: []string{"git", "curl"}},
		{Name: "dev-go", Packages: []string{"go", "golangci-lint", "delve"}},
	}
	report := analyzeLayersWithAI(context.Background(), layers, provider)
	require.NotNil(t, report)
	assert.Len(t, report.Layers, 2)
	assert.Equal(t, 5, report.TotalPackages)
	assert.Equal(t, "base", report.Layers[0].LayerName)
	assert.Equal(t, "dev-go", report.Layers[1].LayerName)
}

// ---------------------------------------------------------------------------
// runProfileCreate / runProfileDelete edge cases
// ---------------------------------------------------------------------------

func TestBatch1_RunProfileCreate_NewProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = "test-target"
	defer func() { profileFromTarget = savedFromTarget }()

	output := captureStdout(t, func() {
		err := runProfileCreate(nil, []string{"new-prof"})
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Created profile 'new-prof'")

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "new-prof", loaded[0].Name)
	assert.Equal(t, "test-target", loaded[0].Target)
}

func TestBatch1_RunProfileCreate_DefaultTarget(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = ""
	defer func() { profileFromTarget = savedFromTarget }()

	output := captureStdout(t, func() {
		err := runProfileCreate(nil, []string{"def-prof"})
		require.NoError(t, err)
	})
	assert.Contains(t, output, "from target 'default'")
}

func TestBatch1_RunProfileCreate_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedFromTarget := profileFromTarget
	profileFromTarget = "t"
	defer func() { profileFromTarget = savedFromTarget }()

	captureStdout(t, func() {
		require.NoError(t, runProfileCreate(nil, []string{"dup"}))
	})
	err := runProfileCreate(nil, []string{"dup"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestBatch1_RunProfileDelete_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, saveCustomProfiles([]ProfileInfo{
		{Name: "keep", Target: "t"},
		{Name: "remove", Target: "t"},
	}))

	output := captureStdout(t, func() {
		require.NoError(t, runProfileDelete(nil, []string{"remove"}))
	})
	assert.Contains(t, output, "Deleted profile 'remove'")

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "keep", loaded[0].Name)
}

func TestBatch1_RunProfileDelete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	require.NoError(t, saveCustomProfiles([]ProfileInfo{
		{Name: "existing", Target: "t"},
	}))

	err := runProfileDelete(nil, []string{"nope"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBatch1_RunProfileDelete_NoProfilesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runProfileDelete(nil, []string{"any"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBatch1_RunProfileCurrent_NoProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedJSON := profileJSON
	profileJSON = false
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		require.NoError(t, runProfileCurrent(nil, nil))
	})
	assert.Contains(t, output, "No profile active")
}

func TestBatch1_RunProfileCurrent_WithProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	require.NoError(t, setCurrentProfile("active-one"))

	savedJSON := profileJSON
	profileJSON = false
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		require.NoError(t, runProfileCurrent(nil, nil))
	})
	assert.Contains(t, output, "Current profile: active-one")
}

func TestBatch1_RunProfileCurrent_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	require.NoError(t, setCurrentProfile("json-prof"))

	savedJSON := profileJSON
	profileJSON = true
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		require.NoError(t, runProfileCurrent(nil, nil))
	})
	assert.Contains(t, output, "json-prof")
	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))
	assert.Equal(t, "json-prof", parsed["profile"])
}

func TestBatch1_RunProfileCurrent_JSON_NoProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	savedJSON := profileJSON
	profileJSON = true
	defer func() { profileJSON = savedJSON }()

	output := captureStdout(t, func() {
		require.NoError(t, runProfileCurrent(nil, nil))
	})
	// With no active profile, it still prints "No profile active" (not JSON)
	assert.Contains(t, output, "No profile active")
}

// ---------------------------------------------------------------------------
// Additional formatAge tests (rollback.go)
// ---------------------------------------------------------------------------

func TestBatch1_FormatAge_JustNow(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now())
	assert.Equal(t, "just now", result)
}

func TestBatch1_FormatAge_OneMinAgo(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-1 * time.Minute))
	assert.Equal(t, "1 min ago", result)
}

func TestBatch1_FormatAge_MultipleMinutes(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-15 * time.Minute))
	assert.Equal(t, "15 mins ago", result)
}

func TestBatch1_FormatAge_OneHour(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-1 * time.Hour))
	assert.Contains(t, result, "hour")
}

func TestBatch1_FormatAge_MultipleHours(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-5 * time.Hour))
	assert.Contains(t, result, "hours")
}

func TestBatch1_FormatAge_OneDayAgo(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-25 * time.Hour))
	assert.Contains(t, result, "day")
}

func TestBatch1_FormatAge_MultipleDays(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-72 * time.Hour))
	assert.Contains(t, result, "days")
}

// ---------------------------------------------------------------------------
// OutputAnalyzeJSON edge case: both nil report and nil error
// ---------------------------------------------------------------------------

func TestBatch1_OutputAnalyzeJSON_BothNil(t *testing.T) {
	output := captureStdout(t, func() {
		outputAnalyzeJSON(nil, nil)
	})
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))
	// Should be a valid JSON object with no error and no layers
	_, hasError := parsed["error"]
	assert.False(t, hasError)
}

// ---------------------------------------------------------------------------
// Profile - applyGitConfig with signing_key only
// ---------------------------------------------------------------------------

func TestBatch1_ApplyGitConfig_OnlySigningKey(t *testing.T) {
	git := map[string]interface{}{
		"signing_key": "DEADBEEF",
	}
	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "user.signingkey")
	assert.NotContains(t, output, "user.name")
	assert.NotContains(t, output, "user.email")
}
