// Package tools provides a knowledge base for tool intelligence including
// categories, capabilities, supersedes relationships, and deprecation info.
package tools

import (
	"embed"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

//go:embed tools.yaml
var embeddedFS embed.FS

// Tool represents metadata about a development tool.
type Tool struct {
	Name            string            `yaml:"-"` // Set from map key
	Category        string            `yaml:"category"`
	Capabilities    []string          `yaml:"capabilities"`
	Supersedes      []SupersedesEntry `yaml:"supersedes"`
	Deprecated      bool              `yaml:"deprecated"`
	DeprecatedSince string            `yaml:"deprecated_since"`
	Successor       string            `yaml:"successor"`
	Reason          string            `yaml:"reason"`
	Docs            string            `yaml:"docs"`
}

// SupersedesEntry represents a tool that is superseded by another.
type SupersedesEntry struct {
	Tool   string `yaml:"tool"`
	Reason string `yaml:"reason"`
}

// ConsolidationSuggestion represents a recommendation to consolidate multiple tools.
type ConsolidationSuggestion struct {
	Target          string   // Tool that can replace others
	ReplacedTools   []string // Tools that can be replaced
	CoveragePercent float64  // Percentage of capabilities covered
	UncoveredCaps   []string // Capabilities not covered by the target
}

// KnowledgeBase provides tool intelligence for configuration analysis.
type KnowledgeBase interface {
	// GetTool returns a tool by name.
	GetTool(name string) (*Tool, bool)

	// GetToolsByCategory returns all tools in a category.
	GetToolsByCategory(category string) []*Tool

	// AllTools returns all tools in the knowledge base.
	AllTools() []*Tool

	// GetDeprecatedTools returns all deprecated tools.
	GetDeprecatedTools() []*Tool

	// FindSupersedes returns the supersedes entries for a tool.
	FindSupersedes(tool string) []SupersedesEntry

	// FindSupersededBy returns the tool that supersedes the given tool.
	FindSupersededBy(tool string) (*Tool, bool)

	// FindConsolidationTarget finds a single tool that can replace multiple tools.
	FindConsolidationTarget(tools []string) (*ConsolidationSuggestion, bool)
}

// knowledgeBase is the in-memory implementation of KnowledgeBase.
type knowledgeBase struct {
	tools          map[string]*Tool
	byCategory     map[string][]*Tool
	supersededByMap map[string]string // tool -> superseding tool
}

// knowledgeBaseFile represents the YAML file structure.
type knowledgeBaseFile struct {
	Tools map[string]*Tool `yaml:"tools"`
}

// LoadKnowledgeBase loads the embedded knowledge base from tools.yaml.
func LoadKnowledgeBase() (KnowledgeBase, error) {
	data, err := embeddedFS.ReadFile("tools.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded tools.yaml: %w", err)
	}
	return ParseKnowledgeBase(data)
}

// ParseKnowledgeBase parses YAML data into a KnowledgeBase.
func ParseKnowledgeBase(data []byte) (KnowledgeBase, error) {
	var file knowledgeBaseFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse tools YAML: %w", err)
	}

	kb := &knowledgeBase{
		tools:           make(map[string]*Tool),
		byCategory:      make(map[string][]*Tool),
		supersededByMap: make(map[string]string),
	}

	// Process tools and set names from map keys
	for name, tool := range file.Tools {
		if tool == nil {
			tool = &Tool{}
		}
		tool.Name = name
		kb.tools[name] = tool

		// Index by category
		if tool.Category != "" {
			kb.byCategory[tool.Category] = append(kb.byCategory[tool.Category], tool)
		}

		// Build supersededBy index
		for _, entry := range tool.Supersedes {
			kb.supersededByMap[entry.Tool] = name
		}
	}

	return kb, nil
}

// GetTool returns a tool by name.
func (kb *knowledgeBase) GetTool(name string) (*Tool, bool) {
	tool, found := kb.tools[name]
	return tool, found
}

