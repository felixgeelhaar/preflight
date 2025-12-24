// Package policy provides configuration policy constraints with deny/allow rules.
package policy

import (
	"fmt"
	"time"
)

// EnforcementMode determines how policy violations are handled.
type EnforcementMode string

const (
	// EnforcementWarn logs violations as warnings but allows execution.
	EnforcementWarn EnforcementMode = "warn"
	// EnforcementBlock prevents execution when violations are detected.
	EnforcementBlock EnforcementMode = "block"
)

// Requirement represents something that must be present.
type Requirement struct {
	// Pattern is a glob pattern for required items (e.g., "git:*", "layer:base")
	Pattern string `yaml:"pattern"`
	// Message explains why this is required
	Message string `yaml:"message,omitempty"`
	// Scope limits the requirement to specific contexts
	Scope string `yaml:"scope,omitempty"`
}

// Forbidden represents something that must not be present.
type Forbidden struct {
	// Pattern is a glob pattern for forbidden items
	Pattern string `yaml:"pattern"`
	// Message explains why this is forbidden
	Message string `yaml:"message,omitempty"`
	// Scope limits the rule to specific contexts
	Scope string `yaml:"scope,omitempty"`
}

// Override represents an exception to policy rules.
type Override struct {
	// Pattern is the pattern being overridden
	Pattern string `yaml:"pattern"`
	// Justification explains why this override exists
	Justification string `yaml:"justification"`
	// ApprovedBy identifies who approved the override
	ApprovedBy string `yaml:"approved_by,omitempty"`
	// ExpiresAt is when the override expires (RFC3339 format)
	ExpiresAt string `yaml:"expires_at,omitempty"`
}

// OrgPolicy represents organizational policy constraints.
type OrgPolicy struct {
	// Name identifies this org policy
	Name string `yaml:"name"`
	// Description explains the policy's purpose
	Description string `yaml:"description,omitempty"`
	// Enforcement determines warn vs block behavior
	Enforcement EnforcementMode `yaml:"enforcement"`
	// Required patterns that must be present
	Required []Requirement `yaml:"required,omitempty"`
	// Forbidden patterns that must not be present
	Forbidden []Forbidden `yaml:"forbidden,omitempty"`
	// Overrides allow exceptions to policy rules
	Overrides []Override `yaml:"overrides,omitempty"`
}

// OrgViolation represents an org policy violation.
type OrgViolation struct {
	// Type is either "missing_required" or "forbidden_present"
	Type string
	// Pattern that was violated
	Pattern string
	// Message explains the violation
	Message string
	// Value is the actual value that triggered the violation (for forbidden)
	Value string
}

// Error returns the violation as an error message.
func (v *OrgViolation) Error() string {
	if v.Type == "missing_required" {
		msg := fmt.Sprintf("missing required: %s", v.Pattern)
		if v.Message != "" {
			msg += fmt.Sprintf(" (%s)", v.Message)
		}
		return msg
	}
	msg := fmt.Sprintf("forbidden: %s matches %s", v.Value, v.Pattern)
	if v.Message != "" {
		msg += fmt.Sprintf(" (%s)", v.Message)
	}
	return msg
}

// OrgResult holds the outcome of org policy evaluation.
type OrgResult struct {
	// Violations is the list of policy violations
	Violations []OrgViolation
	// Warnings is violations when enforcement is "warn"
	Warnings []OrgViolation
	// Enforcement mode that was used
	Enforcement EnforcementMode
	// OverridesApplied lists which overrides were used
	OverridesApplied []Override
}

// HasViolations returns true if there are blocking violations.
func (r *OrgResult) HasViolations() bool {
	return r.Enforcement == EnforcementBlock && len(r.Violations) > 0
}

