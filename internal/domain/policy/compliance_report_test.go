package policy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplianceStatus_Values(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ComplianceStatusCompliant, ComplianceStatus("compliant"))
	assert.Equal(t, ComplianceStatusNonCompliant, ComplianceStatus("non_compliant"))
	assert.Equal(t, ComplianceStatusWarning, ComplianceStatus("warning"))
}

func TestReportGenerator_Generate_NilPolicy(t *testing.T) {
	t.Parallel()

	generator := NewReportGenerator(nil)
	result := &OrgResult{
		Violations:  []OrgViolation{},
		Warnings:    []OrgViolation{},
		Enforcement: EnforcementBlock,
	}

	report := generator.Generate(result, []string{"git:user.name"})

	assert.Equal(t, "none", report.PolicyName)
	assert.Equal(t, EnforcementBlock, report.Enforcement)
	assert.Equal(t, ComplianceStatusCompliant, report.Summary.Status)
}

func TestReportGenerator_Generate_WithPolicy(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "test-policy",
		Description: "Test policy for compliance",
		Enforcement: EnforcementBlock,
	}

	generator := NewReportGenerator(policy)
	result := &OrgResult{
		Violations:  []OrgViolation{},
		Warnings:    []OrgViolation{},
		Enforcement: EnforcementBlock,
	}

	report := generator.Generate(result, []string{"git:user.name", "brew:git"})

	assert.Equal(t, "test-policy", report.PolicyName)
	assert.Equal(t, "Test policy for compliance", report.PolicyDescription)
	assert.Equal(t, EnforcementBlock, report.Enforcement)
	assert.Equal(t, 2, report.Summary.TotalChecks)
	assert.Equal(t, ComplianceStatusCompliant, report.Summary.Status)
}

func TestReportGenerator_Generate_WithViolations(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "strict-policy",
		Enforcement: EnforcementBlock,
	}

	generator := NewReportGenerator(policy)
	result := &OrgResult{
		Violations: []OrgViolation{
			{
				Type:    "missing_required",
				Pattern: "git:*",
				Message: "git config is required",
			},
			{
				Type:    "forbidden_present",
				Pattern: "brew:*-nightly",
				Value:   "brew:rust-nightly",
				Message: "nightly packages not allowed",
			},
		},
		Warnings:    []OrgViolation{},
		Enforcement: EnforcementBlock,
	}

	report := generator.Generate(result, []string{"brew:rust-nightly"})

	assert.Equal(t, ComplianceStatusNonCompliant, report.Summary.Status)
	assert.Equal(t, 2, report.Summary.ViolationCount)
	assert.Len(t, report.Violations, 2)

	// Check first violation details
	assert.Equal(t, "missing_required", report.Violations[0].Type)
	assert.Equal(t, "git:*", report.Violations[0].Pattern)
	assert.Equal(t, "error", report.Violations[0].Severity)
	assert.Contains(t, report.Violations[0].Recommendation, "Add a configuration")

	// Check second violation details
	assert.Equal(t, "forbidden_present", report.Violations[1].Type)
	assert.Equal(t, "brew:rust-nightly", report.Violations[1].Value)
	assert.Contains(t, report.Violations[1].Recommendation, "Remove")
}

func TestReportGenerator_Generate_WithWarnings(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "lenient-policy",
		Enforcement: EnforcementWarn,
	}

	generator := NewReportGenerator(policy)
	result := &OrgResult{
		Violations: []OrgViolation{},
		Warnings: []OrgViolation{
			{
				Type:    "forbidden_present",
				Pattern: "brew:*-nightly",
				Value:   "brew:rust-nightly",
			},
		},
		Enforcement: EnforcementWarn,
	}

	report := generator.Generate(result, []string{"brew:rust-nightly", "brew:git"})

	assert.Equal(t, ComplianceStatusWarning, report.Summary.Status)
	assert.Equal(t, 0, report.Summary.ViolationCount)
	assert.Equal(t, 1, report.Summary.WarningCount)
	assert.Len(t, report.Warnings, 1)
	assert.Equal(t, "warning", report.Warnings[0].Severity)
}

