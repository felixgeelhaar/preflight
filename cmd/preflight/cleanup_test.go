package main

import (
	"encoding/json"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupCommand_Flags(t *testing.T) {
	t.Parallel()

	// Verify command exists and has expected flags
	assert.NotNil(t, cleanupCmd)
	assert.Equal(t, "cleanup", cleanupCmd.Use)

	// Check flags exist
	flags := cleanupCmd.Flags()
	assert.NotNil(t, flags.Lookup("remove"))
	assert.NotNil(t, flags.Lookup("autoremove"))
	assert.NotNil(t, flags.Lookup("all"))
	assert.NotNil(t, flags.Lookup("dry-run"))
	assert.NotNil(t, flags.Lookup("json"))
	assert.NotNil(t, flags.Lookup("quiet"))
	assert.NotNil(t, flags.Lookup("ignore"))
	assert.NotNil(t, flags.Lookup("keep"))
	assert.NotNil(t, flags.Lookup("no-orphans"))
	assert.NotNil(t, flags.Lookup("no-overlaps"))
}

func TestFormatCategory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"security_scanners", "Security Scanners"},
		{"node_package_managers", "Node Package Managers"},
		{"container_runtimes", "Container Runtimes"},
		{"single", "Single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := formatCategory(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedundancySummary_Counts(t *testing.T) {
	t.Parallel()

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Type: security.RedundancyDuplicate, Remove: []string{"go@1.24"}},
			{Type: security.RedundancyDuplicate, Remove: []string{"python@3.12"}},
			{Type: security.RedundancyOverlap, Remove: []string{"trivy"}},
			{Type: security.RedundancyOrphan, Remove: []string{"libpng", "zlib"}},
		},
	}

	summary := result.Summary()

	assert.Equal(t, 4, summary.Total)
	assert.Equal(t, 2, summary.Duplicates)
	assert.Equal(t, 1, summary.Overlaps)
	assert.Equal(t, 1, summary.Orphans)
	assert.Equal(t, 5, summary.Removable)
}

func TestRedundancies_CollectRemovable(t *testing.T) {
	t.Parallel()

	redundancies := security.Redundancies{
		{Type: security.RedundancyDuplicate, Remove: []string{"go@1.24"}},
		{Type: security.RedundancyOverlap, Remove: []string{"trivy"}},
		{Type: security.RedundancyOrphan, Remove: []string{"libpng", "zlib"}},
	}

	// Collect all removable packages
	toRemove := make([]string, 0, len(redundancies))
	for _, red := range redundancies {
		toRemove = append(toRemove, red.Remove...)
	}

	assert.Len(t, toRemove, 4)
	assert.Contains(t, toRemove, "go@1.24")
	assert.Contains(t, toRemove, "trivy")
	assert.Contains(t, toRemove, "libpng")
	assert.Contains(t, toRemove, "zlib")
}

func TestPrintRedundancySummaryBar_EmptyCounts(t *testing.T) {
	t.Helper() // Not parallel: writes to os.Stdout, races with captureStdout in validate_test.go

	summary := security.RedundancySummary{
		Total:      0,
		Duplicates: 0,
		Overlaps:   0,
		Orphans:    0,
		Removable:  0,
	}

	stdoutMu.Lock()
	printRedundancySummaryBar(summary)
	stdoutMu.Unlock()
}

func TestPrintRedundancySummaryBar_AllTypes(t *testing.T) {
	t.Helper() // Not parallel: writes to os.Stdout, races with captureStdout in validate_test.go

	summary := security.RedundancySummary{
		Total:      5,
		Duplicates: 2,
		Overlaps:   1,
		Orphans:    2,
		Removable:  8,
	}

	stdoutMu.Lock()
	printRedundancySummaryBar(summary)
	stdoutMu.Unlock()
}

func TestOutputCleanupJSON_WithResult(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Category:       "go",
				Recommendation: "Keep go (tracks latest)",
				Keep:           []string{"go"},
				Remove:         []string{"go@1.24"},
			},
		},
	}

	output := captureStdout(t, func() {
		outputCleanupJSON(result, nil, nil)
	})

	var parsed security.CleanupResultJSON
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Empty(t, parsed.Error)
	assert.Nil(t, parsed.Cleanup)
	require.NotNil(t, parsed.Summary)
	assert.Equal(t, 1, parsed.Summary.Total)
	assert.Equal(t, 1, parsed.Summary.Duplicates)
	assert.Equal(t, 1, parsed.Summary.Removable)
	require.Len(t, parsed.Redundancies, 1)
	assert.Equal(t, security.RedundancyDuplicate, parsed.Redundancies[0].Type)
	assert.Equal(t, []string{"go", "go@1.24"}, parsed.Redundancies[0].Packages)
}

func TestOutputCleanupJSON_WithCleanup(t *testing.T) {
	cleanup := &security.CleanupResult{
		Removed: []string{"go@1.24", "python@3.12"},
		DryRun:  true,
	}

	output := captureStdout(t, func() {
		outputCleanupJSON(nil, cleanup, nil)
	})

	var parsed security.CleanupResultJSON
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Empty(t, parsed.Error)
	assert.Nil(t, parsed.Summary)
	assert.Empty(t, parsed.Redundancies)
	require.NotNil(t, parsed.Cleanup)
	assert.True(t, parsed.Cleanup.DryRun)
	assert.Equal(t, []string{"go@1.24", "python@3.12"}, parsed.Cleanup.Removed)
}