// HasWarnings returns true if there are warnings.
func (r *OrgResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// AllIssues returns all violations and warnings combined.
func (r *OrgResult) AllIssues() []OrgViolation {
	all := make([]OrgViolation, 0, len(r.Violations)+len(r.Warnings))
	all = append(all, r.Violations...)
	all = append(all, r.Warnings...)
	return all
}

// Errors returns violations as errors.
func (r *OrgResult) Errors() []error {
	errs := make([]error, len(r.Violations))
	for i := range r.Violations {
		errs[i] = &r.Violations[i]
	}
	return errs
}

// OrgEvaluator evaluates org policies.
type OrgEvaluator struct {
	policy *OrgPolicy
}

// NewOrgEvaluator creates a new org policy evaluator.
func NewOrgEvaluator(policy *OrgPolicy) *OrgEvaluator {
	return &OrgEvaluator{policy: policy}
}

// Evaluate checks the given values against the org policy.
func (e *OrgEvaluator) Evaluate(values []string) *OrgResult {
	result := &OrgResult{
		Violations:       []OrgViolation{},
		Warnings:         []OrgViolation{},
		Enforcement:      EnforcementBlock,
		OverridesApplied: []Override{},
	}

	if e.policy == nil {
		return result
	}

	result.Enforcement = e.policy.Enforcement

	// Check required patterns
	for _, req := range e.policy.Required {
		if !e.hasMatch(values, req.Pattern, req.Scope) {
			violation := OrgViolation{
				Type:    "missing_required",
				Pattern: req.Pattern,
				Message: req.Message,
			}
			e.addViolation(result, violation)
		}
	}

	// Check forbidden patterns
	for _, forbidden := range e.policy.Forbidden {
		for _, value := range values {
			if e.matchesForbidden(value, forbidden) {
				// Check for override
				if override := e.findOverride(value); override != nil {
					if e.isOverrideValid(override) {
						result.OverridesApplied = append(result.OverridesApplied, *override)
						continue
					}
				}
				violation := OrgViolation{
					Type:    "forbidden_present",
					Pattern: forbidden.Pattern,
					Message: forbidden.Message,
					Value:   value,
				}
				e.addViolation(result, violation)
			}
		}
	}

	return result
}

// addViolation adds a violation to the appropriate list based on enforcement.
func (e *OrgEvaluator) addViolation(result *OrgResult, violation OrgViolation) {
	if e.policy.Enforcement == EnforcementWarn {
		result.Warnings = append(result.Warnings, violation)
	} else {
		result.Violations = append(result.Violations, violation)
	}
}

// hasMatch checks if any value matches the required pattern.
func (e *OrgEvaluator) hasMatch(values []string, pattern, scope string) bool {
	for _, value := range values {
		if scope != "" {
			// Extract provider from value (first part before :)
			parts := splitFirst(value, ":")
			if len(parts) > 0 && !matchGlob(parts[0], scope) {
				continue
			}
		}
		if matchGlob(value, pattern) {
			return true
		}
	}
	return false
}

// matchesForbidden checks if a value matches a forbidden rule.
func (e *OrgEvaluator) matchesForbidden(value string, forbidden Forbidden) bool {
	if forbidden.Scope != "" {
		parts := splitFirst(value, ":")
		if len(parts) > 0 && !matchGlob(parts[0], forbidden.Scope) {
			return false
		}
	}
	return matchGlob(value, forbidden.Pattern)
}

// findOverride finds an override for the given value.
func (e *OrgEvaluator) findOverride(value string) *Override {
	for i := range e.policy.Overrides {
		if matchGlob(value, e.policy.Overrides[i].Pattern) {
			return &e.policy.Overrides[i]
		}
	}
	return nil
}

// isOverrideValid checks if an override is still valid (not expired).
func (e *OrgEvaluator) isOverrideValid(override *Override) bool {
	if override.ExpiresAt == "" {
		return true // No expiration
	}
	expiresAt, err := time.Parse(time.RFC3339, override.ExpiresAt)
	if err != nil {
		return false // Invalid date format = expired
	}
	return time.Now().Before(expiresAt)
}

// splitFirst splits a string on the first occurrence of sep.
func splitFirst(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}
