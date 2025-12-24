// Package policy provides configuration policy constraints with deny/allow rules.
package policy

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Rule represents a single policy rule.
type Rule struct {
	// Pattern is a glob pattern to match against (e.g., "brew:*", "*.dmg")
	Pattern string `yaml:"pattern"`
	// Action is either "deny" or "allow"
	Action string `yaml:"action"`
	// Message is an optional explanation for why this rule exists
	Message string `yaml:"message,omitempty"`
	// Scope limits the rule to specific providers or contexts
	Scope string `yaml:"scope,omitempty"`
}

// Policy represents a set of policy constraints.
type Policy struct {
	// Name identifies this policy
	Name string `yaml:"name"`
	// Description explains the policy's purpose
	Description string `yaml:"description,omitempty"`
	// Rules are evaluated in order; first match wins
	Rules []Rule `yaml:"rules"`
}

// Violation represents a policy violation.
type Violation struct {
	// StepID is the step that violated the policy
	StepID string
	// Rule is the rule that was violated
	Rule Rule
	// Value is the value that matched the rule
	Value string
}

// Error returns the violation as an error message.
func (v *Violation) Error() string {
	msg := fmt.Sprintf("policy violation: %s is denied by rule %q", v.StepID, v.Rule.Pattern)
	if v.Rule.Message != "" {
		msg += fmt.Sprintf(" (%s)", v.Rule.Message)
	}
	return msg
}

// Result holds the outcome of policy evaluation.
type Result struct {
	Violations []Violation
	Allowed    []string
}

// HasViolations returns true if there are any policy violations.
func (r *Result) HasViolations() bool {
	return len(r.Violations) > 0
}

// Errors returns all violations as errors.
func (r *Result) Errors() []error {
	errs := make([]error, len(r.Violations))
	for i := range r.Violations {
		errs[i] = &r.Violations[i]
	}
	return errs
}

// Evaluator evaluates policies against values.
type Evaluator struct {
	policies []Policy
}

// NewEvaluator creates a new policy evaluator.
func NewEvaluator(policies ...Policy) *Evaluator {
	return &Evaluator{policies: policies}
}

// Evaluate checks the given values against all policies.
// Values should be in the format "provider:action" (e.g., "brew:install:git").
func (e *Evaluator) Evaluate(values []string) *Result {
	result := &Result{
		Violations: []Violation{},
		Allowed:    []string{},
	}

	for _, value := range values {
		violation := e.checkValue(value)
		if violation != nil {
			result.Violations = append(result.Violations, *violation)
		} else {
			result.Allowed = append(result.Allowed, value)
		}
	}

	return result
}

// EvaluateSteps checks step IDs against policies.
func (e *Evaluator) EvaluateSteps(stepIDs []string) *Result {
	return e.Evaluate(stepIDs)
}

// checkValue checks a single value against all policies.
// Returns a violation if denied, nil if allowed.
func (e *Evaluator) checkValue(value string) *Violation {
	for _, policy := range e.policies {
		for _, rule := range policy.Rules {
			if e.matchesRule(value, rule) {
				if rule.Action == "deny" {
					return &Violation{
						StepID: value,
						Rule:   rule,
						Value:  value,
					}
				}
				// Explicit allow - stop checking other rules
				return nil
			}
		}
	}
	// No matching rules - default allow
	return nil
}

// matchesRule checks if a value matches a rule's pattern and scope.
func (e *Evaluator) matchesRule(value string, rule Rule) bool {
	// Check scope first if specified
	if rule.Scope != "" {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) > 0 && !matchGlob(parts[0], rule.Scope) {
			return false
		}
	}

	return matchGlob(value, rule.Pattern)
}

// matchGlob performs glob-style pattern matching.
func matchGlob(value, pattern string) bool {
	// Use filepath.Match for glob matching
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	// Handle "**" patterns for nested matching
	if strings.Contains(pattern, "**") {
		// Convert ** to * for simple matching
		simplePattern := strings.ReplaceAll(pattern, "**", "*")
		matched, _ = filepath.Match(simplePattern, value)
		if matched {
			return true
		}
	}

	// Handle patterns like "provider:*" matching "provider:action:target"
	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}

	return false
}

// DefaultDenyAll creates a policy that denies everything by default.
func DefaultDenyAll(exceptions ...string) Policy {
	rules := make([]Rule, 0, len(exceptions)+1)
	for _, exception := range exceptions {
		rules = append(rules, Rule{
			Pattern: exception,
			Action:  "allow",
			Message: "explicitly allowed",
		})
	}
	rules = append(rules, Rule{
		Pattern: "*",
		Action:  "deny",
		Message: "default deny",
	})
	return Policy{
		Name:        "default-deny",
		Description: "Deny all except explicit allows",
		Rules:       rules,
	}
}

// DefaultAllowAll creates a policy that allows everything by default.
func DefaultAllowAll(denials ...string) Policy {
	rules := make([]Rule, 0, len(denials)+1)
	for _, denial := range denials {
		rules = append(rules, Rule{
			Pattern: denial,
			Action:  "deny",
			Message: "explicitly denied",
		})
	}
	rules = append(rules, Rule{
		Pattern: "*",
		Action:  "allow",
		Message: "default allow",
	})
	return Policy{
		Name:        "default-allow",
		Description: "Allow all except explicit denials",
		Rules:       rules,
	}
}
