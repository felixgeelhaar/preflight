package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// batch2CaptureStdout captures stdout during function execution.
// Delegates to the shared captureStdout defined in validate_test.go.
func batch2CaptureStdout(t *testing.T, f func()) string {
	t.Helper()
	return captureStdout(t, f)
}

// ---------------------------------------------------------------------------
// doctor.go -- printDoctorQuiet
// ---------------------------------------------------------------------------

func TestBatch2_PrintDoctorQuiet_EmptyReport(t *testing.T) {
	report := &app.DoctorReport{}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "No issues found")
	assert.Contains(t, output, "Doctor Report")
	assert.Contains(t, output, "=============")
}

func TestBatch2_PrintDoctorQuiet_SingleErrorIssue(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityError,
				Message:  "package git not installed",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Found 1 issue(s)")
	assert.Contains(t, output, "[error]")
	assert.Contains(t, output, "package git not installed")
	// Error issues use the cross marker
	assert.Contains(t, output, "\u2717") // ✗
}

func TestBatch2_PrintDoctorQuiet_WarningIssue(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityWarning,
				Message:  "config drift detected",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "[warning]")
	assert.Contains(t, output, "config drift detected")
	// Warning issues use the exclamation marker
	assert.Contains(t, output, "!")
}

func TestBatch2_PrintDoctorQuiet_IssueWithProvider(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityError,
				Message:  "missing package",
				Provider: "brew",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Provider: brew")
}

func TestBatch2_PrintDoctorQuiet_IssueWithExpectedActual(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityWarning,
				Message:  "version mismatch",
				Expected: "1.0.0",
				Actual:   "0.9.0",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Expected: 1.0.0")
	assert.Contains(t, output, "Actual: 0.9.0")
}

func TestBatch2_PrintDoctorQuiet_IssueWithFixCommand(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityWarning,
				Message:    "package outdated",
				FixCommand: "brew upgrade git",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Fix: brew upgrade git")
}

func TestBatch2_PrintDoctorQuiet_FixableIssues(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityWarning,
				Message:    "fixable issue",
				Fixable:    true,
				FixCommand: "preflight apply",
			},
			{
				Severity: app.SeverityError,
				Message:  "not fixable",
				Fixable:  false,
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Found 2 issue(s)")
	assert.Contains(t, output, "1 issue(s) can be auto-fixed")
	assert.Contains(t, output, "preflight doctor --fix")
}

func TestBatch2_PrintDoctorQuiet_WithPatches(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityWarning,
				Message:  "drift detected",
			},
		},
		SuggestedPatches: []app.ConfigPatch{
			app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpAdd, nil, "git", "doctor"),
			app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpModify, "old", "new", "doctor"),
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "2 config patches suggested")
	assert.Contains(t, output, "preflight doctor --update-config")
}

func TestBatch2_PrintDoctorQuiet_MultipleIssuesMixed(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityError,
				Message:    "error one",
				Provider:   "brew",
				Expected:   "installed",
				Actual:     "missing",
				FixCommand: "brew install git",
				Fixable:    true,
			},
			{
				Severity: app.SeverityWarning,
				Message:  "warning one",
			},
			{
				Severity: app.SeverityError,
				Message:  "error two",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "Found 3 issue(s)")
	assert.Contains(t, output, "error one")
	assert.Contains(t, output, "warning one")
	assert.Contains(t, output, "error two")
	assert.Contains(t, output, "Provider: brew")
	assert.Contains(t, output, "Expected: installed")
	assert.Contains(t, output, "Actual: missing")
	assert.Contains(t, output, "Fix: brew install git")
}

func TestBatch2_PrintDoctorQuiet_NoExpectedWithoutActual(t *testing.T) {
	// When Expected is set but Actual is empty, the Expected/Actual section
	// should not be printed
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityWarning,
				Message:  "only expected set",
				Expected: "1.0.0",
				Actual:   "",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.NotContains(t, output, "Expected:")
	assert.NotContains(t, output, "Actual:")
}

func TestBatch2_PrintDoctorQuiet_EmptyProviderNotPrinted(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityWarning,
				Message:  "no provider",
				Provider: "",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.NotContains(t, output, "Provider:")
}

func TestBatch2_PrintDoctorQuiet_EmptyFixCommandNotPrinted(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityWarning,
				Message:    "no fix",
				FixCommand: "",
			},
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.NotContains(t, output, "Fix:")
}

func TestBatch2_PrintDoctorQuiet_PatchesWithoutFixable(t *testing.T) {
	// Patches present but no fixable issues - should show patches but not auto-fix
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity: app.SeverityError,
				Message:  "not fixable",
				Fixable:  false,
			},
		},
		SuggestedPatches: []app.ConfigPatch{
			app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpAdd, nil, "git", "doctor"),
		},
	}
	output := batch2CaptureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.NotContains(t, output, "can be auto-fixed")
	assert.Contains(t, output, "1 config patches suggested")
}

// ---------------------------------------------------------------------------
// catalog.go -- deriveCatalogName
// ---------------------------------------------------------------------------

func TestBatch2_DeriveCatalogName_URL(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("https://example.com/presets/my-catalog")
	assert.Equal(t, "my-catalog", result)
}

func TestBatch2_DeriveCatalogName_Path(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("/home/user/presets/my-catalog")
	assert.Equal(t, "my-catalog", result)
}

func TestBatch2_DeriveCatalogName_TrailingSlash(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("https://example.com/catalog-repo/")
	assert.Equal(t, "catalog-repo", result)
}

