package config

import (
	"os"
	"regexp"
	"runtime"
	"strings"
)

// Condition represents a conditional expression for layer application.
type Condition struct {
	OS       string      `yaml:"os,omitempty"`       // Operating system (darwin, linux, windows)
	Arch     string      `yaml:"arch,omitempty"`     // Architecture (amd64, arm64)
	Hostname string      `yaml:"hostname,omitempty"` // Hostname pattern (supports glob)
	EnvVar   string      `yaml:"env,omitempty"`      // Environment variable check (VAR or VAR=value)
	EnvSet   string      `yaml:"env_set,omitempty"`  // Environment variable exists
	EnvUnset string      `yaml:"env_unset,omitempty"`
	Command  string      `yaml:"command,omitempty"` // Command exists in PATH
	File     string      `yaml:"file,omitempty"`    // File/directory exists
	Not      *Condition  `yaml:"not,omitempty"`     // Negate condition
	All      []Condition `yaml:"all,omitempty"`     // All conditions must match
	Any      []Condition `yaml:"any,omitempty"`     // Any condition must match
}

// ConditionalLayer extends a layer reference with conditions.
type ConditionalLayer struct {
	Name   string    `yaml:"name"`
	When   Condition `yaml:"when,omitempty"`
	Unless Condition `yaml:"unless,omitempty"`
}

// ConditionEvaluator evaluates conditions against the current environment.
type ConditionEvaluator struct {
	hostname string
	osName   string
	arch     string
}

// NewConditionEvaluator creates a new condition evaluator.
func NewConditionEvaluator() *ConditionEvaluator {
	hostname, _ := os.Hostname()
	return &ConditionEvaluator{
		hostname: hostname,
		osName:   runtime.GOOS,
		arch:     runtime.GOARCH,
	}
}