func TestOutputCleanupJSON_WithError(t *testing.T) {
	output := captureStdout(t, func() {
		outputCleanupJSON(nil, nil, assert.AnError)
	})

	var parsed security.CleanupResultJSON
	err := json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)

	assert.Equal(t, assert.AnError.Error(), parsed.Error)
	assert.Nil(t, parsed.Cleanup)
	assert.Nil(t, parsed.Summary)
	assert.Empty(t, parsed.Redundancies)
}

func TestOutputCleanupText_NoRedundancies(t *testing.T) {
	result := &security.RedundancyResult{
		Checker:      "brew",
		Redundancies: security.Redundancies{},
	}

	output := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, output, "No redundancies detected")
	assert.Contains(t, output, "brew")
}

func TestOutputCleanupText_AllTypes(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Category:       "go",
				Recommendation: "Keep go (tracks latest)",
				Keep:           []string{"go"},
				Remove:         []string{"go@1.24"},
			},
			{
				Type:           security.RedundancyOverlap,
				Packages:       []string{"grype", "trivy"},
				Category:       "security_scanners",
				Recommendation: "Vulnerability scanners - consider keeping only one",
				Keep:           []string{"grype"},
				Remove:         []string{"trivy"},
			},
			{
				Type:           security.RedundancyOrphan,
				Packages:       []string{"libpng", "zlib"},
				Category:       "orphaned_dependencies",
				Recommendation: "2 orphaned dependencies can be removed",
				Action:         "preflight cleanup --autoremove",
				Remove:         []string{"libpng", "zlib"},
			},
		},
	}

	output := captureStdout(t, func() {
		outputCleanupText(result, false)
	})

	assert.Contains(t, output, "Redundancy Analysis (brew)")
	assert.Contains(t, output, "3 redundancies found")
	// Duplicates section
	assert.Contains(t, output, "Version Duplicates")
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "go@1.24")
	// Overlaps section
	assert.Contains(t, output, "Overlapping Tools")
	assert.Contains(t, output, "Security Scanners")
	assert.Contains(t, output, "grype")
	assert.Contains(t, output, "trivy")
	// Orphans section
	assert.Contains(t, output, "Orphaned Dependencies")
	assert.Contains(t, output, "libpng")
	assert.Contains(t, output, "zlib")
	// Actions section
	assert.Contains(t, output, "Actions")
	assert.Contains(t, output, "preflight cleanup --autoremove")
	assert.Contains(t, output, "preflight cleanup --all")
}

func TestOutputCleanupText_Quiet(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{
				Type:           security.RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Category:       "go",
				Recommendation: "Keep go (tracks latest)",
				Keep:           []string{"go"},
				Remove:         []string{"go@1.24"},
			},
		},
	}

	output := captureStdout(t, func() {
		outputCleanupText(result, true)
	})

	assert.Contains(t, output, "1 redundancies found")
	assert.Contains(t, output, "preflight cleanup --all")
	// Quiet mode should not show detailed tables
	assert.NotContains(t, output, "Version Duplicates")
	assert.NotContains(t, output, "Actions:")
}

func TestPrintRedundancyTable(t *testing.T) {
	redundancies := security.Redundancies{
		{
			Type:           security.RedundancyDuplicate,
			Packages:       []string{"go", "go@1.24"},
			Category:       "go",
			Recommendation: "Keep go (tracks latest)",
			Keep:           []string{"go"},
			Remove:         []string{"go@1.24"},
		},
		{
			Type:           security.RedundancyDuplicate,
			Packages:       []string{"python", "python@3.12"},
			Category:       "python",
			Recommendation: "Keep python (tracks latest)",
			Keep:           []string{"python"},
			Remove:         []string{"python@3.12"},
		},
	}

	output := captureStdout(t, func() {
		printRedundancyTable(redundancies)
	})

	// Verify package groups are listed
	assert.Contains(t, output, "go + go@1.24")
	assert.Contains(t, output, "python + python@3.12")
	// Verify recommendations
	assert.Contains(t, output, "Keep go (tracks latest)")
	assert.Contains(t, output, "Keep python (tracks latest)")
	// Verify remove suggestions
	assert.Contains(t, output, "Remove: go@1.24")
	assert.Contains(t, output, "Remove: python@3.12")
}

func TestPrintOverlapTable(t *testing.T) {
	redundancies := security.Redundancies{
		{
			Type:           security.RedundancyOverlap,
			Packages:       []string{"grype", "trivy"},
			Category:       "security_scanners",
			Recommendation: "Vulnerability scanners - consider keeping only one",
			Keep:           []string{"grype"},
			Remove:         []string{"trivy"},
		},
		{
			Type:           security.RedundancyOverlap,
			Packages:       []string{"npm", "yarn", "pnpm"},
			Category:       "node_package_managers",
			Recommendation: "Node.js package managers - consider keeping only one",
			Keep:           []string{"npm"},
			Remove:         []string{"yarn", "pnpm"},
		},
	}

	output := captureStdout(t, func() {
		printOverlapTable(redundancies)
	})

	// Verify categories are formatted
	assert.Contains(t, output, "Security Scanners")
	assert.Contains(t, output, "Node Package Managers")
	// Verify packages listed
	assert.Contains(t, output, "grype, trivy")
	assert.Contains(t, output, "npm, yarn, pnpm")
	// Verify recommendations
	assert.Contains(t, output, "consider keeping only one")
	// Verify keep/remove
	assert.Contains(t, output, "Keep: grype")
	assert.Contains(t, output, "Remove: trivy")
	assert.Contains(t, output, "Keep: npm")
	assert.Contains(t, output, "Remove: yarn, pnpm")
}
