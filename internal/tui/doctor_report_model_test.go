package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDoctorReportModel_Init(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReport(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
}

func TestDoctorReportModel_View(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReport(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "Doctor Report", "should contain header")
}

func TestDoctorReportModel_NoIssues(t *testing.T) {
	t.Parallel()

	report := &DoctorReport{Issues: []DoctorIssue{}}
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "No issues", "should show no issues message")
}

func TestDoctorReportModel_WithIssues(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReport(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "drift", "should show drift issue")
}

func TestDoctorReportModel_Navigation(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReportWithMultipleIssues(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	// Initial cursor should be at 0
	assert.Equal(t, 0, model.cursor)

	// Navigate down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(doctorReportModel)
	assert.Equal(t, 1, m.cursor)
}

func TestDoctorReportModel_WindowResize(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReport(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(doctorReportModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestDoctorReportModel_Quit(t *testing.T) {
	t.Parallel()

	report := createTestDoctorReport(t)
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	// Press 'q' to quit
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd, "should return quit command")
}

func TestDoctorReportModel_FixableIssues(t *testing.T) {
	t.Parallel()

	report := &DoctorReport{
		Issues: []DoctorIssue{
			{
				Severity:   IssueSeverityWarning,
				Category:   "packages",
				Message:    "Package git version mismatch",
				Details:    "Expected 2.42.0, got 2.41.0",
				CanAutoFix: true,
				FixCommand: "brew upgrade git",
			},
		},
	}
	model := newDoctorReportModel(report, DoctorReportOptions{Verbose: true})
	model.width = 100
	model.height = 24

	view := model.View()

	assert.Contains(t, view, "fixable", "should indicate fixable issue")
}

func TestDoctorReportModel_AutoFix(t *testing.T) {
	t.Parallel()

	report := &DoctorReport{
		Issues: []DoctorIssue{
			{
				Severity:   IssueSeverityWarning,
				Category:   "packages",
				Message:    "Package git version mismatch",
				CanAutoFix: true,
			},
		},
	}
	model := newDoctorReportModel(report, DoctorReportOptions{AutoFix: true, Verbose: true})
	model.width = 100
	model.height = 24

	// Press 'f' to fix
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m := newModel.(doctorReportModel)

	assert.True(t, m.fixing, "should be in fixing state")
}

// Helper functions to create test doctor reports

func createTestDoctorReport(t *testing.T) *DoctorReport {
	t.Helper()

	return &DoctorReport{
		Issues: []DoctorIssue{
			{
				Severity:   IssueSeverityWarning,
				Category:   "config",
				Message:    "Configuration drift detected",
				Details:    "git.user.email differs from expected",
				CanAutoFix: false,
			},
		},
	}
}

func createTestDoctorReportWithMultipleIssues(t *testing.T) *DoctorReport {
	t.Helper()

	return &DoctorReport{
		Issues: []DoctorIssue{
			{
				Severity: IssueSeverityWarning,
				Category: "config",
				Message:  "Configuration drift detected",
			},
			{
				Severity: IssueSeverityError,
				Category: "packages",
				Message:  "Package missing: jq",
			},
			{
				Severity: IssueSeverityInfo,
				Category: "files",
				Message:  "File needs update: ~/.bashrc",
			},
		},
	}
}