func TestReportGenerator_Generate_WithOverrides(t *testing.T) {
	t.Parallel()

	futureDate := time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	policy := &OrgPolicy{
		Name:        "override-policy",
		Enforcement: EnforcementBlock,
	}

	generator := NewReportGenerator(policy)
	result := &OrgResult{
		Violations:  []OrgViolation{},
		Warnings:    []OrgViolation{},
		Enforcement: EnforcementBlock,
		OverridesApplied: []Override{
			{
				Pattern:       "brew:rust-nightly",
				Justification: "needed for testing",
				ApprovedBy:    "admin@example.com",
				ExpiresAt:     futureDate,
			},
		},
	}

	report := generator.Generate(result, []string{"brew:rust-nightly"})

	assert.Equal(t, 1, report.Summary.OverrideCount)
	require.Len(t, report.Overrides, 1)
	assert.Equal(t, "brew:rust-nightly", report.Overrides[0].Pattern)
	assert.Equal(t, "needed for testing", report.Overrides[0].Justification)
	assert.Equal(t, "admin@example.com", report.Overrides[0].ApprovedBy)
	assert.GreaterOrEqual(t, report.Overrides[0].DaysUntilExpiry, 6) // ~7 days
}

func TestReportGenerator_CalculateDaysUntilExpiry(t *testing.T) {
	t.Parallel()

	generator := NewReportGenerator(nil)

	tests := []struct {
		name      string
		expiresAt string
		wantMin   int
		wantMax   int
	}{
		{
			name:      "no expiry",
			expiresAt: "",
			wantMin:   -1,
			wantMax:   -1,
		},
		{
			name:      "invalid date",
			expiresAt: "not-a-date",
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "expired",
			expiresAt: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			wantMin:   0,
			wantMax:   0,
		},
		{
			name:      "future date",
			expiresAt: time.Now().Add(10 * 24 * time.Hour).Format(time.RFC3339),
			wantMin:   9,
			wantMax:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			days := generator.calculateDaysUntilExpiry(tt.expiresAt)
			assert.GreaterOrEqual(t, days, tt.wantMin)
			assert.LessOrEqual(t, days, tt.wantMax)
		})
	}
}

func TestReportGenerator_GenerateRecommendation(t *testing.T) {
	t.Parallel()

	generator := NewReportGenerator(nil)

	tests := []struct {
		name      string
		violation OrgViolation
		wantIn    string
	}{
		{
			name:      "missing required",
			violation: OrgViolation{Type: "missing_required", Pattern: "git:*"},
			wantIn:    "Add a configuration",
		},
		{
			name:      "forbidden present",
			violation: OrgViolation{Type: "forbidden_present", Pattern: "brew:*", Value: "brew:rust"},
			wantIn:    "Remove",
		},
		{
			name:      "unknown type",
			violation: OrgViolation{Type: "unknown"},
			wantIn:    "Review the policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := generator.generateRecommendation(tt.violation)
			assert.Contains(t, rec, tt.wantIn)
		})
	}
}

func TestComplianceReport_ToJSON(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ViolationCount:  0,
			WarningCount:    0,
			OverrideCount:   0,
			ComplianceScore: 100.0,
		},
	}

	jsonBytes, err := report.ToJSON()
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-policy", parsed["policy_name"])
	assert.Equal(t, "block", parsed["enforcement"])
}

