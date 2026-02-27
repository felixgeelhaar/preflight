package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		{"1_hour", "1h", time.Hour, false},
		{"24_hours", "24h", 24 * time.Hour, false},
		{"7_days", "7d", 7 * 24 * time.Hour, false},
		{"2_weeks", "2w", 2 * 7 * 24 * time.Hour, false},
		{"1_month", "1m", 30 * 24 * time.Hour, false},
		{"3_months", "3m", 3 * 30 * 24 * time.Hour, false},
		{"leading_trailing_spaces", "  5d  ", 5 * 24 * time.Hour, false},
		{"uppercase_H", "6H", 6 * time.Hour, false},
		{"uppercase_D", "3D", 3 * 24 * time.Hour, false},
		{"uppercase_W", "1W", 7 * 24 * time.Hour, false},
		{"uppercase_M", "2M", 2 * 30 * 24 * time.Hour, false},
		{"single_char_invalid", "x", 0, true},
		{"empty_string", "", 0, true},
		{"only_unit_no_number", "d", 0, true},
		{"unknown_unit_y", "5y", 0, true},
		{"unknown_unit_s", "30s", 0, true},
		{"non_numeric_value", "abcd", 0, true},
		{"float_value_truncates", "1.5h", time.Hour, false},
		{"negative_value_accepted", "-3d", -3 * 24 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseDuration(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatStatus_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"success_check", "success", "✓ success"},
		{"failed_cross", "failed", "✗ failed"},
		{"partial_tilde", "partial", "~ partial"},
		{"unknown_passthrough", "unknown", "unknown"},
		{"empty_passthrough", "", ""},
		{"arbitrary_value", "running", "running"},
		{"cancelled_value", "cancelled", "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHistoryAge_AllRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		age      time.Duration
		contains string
	}{
		{"just_now", 10 * time.Second, "just now"},
		{"30_seconds", 30 * time.Second, "just now"},
		{"5_minutes", 5 * time.Minute, "5m ago"},
		{"45_minutes", 45 * time.Minute, "45m ago"},
		{"2_hours", 2 * time.Hour, "2h ago"},
		{"23_hours", 23 * time.Hour, "23h ago"},
		{"3_days", 3 * 24 * time.Hour, "3d ago"},
		{"6_days", 6 * 24 * time.Hour, "6d ago"},
		{"2_weeks", 14 * 24 * time.Hour, "2w ago"},
		{"4_weeks", 28 * 24 * time.Hour, "4w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := time.Now().Add(-tt.age)
			result := formatHistoryAge(ts)
			assert.Equal(t, tt.contains, result)
		})
	}
}

func TestFormatHistoryAge_OldDate_UsesMonthFormat(t *testing.T) {
	t.Parallel()

	// 60 days ago should use "Jan 2" style format
	ts := time.Now().Add(-60 * 24 * time.Hour)
	result := formatHistoryAge(ts)
	expected := ts.Format("Jan 2")
	assert.Equal(t, expected, result)
}

func TestOutputHistoryText_TableFormat(t *testing.T) {
	// Not parallel - writes to stdout
	entries := []HistoryEntry{
		{
			ID:        "entry-1",
			Timestamp: time.Now().Add(-5 * time.Minute),
			Command:   "apply",
			Target:    "work",
			Status:    "success",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "ripgrep"},
				{Provider: "brew", Action: "install", Item: "fd"},
			},
		},
		{
			ID:        "entry-2",
			Timestamp: time.Now().Add(-2 * time.Hour),
			Command:   "doctor --fix",
			Target:    "personal",
			Status:    "failed",
			Error:     "connection timeout",
			Changes: []Change{
				{Provider: "files", Action: "create", Item: "~/.zshrc"},
			},
		},
	}

	// Ensure non-verbose mode
	savedVerbose := historyVerbose
	historyVerbose = false
	defer func() { historyVerbose = savedVerbose }()

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	// Verify table headers
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "COMMAND")
	assert.Contains(t, output, "TARGET")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "CHANGES")

	// Verify entry data appears
	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "✓ success")
	assert.Contains(t, output, "doctor --fix")
	assert.Contains(t, output, "personal")
	assert.Contains(t, output, "✗ failed")

	// Verify footer
	assert.Contains(t, output, "Showing 2 entries")
}

