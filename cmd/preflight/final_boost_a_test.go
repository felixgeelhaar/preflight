package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// sync_conflicts.go - relationString
// ---------------------------------------------------------------------------

func TestBoostA_RelationString_AllCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		relation sync.CausalRelation
		expected string
	}{
		{"equal", sync.Equal, "equal (in sync)"},
		{"before", sync.Before, "behind (pull needed)"},
		{"after", sync.After, "ahead (push needed)"},
		{"concurrent", sync.Concurrent, "concurrent (merge needed)"},
		{"unknown", sync.CausalRelation(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := relationString(tc.relation)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// sync_conflicts.go - printJSONOutput
// ---------------------------------------------------------------------------

func TestBoostA_PrintJSONOutput_Populated(t *testing.T) {
	t.Parallel()

	output := ConflictsOutputJSON{
		Relation:       "concurrent (merge needed)",
		TotalConflicts: 2,
		AutoResolvable: 1,
		ManualConflicts: []ConflictJSON{
			{
				PackageKey:    "brew:ripgrep",
				Type:          "both_modified",
				LocalVersion:  "14.0.0",
				RemoteVersion: "14.1.0",
				Resolvable:    false,
			},
		},
		NeedsMerge: true,
	}

	out := captureStdout(t, func() {
		err := printJSONOutput(output)
		require.NoError(t, err)
	})

	var parsed ConflictsOutputJSON
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, 2, parsed.TotalConflicts)
	assert.Equal(t, 1, parsed.AutoResolvable)
	assert.True(t, parsed.NeedsMerge)
	assert.Len(t, parsed.ManualConflicts, 1)
	assert.Equal(t, "brew:ripgrep", parsed.ManualConflicts[0].PackageKey)
}

func TestBoostA_PrintJSONOutput_Empty(t *testing.T) {
	t.Parallel()

	output := ConflictsOutputJSON{
		Relation:        "equal (in sync)",
		TotalConflicts:  0,
		AutoResolvable:  0,
		ManualConflicts: []ConflictJSON{},
		NeedsMerge:      false,
	}

	out := captureStdout(t, func() {
		err := printJSONOutput(output)
		require.NoError(t, err)
	})

	assert.Contains(t, out, `"total_conflicts": 0`)
	assert.Contains(t, out, `"needs_merge": false`)
}

// ---------------------------------------------------------------------------
// rollback.go - formatAge
// ---------------------------------------------------------------------------

func TestBoostA_FormatAge_AllRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ago      time.Duration
		expected string
	}{
		{"just_now", 5 * time.Second, "just now"},
		{"1_min", 1 * time.Minute, "1 min ago"},
		{"5_mins", 5 * time.Minute, "5 mins ago"},
		{"1_hour", 1 * time.Hour, "1 hour ago"},
		{"3_hours", 3 * time.Hour, "3 hours ago"},
		{"1_day", 25 * time.Hour, "1 day ago"},
		{"3_days", 3 * 24 * time.Hour, "3 days ago"},
		{"1_week", 8 * 24 * time.Hour, "1 week ago"},
		{"3_weeks", 22 * 24 * time.Hour, "3 weeks ago"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := formatAge(time.Now().Add(-tc.ago))
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// rollback.go - listSnapshots
// ---------------------------------------------------------------------------

func TestBoostA_ListSnapshots(t *testing.T) {
	t.Parallel()

	sets := []snapshot.Set{
		{
			ID:        "abcdef1234567890",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Reason:    "pre-apply backup",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.gitconfig"},
				{Path: "/home/user/.zshrc"},
			},
		},
		{
			ID:        "1234567890abcdef",
			CreatedAt: time.Now().Add(-48 * time.Hour),
			Reason:    "",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.vimrc"},
			},
		},
	}

	out := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Available Snapshots")
	assert.Contains(t, out, "abcdef12")
	assert.Contains(t, out, "12345678")
	assert.Contains(t, out, "2 files")
	assert.Contains(t, out, "1 files")
	assert.Contains(t, out, "pre-apply backup")
	assert.Contains(t, out, "preflight rollback --to")
	assert.Contains(t, out, "preflight rollback --latest")
}

// ---------------------------------------------------------------------------
// discover.go - getPatternIcon
// ---------------------------------------------------------------------------

func TestBoostA_GetPatternIcon_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ptype    discover.PatternType
		expected string
	}{
		{"shell", discover.PatternTypeShell, "🐚"},
		{"editor", discover.PatternTypeEditor, "📝"},
		{"git", discover.PatternTypeGit, "📦"},
		{"ssh", discover.PatternTypeSSH, "🔐"},
		{"tmux", discover.PatternTypeTmux, "🖥️"},
		{"package_manager", discover.PatternTypePackageManager, "📦"},
		{"unknown", discover.PatternType("unknown_type"), "•"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getPatternIcon(tc.ptype)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// compliance.go - outputComplianceText with expiring overrides
// ---------------------------------------------------------------------------

func TestBoostA_OutputComplianceText_WithExpiringOverrides(t *testing.T) {
	t.Parallel()

	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ComplianceScore: 100,
		},
		Overrides: []policy.OverrideDetail{
			{
				Pattern:         "require_gpg",
				Justification:   "GPG setup pending",
				ExpiresAt:       time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
				DaysUntilExpiry: 3,
			},
		},
	}

	out := captureStdout(t, func() {
		outputComplianceText(report)
	})

	assert.Contains(t, out, "expiring within 7 days")
}

func TestBoostA_OutputComplianceText_NoOverrides(t *testing.T) {
	t.Parallel()

	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     3,
			PassedChecks:    3,
			ComplianceScore: 100,
		},
	}

	out := captureStdout(t, func() {
		outputComplianceText(report)
	})

	assert.NotContains(t, out, "expiring")
}

// ---------------------------------------------------------------------------
// compliance.go - outputComplianceError
// ---------------------------------------------------------------------------

func TestBoostA_OutputComplianceError(t *testing.T) {
	t.Parallel()

	out := captureStdout(t, func() {
		outputComplianceError(assert.AnError)
	})

	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, assert.AnError.Error(), parsed["error"])
}

// ---------------------------------------------------------------------------
// sync_conflicts.go - ConflictJSON and ConflictsOutputJSON structs
// ---------------------------------------------------------------------------

func TestBoostA_ConflictJSON_Serialization(t *testing.T) {
	t.Parallel()

	c := ConflictJSON{
		PackageKey:    "brew:fzf",
		Type:          "version_conflict",
		LocalVersion:  "0.44.0",
		RemoteVersion: "0.45.0",
		Resolvable:    true,
	}

	data, err := json.Marshal(c)
	require.NoError(t, err)

	var parsed ConflictJSON
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "brew:fzf", parsed.PackageKey)
	assert.True(t, parsed.Resolvable)
}