func TestBatch2_DeriveCatalogName_NoSlashes(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("simple-name")
	assert.Equal(t, "simple-name", result)
}

func TestBatch2_DeriveCatalogName_Empty(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("")
	// Empty string falls to the time-based default
	assert.True(t, strings.HasPrefix(result, "catalog-"))
}

func TestBatch2_DeriveCatalogName_SingleSegment(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("standalone")
	assert.Equal(t, "standalone", result)
}

func TestBatch2_DeriveCatalogName_DeepPath(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("/a/b/c/d/e/my-preset")
	assert.Equal(t, "my-preset", result)
}

func TestBatch2_DeriveCatalogName_GitURL(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("https://github.com/user/preflight-catalog.git")
	// Should return "preflight-catalog.git" because it doesn't strip extensions
	assert.Equal(t, "preflight-catalog.git", result)
}

// ---------------------------------------------------------------------------
// catalog.go -- filterBySeverity
// ---------------------------------------------------------------------------

func TestBatch2_FilterBySeverity_Empty(t *testing.T) {
	t.Parallel()
	// filterBySeverity is already at 100% but we add one more test for completeness
	result := filterBySeverity(nil, catalog.AuditSeverityCritical)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// catalog.go -- verifyCatalogSignatures
// ---------------------------------------------------------------------------

func TestBatch2_VerifyCatalogSignatures_ReturnsNoSignature(t *testing.T) {
	t.Parallel()
	result := verifyCatalogSignatures(nil, nil)
	assert.False(t, result.hasSignature)
	assert.False(t, result.verified)
	assert.Empty(t, result.signer)
	assert.Empty(t, result.issuer)
	assert.NoError(t, result.err)
}

// ---------------------------------------------------------------------------
// catalog.go -- signatureVerifyResult fields
// ---------------------------------------------------------------------------

func TestBatch2_SignatureVerifyResult_AllFields(t *testing.T) {
	t.Parallel()
	result := signatureVerifyResult{
		hasSignature: true,
		verified:     true,
		signer:       "alice@example.com",
		issuer:       "https://github.com/login/oauth",
		err:          nil,
	}
	assert.True(t, result.hasSignature)
	assert.True(t, result.verified)
	assert.Equal(t, "alice@example.com", result.signer)
	assert.Equal(t, "https://github.com/login/oauth", result.issuer)
}

// ---------------------------------------------------------------------------
// tour.go -- runTour (25% coverage)
// ---------------------------------------------------------------------------

func TestBatch2_RunTour_ListFlag(t *testing.T) {
	// Save and restore global
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = true

	output := batch2CaptureStdout(t, func() {
		err := runTour(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Available tour topics")
	assert.Contains(t, output, "preflight tour")
}

func TestBatch2_RunTour_InvalidTopic(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = false

	err := runTour(nil, []string{"nonexistent-topic"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic: nonexistent-topic")
}

func TestBatch2_PrintTourTopics(t *testing.T) {
	output := batch2CaptureStdout(t, func() {
		printTourTopics()
	})
	assert.Contains(t, output, "Available tour topics")
	// Should contain at least one topic
	topics := tui.GetAllTopics()
	if len(topics) > 0 {
		assert.Contains(t, output, topics[0].ID)
	}
}

func TestBatch2_RunTour_ValidTopic_ButRequiresTUI(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = false

	topicIDs := tui.GetTopicIDs()
	if len(topicIDs) == 0 {
		t.Skip("No tour topics available")
	}

	// This will fail because it tries to start TUI, but it exercises the topic validation path
	err := runTour(nil, []string{topicIDs[0]})
	// It will either succeed or fail with TUI error, both are acceptable
	// The important thing is it passes the topic validation check
	if err != nil {
		assert.Contains(t, err.Error(), "tour failed")
	}
}

// ---------------------------------------------------------------------------
// tour.go -- tui.GetTopic / tui.GetTopicIDs
// ---------------------------------------------------------------------------

func TestBatch2_GetTopic_Valid(t *testing.T) {
	t.Parallel()
	topicIDs := tui.GetTopicIDs()
	if len(topicIDs) == 0 {
		t.Skip("No topics available")
	}
	topic, found := tui.GetTopic(topicIDs[0])
	assert.True(t, found)
	assert.Equal(t, topicIDs[0], topic.ID)
	assert.NotEmpty(t, topic.Description)
}

func TestBatch2_GetTopic_Invalid(t *testing.T) {
	t.Parallel()
	_, found := tui.GetTopic("this-topic-does-not-exist")
	assert.False(t, found)
}

func TestBatch2_GetTopicIDs_NonEmpty(t *testing.T) {
	t.Parallel()
	ids := tui.GetTopicIDs()
	assert.NotEmpty(t, ids)
}

func TestBatch2_GetAllTopics_NonEmpty(t *testing.T) {
	t.Parallel()
	topics := tui.GetAllTopics()
	assert.NotEmpty(t, topics)
	for _, topic := range topics {
		assert.NotEmpty(t, topic.ID)
	}
}

// ---------------------------------------------------------------------------
// secrets.go -- resolveAge, resolveSecret, findSecretRefs, setSecret
// ---------------------------------------------------------------------------

func TestBatch2_ResolveAge_MissingKeyFile(t *testing.T) {
	// resolveAge should fail when the .age file does not exist
	_, err := resolveAge("nonexistent-key-id-batch2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBatch2_ResolveSecret_EnvBackend(t *testing.T) {
	key := "BATCH2_TEST_SECRET_VALUE"
	t.Setenv(key, "mysecretvalue")
	val, err := resolveSecret("env", key)
	assert.NoError(t, err)
	assert.Equal(t, "mysecretvalue", val)
}

func TestBatch2_ResolveSecret_UnknownBackend(t *testing.T) {
	t.Parallel()
	_, err := resolveSecret("unknown-backend", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

func TestBatch2_ResolveSecret_EnvBackendEmpty(t *testing.T) {
	t.Parallel()
	val, err := resolveSecret("env", "BATCH2_NONEXISTENT_ENV_VAR")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestBatch2_SetSecret_EnvBackend(t *testing.T) {
	t.Parallel()
	err := setSecret("env", "test", "val")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment variables")
}

func TestBatch2_SetSecret_UnsupportedBackend(t *testing.T) {
	t.Parallel()
	err := setSecret("1password", "test", "val")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestBatch2_FindSecretRefs_WithRefs(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "preflight.yaml")
	content := `git:
  signing_key: "secret://1password/GitHub/signing-key"
ssh:
  passphrase: "secret://keychain/ssh-key"
env:
  token: "secret://env/GITHUB_TOKEN"
`
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

	refs, err := findSecretRefs(configFile)
	assert.NoError(t, err)
	assert.Len(t, refs, 3)

	assert.Equal(t, "1password", refs[0].Backend)
	assert.Equal(t, "GitHub/signing-key", refs[0].Key)

	assert.Equal(t, "keychain", refs[1].Backend)
	assert.Equal(t, "ssh-key", refs[1].Key)

	assert.Equal(t, "env", refs[2].Backend)
	assert.Equal(t, "GITHUB_TOKEN", refs[2].Key)
}

func TestBatch2_FindSecretRefs_NoRefs(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "preflight.yaml")
	content := `git:
  name: "Test User"
  email: "test@example.com"
`
	require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

	refs, err := findSecretRefs(configFile)
	assert.NoError(t, err)
	assert.Empty(t, refs)
}

func TestBatch2_FindSecretRefs_MissingFile(t *testing.T) {
	_, err := findSecretRefs("/nonexistent/path/batch2.yaml")
	assert.Error(t, err)
}

func TestBatch2_CheckBackendsCLI(t *testing.T) {
	t.Parallel()
	// These functions just check LookPath; they should not panic
	_ = check1PasswordCLI()
	_ = checkBitwardenCLI()
	_ = checkKeychain()
	_ = checkAgeCLI()
}

// ---------------------------------------------------------------------------
// history.go -- parseDuration, formatStatus, formatHistoryAge, outputHistoryText
// ---------------------------------------------------------------------------

func TestBatch2_ParseDuration_Hours(t *testing.T) {
	t.Parallel()
	d, err := parseDuration("24h")
	assert.NoError(t, err)
	assert.Equal(t, 24*time.Hour, d)
}

func TestBatch2_ParseDuration_Days(t *testing.T) {
	t.Parallel()
	d, err := parseDuration("7d")
	assert.NoError(t, err)
	assert.Equal(t, 7*24*time.Hour, d)
}

func TestBatch2_ParseDuration_Weeks(t *testing.T) {
	t.Parallel()
	d, err := parseDuration("2w")
	assert.NoError(t, err)
	assert.Equal(t, 14*24*time.Hour, d)
}

func TestBatch2_ParseDuration_Months(t *testing.T) {
	t.Parallel()
	d, err := parseDuration("3m")
	assert.NoError(t, err)
	assert.Equal(t, 90*24*time.Hour, d)
}

func TestBatch2_ParseDuration_Invalid(t *testing.T) {
	t.Parallel()
	_, err := parseDuration("x")
	assert.Error(t, err)
}

func TestBatch2_ParseDuration_UnknownUnit(t *testing.T) {
	t.Parallel()
	_, err := parseDuration("5z")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown unit")
}

func TestBatch2_FormatStatus_AllCases(t *testing.T) {
	t.Parallel()
	assert.Contains(t, formatStatus("success"), "success")
	assert.Contains(t, formatStatus("failed"), "failed")
	assert.Contains(t, formatStatus("partial"), "partial")
	assert.Equal(t, "other", formatStatus("other"))
}

func TestBatch2_FormatHistoryAge_JustNow(t *testing.T) {
	t.Parallel()
	result := formatHistoryAge(time.Now())
	assert.Equal(t, "just now", result)
}

func TestBatch2_FormatHistoryAge_Minutes(t *testing.T) {
	t.Parallel()
	result := formatHistoryAge(time.Now().Add(-5 * time.Minute))
	assert.Contains(t, result, "m ago")
}

func TestBatch2_FormatHistoryAge_Hours(t *testing.T) {
	t.Parallel()
	result := formatHistoryAge(time.Now().Add(-3 * time.Hour))
	assert.Contains(t, result, "h ago")
}

func TestBatch2_FormatHistoryAge_Days(t *testing.T) {
	t.Parallel()
	result := formatHistoryAge(time.Now().Add(-3 * 24 * time.Hour))
	assert.Contains(t, result, "d ago")
}

func TestBatch2_FormatHistoryAge_Weeks(t *testing.T) {
	t.Parallel()
	result := formatHistoryAge(time.Now().Add(-10 * 24 * time.Hour))
	assert.Contains(t, result, "w ago")
}

func TestBatch2_FormatHistoryAge_OldDate(t *testing.T) {
	t.Parallel()
	old := time.Now().Add(-60 * 24 * time.Hour)
	result := formatHistoryAge(old)
	// Should return a month-day format like "Jan 2"
	assert.NotContains(t, result, "ago")
}

func TestBatch2_OutputHistoryText_TableMode(t *testing.T) {
	old := historyVerbose
	defer func() { historyVerbose = old }()
	historyVerbose = false

	entries := []HistoryEntry{
		{
			ID:        "entry1",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Command:   "apply",
			Target:    "default",
			Status:    "success",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
		},
	}

	output := batch2CaptureStdout(t, func() {
		outputHistoryText(entries)
	})
	assert.Contains(t, output, "TIME")
	assert.Contains(t, output, "COMMAND")
	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "Showing 1 entries")
}

func TestBatch2_OutputHistoryText_VerboseMode(t *testing.T) {
	old := historyVerbose
	defer func() { historyVerbose = old }()
	historyVerbose = true

	entries := []HistoryEntry{
		{
			ID:        "entry1",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Command:   "apply",
			Target:    "default",
			Status:    "success",
			Duration:  "2.5s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
		},
	}

	output := batch2CaptureStdout(t, func() {
		outputHistoryText(entries)
	})
	assert.Contains(t, output, "entry1")
	assert.Contains(t, output, "Time:")
	assert.Contains(t, output, "Command:")
	assert.Contains(t, output, "Target:")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "Duration:")
	assert.Contains(t, output, "Changes:")
	assert.Contains(t, output, "[brew]")
}

func TestBatch2_OutputHistoryText_VerboseWithError(t *testing.T) {
	old := historyVerbose
	defer func() { historyVerbose = old }()
	historyVerbose = true

	entries := []HistoryEntry{
		{
			ID:        "entry-err",
			Timestamp: time.Now(),
			Command:   "apply",
			Status:    "failed",
			Error:     "something went wrong",
		},
	}

	output := batch2CaptureStdout(t, func() {
		outputHistoryText(entries)
	})
	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "something went wrong")
}

func TestBatch2_OutputHistoryText_VerboseMultipleEntries(t *testing.T) {
	old := historyVerbose
	defer func() { historyVerbose = old }()
	historyVerbose = true

	entries := []HistoryEntry{
		{
			ID:        "a1",
			Timestamp: time.Now(),
			Command:   "apply",
			Status:    "success",
		},
		{
			ID:        "a2",
			Timestamp: time.Now(),
			Command:   "doctor",
			Status:    "success",
		},
	}

	output := batch2CaptureStdout(t, func() {
		outputHistoryText(entries)
	})
	assert.Contains(t, output, "a1")
	assert.Contains(t, output, "a2")
	assert.Contains(t, output, "Showing 2 entries")
}

func TestBatch2_SaveHistoryEntry_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	histDir := filepath.Join(tmpDir, ".preflight", "history")

	// Patch the getHistoryDir to use tmpDir
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "test-batch2",
		Timestamp: time.Now().Truncate(time.Millisecond),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "1.5s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "git"},
		},
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify file was created
	files, err := os.ReadDir(histDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)

	// Read back and verify
	data, err := os.ReadFile(filepath.Join(histDir, files[0].Name()))
	require.NoError(t, err)

	var loaded HistoryEntry
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, entry.ID, loaded.ID)
	assert.Equal(t, entry.Command, loaded.Command)
	assert.Equal(t, entry.Target, loaded.Target)
	assert.Equal(t, entry.Status, loaded.Status)
}

func TestBatch2_SaveHistoryEntry_AutoID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		Command: "doctor",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Load and verify auto-generated fields
	entries, err := loadHistory()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.NotEmpty(t, entries[0].ID)
	assert.False(t, entries[0].Timestamp.IsZero())
}

func TestBatch2_LoadHistory_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestBatch2_LoadHistory_NonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmpDir, "nonexistent"))

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Nil(t, entries)
}

