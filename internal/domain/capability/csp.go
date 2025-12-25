package capability

import (
	"fmt"
	"regexp"
)

// CSPSeverity represents the severity of a CSP rule violation.
type CSPSeverity string

// CSP severity levels.
const (
	CSPSeverityDeny CSPSeverity = "deny"
	CSPSeverityWarn CSPSeverity = "warn"
)

// CSPRule represents a content security policy rule.
type CSPRule struct {
	Pattern  string
	Reason   string
	Severity CSPSeverity
	compiled *regexp.Regexp
}

// NewDenyRule creates a deny rule.
func NewDenyRule(pattern, reason string) CSPRule {
	return CSPRule{
		Pattern:  pattern,
		Reason:   reason,
		Severity: CSPSeverityDeny,
	}
}

// NewWarnRule creates a warn rule.
func NewWarnRule(pattern, reason string) CSPRule {
	return CSPRule{
		Pattern:  pattern,
		Reason:   reason,
		Severity: CSPSeverityWarn,
	}
}

// Compile compiles the pattern regex.
func (r *CSPRule) Compile() error {
	if r.compiled != nil {
		return nil
	}
	re, err := regexp.Compile(r.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", r.Pattern, err)
	}
	r.compiled = re
	return nil
}

// Match checks if the content matches this rule.
func (r *CSPRule) Match(content string) bool {
	if r.compiled == nil {
		if err := r.Compile(); err != nil {
			return false
		}
	}
	return r.compiled.MatchString(content)
}

// CSP represents a Content Security Policy for command validation.
type CSP struct {
	rules []CSPRule
}

// NewCSP creates an empty CSP.
func NewCSP() *CSP {
	return &CSP{
		rules: make([]CSPRule, 0),
	}
}

// AddRule adds a rule to the policy.
func (c *CSP) AddRule(rule CSPRule) error {
	if err := rule.Compile(); err != nil {
		return err
	}
	c.rules = append(c.rules, rule)
	return nil
}

// AddDeny adds a deny rule.
func (c *CSP) AddDeny(pattern, reason string) error {
	return c.AddRule(NewDenyRule(pattern, reason))
}

// AddWarn adds a warn rule.
func (c *CSP) AddWarn(pattern, reason string) error {
	return c.AddRule(NewWarnRule(pattern, reason))
}

// Validate checks content against all rules.
func (c *CSP) Validate(content string) *CSPResult {
	result := &CSPResult{
		Content:    content,
		Violations: make([]CSPViolation, 0),
	}

	for _, rule := range c.rules {
		if rule.Match(content) {
			result.Violations = append(result.Violations, CSPViolation{
				Rule:     rule,
				Content:  content,
				Severity: rule.Severity,
			})
		}
	}

	return result
}

// ValidateAll checks multiple content strings.
func (c *CSP) ValidateAll(contents []string) []*CSPResult {
	results := make([]*CSPResult, 0, len(contents))
	for _, content := range contents {
		result := c.Validate(content)
		if len(result.Violations) > 0 {
			results = append(results, result)
		}
	}
	return results
}

// RuleCount returns the number of rules.
func (c *CSP) RuleCount() int {
	return len(c.rules)
}

// CSPViolation represents a policy violation.
type CSPViolation struct {
	Rule     CSPRule
	Content  string
	Severity CSPSeverity
}

// CSPResult contains validation results for a single piece of content.
type CSPResult struct {
	Content    string
	Violations []CSPViolation
}

// IsAllowed returns true if no deny violations occurred.
func (r *CSPResult) IsAllowed() bool {
	for _, v := range r.Violations {
		if v.Severity == CSPSeverityDeny {
			return false
		}
	}
	return true
}

// HasWarnings returns true if there are warning violations.
func (r *CSPResult) HasWarnings() bool {
	for _, v := range r.Violations {
		if v.Severity == CSPSeverityWarn {
			return true
		}
	}
	return false
}

// DenyViolations returns only deny violations.
func (r *CSPResult) DenyViolations() []CSPViolation {
	var result []CSPViolation
	for _, v := range r.Violations {
		if v.Severity == CSPSeverityDeny {
			result = append(result, v)
		}
	}
	return result
}

// WarnViolations returns only warning violations.
func (r *CSPResult) WarnViolations() []CSPViolation {
	var result []CSPViolation
	for _, v := range r.Violations {
		if v.Severity == CSPSeverityWarn {
			result = append(result, v)
		}
	}
	return result
}

// DefaultCSP returns a CSP with common security rules.
func DefaultCSP() *CSP {
	csp := NewCSP()

	// Deny rules for dangerous patterns
	_ = csp.AddDeny(`curl\s+.*\|\s*(ba)?sh`, "Piped curl to shell is dangerous")
	_ = csp.AddDeny(`wget\s+.*\|\s*(ba)?sh`, "Piped wget to shell is dangerous")
	_ = csp.AddDeny(`chmod\s+.*777`, "World-writable permissions are dangerous")
	_ = csp.AddDeny(`sudo\s+`, "Sudo commands not allowed in presets")
	_ = csp.AddDeny(`rm\s+(-rf?|--recursive)\s+/[^/\s]*$`, "Recursive delete of root paths")
	_ = csp.AddDeny(`>\s*/etc/`, "Direct write to /etc not allowed")
	_ = csp.AddDeny(`>\s*/usr/`, "Direct write to /usr not allowed")

	// Warn rules for potentially problematic patterns
	_ = csp.AddWarn(`eval\s+`, "Use of eval may be dangerous")
	_ = csp.AddWarn(`\$\(.*\)`, "Command substitution should be reviewed")
	_ = csp.AddWarn(`source\s+/dev/stdin`, "Sourcing from stdin is suspicious")
	_ = csp.AddWarn(`base64\s+(-d|--decode)`, "Base64 decode may hide malicious content")

	return csp
}

// StrictCSP returns a stricter CSP for untrusted content.
func StrictCSP() *CSP {
	csp := DefaultCSP()

	// Additional strict rules
	_ = csp.AddDeny(`\$\{.*\}`, "Variable expansion not allowed in strict mode")
	_ = csp.AddDeny(`\|`, "Pipes not allowed in strict mode")
	_ = csp.AddDeny(`&`, "Background execution not allowed in strict mode")
	_ = csp.AddDeny(`;`, "Command chaining not allowed in strict mode")

	return csp
}
