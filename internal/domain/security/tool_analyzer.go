package security

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog/tools"
)

// FindingType indicates the type of tool finding.
type FindingType string

const (
	// FindingRedundancy indicates overlapping tools where one supersedes another.
	FindingRedundancy FindingType = "redundancy"
	// FindingDeprecated indicates a deprecated tool.
	FindingDeprecated FindingType = "deprecated"
	// FindingConsolidation indicates tools that can be consolidated.
	FindingConsolidation FindingType = "consolidation"
	// FindingUnknown indicates an unknown tool not in the knowledge base.
	FindingUnknown FindingType = "unknown"
)

// FindingSeverity indicates the severity of a tool finding.
type FindingSeverity string

const (
	// SeverityInfo is informational only.
	SeverityInfo FindingSeverity = "info"
	// SeverityWarning suggests action should be taken.
	SeverityWarning FindingSeverity = "warning"
	// SeverityError indicates a problem that should be fixed.
	SeverityError FindingSeverity = "error"
)

// ToolFinding represents a finding from tool analysis.
type ToolFinding struct {
	Type        FindingType     `json:"type"`
	Severity    FindingSeverity `json:"severity"`
	Tools       []string    `json:"tools"`
	Message     string      `json:"message"`
	Suggestion  string      `json:"suggestion,omitempty"`
	Replacement string      `json:"replacement,omitempty"`
	Category    string      `json:"category,omitempty"`
	Docs        string      `json:"docs,omitempty"`
}

// ToolAnalysisResult contains the results of tool analysis.
type ToolAnalysisResult struct {
	Findings       []ToolFinding `json:"findings"`
	ToolsAnalyzed  int           `json:"tools_analyzed"`
	IssuesFound    int           `json:"issues_found"`
	Consolidations int           `json:"consolidations"`
}

// ToolAnalyzer analyzes tools for redundancy, deprecation, and consolidation opportunities.
type ToolAnalyzer interface {
	Analyze(ctx context.Context, toolNames []string) (*ToolAnalysisResult, error)
}

// toolAnalyzer is the default implementation of ToolAnalyzer.
type toolAnalyzer struct {
	kb tools.KnowledgeBase
}

// NewToolAnalyzer creates a new ToolAnalyzer with the given knowledge base.
func NewToolAnalyzer(kb tools.KnowledgeBase) ToolAnalyzer {
	return &toolAnalyzer{kb: kb}
}

// Analyze analyzes the given tools for issues.
func (a *toolAnalyzer) Analyze(ctx context.Context, toolNames []string) (*ToolAnalysisResult, error) {
	result := &ToolAnalysisResult{
		Findings:      make([]ToolFinding, 0),
		ToolsAnalyzed: len(toolNames),
	}

	if len(toolNames) == 0 {
		return result, nil
	}

	// Create a set for fast lookup
	toolSet := make(map[string]bool)
	for _, name := range toolNames {
		toolSet[name] = true
	}

	// Check for deprecations
	deprecations := a.findDeprecations(toolNames)
	result.Findings = append(result.Findings, deprecations...)

	// Check for redundancies (tool present that is superseded by another present tool)
	redundancies := a.findRedundancies(toolNames, toolSet)
	result.Findings = append(result.Findings, redundancies...)

	// Check for consolidation opportunities (only if consolidating tool not already present)
	consolidations := a.findConsolidations(toolNames, toolSet)
	result.Findings = append(result.Findings, consolidations...)
	result.Consolidations = len(consolidations)

	// Count issues (warnings and errors only)
	for _, f := range result.Findings {
		if f.Severity == SeverityWarning || f.Severity == SeverityError {
			result.IssuesFound++
		}
	}

	// Sort findings by severity (error > warning > info) then by type
	sort.Slice(result.Findings, func(i, j int) bool {
		si := findingSeverityOrder(result.Findings[i].Severity)
		sj := findingSeverityOrder(result.Findings[j].Severity)
		if si != sj {
			return si < sj
		}
		return result.Findings[i].Type < result.Findings[j].Type
	})

	return result, nil
}