// ---------------------------------------------------------------------------
// history.go -- HistoryEntry JSON serialization
// ---------------------------------------------------------------------------

func TestBatch2_HistoryEntry_JSONSerialization(t *testing.T) {
	t.Parallel()
	entry := HistoryEntry{
		ID:        "test123",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
		Duration:  "3.2s",
		Changes: []Change{
			{Provider: "brew", Action: "install", Item: "git", Details: "v2.40"},
		},
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded HistoryEntry
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.Command, decoded.Command)
	assert.Equal(t, entry.Target, decoded.Target)
	assert.Equal(t, entry.Status, decoded.Status)
	assert.Equal(t, entry.Duration, decoded.Duration)
	assert.Len(t, decoded.Changes, 1)
	assert.Equal(t, "git", decoded.Changes[0].Item)
}

func TestBatch2_HistoryEntry_JSONOmitsEmptyFields(t *testing.T) {
	t.Parallel()
	entry := HistoryEntry{
		ID:        "minimal",
		Timestamp: time.Now(),
		Command:   "doctor",
		Status:    "success",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "target")
	assert.NotContains(t, jsonStr, "duration")
	assert.NotContains(t, jsonStr, "changes")
	assert.NotContains(t, jsonStr, "error")
}

// ---------------------------------------------------------------------------
// history.go -- Change struct
// ---------------------------------------------------------------------------