// GetToolsByCategory returns all tools in a category.
func (kb *knowledgeBase) GetToolsByCategory(category string) []*Tool {
	tools := kb.byCategory[category]
	if tools == nil {
		return []*Tool{}
	}
	// Return a copy to prevent modification
	result := make([]*Tool, len(tools))
	copy(result, tools)
	return result
}

// AllTools returns all tools in the knowledge base.
func (kb *knowledgeBase) AllTools() []*Tool {
	result := make([]*Tool, 0, len(kb.tools))
	for _, tool := range kb.tools {
		result = append(result, tool)
	}
	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetDeprecatedTools returns all deprecated tools.
func (kb *knowledgeBase) GetDeprecatedTools() []*Tool {
	result := make([]*Tool, 0)
	for _, tool := range kb.tools {
		if tool.Deprecated {
			result = append(result, tool)
		}
	}
	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// FindSupersedes returns the supersedes entries for a tool.
func (kb *knowledgeBase) FindSupersedes(name string) []SupersedesEntry {
	tool, found := kb.tools[name]
	if !found {
		return []SupersedesEntry{}
	}
	// Return a copy
	result := make([]SupersedesEntry, len(tool.Supersedes))
	copy(result, tool.Supersedes)
	return result
}

// FindSupersededBy returns the tool that supersedes the given tool.
func (kb *knowledgeBase) FindSupersededBy(name string) (*Tool, bool) {
	supersederName, found := kb.supersededByMap[name]
	if !found {
		return nil, false
	}
	return kb.GetTool(supersederName)
}

// FindConsolidationTarget finds a single tool that can replace multiple tools.
func (kb *knowledgeBase) FindConsolidationTarget(inputTools []string) (*ConsolidationSuggestion, bool) {
	// Need at least 2 tools to consolidate
	if len(inputTools) < 2 {
		return nil, false
	}

	// Create a set of input tools for fast lookup
	inputSet := make(map[string]bool)
	for _, t := range inputTools {
		inputSet[t] = true
	}

	// Find potential consolidation targets
	// A tool is a target if it supersedes at least 2 of the input tools
	candidates := make(map[string][]string) // candidate -> replaced tools

	for _, inputTool := range inputTools {
		superseder, found := kb.FindSupersededBy(inputTool)
		if found {
			candidates[superseder.Name] = append(candidates[superseder.Name], inputTool)
		}
	}

	// Find the best candidate (one that replaces the most tools)
	var bestCandidate string
	var bestReplaced []string
	for candidate, replaced := range candidates {
		if len(replaced) >= 2 && len(replaced) > len(bestReplaced) {
			bestCandidate = candidate
			bestReplaced = replaced
		}
	}

	if bestCandidate == "" {
		return nil, false
	}

	// Calculate coverage
	targetTool, _ := kb.GetTool(bestCandidate)
	targetCaps := make(map[string]bool)
	for _, cap := range targetTool.Capabilities {
		targetCaps[cap] = true
	}

	// Collect all capabilities from replaced tools
	var allReplacedCaps []string
	for _, replacedName := range bestReplaced {
		if replacedTool, found := kb.GetTool(replacedName); found {
			allReplacedCaps = append(allReplacedCaps, replacedTool.Capabilities...)
		}
	}

	// Remove duplicates from replaced caps
	replacedCapsSet := make(map[string]bool)
	for _, cap := range allReplacedCaps {
		replacedCapsSet[cap] = true
	}

	// Calculate coverage
	coveredCount := 0
	var uncovered []string
	for cap := range replacedCapsSet {
		if targetCaps[cap] {
			coveredCount++
		} else {
			uncovered = append(uncovered, cap)
		}
	}

	coverage := 1.0
	if len(replacedCapsSet) > 0 {
		coverage = float64(coveredCount) / float64(len(replacedCapsSet))
	}

	return &ConsolidationSuggestion{
		Target:          bestCandidate,
		ReplacedTools:   bestReplaced,
		CoveragePercent: coverage,
		UncoveredCaps:   uncovered,
	}, true
}