// findDeprecations finds deprecated tools.
func (a *toolAnalyzer) findDeprecations(toolNames []string) []ToolFinding {
	findings := make([]ToolFinding, 0)

	for _, name := range toolNames {
		tool, found := a.kb.GetTool(name)
		if !found || !tool.Deprecated {
			continue
		}

		message := fmt.Sprintf("%s is deprecated", name)
		if tool.DeprecatedSince != "" {
			message = fmt.Sprintf("%s is deprecated (since %s)", name, tool.DeprecatedSince)
		}

		suggestion := ""
		if tool.Successor != "" {
			suggestion = fmt.Sprintf("Replace with %s", tool.Successor)
		}

		finding := ToolFinding{
			Type:        FindingDeprecated,
			Severity:    SeverityWarning,
			Tools:       []string{name},
			Message:     message,
			Suggestion:  suggestion,
			Replacement: tool.Successor,
			Category:    tool.Category,
		}

		if tool.Reason != "" {
			finding.Message = fmt.Sprintf("%s: %s", message, tool.Reason)
		}

		if tool.Docs != "" {
			finding.Docs = tool.Docs
		}

		findings = append(findings, finding)
	}

	return findings
}

// findRedundancies finds tools that are superseded by other tools in the list.
func (a *toolAnalyzer) findRedundancies(toolNames []string, toolSet map[string]bool) []ToolFinding {
	findings := make([]ToolFinding, 0)
	reported := make(map[string]bool) // Avoid duplicate reports

	for _, name := range toolNames {
		if reported[name] {
			continue
		}

		// Check if this tool is superseded by another tool in the list
		superseder, found := a.kb.FindSupersededBy(name)
		if !found {
			continue
		}

		// Check if the superseding tool is in our list
		if !toolSet[superseder.Name] {
			continue
		}

		// Found redundancy
		reported[name] = true

		// Find the reason from the superseder's supersedes list
		reason := ""
		for _, entry := range superseder.Supersedes {
			if entry.Tool == name {
				reason = entry.Reason
				break
			}
		}

		message := fmt.Sprintf("%s is redundant with %s", name, superseder.Name)
		suggestion := fmt.Sprintf("Remove %s (functionality covered by %s)", name, superseder.Name)
		if reason != "" {
			suggestion = fmt.Sprintf("Remove %s (%s is covered by %s)", name, reason, superseder.Name)
		}

		finding := ToolFinding{
			Type:        FindingRedundancy,
			Severity:    SeverityWarning,
			Tools:       []string{name},
			Message:     message,
			Suggestion:  suggestion,
			Replacement: superseder.Name,
			Category:    superseder.Category,
		}

		if superseder.Docs != "" {
			finding.Docs = superseder.Docs
		}

		findings = append(findings, finding)
	}

	return findings
}

// findConsolidations finds opportunities to consolidate multiple tools into one.
func (a *toolAnalyzer) findConsolidations(toolNames []string, toolSet map[string]bool) []ToolFinding {
	findings := make([]ToolFinding, 0)

	suggestion, found := a.kb.FindConsolidationTarget(toolNames)
	if !found {
		return findings
	}

	// Don't suggest consolidation if the target tool is already present
	if toolSet[suggestion.Target] {
		return findings
	}

	// Build a human-readable message
	toolList := strings.Join(suggestion.ReplacedTools, ", ")
	message := fmt.Sprintf("[%s] can be consolidated to %s", toolList, suggestion.Target)

	suggestionText := fmt.Sprintf("Replace with %s for simplified toolchain", suggestion.Target)
	if suggestion.CoveragePercent < 1.0 {
		coverage := int(suggestion.CoveragePercent * 100)
		suggestionText = fmt.Sprintf("Replace with %s (%d%% capability coverage)", suggestion.Target, coverage)
	}

	finding := ToolFinding{
		Type:        FindingConsolidation,
		Severity:    SeverityInfo,
		Tools:       suggestion.ReplacedTools,
		Message:     message,
		Suggestion:  suggestionText,
		Replacement: suggestion.Target,
	}

	// Get docs from target tool
	if targetTool, ok := a.kb.GetTool(suggestion.Target); ok {
		finding.Category = targetTool.Category
		finding.Docs = targetTool.Docs
	}

	findings = append(findings, finding)
	return findings
}

// findingSeverityOrder returns a sort order for finding severities (lower = more severe).
func findingSeverityOrder(s FindingSeverity) int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}