func TestBatch2_Change_StructFields(t *testing.T) {
	t.Parallel()
	change := Change{
		Provider: "brew",
		Action:   "install",
		Item:     "ripgrep",
		Details:  "version 14.0",
	}
	assert.Equal(t, "brew", change.Provider)
	assert.Equal(t, "install", change.Action)
	assert.Equal(t, "ripgrep", change.Item)
	assert.Equal(t, "version 14.0", change.Details)
}

// ---------------------------------------------------------------------------
// secrets.go -- SecretRef struct
// ---------------------------------------------------------------------------

func TestBatch2_SecretRef_Fields(t *testing.T) {
	t.Parallel()
	ref := SecretRef{
		Path:     "git.signing_key",
		Backend:  "1password",
		Key:      "GitHub/signing-key",
		Resolved: true,
	}
	assert.Equal(t, "git.signing_key", ref.Path)
	assert.Equal(t, "1password", ref.Backend)
	assert.Equal(t, "GitHub/signing-key", ref.Key)
	assert.True(t, ref.Resolved)
}

// ---------------------------------------------------------------------------
// plugin.go -- ValidationResult struct
// ---------------------------------------------------------------------------

func TestBatch2_ValidationResult_JSONSerialization(t *testing.T) {
	t.Parallel()
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{"missing description"},
		Plugin:   "my-plugin",
		Version:  "1.0.0",
		Path:     "/path/to/plugin",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded ValidationResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, result.Valid, decoded.Valid)
	assert.Equal(t, result.Plugin, decoded.Plugin)
	assert.Equal(t, result.Version, decoded.Version)
	assert.Len(t, decoded.Warnings, 1)
}

