// Package policy provides configuration policy constraints with deny/allow rules.
package policy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ComplianceStatus represents the overall compliance state.
type ComplianceStatus string

const (
	// ComplianceStatusCompliant indicates all policy checks passed.
	ComplianceStatusCompliant ComplianceStatus = "compliant"
	// ComplianceStatusNonCompliant indicates blocking violations exist.
	ComplianceStatusNonCompliant ComplianceStatus = "non_compliant"
	// ComplianceStatusWarning indicates non-blocking issues exist.
	ComplianceStatusWarning ComplianceStatus = "warning"
)

// ViolationDetail provides detailed information about a policy violation.
type ViolationDetail struct {
	// Type is "missing_required" or "forbidden_present"
	Type string `json:"type"`
	// Pattern that was violated
	Pattern string `json:"pattern"`
	// Value that triggered the violation (for forbidden)
	Value string `json:"value,omitempty"`
	// Message explains the violation
	Message string `json:"message,omitempty"`
	// Severity is "error" for block mode, "warning" for warn mode
	Severity string `json:"severity"`
	// Recommendation suggests how to fix the violation
	Recommendation string `json:"recommendation,omitempty"`
}

// OverrideDetail provides information about an applied override.
type OverrideDetail struct {
	// Pattern that was overridden
	Pattern string `json:"pattern"`
	// Justification for the override
	Justification string `json:"justification"`
	// ApprovedBy identifies the approver
	ApprovedBy string `json:"approved_by,omitempty"`
	// ExpiresAt is when the override expires
	ExpiresAt string `json:"expires_at,omitempty"`
	// DaysUntilExpiry shows remaining days (-1 if no expiry)
	DaysUntilExpiry int `json:"days_until_expiry"`
}

// ComplianceSummary provides aggregate compliance statistics.
type ComplianceSummary struct {
	// Status is the overall compliance status
	Status ComplianceStatus `json:"status"`
	// TotalChecks is the number of items checked
	TotalChecks int `json:"total_checks"`
	// PassedChecks is the number of items that passed
	PassedChecks int `json:"passed_checks"`
	// ViolationCount is the number of blocking violations
	ViolationCount int `json:"violation_count"`
	// WarningCount is the number of non-blocking warnings
	WarningCount int `json:"warning_count"`
	// OverrideCount is the number of overrides applied
	OverrideCount int `json:"override_count"`
	// ComplianceScore is the percentage of passed checks (0-100)
	ComplianceScore float64 `json:"compliance_score"`
}

// ComplianceReport is a structured report of policy evaluation results.
type ComplianceReport struct {
	// PolicyName is the name of the evaluated policy
	PolicyName string `json:"policy_name"`
	// PolicyDescription describes the policy
	PolicyDescription string `json:"policy_description,omitempty"`
	// Enforcement mode used
	Enforcement EnforcementMode `json:"enforcement"`
	// GeneratedAt is when the report was generated
	GeneratedAt time.Time `json:"generated_at"`
	// Summary provides aggregate statistics
	Summary ComplianceSummary `json:"summary"`
	// Violations lists all blocking violations
	Violations []ViolationDetail `json:"violations,omitempty"`
	// Warnings lists all non-blocking warnings
	Warnings []ViolationDetail `json:"warnings,omitempty"`
	// Overrides lists all applied overrides
	Overrides []OverrideDetail `json:"overrides,omitempty"`
	// EvaluatedItems lists all items that were checked
	EvaluatedItems []string `json:"evaluated_items,omitempty"`
}

// ReportGenerator generates compliance reports from policy evaluation results.
type ReportGenerator struct {
	policy *OrgPolicy
}

// NewReportGenerator creates a new compliance report generator.
func NewReportGenerator(policy *OrgPolicy) *ReportGenerator {
	return &ReportGenerator{policy: policy}
}

