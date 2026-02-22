package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
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
