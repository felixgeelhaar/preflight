package tui

import (
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/app"
)

// ConvertDoctorReport converts an app.DoctorReport to a tui.DoctorReport.
func ConvertDoctorReport(report *app.DoctorReport) *DoctorReport {
	if report == nil {
		return &DoctorReport{
			Issues: []DoctorIssue{},
		}
	}

	issues := make([]DoctorIssue, len(report.Issues))
	for i, issue := range report.Issues {
		issues[i] = ConvertDoctorIssue(issue)
	}

	return &DoctorReport{
		Issues: issues,
	}
}

// ConvertDoctorIssue converts an app.DoctorIssue to a tui.DoctorIssue.
func ConvertDoctorIssue(issue app.DoctorIssue) DoctorIssue {
	var details string
	switch {
	case issue.Expected != "" && issue.Actual != "":
		details = fmt.Sprintf("Expected: %s, Actual: %s", issue.Expected, issue.Actual)
	case issue.Expected != "":
		details = fmt.Sprintf("Expected: %s", issue.Expected)
	case issue.Actual != "":
		details = fmt.Sprintf("Actual: %s", issue.Actual)
	}

	return DoctorIssue{
		Severity:   convertSeverity(issue.Severity),
		Category:   issue.Provider,
		Message:    issue.Message,
		Details:    details,
		CanAutoFix: issue.Fixable,
		FixCommand: issue.FixCommand,
	}
}

// convertSeverity converts app.IssueSeverity to tui.IssueSeverity.
func convertSeverity(severity app.IssueSeverity) IssueSeverity {
	switch severity {
	case app.SeverityError:
		return IssueSeverityError
	case app.SeverityWarning:
		return IssueSeverityWarning
	case app.SeverityInfo:
		return IssueSeverityInfo
	default:
		return IssueSeverityInfo
	}
}

// ConvertCapturedItems converts app.CapturedItems to tui.CaptureItems.
func ConvertCapturedItems(items []app.CapturedItem) []CaptureItem {
	result := make([]CaptureItem, len(items))
	for i, item := range items {
		result[i] = ConvertCapturedItem(item)
	}
	return result
}

// ConvertCapturedItem converts a single app.CapturedItem to tui.CaptureItem.
func ConvertCapturedItem(item app.CapturedItem) CaptureItem {
	// Determine capture type based on provider
	captureType := CaptureTypeFile
	switch item.Provider {
	case "brew":
		captureType = CaptureTypeFormula
	case "git":
		captureType = CaptureTypeGit
	case "ssh":
		captureType = CaptureTypeSSH
	case "shell":
		captureType = CaptureTypeShell
	case "nvim":
		captureType = CaptureTypeNvim
	case "vscode":
		captureType = CaptureTypeExtension
	case "runtime":
		captureType = CaptureTypeRuntime
	}

	// Format value for display
	valueStr := ""
	if item.Value != nil {
		valueStr = fmt.Sprintf("%v", item.Value)
	}

	// Add redaction indicator
	details := item.Source
	if item.Redacted {
		details = "[REDACTED] " + details
	}

	return CaptureItem{
		Category: item.Provider,
		Name:     item.Name,
		Type:     captureType,
		Details:  details,
		Value:    valueStr,
	}
}