// Generate creates a compliance report from an OrgResult.
func (g *ReportGenerator) Generate(result *OrgResult, evaluatedItems []string) *ComplianceReport {
	report := &ComplianceReport{
		GeneratedAt:    time.Now(),
		EvaluatedItems: evaluatedItems,
	}

	if g.policy != nil {
		report.PolicyName = g.policy.Name
		report.PolicyDescription = g.policy.Description
		report.Enforcement = g.policy.Enforcement
	} else {
		report.PolicyName = "none"
		report.Enforcement = EnforcementBlock
	}

	// Convert violations to detailed format
	for _, v := range result.Violations {
		detail := ViolationDetail{
			Type:           v.Type,
			Pattern:        v.Pattern,
			Value:          v.Value,
			Message:        v.Message,
			Severity:       "error",
			Recommendation: g.generateRecommendation(v),
		}
		report.Violations = append(report.Violations, detail)
	}

	// Convert warnings to detailed format
	for _, w := range result.Warnings {
		detail := ViolationDetail{
			Type:           w.Type,
			Pattern:        w.Pattern,
			Value:          w.Value,
			Message:        w.Message,
			Severity:       "warning",
			Recommendation: g.generateRecommendation(w),
		}
		report.Warnings = append(report.Warnings, detail)
	}

	// Convert overrides to detailed format
	for _, o := range result.OverridesApplied {
		detail := OverrideDetail{
			Pattern:         o.Pattern,
			Justification:   o.Justification,
			ApprovedBy:      o.ApprovedBy,
			ExpiresAt:       o.ExpiresAt,
			DaysUntilExpiry: g.calculateDaysUntilExpiry(o.ExpiresAt),
		}
		report.Overrides = append(report.Overrides, detail)
	}

	// Calculate summary
	report.Summary = g.calculateSummary(result, evaluatedItems)

	return report
}

// generateRecommendation creates an actionable recommendation for a violation.
func (g *ReportGenerator) generateRecommendation(v OrgViolation) string {
	switch v.Type {
	case "missing_required":
		return fmt.Sprintf("Add a configuration matching pattern '%s' to satisfy the requirement.", v.Pattern)
	case "forbidden_present":
		return fmt.Sprintf("Remove '%s' from your configuration or request an override.", v.Value)
	default:
		return "Review the policy violation and update your configuration accordingly."
	}
}