func TestComplianceReport_ToText(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:        "security-policy",
		PolicyDescription: "Security compliance requirements",
		Enforcement:       EnforcementBlock,
		GeneratedAt:       time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusNonCompliant,
			TotalChecks:     10,
			PassedChecks:    8,
			ViolationCount:  2,
			WarningCount:    1,
			OverrideCount:   1,
			ComplianceScore: 80.0,
		},
		Violations: []ViolationDetail{
			{
				Type:           "missing_required",
				Pattern:        "git:*",
				Message:        "git configuration required",
				Severity:       "error",
				Recommendation: "Add git configuration",
			},
		},
		Warnings: []ViolationDetail{
			{
				Type:           "forbidden_present",
				Pattern:        "brew:*-beta",
				Value:          "brew:go-beta",
				Severity:       "warning",
				Recommendation: "Consider using stable version",
			},
		},
		Overrides: []OverrideDetail{
			{
				Pattern:         "brew:rust-nightly",
				Justification:   "needed for testing",
				ApprovedBy:      "admin",
				ExpiresAt:       "2025-02-01T00:00:00Z",
				DaysUntilExpiry: 30,
			},
		},
	}

	text := report.ToText()

	// Check headers
	assert.Contains(t, text, "POLICY COMPLIANCE REPORT")
	assert.Contains(t, text, "security-policy")
	assert.Contains(t, text, "Security compliance requirements")

	// Check summary
	assert.Contains(t, text, "[FAIL]")
	assert.Contains(t, text, "80.0%")
	assert.Contains(t, text, "Violations:")

	// Check violations section
	assert.Contains(t, text, "missing_required")
	assert.Contains(t, text, "git:*")

	// Check warnings section
	assert.Contains(t, text, "Warnings")
	assert.Contains(t, text, "brew:go-beta")

	// Check overrides section
	assert.Contains(t, text, "Overrides")
	assert.Contains(t, text, "needed for testing")
}

func TestComplianceReport_GetStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   ComplianceStatus
		wantIcon string
	}{
		{ComplianceStatusCompliant, "[PASS]"},
		{ComplianceStatusWarning, "[WARN]"},
		{ComplianceStatusNonCompliant, "[FAIL]"},
		{ComplianceStatus("unknown"), "[????]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			report := &ComplianceReport{
				Summary: ComplianceSummary{Status: tt.status},
			}
			assert.Equal(t, tt.wantIcon, report.getStatusIcon())
		})
	}
}

func TestComplianceReport_IsCompliant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status ComplianceStatus
		want   bool
	}{
		{ComplianceStatusCompliant, true},
		{ComplianceStatusWarning, false},
		{ComplianceStatusNonCompliant, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			report := &ComplianceReport{
				Summary: ComplianceSummary{Status: tt.status},
			}
			assert.Equal(t, tt.want, report.IsCompliant())
		})
	}
}

func TestComplianceReport_HasBlockingViolations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status ComplianceStatus
		want   bool
	}{
		{ComplianceStatusCompliant, false},
		{ComplianceStatusWarning, false},
		{ComplianceStatusNonCompliant, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			report := &ComplianceReport{
				Summary: ComplianceSummary{Status: tt.status},
			}
			assert.Equal(t, tt.want, report.HasBlockingViolations())
		})
	}
}

func TestComplianceReport_ExpiringOverrides(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		Overrides: []OverrideDetail{
			{Pattern: "pkg:a", DaysUntilExpiry: -1}, // No expiry
			{Pattern: "pkg:b", DaysUntilExpiry: 3},  // Expires in 3 days
			{Pattern: "pkg:c", DaysUntilExpiry: 7},  // Expires in 7 days
			{Pattern: "pkg:d", DaysUntilExpiry: 30}, // Expires in 30 days
			{Pattern: "pkg:e", DaysUntilExpiry: 0},  // Already expired
		},
	}

	// Overrides expiring within 7 days
	expiring := report.ExpiringOverrides(7)
	assert.Len(t, expiring, 3) // pkg:b, pkg:c, pkg:e

	// Verify patterns
	patterns := make([]string, len(expiring))
	for i, o := range expiring {
		patterns[i] = o.Pattern
	}
	assert.Contains(t, patterns, "pkg:b")
	assert.Contains(t, patterns, "pkg:c")
	assert.Contains(t, patterns, "pkg:e")
}

func TestComplianceReportService_EvaluateAndReport(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "service-test",
		Enforcement: EnforcementBlock,
		Required: []Requirement{
			{Pattern: "git:*", Message: "git config required"},
		},
		Forbidden: []Forbidden{
			{Pattern: "brew:*-nightly", Message: "nightly packages forbidden"},
		},
	}

	service := NewComplianceReportService(policy)
	items := []string{"brew:git", "brew:rust-nightly"}

	report := service.EvaluateAndReport(items)

	assert.Equal(t, "service-test", report.PolicyName)
	assert.Equal(t, ComplianceStatusNonCompliant, report.Summary.Status)
	assert.Equal(t, 2, report.Summary.ViolationCount) // missing git:*, forbidden rust-nightly
}