func TestBatch2_ValidationResult_Invalid(t *testing.T) {
	t.Parallel()
	result := ValidationResult{
		Valid:  false,
		Errors: []string{"missing apiVersion", "missing name"},
		Path:   "/tmp/bad-plugin",
	}
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 2)
}

func TestBatch2_ValidationResult_OmitsEmptyArrays(t *testing.T) {
	t.Parallel()
	result := ValidationResult{
		Valid: true,
		Path:  "/tmp/plugin",
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)
	jsonStr := string(data)
	// errors and warnings should be omitted when nil
	assert.NotContains(t, jsonStr, "errors")
	assert.NotContains(t, jsonStr, "warnings")
}

// ---------------------------------------------------------------------------
// compare.go -- compareConfigs, compareProviderConfig, outputCompareText
// ---------------------------------------------------------------------------

func TestBatch2_CompareConfigs_NoChanges(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"brew": map[string]interface{}{"formulae": "git"}}
	dest := map[string]interface{}{"brew": map[string]interface{}{"formulae": "git"}}
	diffs := compareConfigs(source, dest, nil)
	assert.Empty(t, diffs)
}

func TestBatch2_CompareConfigs_Added(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{}
	dest := map[string]interface{}{"brew": map[string]interface{}{"formulae": "git"}}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "added", diffs[0].Type)
	assert.Equal(t, "brew", diffs[0].Provider)
}

func TestBatch2_CompareConfigs_Removed(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"brew": map[string]interface{}{"formulae": "git"}}
	dest := map[string]interface{}{}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "removed", diffs[0].Type)
}

func TestBatch2_CompareConfigs_Changed(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{"brew": map[string]interface{}{"formulae": "git"}}
	dest := map[string]interface{}{"brew": map[string]interface{}{"formulae": "curl"}}
	diffs := compareConfigs(source, dest, nil)
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
}

func TestBatch2_CompareConfigs_WithProviderFilter(t *testing.T) {
	t.Parallel()
	source := map[string]interface{}{
		"brew": "value1",
		"git":  "value2",
	}
	dest := map[string]interface{}{
		"brew": "changed",
		"git":  "changed",
	}
	diffs := compareConfigs(source, dest, []string{"brew"})
	require.Len(t, diffs, 1)
	assert.Equal(t, "brew", diffs[0].Provider)
}

func TestBatch2_CompareProviderConfig_NonMaps(t *testing.T) {
	t.Parallel()
	diffs := compareProviderConfig("test", "value1", "value2")
	require.Len(t, diffs, 1)
	assert.Equal(t, "changed", diffs[0].Type)
}

func TestBatch2_CompareProviderConfig_EqualNonMaps(t *testing.T) {
	t.Parallel()
	diffs := compareProviderConfig("test", "same", "same")
	assert.Empty(t, diffs)
}

func TestBatch2_OutputCompareText_NoDiffs(t *testing.T) {
	output := batch2CaptureStdout(t, func() {
		outputCompareText("work", "personal", nil)
	})
	assert.Contains(t, output, "No differences")
}

func TestBatch2_OutputCompareText_WithDiffs(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Dest: "git"},
		{Provider: "git", Key: "name", Type: "removed", Source: "old"},
		{Provider: "ssh", Key: "keys", Type: "changed", Source: "a", Dest: "b"},
	}
	output := batch2CaptureStdout(t, func() {
		outputCompareText("work", "personal", diffs)
	})
	assert.Contains(t, output, "Comparing work")
	assert.Contains(t, output, "+ added")
	assert.Contains(t, output, "- removed")
	assert.Contains(t, output, "~ changed")
	assert.Contains(t, output, "3 difference(s)")
}

func TestBatch2_OutputCompareText_EmptyKey(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "", Type: "added", Dest: "value"},
	}
	output := batch2CaptureStdout(t, func() {
		outputCompareText("src", "dst", diffs)
	})
	assert.Contains(t, output, "(entire section)")
}