func TestOutputHistoryText_VerboseFormat(t *testing.T) {
	// Not parallel - writes to stdout and modifies global
	savedVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = savedVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "verbose-entry-1",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Command:   "apply",
			Target:    "work",
			Status:    "success",
			Duration:  "2.3s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "ripgrep"},
				{Provider: "files", Action: "create", Item: "~/.gitconfig"},
			},
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	// Verbose format includes labeled sections
	assert.Contains(t, output, "verbose-entry-1")
	assert.Contains(t, output, "Time:")
	assert.Contains(t, output, "Command:  apply")
	assert.Contains(t, output, "Target:   work")
	assert.Contains(t, output, "Status:   ✓ success")
	assert.Contains(t, output, "Duration: 2.3s")
	assert.Contains(t, output, "Changes:")
	assert.Contains(t, output, "[brew] install: ripgrep")
	assert.Contains(t, output, "[files] create: ~/.gitconfig")

	// Footer still present
	assert.Contains(t, output, "Showing 1 entries")
}

func TestOutputHistoryText_VerboseWithErrorField(t *testing.T) {
	// Not parallel - writes to stdout and modifies global
	savedVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = savedVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "error-entry",
			Timestamp: time.Now().Add(-1 * time.Minute),
			Command:   "apply",
			Status:    "failed",
			Error:     "package not found: nonexistent",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "Error:    package not found: nonexistent")
	assert.Contains(t, output, "✗ failed")
}

func TestOutputHistoryText_VerboseNoTargetOmitted(t *testing.T) {
	// Not parallel - writes to stdout and modifies global
	savedVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = savedVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "no-target-entry",
			Timestamp: time.Now(),
			Command:   "doctor",
			Status:    "success",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	// Target line should not appear when target is empty
	assert.NotContains(t, output, "Target:")
	// Duration should not appear either
	assert.NotContains(t, output, "Duration:")
}

func TestOutputHistoryText_VerboseMultipleEntriesSeparator(t *testing.T) {
	// Not parallel - writes to stdout and modifies global
	savedVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = savedVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "sep-entry-1",
			Timestamp: time.Now(),
			Command:   "apply",
			Status:    "success",
		},
		{
			ID:        "sep-entry-2",
			Timestamp: time.Now(),
			Command:   "rollback",
			Status:    "success",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "sep-entry-1")
	assert.Contains(t, output, "sep-entry-2")
	assert.Contains(t, output, "Showing 2 entries")
}

func TestOutputHistoryText_SingleEntry(t *testing.T) {
	// Not parallel - writes to stdout
	savedVerbose := historyVerbose
	historyVerbose = false
	defer func() { historyVerbose = savedVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "single-entry",
			Timestamp: time.Now(),
			Command:   "apply",
			Target:    "default",
			Status:    "partial",
			Changes:   []Change{{Provider: "apt", Action: "install", Item: "curl"}},
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "~ partial")
	assert.Contains(t, output, "Showing 1 entries")
}

func TestSaveHistoryEntry_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "roundtrip-test-001",
		Timestamp: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "1.5s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "ripgrep", Details: "version 14.0"},
			{Provider: "files", Action: "create", Item: "~/.zshrc"},
		},
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Read back the file
	histDir := filepath.Join(tmpDir, ".preflight", "history")
	data, err := os.ReadFile(filepath.Join(histDir, "roundtrip-test-001.json"))
	require.NoError(t, err)

	var loaded HistoryEntry
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, entry.ID, loaded.ID)
	assert.Equal(t, entry.Command, loaded.Command)
	assert.Equal(t, entry.Target, loaded.Target)
	assert.Equal(t, entry.Status, loaded.Status)
	assert.Equal(t, entry.Duration, loaded.Duration)
	assert.Len(t, loaded.Changes, 2)
	assert.Equal(t, "brew", loaded.Changes[0].Provider)
	assert.Equal(t, "version 14.0", loaded.Changes[0].Details)
}

func TestSaveHistoryEntry_AutoGeneratesIDAndTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		Command: "doctor --fix",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify a file was created in the history dir
	histDir := filepath.Join(tmpDir, ".preflight", "history")
	files, err := os.ReadDir(histDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0].Name(), ".json")
}