func TestComplianceReport_CompareTo(t *testing.T) {
	t.Parallel()

	previous := &ComplianceReport{
		Summary: ComplianceSummary{
			Status:          ComplianceStatusNonCompliant,
			ComplianceScore: 60.0,
		},
		Violations: []ViolationDetail{
			{Type: "missing_required", Pattern: "git:*", Value: ""},
			{Type: "forbidden_present", Pattern: "brew:*-nightly", Value: "brew:rust-nightly"},
		},
	}

	current := &ComplianceReport{
		Summary: ComplianceSummary{
			Status:          ComplianceStatusWarning,
			ComplianceScore: 80.0,
		},
		Violations: []ViolationDetail{
			{Type: "forbidden_present", Pattern: "brew:*-beta", Value: "brew:go-beta"},
		},
	}

	delta := current.CompareTo(previous)

	assert.InDelta(t, 20.0, delta.ScoreChange, 0.001)
	assert.Equal(t, "non_compliant → warning", delta.StatusChange)
	assert.Len(t, delta.NewViolations, 1)
	assert.Equal(t, "brew:go-beta", delta.NewViolations[0].Value)
	assert.Len(t, delta.ResolvedViolations, 2)
}

func TestComplianceReport_CompareTo_NoChange(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			ComplianceScore: 100.0,
		},
		Violations: []ViolationDetail{},
	}

	delta := report.CompareTo(report)

	assert.InDelta(t, 0.0, delta.ScoreChange, 0.001)
	assert.Empty(t, delta.StatusChange)
	assert.Empty(t, delta.NewViolations)
	assert.Empty(t, delta.ResolvedViolations)
}

func TestComplianceSummary_CalculateScore(t *testing.T) {
	t.Parallel()

	policy := &OrgPolicy{
		Name:        "score-test",
		Enforcement: EnforcementBlock,
	}

	generator := NewReportGenerator(policy)

	tests := []struct {
		name         string
		result       *OrgResult
		items        []string
		wantScoreMin float64
		wantScoreMax float64
	}{
		{
			name: "all passed",
			result: &OrgResult{
				Violations:  []OrgViolation{},
				Warnings:    []OrgViolation{},
				Enforcement: EnforcementBlock,
			},
			items:        []string{"git:user.name", "brew:git"},
			wantScoreMin: 100.0,
			wantScoreMax: 100.0,
		},
		{
			name: "some failed",
			result: &OrgResult{
				Violations: []OrgViolation{
					{Type: "forbidden_present", Pattern: "brew:*", Value: "brew:bad"},
				},
				Warnings:    []OrgViolation{},
				Enforcement: EnforcementBlock,
			},
			items:        []string{"git:user.name", "brew:bad"},
			wantScoreMin: 40.0,
			wantScoreMax: 60.0,
		},
		{
			name: "empty items",
			result: &OrgResult{
				Violations:  []OrgViolation{},
				Warnings:    []OrgViolation{},
				Enforcement: EnforcementBlock,
			},
			items:        []string{},
			wantScoreMin: 100.0,
			wantScoreMax: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := generator.Generate(tt.result, tt.items)
			assert.GreaterOrEqual(t, report.Summary.ComplianceScore, tt.wantScoreMin)
			assert.LessOrEqual(t, report.Summary.ComplianceScore, tt.wantScoreMax)
		})
	}
}

func TestComplianceReport_ToText_AllSections(t *testing.T) {
	t.Parallel()

	report := &ComplianceReport{
		PolicyName:  "full-report",
		Enforcement: EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: ComplianceSummary{
			Status:          ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ComplianceScore: 100.0,
		},
	}

	text := report.ToText()

	// Should have main sections
	assert.Contains(t, text, "Summary")
	assert.Contains(t, text, "Status:")
	assert.Contains(t, text, "[PASS]")

	// Should NOT have empty sections
	assert.NotContains(t, text, "Violations (Blocking)")
	assert.NotContains(t, text, "Warnings (Non-Blocking)")
	assert.NotContains(t, text, "Applied Overrides")
}