func TestBatch2_OutputCompareJSON(t *testing.T) {
	diffs := []configDiff{
		{Provider: "brew", Key: "formulae", Type: "added", Dest: "git"},
	}
	output := batch2CaptureStdout(t, func() {
		err := outputCompareJSON(diffs)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "formulae")
	assert.Contains(t, output, "added")
}

// ---------------------------------------------------------------------------
// compare.go -- helper functions
// ---------------------------------------------------------------------------

func TestBatch2_EqualValues(t *testing.T) {
	t.Parallel()
	assert.True(t, equalValues("hello", "hello"))
	assert.False(t, equalValues("hello", "world"))
	assert.True(t, equalValues(42, 42))
	assert.True(t, equalValues(nil, nil))
}

func TestBatch2_ContainsProvider(t *testing.T) {
	t.Parallel()
	assert.True(t, containsProvider([]string{"brew", "git"}, "brew"))
	assert.False(t, containsProvider([]string{"brew", "git"}, "ssh"))
	assert.False(t, containsProvider([]string{}, "brew"))
}

func TestBatch2_FormatValue(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "<nil>", formatValue(nil))
	assert.Equal(t, "hello", formatValue("hello"))
	assert.Equal(t, "{2 keys}", formatValue(map[string]interface{}{"a": 1, "b": 2}))
}

func TestBatch2_Truncate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 6))
}

// ---------------------------------------------------------------------------
// rollback.go -- formatAge
// ---------------------------------------------------------------------------

func TestBatch2_FormatAge_JustNow(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now())
	assert.Equal(t, "just now", result)
}

func TestBatch2_FormatAge_OneMinute(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-1*time.Minute - 10*time.Second))
	assert.Equal(t, "1 min ago", result)
}

func TestBatch2_FormatAge_MultipleMinutes(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-5 * time.Minute))
	assert.Equal(t, "5 mins ago", result)
}

func TestBatch2_FormatAge_OneHour(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-1*time.Hour - 10*time.Minute))
	assert.Equal(t, "1 hour ago", result)
}

func TestBatch2_FormatAge_MultipleHours(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-5 * time.Hour))
	assert.Equal(t, "5 hours ago", result)
}

func TestBatch2_FormatAge_OneDay(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-25 * time.Hour))
	assert.Equal(t, "1 day ago", result)
}

func TestBatch2_FormatAge_MultipleDays(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-3 * 24 * time.Hour))
	assert.Equal(t, "3 days ago", result)
}

func TestBatch2_FormatAge_OneWeek(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-8 * 24 * time.Hour))
	assert.Equal(t, "1 week ago", result)
}

func TestBatch2_FormatAge_MultipleWeeks(t *testing.T) {
	t.Parallel()
	result := formatAge(time.Now().Add(-21 * 24 * time.Hour))
	assert.Equal(t, "3 weeks ago", result)
}

// ---------------------------------------------------------------------------
// compliance.go -- collectEvaluatedItems
// ---------------------------------------------------------------------------

func TestBatch2_CollectEvaluatedItems_Nil(t *testing.T) {
	t.Parallel()
	result := collectEvaluatedItems(nil)
	assert.Nil(t, result)
}

func TestBatch2_CollectEvaluatedItems_WithData(t *testing.T) {
	t.Parallel()
	vr := &app.ValidationResult{
		Info:   []string{"item1", "item2"},
		Errors: []string{"error1"},
	}
	items := collectEvaluatedItems(vr)
	assert.Len(t, items, 3)
	assert.Contains(t, items, "item1")
	assert.Contains(t, items, "error1")
}

func TestBatch2_CollectEvaluatedItems_Empty(t *testing.T) {
	t.Parallel()
	vr := &app.ValidationResult{}
	items := collectEvaluatedItems(vr)
	assert.Empty(t, items)
}

// ---------------------------------------------------------------------------
// discover.go -- getPatternIcon
// ---------------------------------------------------------------------------

func TestBatch2_GetPatternIcon_AllTypes(t *testing.T) {
	t.Parallel()
	// Just verify they return non-empty strings
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypeShell))
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypeEditor))
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypeGit))
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypeSSH))
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypeTmux))
	assert.NotEmpty(t, getPatternIcon(discover.PatternTypePackageManager))
	assert.NotEmpty(t, getPatternIcon("unknown"))
}

// ---------------------------------------------------------------------------
// export.go -- exportToNix, exportToBrewfile, exportToShell
// ---------------------------------------------------------------------------

func TestBatch2_ExportToNix_WithBrewAndGit(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "curl"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	}
	output, err := exportToNix(config)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "home.packages")
	assert.Contains(t, string(output), "programs.git")
	assert.Contains(t, string(output), "Test User")
	assert.Contains(t, string(output), "test@example.com")
}

func TestBatch2_ExportToNix_WithShell(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"shell": map[string]interface{}{
			"shell":   "zsh",
			"plugins": []interface{}{"zsh-autosuggestions"},
		},
	}
	output, err := exportToNix(config)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "programs.zsh")
	assert.Contains(t, string(output), "zsh-autosuggestions")
}

func TestBatch2_ExportToNix_Empty(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToNix(config)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Generated by preflight")
}

func TestBatch2_ExportToBrewfile_Full(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"git", "curl"},
			"casks":    []interface{}{"firefox"},
		},
	}
	output, err := exportToBrewfile(config)
	assert.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "tap \"homebrew/cask\"")
	assert.Contains(t, s, "brew \"git\"")
	assert.Contains(t, s, "brew \"curl\"")
	assert.Contains(t, s, "cask \"firefox\"")
}

func TestBatch2_ExportToBrewfile_Empty(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToBrewfile(config)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Generated by preflight")
}

