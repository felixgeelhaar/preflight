package catalog

import (
	"regexp"
	"strings"
)

// AuditSeverity represents the severity of an audit finding.
type AuditSeverity string

// AuditSeverity constants.
const (
	AuditSeverityCritical AuditSeverity = "critical"
	AuditSeverityHigh     AuditSeverity = "high"
	AuditSeverityMedium   AuditSeverity = "medium"
	AuditSeverityLow      AuditSeverity = "low"
	AuditSeverityInfo     AuditSeverity = "info"
)

// AuditFinding represents a security finding from the audit.
type AuditFinding struct {
	Severity   AuditSeverity
	Category   string
	Message    string
	Location   string
	Suggestion string
}

// AuditResult contains the results of a security audit.
type AuditResult struct {
	CatalogName string
	Findings    []AuditFinding
	Passed      bool
}

// CriticalCount returns the number of critical findings.
func (r AuditResult) CriticalCount() int {
	return r.countBySeverity(AuditSeverityCritical)
}

// HighCount returns the number of high severity findings.
func (r AuditResult) HighCount() int {
	return r.countBySeverity(AuditSeverityHigh)
}

// MediumCount returns the number of medium severity findings.
func (r AuditResult) MediumCount() int {
	return r.countBySeverity(AuditSeverityMedium)
}

// LowCount returns the number of low severity findings.
func (r AuditResult) LowCount() int {
	return r.countBySeverity(AuditSeverityLow)
}

func (r AuditResult) countBySeverity(severity AuditSeverity) int {
	count := 0
	for _, f := range r.Findings {
		if f.Severity == severity {
			count++
		}
	}
	return count
}

// Auditor performs security audits on catalogs.
type Auditor struct {
	rules []auditRule
}

// auditRule represents a security check rule.
type auditRule struct {
	name       string
	category   string
	severity   AuditSeverity
	pattern    *regexp.Regexp
	message    string
	suggestion string
}

// NewAuditor creates a new auditor with default rules.
func NewAuditor() *Auditor {
	return &Auditor{
		rules: defaultAuditRules(),
	}
}

// defaultAuditRules returns the default security audit rules.
func defaultAuditRules() []auditRule {
	return []auditRule{
		// Critical: Remote code execution patterns
		{
			name:       "curl-pipe-shell",
			category:   "code-execution",
			severity:   AuditSeverityCritical,
			pattern:    regexp.MustCompile(`curl\s+.*\|\s*(ba)?sh`),
			message:    "Piped curl to shell detected - potential remote code execution",
			suggestion: "Download scripts first, review them, then execute separately",
		},
		{
			name:       "wget-pipe-shell",
			category:   "code-execution",
			severity:   AuditSeverityCritical,
			pattern:    regexp.MustCompile(`wget\s+.*\|\s*(ba)?sh`),
			message:    "Piped wget to shell detected - potential remote code execution",
			suggestion: "Download scripts first, review them, then execute separately",
		},

		// High: Dangerous operations
		{
			name:       "sudo-usage",
			category:   "privilege-escalation",
			severity:   AuditSeverityHigh,
			pattern:    regexp.MustCompile(`sudo\s+`),
			message:    "Sudo usage detected - requires elevated privileges",
			suggestion: "Avoid sudo in presets; document privilege requirements instead",
		},
		{
			name:       "rm-rf-root",
			category:   "destructive",
			severity:   AuditSeverityCritical,
			pattern:    regexp.MustCompile(`rm\s+-rf\s+/[^/\s]*\s`),
			message:    "Recursive deletion of system paths detected",
			suggestion: "Use specific paths and avoid wildcards near root",
		},
		{
			name:       "chmod-777",
			category:   "permissions",
			severity:   AuditSeverityHigh,
			pattern:    regexp.MustCompile(`chmod\s+777`),
			message:    "World-writable permissions detected",
			suggestion: "Use more restrictive permissions (e.g., 755 or 644)",
		},

		// Medium: Potentially unsafe patterns
		{
			name:       "eval-usage",
			category:   "code-execution",
			severity:   AuditSeverityMedium,
			pattern:    regexp.MustCompile(`\beval\s+`),
			message:    "Eval usage detected - can execute arbitrary code",
			suggestion: "Avoid eval; use explicit commands instead",
		},
		{
			name:       "env-secrets",
			category:   "secrets",
			severity:   AuditSeverityMedium,
			pattern:    regexp.MustCompile(`(?i)(password|secret|api_key|token)\s*[=:]\s*[^\s]+`),
			message:    "Potential hardcoded secret detected",
			suggestion: "Use environment variables or secret references instead",
		},
		{
			name:       "private-key",
			category:   "secrets",
			severity:   AuditSeverityHigh,
			pattern:    regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
			message:    "Private key content detected",
			suggestion: "Never include private keys in catalogs",
		},

		// Low: Best practice violations
		{
			name:       "hardcoded-path",
			category:   "portability",
			severity:   AuditSeverityLow,
			pattern:    regexp.MustCompile(`/Users/[^/]+/`),
			message:    "Hardcoded user path detected",
			suggestion: "Use $HOME or ~ for user-relative paths",
		},
		{
			name:       "external-script",
			category:   "transparency",
			severity:   AuditSeverityMedium,
			pattern:    regexp.MustCompile(`https?://[^\s]+\.(sh|bash|zsh)`),
			message:    "Reference to external shell script",
			suggestion: "Include scripts in catalog or document external dependencies",
		},
	}
}

