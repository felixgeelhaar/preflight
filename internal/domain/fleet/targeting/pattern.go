// Package targeting provides host selection patterns and selectors for fleet operations.
package targeting

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// PatternType indicates the type of pattern matching.
type PatternType string

const (
	// PatternTypeGlob uses glob/wildcard matching (*, ?).
	PatternTypeGlob PatternType = "glob"
	// PatternTypeRegex uses regular expression matching.
	PatternTypeRegex PatternType = "regex"
	// PatternTypeLiteral matches exact strings.
	PatternTypeLiteral PatternType = "literal"
)

// Pattern represents a host matching pattern.
type Pattern struct {
	raw         string
	patternType PatternType
	regex       *regexp.Regexp
}

// NewPattern creates a pattern from a string.
// Patterns starting with ~ are treated as regex.
// Patterns containing * or ? are treated as glob.
// Otherwise, the pattern is a literal match.
func NewPattern(raw string) (*Pattern, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	p := &Pattern{raw: raw}

	switch {
	case strings.HasPrefix(raw, "~"):
		// Regex pattern
		p.patternType = PatternTypeRegex
		regex, err := regexp.Compile(raw[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		p.regex = regex
	case strings.ContainsAny(raw, "*?["):
		// Glob pattern
		p.patternType = PatternTypeGlob
		// Pre-validate the glob pattern
		_, err := filepath.Match(raw, "test")
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern: %w", err)
		}
	default:
		// Literal pattern
		p.patternType = PatternTypeLiteral
	}

	return p, nil
}

// Raw returns the original pattern string.
func (p *Pattern) Raw() string {
	return p.raw
}

// Type returns the pattern type.
func (p *Pattern) Type() PatternType {
	return p.patternType
}

// Match checks if a string matches the pattern.
func (p *Pattern) Match(s string) bool {
	switch p.patternType {
	case PatternTypeRegex:
		return p.regex.MatchString(s)
	case PatternTypeGlob:
		matched, _ := filepath.Match(p.raw, s)
		return matched
	case PatternTypeLiteral:
		return p.raw == s
	default:
		return false
	}
}

// MatchAny checks if any of the strings match the pattern.
func (p *Pattern) MatchAny(strings []string) bool {
	for _, s := range strings {
		if p.Match(s) {
			return true
		}
	}
	return false
}

// Patterns is a collection of patterns.
type Patterns []*Pattern

// NewPatterns creates a Patterns collection from strings.
func NewPatterns(raws ...string) (Patterns, error) {
	patterns := make(Patterns, 0, len(raws))
	for _, raw := range raws {
		p, err := NewPattern(raw)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, p)
	}
	return patterns, nil
}

// MatchAny returns true if any pattern matches the string.
func (ps Patterns) MatchAny(s string) bool {
	for _, p := range ps {
		if p.Match(s) {
			return true
		}
	}
	return false
}

// MatchAll returns true if all patterns match the string.
func (ps Patterns) MatchAll(s string) bool {
	for _, p := range ps {
		if !p.Match(s) {
			return false
		}
	}
	return true
}

// FilterMatching returns all strings that match any pattern.
func (ps Patterns) FilterMatching(strings []string) []string {
	var result []string
	for _, s := range strings {
		if ps.MatchAny(s) {
			result = append(result, s)
		}
	}
	return result
}