// calculateDaysUntilExpiry calculates days until an override expires.
func (g *ReportGenerator) calculateDaysUntilExpiry(expiresAt string) int {
	if expiresAt == "" {
		return -1 // No expiry
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0 // Invalid date = expired
	}

	days := int(time.Until(expiry).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// calculateSummary computes summary statistics from the result.
func (g *ReportGenerator) calculateSummary(result *OrgResult, evaluatedItems []string) ComplianceSummary {
	totalChecks := len(evaluatedItems)
	violationCount := len(result.Violations)
	warningCount := len(result.Warnings)
	overrideCount := len(result.OverridesApplied)

	// Calculate passed checks (items that didn't trigger violations/warnings)
	failedItems := make(map[string]bool)
	for _, v := range result.Violations {
		if v.Value != "" {
			failedItems[v.Value] = true
		}
	}
	for _, w := range result.Warnings {
		if w.Value != "" {
			failedItems[w.Value] = true
		}
	}

	// Add missing required as failures (they affect the overall count)
	missingRequired := 0
	for _, v := range result.Violations {
		if v.Type == "missing_required" {
			missingRequired++
		}
	}
	for _, w := range result.Warnings {
		if w.Type == "missing_required" {
			missingRequired++
		}
	}

	passedChecks := totalChecks - len(failedItems)
	if passedChecks < 0 {
		passedChecks = 0
	}

	// Calculate compliance score
	var complianceScore float64
	effectiveTotal := totalChecks + missingRequired
	if effectiveTotal > 0 {
		complianceScore = float64(passedChecks) / float64(effectiveTotal) * 100
	} else {
		complianceScore = 100 // No checks = fully compliant
	}

	// Determine overall status
	var status ComplianceStatus
	switch {
	case violationCount > 0 && result.Enforcement == EnforcementBlock:
		status = ComplianceStatusNonCompliant
	case warningCount > 0 || (violationCount > 0 && result.Enforcement == EnforcementWarn):
		status = ComplianceStatusWarning
	default:
		status = ComplianceStatusCompliant
	}

	return ComplianceSummary{
		Status:          status,
		TotalChecks:     totalChecks,
		PassedChecks:    passedChecks,
		ViolationCount:  violationCount,
		WarningCount:    warningCount,
		OverrideCount:   overrideCount,
		ComplianceScore: complianceScore,
	}
}

// ToJSON serializes the report to JSON.
func (r *ComplianceReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToText formats the report as human-readable text.
func (r *ComplianceReport) ToText() string {
	var sb strings.Builder

	// Header
	sb.WriteString("╭───────────────────────────────────────────────────────────────╮\n")
	sb.WriteString("│               POLICY COMPLIANCE REPORT                        │\n")
	sb.WriteString("╰───────────────────────────────────────────────────────────────╯\n\n")

	// Policy info
	sb.WriteString(fmt.Sprintf("Policy:       %s\n", r.PolicyName))
	if r.PolicyDescription != "" {
		sb.WriteString(fmt.Sprintf("Description:  %s\n", r.PolicyDescription))
	}
	sb.WriteString(fmt.Sprintf("Enforcement:  %s\n", r.Enforcement))
	sb.WriteString(fmt.Sprintf("Generated:    %s\n\n", r.GeneratedAt.Format(time.RFC3339)))

	// Summary
	sb.WriteString("─── Summary ───────────────────────────────────────────────────\n")
	statusIcon := r.getStatusIcon()
	sb.WriteString(fmt.Sprintf("Status:           %s %s\n", statusIcon, r.Summary.Status))
	sb.WriteString(fmt.Sprintf("Compliance Score: %.1f%%\n", r.Summary.ComplianceScore))
	sb.WriteString(fmt.Sprintf("Total Checks:     %d\n", r.Summary.TotalChecks))
	sb.WriteString(fmt.Sprintf("Passed:           %d\n", r.Summary.PassedChecks))
	sb.WriteString(fmt.Sprintf("Violations:       %d\n", r.Summary.ViolationCount))
	sb.WriteString(fmt.Sprintf("Warnings:         %d\n", r.Summary.WarningCount))
	sb.WriteString(fmt.Sprintf("Overrides:        %d\n\n", r.Summary.OverrideCount))

	// Violations
	if len(r.Violations) > 0 {
		sb.WriteString("─── Violations (Blocking) ─────────────────────────────────────\n")
		for i, v := range r.Violations {
			sb.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, v.Type))
			sb.WriteString(fmt.Sprintf("    Pattern: %s\n", v.Pattern))
			if v.Value != "" {
				sb.WriteString(fmt.Sprintf("    Value:   %s\n", v.Value))
			}
			if v.Message != "" {
				sb.WriteString(fmt.Sprintf("    Message: %s\n", v.Message))
			}
			sb.WriteString(fmt.Sprintf("    Fix:     %s\n", v.Recommendation))
		}
		sb.WriteString("\n")
	}

	// Warnings
	if len(r.Warnings) > 0 {
		sb.WriteString("─── Warnings (Non-Blocking) ───────────────────────────────────\n")
		for i, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, w.Type))
			sb.WriteString(fmt.Sprintf("    Pattern: %s\n", w.Pattern))
			if w.Value != "" {
				sb.WriteString(fmt.Sprintf("    Value:   %s\n", w.Value))
			}
			if w.Message != "" {
				sb.WriteString(fmt.Sprintf("    Message: %s\n", w.Message))
			}
			sb.WriteString(fmt.Sprintf("    Fix:     %s\n", w.Recommendation))
		}
		sb.WriteString("\n")
	}

	// Overrides
	if len(r.Overrides) > 0 {
		sb.WriteString("─── Applied Overrides ─────────────────────────────────────────\n")
		for i, o := range r.Overrides {
			sb.WriteString(fmt.Sprintf("\n[%d] %s\n", i+1, o.Pattern))
			sb.WriteString(fmt.Sprintf("    Justification: %s\n", o.Justification))
			if o.ApprovedBy != "" {
				sb.WriteString(fmt.Sprintf("    Approved By:   %s\n", o.ApprovedBy))
			}
			if o.ExpiresAt != "" {
				expiryStatus := "active"
				if o.DaysUntilExpiry == 0 {
					expiryStatus = "EXPIRED"
				} else if o.DaysUntilExpiry > 0 && o.DaysUntilExpiry <= 7 {
					expiryStatus = fmt.Sprintf("expires in %d days", o.DaysUntilExpiry)
				}
				sb.WriteString(fmt.Sprintf("    Expires:       %s (%s)\n", o.ExpiresAt, expiryStatus))
			}
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("───────────────────────────────────────────────────────────────\n")

	return sb.String()
}

// getStatusIcon returns an icon representing the compliance status.
func (r *ComplianceReport) getStatusIcon() string {
	switch r.Summary.Status {
	case ComplianceStatusCompliant:
		return "[PASS]"
	case ComplianceStatusWarning:
		return "[WARN]"
	case ComplianceStatusNonCompliant:
		return "[FAIL]"
	default:
		return "[????]"
	}
}

// IsCompliant returns true if the report shows full compliance.
func (r *ComplianceReport) IsCompliant() bool {
	return r.Summary.Status == ComplianceStatusCompliant
}

// HasBlockingViolations returns true if there are blocking violations.
func (r *ComplianceReport) HasBlockingViolations() bool {
	return r.Summary.Status == ComplianceStatusNonCompliant
}

// ExpiringOverrides returns overrides that expire within the given days.
func (r *ComplianceReport) ExpiringOverrides(withinDays int) []OverrideDetail {
	var expiring []OverrideDetail
	for _, o := range r.Overrides {
		if o.DaysUntilExpiry >= 0 && o.DaysUntilExpiry <= withinDays {
			expiring = append(expiring, o)
		}
	}
	return expiring
}

// ComplianceReportService provides high-level compliance reporting.
type ComplianceReportService struct {
	evaluator *OrgEvaluator
	generator *ReportGenerator
}

// NewComplianceReportService creates a new compliance report service.
func NewComplianceReportService(policy *OrgPolicy) *ComplianceReportService {
	return &ComplianceReportService{
		evaluator: NewOrgEvaluator(policy),
		generator: NewReportGenerator(policy),
	}
}

// EvaluateAndReport evaluates items against the policy and generates a report.
func (s *ComplianceReportService) EvaluateAndReport(items []string) *ComplianceReport {
	result := s.evaluator.Evaluate(items)
	return s.generator.Generate(result, items)
}

// ComplianceDelta represents the difference between two compliance reports.
type ComplianceDelta struct {
	// NewViolations are violations that appeared since last report
	NewViolations []ViolationDetail `json:"new_violations,omitempty"`
	// ResolvedViolations are violations that were fixed since last report
	ResolvedViolations []ViolationDetail `json:"resolved_violations,omitempty"`
	// ScoreChange is the change in compliance score
	ScoreChange float64 `json:"score_change"`
	// StatusChange describes any status change
	StatusChange string `json:"status_change,omitempty"`
}

// CompareTo compares this report with a previous report.
func (r *ComplianceReport) CompareTo(previous *ComplianceReport) *ComplianceDelta {
	delta := &ComplianceDelta{
		ScoreChange: r.Summary.ComplianceScore - previous.Summary.ComplianceScore,
	}

	// Find new violations
	prevViolations := make(map[string]bool)
	for _, v := range previous.Violations {
		key := v.Type + ":" + v.Pattern + ":" + v.Value
		prevViolations[key] = true
	}
	for _, v := range r.Violations {
		key := v.Type + ":" + v.Pattern + ":" + v.Value
		if !prevViolations[key] {
			delta.NewViolations = append(delta.NewViolations, v)
		}
	}

	// Find resolved violations
	currViolations := make(map[string]bool)
	for _, v := range r.Violations {
		key := v.Type + ":" + v.Pattern + ":" + v.Value
		currViolations[key] = true
	}
	for _, v := range previous.Violations {
		key := v.Type + ":" + v.Pattern + ":" + v.Value
		if !currViolations[key] {
			delta.ResolvedViolations = append(delta.ResolvedViolations, v)
		}
	}

	// Describe status change
	if r.Summary.Status != previous.Summary.Status {
		delta.StatusChange = fmt.Sprintf("%s → %s", previous.Summary.Status, r.Summary.Status)
	}

	return delta
}