func TestBatch2_ExportToShell_Full(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"git", "curl"},
			"casks":    []interface{}{"firefox"},
		},
		"git": map[string]interface{}{
			"name":  "Test User",
			"email": "test@example.com",
		},
	}
	output, err := exportToShell(config)
	assert.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	assert.Contains(t, s, "brew tap homebrew/cask")
	assert.Contains(t, s, "brew install")
	assert.Contains(t, s, "brew install --cask")
	assert.Contains(t, s, "git config --global user.name")
	assert.Contains(t, s, "Setup complete!")
}

func TestBatch2_ExportToShell_Empty(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{}
	output, err := exportToShell(config)
	assert.NoError(t, err)
	assert.Contains(t, string(output), "set -euo pipefail")
}

// ---------------------------------------------------------------------------
// deprecated.go -- outputDeprecatedJSON, toDeprecatedPackagesJSON, formatDeprecationStatus
// ---------------------------------------------------------------------------

func TestBatch2_FormatDeprecationStatus_AllTypes(t *testing.T) {
	t.Parallel()
	// These return ANSI-colored strings
	assert.Contains(t, formatDeprecationStatus(security.ReasonDisabled), "DISABLED")
	assert.Contains(t, formatDeprecationStatus(security.ReasonDeprecated), "DEPRECATED")
	assert.Contains(t, formatDeprecationStatus(security.ReasonEOL), "EOL")
	assert.Contains(t, formatDeprecationStatus(security.ReasonUnmaintained), "UNMAINTAINED")
	assert.Equal(t, "other", formatDeprecationStatus("other"))
}

// ---------------------------------------------------------------------------
// catalog.go and plugin.go -- command registration tests
// ---------------------------------------------------------------------------

func TestBatch2_CatalogCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	assert.True(t, catalogCmd.HasSubCommands())
	subCmds := catalogCmd.Commands()
	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "verify")
	assert.Contains(t, names, "audit")
}

func TestBatch2_PluginCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	assert.True(t, pluginCmd.HasSubCommands())
	subCmds := pluginCmd.Commands()
	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "install")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "info")
	assert.Contains(t, names, "search")
	assert.Contains(t, names, "validate")
	assert.Contains(t, names, "upgrade")
}

func TestBatch2_SecretsCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	assert.True(t, secretsCmd.HasSubCommands())
	subCmds := secretsCmd.Commands()
	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "check")
	assert.Contains(t, names, "set")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "backends")
}

func TestBatch2_HistoryCmd_HasClearSubcommand(t *testing.T) {
	t.Parallel()
	subCmds := historyCmd.Commands()
	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "clear")
}

func TestBatch2_TourCmd_Flags(t *testing.T) {
	t.Parallel()
	f := tourCmd.Flags()
	assert.NotNil(t, f.Lookup("list"))
}

func TestBatch2_DoctorCmd_Flags(t *testing.T) {
	t.Parallel()
	f := doctorCmd.Flags()
	assert.NotNil(t, f.Lookup("fix"))
	assert.NotNil(t, f.Lookup("verbose"))
	assert.NotNil(t, f.Lookup("update-config"))
	assert.NotNil(t, f.Lookup("dry-run"))
	assert.NotNil(t, f.Lookup("quiet"))
}

func TestBatch2_RollbackCmd_Flags(t *testing.T) {
	t.Parallel()
	f := rollbackCmd.Flags()
	assert.NotNil(t, f.Lookup("to"))
	assert.NotNil(t, f.Lookup("latest"))
	assert.NotNil(t, f.Lookup("dry-run"))
}

func TestBatch2_SyncCmd_Flags(t *testing.T) {
	t.Parallel()
	f := syncCmd.Flags()
	assert.NotNil(t, f.Lookup("config"))
	assert.NotNil(t, f.Lookup("target"))
	assert.NotNil(t, f.Lookup("remote"))
	assert.NotNil(t, f.Lookup("branch"))
	assert.NotNil(t, f.Lookup("push"))
	assert.NotNil(t, f.Lookup("dry-run"))
	assert.NotNil(t, f.Lookup("force"))
}

func TestBatch2_CompareCmd_Flags(t *testing.T) {
	t.Parallel()
	f := compareCmd.Flags()
	assert.NotNil(t, f.Lookup("config"))
	assert.NotNil(t, f.Lookup("config2"))
	assert.NotNil(t, f.Lookup("json"))
	assert.NotNil(t, f.Lookup("verbose"))
}

func TestBatch2_ExportCmd_Flags(t *testing.T) {
	t.Parallel()
	f := exportCmd.Flags()
	assert.NotNil(t, f.Lookup("config"))
	assert.NotNil(t, f.Lookup("target"))
	assert.NotNil(t, f.Lookup("format"))
	assert.NotNil(t, f.Lookup("output"))
	assert.NotNil(t, f.Lookup("flatten"))
}

func TestBatch2_DeprecatedCmd_Flags(t *testing.T) {
	t.Parallel()
	f := deprecatedCmd.Flags()
	assert.NotNil(t, f.Lookup("ignore"))
	assert.NotNil(t, f.Lookup("json"))
	assert.NotNil(t, f.Lookup("quiet"))
}

func TestBatch2_DiscoverCmd_Flags(t *testing.T) {
	t.Parallel()
	f := discoverCmd.Flags()
	assert.NotNil(t, f.Lookup("max-repos"))
	assert.NotNil(t, f.Lookup("min-stars"))
	assert.NotNil(t, f.Lookup("language"))
	assert.NotNil(t, f.Lookup("all"))
}

func TestBatch2_ComplianceCmd_Flags(t *testing.T) {
	t.Parallel()
	f := complianceCmd.Flags()
	assert.NotNil(t, f.Lookup("config"))
	assert.NotNil(t, f.Lookup("target"))
	assert.NotNil(t, f.Lookup("policy"))
	assert.NotNil(t, f.Lookup("json"))
	assert.NotNil(t, f.Lookup("strict"))
}