// Audit performs a security audit on a registered catalog.
func (a *Auditor) Audit(rc *RegisteredCatalog) AuditResult {
	result := AuditResult{
		CatalogName: rc.Name(),
		Findings:    []AuditFinding{},
		Passed:      true,
	}

	// Audit manifest metadata
	a.auditManifest(rc.Manifest(), &result)

	// Audit presets
	for _, preset := range rc.Catalog().ListPresets() {
		a.auditPreset(preset, &result)
	}

	// Audit capability packs
	for _, pack := range rc.Catalog().ListPacks() {
		a.auditPack(pack, &result)
	}

	// Determine if audit passed (no critical or high findings)
	result.Passed = result.CriticalCount() == 0 && result.HighCount() == 0

	return result
}

// auditManifest audits the manifest metadata.
func (a *Auditor) auditManifest(manifest Manifest, result *AuditResult) {
	// Check for missing metadata
	if manifest.Author() == "" {
		result.Findings = append(result.Findings, AuditFinding{
			Severity:   AuditSeverityInfo,
			Category:   "metadata",
			Message:    "Missing author information",
			Location:   "manifest",
			Suggestion: "Add author to catalog manifest for accountability",
		})
	}

	if manifest.Repository() == "" {
		result.Findings = append(result.Findings, AuditFinding{
			Severity:   AuditSeverityLow,
			Category:   "metadata",
			Message:    "Missing repository URL",
			Location:   "manifest",
			Suggestion: "Add repository URL for source verification",
		})
	}

	if manifest.License() == "" {
		result.Findings = append(result.Findings, AuditFinding{
			Severity:   AuditSeverityInfo,
			Category:   "metadata",
			Message:    "Missing license information",
			Location:   "manifest",
			Suggestion: "Add license to clarify usage terms",
		})
	}
}

// auditPreset audits a preset's configuration.
func (a *Auditor) auditPreset(preset Preset, result *AuditResult) {
	// Convert config to string for pattern matching
	configStr := configToString(preset.Config())
	location := "preset:" + preset.ID().String()

	for _, rule := range a.rules {
		if rule.pattern.MatchString(configStr) {
			result.Findings = append(result.Findings, AuditFinding{
				Severity:   rule.severity,
				Category:   rule.category,
				Message:    rule.message,
				Location:   location,
				Suggestion: rule.suggestion,
			})
		}
	}
}

// auditPack audits a capability pack.
func (a *Auditor) auditPack(pack CapabilityPack, result *AuditResult) {
	location := "pack:" + pack.ID()

	// Check for external tools with potential risks
	for _, tool := range pack.Tools() {
		// Check tool names against patterns
		toolLower := strings.ToLower(tool)
		if strings.Contains(toolLower, "hack") || strings.Contains(toolLower, "exploit") {
			result.Findings = append(result.Findings, AuditFinding{
				Severity:   AuditSeverityMedium,
				Category:   "suspicious-tool",
				Message:    "Tool with potentially suspicious name: " + tool,
				Location:   location,
				Suggestion: "Verify this tool is legitimate and intended",
			})
		}
	}
}

// configToString converts a config map to a string for pattern matching.
func configToString(config map[string]interface{}) string {
	var sb strings.Builder
	configToStringRecursive(config, &sb)
	return sb.String()
}

func configToStringRecursive(v interface{}, sb *strings.Builder) {
	switch val := v.(type) {
	case string:
		sb.WriteString(val)
		sb.WriteString("\n")
	case []interface{}:
		for _, item := range val {
			configToStringRecursive(item, sb)
		}
	case map[string]interface{}:
		for _, item := range val {
			configToStringRecursive(item, sb)
		}
	}
}