// Evaluate checks if a condition is satisfied.
func (e *ConditionEvaluator) Evaluate(c Condition) bool {
	// Empty condition is always true
	if e.isEmpty(c) {
		return true
	}

	// Check OS
	if c.OS != "" && !e.matchOS(c.OS) {
		return false
	}

	// Check architecture
	if c.Arch != "" && !e.matchArch(c.Arch) {
		return false
	}

	// Check hostname
	if c.Hostname != "" && !e.matchHostname(c.Hostname) {
		return false
	}

	// Check environment variable
	if c.EnvVar != "" && !e.matchEnvVar(c.EnvVar) {
		return false
	}

	// Check environment variable exists
	if c.EnvSet != "" && !e.matchEnvSet(c.EnvSet) {
		return false
	}

	// Check environment variable not set
	if c.EnvUnset != "" && !e.matchEnvUnset(c.EnvUnset) {
		return false
	}

	// Check command exists
	if c.Command != "" && !e.matchCommand(c.Command) {
		return false
	}

	// Check file exists
	if c.File != "" && !e.matchFile(c.File) {
		return false
	}

	// Check NOT condition
	if c.Not != nil && e.Evaluate(*c.Not) {
		return false
	}

	// Check ALL conditions
	if len(c.All) > 0 {
		for _, sub := range c.All {
			if !e.Evaluate(sub) {
				return false
			}
		}
	}

	// Check ANY conditions
	if len(c.Any) > 0 {
		anyMatch := false
		for _, sub := range c.Any {
			if e.Evaluate(sub) {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			return false
		}
	}

	return true
}

// ShouldApplyLayer checks if a conditional layer should be applied.
func (e *ConditionEvaluator) ShouldApplyLayer(cl ConditionalLayer) bool {
	// Check "when" condition (must be true if specified)
	if !e.isEmpty(cl.When) && !e.Evaluate(cl.When) {
		return false
	}

	// Check "unless" condition (must be false if specified)
	if !e.isEmpty(cl.Unless) && e.Evaluate(cl.Unless) {
		return false
	}

	return true
}

// isEmpty checks if a condition has no constraints.
func (e *ConditionEvaluator) isEmpty(c Condition) bool {
	return c.OS == "" &&
		c.Arch == "" &&
		c.Hostname == "" &&
		c.EnvVar == "" &&
		c.EnvSet == "" &&
		c.EnvUnset == "" &&
		c.Command == "" &&
		c.File == "" &&
		c.Not == nil &&
		len(c.All) == 0 &&
		len(c.Any) == 0
}

// matchOS checks if the current OS matches the pattern.
func (e *ConditionEvaluator) matchOS(pattern string) bool {
	patterns := strings.Split(pattern, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		// Handle aliases
		switch strings.ToLower(p) {
		case "mac", "macos", "darwin":
			if e.osName == "darwin" {
				return true
			}
		case "win", "windows":
			if e.osName == "windows" {
				return true
			}
		default:
			if strings.EqualFold(e.osName, p) {
				return true
			}
		}
	}
	return false
}

// matchArch checks if the current architecture matches the pattern.
func (e *ConditionEvaluator) matchArch(pattern string) bool {
	patterns := strings.Split(pattern, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		// Handle aliases
		switch strings.ToLower(p) {
		case "x64", "x86_64", "amd64":
			if e.arch == "amd64" {
				return true
			}
		case "arm", "arm64", "aarch64":
			if e.arch == "arm64" {
				return true
			}
		default:
			if strings.EqualFold(e.arch, p) {
				return true
			}
		}
	}
	return false
}

// matchHostname checks if the hostname matches the pattern.
func (e *ConditionEvaluator) matchHostname(pattern string) bool {
	// Convert glob pattern to regex
	regexPattern := globToRegex(pattern)
	matched, _ := regexp.MatchString("(?i)^"+regexPattern+"$", e.hostname)
	return matched
}

// matchEnvVar checks if an environment variable matches.
func (e *ConditionEvaluator) matchEnvVar(expr string) bool {
	parts := strings.SplitN(expr, "=", 2)
	varName := parts[0]

	value, exists := os.LookupEnv(varName)
	if !exists {
		return false
	}

	// If just VAR name, check that it's set and non-empty
	if len(parts) == 1 {
		return value != ""
	}

	// If VAR=value, check exact match
	return value == parts[1]
}

// matchEnvSet checks if an environment variable is set.
func (e *ConditionEvaluator) matchEnvSet(varName string) bool {
	_, exists := os.LookupEnv(varName)
	return exists
}

// matchEnvUnset checks if an environment variable is not set.
func (e *ConditionEvaluator) matchEnvUnset(varName string) bool {
	_, exists := os.LookupEnv(varName)
	return !exists
}

// matchCommand checks if a command exists in PATH.
func (e *ConditionEvaluator) matchCommand(cmd string) bool {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, dir := range paths {
		path := strings.Join([]string{dir, cmd}, string(os.PathSeparator))
		if _, err := os.Stat(path); err == nil {
			return true
		}
		// Try with common extensions on Windows
		if runtime.GOOS == "windows" {
			for _, ext := range []string{".exe", ".cmd", ".bat"} {
				if _, err := os.Stat(path + ext); err == nil {
					return true
				}
			}
		}
	}
	return false
}

// matchFile checks if a file or directory exists.
func (e *ConditionEvaluator) matchFile(path string) bool {
	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = home + path[1:]
	}
	// Expand environment variables
	path = os.ExpandEnv(path)

	_, err := os.Stat(path)
	return err == nil
}

// globToRegex converts a glob pattern to a regex pattern.
func globToRegex(glob string) string {
	var result strings.Builder
	for _, c := range glob {
		switch c {
		case '*':
			result.WriteString(".*")
		case '?':
			result.WriteString(".")
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			result.WriteRune('\\')
			result.WriteRune(c)
		default:
			result.WriteRune(c)
		}
	}
	return result.String()
}