func TestLoadHistory_RoundTripWithSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save two entries
	entry1 := HistoryEntry{
		ID:        "load-test-001",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Command:   "apply",
		Status:    "success",
	}
	entry2 := HistoryEntry{
		ID:        "load-test-002",
		Timestamp: time.Now(),
		Command:   "rollback",
		Status:    "failed",
		Error:     "rollback failed",
	}

	require.NoError(t, SaveHistoryEntry(entry1))
	require.NoError(t, SaveHistoryEntry(entry2))

	entries, err := loadHistory()
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify both entries are present
	ids := make(map[string]bool)
	for _, e := range entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["load-test-001"])
	assert.True(t, ids["load-test-002"])
}

func TestRunHistoryClear_RemovesHistory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save an entry first
	entry := HistoryEntry{
		ID:      "clear-test-001",
		Command: "apply",
		Status:  "success",
	}
	require.NoError(t, SaveHistoryEntry(entry))

	// Verify history dir exists
	histDir := filepath.Join(tmpDir, ".preflight", "history")
	_, err := os.Stat(histDir)
	require.NoError(t, err)

	// Clear history
	output := captureStdout(t, func() {
		err = runHistoryClear(nil, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, output, "History cleared.")

	// Verify directory is removed
	_, err = os.Stat(histDir)
	assert.True(t, os.IsNotExist(err))
}

func TestRunHistoryClear_NoHistoryDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Clearing when no history dir exists should not error
	output := captureStdout(t, func() {
		err := runHistoryClear(nil, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "History cleared.")
}

func TestGetHistoryDir_UsesHome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getHistoryDir()
	assert.Equal(t, filepath.Join(tmpDir, ".preflight", "history"), dir)
}

func TestHistoryEntry_StructFields(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	entry := HistoryEntry{
		ID:        "struct-test",
		Timestamp: ts,
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "3.5s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "git", Details: "v2.40"},
		},
		Error: "",
	}

	assert.Equal(t, "struct-test", entry.ID)
	assert.Equal(t, ts, entry.Timestamp)
	assert.Equal(t, "apply", entry.Command)
	assert.Equal(t, "work", entry.Target)
	assert.Equal(t, "success", entry.Status)
	assert.Equal(t, "3.5s", entry.Duration)
	assert.Len(t, entry.Changes, 1)
	assert.Empty(t, entry.Error)
}

func TestChange_StructFields(t *testing.T) {
	t.Parallel()

	c := Change{
		Provider: "apt",
		Action:   "remove",
		Item:     "vim",
		Details:  "version 8.2",
	}

	assert.Equal(t, "apt", c.Provider)
	assert.Equal(t, "remove", c.Action)
	assert.Equal(t, "vim", c.Item)
	assert.Equal(t, "version 8.2", c.Details)
}

func TestHistoryCmd_Registered(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "history" {
			found = true
			break
		}
	}
	assert.True(t, found, "history command should be registered on root")
}

func TestHistoryCmd_FlagDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{"limit", "limit", "20"},
		{"since", "since", ""},
		{"json", "json", "false"},
		{"provider", "provider", ""},
		{"verbose", "verbose", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := historyCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %s should exist", tt.flag)
			assert.Equal(t, tt.defValue, f.DefValue)
		})
	}
}

func TestHistoryCmd_ClearSubcommand(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range historyCmd.Commands() {
		if cmd.Name() == "clear" {
			found = true
			assert.Equal(t, "Clear history", cmd.Short)
			break
		}
	}
	assert.True(t, found, "clear subcommand should exist on history")
}

func TestHistoryEntry_JSONSerialization(t *testing.T) {
	t.Parallel()

	entry := HistoryEntry{
		ID:        "json-test",
		Timestamp: time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC),
		Command:   "apply",
		Target:    "personal",
		Status:    "success",
		Duration:  "1.2s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "jq"},
		},
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded HistoryEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.Command, decoded.Command)
	assert.Equal(t, entry.Target, decoded.Target)
	assert.Equal(t, entry.Status, decoded.Status)
	assert.Equal(t, entry.Duration, decoded.Duration)
	assert.Len(t, decoded.Changes, 1)
}

func TestHistoryEntry_JSONOmitsEmpty(t *testing.T) {
	t.Parallel()

	entry := HistoryEntry{
		ID:      "omit-test",
		Command: "doctor",
		Status:  "success",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	// These fields should be omitted when empty
	assert.NotContains(t, string(data), `"target"`)
	assert.NotContains(t, string(data), `"duration"`)
	assert.NotContains(t, string(data), `"changes"`)
	assert.NotContains(t, string(data), `"error"`)
}