func TestBatch2_MCPCmd_Flags(t *testing.T) {
	t.Parallel()
	f := mcpCmd.Flags()
	assert.NotNil(t, f.Lookup("http"))
	assert.NotNil(t, f.Lookup("config"))
	assert.NotNil(t, f.Lookup("target"))
}

func TestBatch2_DiffCmd_Flags(t *testing.T) {
	t.Parallel()
	f := diffCmd.Flags()
	assert.NotNil(t, f.Lookup("provider"))
}

// ---------------------------------------------------------------------------
// Global variables save/restore tests for doctor flags
// ---------------------------------------------------------------------------

func TestBatch2_DoctorFlags_DefaultValues(t *testing.T) {
	t.Parallel()
	// Verify the doctorCmd flag defaults
	cmd := doctorCmd
	fix, _ := cmd.Flags().GetBool("fix")
	assert.False(t, fix)
	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose)
	updateConfig, _ := cmd.Flags().GetBool("update-config")
	assert.False(t, updateConfig)
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	assert.False(t, dryRun)
	quiet, _ := cmd.Flags().GetBool("quiet")
	assert.False(t, quiet)
}

// ---------------------------------------------------------------------------
// Comprehensive DoctorReport method tests
// ---------------------------------------------------------------------------

func TestBatch2_DoctorReport_IssueCount(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{
		Issues: []app.DoctorIssue{
			{Severity: app.SeverityError, Message: "a"},
			{Severity: app.SeverityWarning, Message: "b"},
		},
	}
	assert.Equal(t, 2, report.IssueCount())
}

func TestBatch2_DoctorReport_FixableCount(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{
		Issues: []app.DoctorIssue{
			{Fixable: true},
			{Fixable: false},
			{Fixable: true},
		},
	}
	assert.Equal(t, 2, report.FixableCount())
}

func TestBatch2_DoctorReport_HasPatches(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{}
	assert.False(t, report.HasPatches())

	report.SuggestedPatches = []app.ConfigPatch{
		app.NewConfigPatch("layer.yaml", "key", app.PatchOpAdd, nil, "value", "test"),
	}
	assert.True(t, report.HasPatches())
	assert.Equal(t, 1, report.PatchCount())
}

func TestBatch2_DoctorReport_PatchesByLayer(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{
		SuggestedPatches: []app.ConfigPatch{
			app.NewConfigPatch("layers/base.yaml", "key1", app.PatchOpAdd, nil, "v1", "test"),
			app.NewConfigPatch("layers/base.yaml", "key2", app.PatchOpModify, "old", "new", "test"),
			app.NewConfigPatch("layers/work.yaml", "key3", app.PatchOpRemove, "old", nil, "test"),
		},
	}
	byLayer := report.PatchesByLayer()
	assert.Len(t, byLayer["layers/base.yaml"], 2)
	assert.Len(t, byLayer["layers/work.yaml"], 1)
}

func TestBatch2_DoctorReport_ErrorCount(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{
		Issues: []app.DoctorIssue{
			{Severity: app.SeverityError},
			{Severity: app.SeverityError},
			{Severity: app.SeverityWarning},
		},
	}
	assert.Equal(t, 2, report.ErrorCount())
}

func TestBatch2_DoctorReport_WarningCount(t *testing.T) {
	t.Parallel()
	report := app.DoctorReport{
		Issues: []app.DoctorIssue{
			{Severity: app.SeverityError},
			{Severity: app.SeverityWarning},
			{Severity: app.SeverityWarning},
		},
	}
	assert.Equal(t, 2, report.WarningCount())
}

// ---------------------------------------------------------------------------
// ConfigPatch.Description
// ---------------------------------------------------------------------------

func TestBatch2_ConfigPatch_Description(t *testing.T) {
	t.Parallel()
	pAdd := app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpAdd, nil, "git", "test")
	assert.Contains(t, pAdd.Description(), "Add")
	assert.Contains(t, pAdd.Description(), "brew.formulae")

	pModify := app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpModify, "old", "new", "test")
	assert.Contains(t, pModify.Description(), "Modify")

	pRemove := app.NewConfigPatch("layers/base.yaml", "brew.formulae", app.PatchOpRemove, "old", nil, "test")
	assert.Contains(t, pRemove.Description(), "Remove")
}

// ---------------------------------------------------------------------------
// history.go -- runHistoryClear
// ---------------------------------------------------------------------------

func TestBatch2_RunHistoryClear_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	output := batch2CaptureStdout(t, func() {
		err := runHistoryClear(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "History cleared")
}

func TestBatch2_RunHistoryClear_WithEntries(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a history entry first
	err := SaveHistoryEntry(HistoryEntry{
		ID:      "to-clear",
		Command: "apply",
		Status:  "success",
	})
	require.NoError(t, err)

	output := batch2CaptureStdout(t, func() {
		err := runHistoryClear(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "History cleared")

	// Verify directory is gone
	histDir := filepath.Join(tmpDir, ".preflight", "history")
	_, err = os.Stat(histDir)
	assert.True(t, os.IsNotExist(err))
}

// ---------------------------------------------------------------------------
// history.go -- getHistoryDir
// ---------------------------------------------------------------------------

func TestBatch2_GetHistoryDir_ContainsPreflight(t *testing.T) {
	dir := getHistoryDir()
	assert.Contains(t, dir, ".preflight")
	assert.Contains(t, dir, "history")
}
