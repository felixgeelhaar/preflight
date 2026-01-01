package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// DeprecationReason indicates why a package is deprecated.
type DeprecationReason string

const (
	// ReasonDeprecated indicates the package is officially deprecated.
	ReasonDeprecated DeprecationReason = "deprecated"
	// ReasonDisabled indicates the package is disabled (can't be installed).
	ReasonDisabled DeprecationReason = "disabled"
	// ReasonEOL indicates the package has reached end of life.
	ReasonEOL DeprecationReason = "end-of-life"
	// ReasonUnmaintained indicates the package is no longer maintained.
	ReasonUnmaintained DeprecationReason = "unmaintained"
)

// DeprecatedPackage represents a package that is deprecated or disabled.
type DeprecatedPackage struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Provider    string            `json:"provider"`
	Reason      DeprecationReason `json:"reason"`
	Date        *time.Time        `json:"date,omitempty"`
	Alternative string            `json:"alternative,omitempty"`
	Message     string            `json:"message,omitempty"`
}

// DeprecatedPackages is a collection of deprecated packages.
type DeprecatedPackages []DeprecatedPackage

// ExcludeNames filters out packages with the given names.
func (d DeprecatedPackages) ExcludeNames(names []string) DeprecatedPackages {
	if len(names) == 0 {
		return d
	}

	exclude := make(map[string]bool)
	for _, name := range names {
		exclude[name] = true
	}

	result := make(DeprecatedPackages, 0, len(d))
	for _, pkg := range d {
		if !exclude[pkg.Name] {
			result = append(result, pkg)
		}
	}
	return result
}

// ByReason filters packages to only those with the given reason.
func (d DeprecatedPackages) ByReason(reason DeprecationReason) DeprecatedPackages {
	result := make(DeprecatedPackages, 0, len(d))
	for _, pkg := range d {
		if pkg.Reason == reason {
			result = append(result, pkg)
		}
	}
	return result
}

// String returns the string representation of the deprecation reason.
func (r DeprecationReason) String() string {
	if r == "" {
		return "unknown"
	}
	return string(r)
}

// DeprecatedResult contains the results of a deprecation check.
type DeprecatedResult struct {
	Checker   string             `json:"checker"`
	CheckedAt time.Time          `json:"checked_at"`
	Packages  DeprecatedPackages `json:"packages"`
}

// Summary returns a summary of deprecated packages.
func (r *DeprecatedResult) Summary() DeprecatedSummary {
	summary := DeprecatedSummary{
		Total: len(r.Packages),
	}

	for _, pkg := range r.Packages {
		switch pkg.Reason {
		case ReasonDeprecated:
			summary.Deprecated++
		case ReasonDisabled:
			summary.Disabled++
		case ReasonEOL:
			summary.EOL++
		case ReasonUnmaintained:
			summary.Unmaintained++
		}
	}

	return summary
}

// DeprecatedSummary provides aggregate information about deprecated packages.
type DeprecatedSummary struct {
	Total        int `json:"total"`
	Deprecated   int `json:"deprecated"`
	Disabled     int `json:"disabled"`
	EOL          int `json:"eol"`
	Unmaintained int `json:"unmaintained"`
}

// DeprecationChecker checks for deprecated packages.
type DeprecationChecker interface {
	// Name returns the checker name.
	Name() string
	// Available returns true if the checker can run.
	Available() bool
	// Check returns deprecated packages.
	Check(ctx context.Context, opts DeprecationOptions) (*DeprecatedResult, error)
}

// DeprecationOptions configures deprecation checking.
type DeprecationOptions struct {
	IgnorePackages []string `json:"ignore_packages"`
}

// BrewDeprecationChecker checks for deprecated Homebrew packages.
type BrewDeprecationChecker struct {
	execCommand func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewBrewDeprecationChecker creates a new Homebrew deprecation checker.
func NewBrewDeprecationChecker() *BrewDeprecationChecker {
	return &BrewDeprecationChecker{
		execCommand: exec.CommandContext,
	}
}

// Name returns the checker name.
func (b *BrewDeprecationChecker) Name() string {
	return "brew"
}

// Available returns true if brew is installed.
func (b *BrewDeprecationChecker) Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// Check returns deprecated Homebrew packages from installed formulae.
func (b *BrewDeprecationChecker) Check(ctx context.Context, opts DeprecationOptions) (*DeprecatedResult, error) {
	if !b.Available() {
		return nil, ErrScannerNotAvailable
	}

	result := &DeprecatedResult{
		Checker:   b.Name(),
		CheckedAt: time.Now(),
		Packages:  make(DeprecatedPackages, 0),
	}

	// Get list of installed formulae
	cmd := b.execCommand(ctx, "brew", "info", "--installed", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run brew info: %w", err)
	}

	packages, err := b.parseOutput(stdout.Bytes())
	if err != nil {
		return nil, err
	}

	result.Packages = packages

	// Apply filters
	if len(opts.IgnorePackages) > 0 {
		result.Packages = result.Packages.ExcludeNames(opts.IgnorePackages)
	}

	return result, nil
}

// brewInfoOutput represents the JSON output from brew info --json.
type brewInfoOutput []brewInfoFormula

type brewInfoFormula struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Deprecated      bool   `json:"deprecated"`
	DeprecateDate   string `json:"deprecation_date,omitempty"`
	DeprecateReason string `json:"deprecation_reason,omitempty"`
	Disabled        bool   `json:"disabled"`
	DisableDate     string `json:"disable_date,omitempty"`
	DisableReason   string `json:"disable_reason,omitempty"`
	Installed       []struct {
		Version string `json:"version"`
	} `json:"installed"`
}

// parseOutput parses brew info --json output.
func (b *BrewDeprecationChecker) parseOutput(data []byte) (DeprecatedPackages, error) {
	if len(data) == 0 {
		return DeprecatedPackages{}, nil
	}

	var output brewInfoOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse brew info output: %w", err)
	}

	packages := make(DeprecatedPackages, 0)

	for _, f := range output {
		// Skip if not deprecated or disabled
		if !f.Deprecated && !f.Disabled {
			continue
		}

		version := ""
		if len(f.Installed) > 0 {
			version = f.Installed[0].Version
		}

		var reason DeprecationReason
		var message string
		var date *time.Time

		if f.Disabled {
			reason = ReasonDisabled
			message = f.DisableReason
			if f.DisableDate != "" {
				if t, err := time.Parse("2006-01-02", f.DisableDate); err == nil {
					date = &t
				}
			}
		} else if f.Deprecated {
			reason = ReasonDeprecated
			message = f.DeprecateReason
			if f.DeprecateDate != "" {
				if t, err := time.Parse("2006-01-02", f.DeprecateDate); err == nil {
					date = &t
				}
			}
		}

		pkg := DeprecatedPackage{
			Name:     f.Name,
			Version:  version,
			Provider: "brew",
			Reason:   reason,
			Date:     date,
			Message:  message,
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// DeprecationCheckerRegistry manages available deprecation checkers.
type DeprecationCheckerRegistry struct {
	checkers []DeprecationChecker
}

// NewDeprecationCheckerRegistry creates a new registry.
func NewDeprecationCheckerRegistry() *DeprecationCheckerRegistry {
	return &DeprecationCheckerRegistry{
		checkers: make([]DeprecationChecker, 0),
	}
}

// Register adds a checker to the registry.
func (r *DeprecationCheckerRegistry) Register(checker DeprecationChecker) {
	r.checkers = append(r.checkers, checker)
}

// Get returns a checker by name, or nil if not found or not available.
func (r *DeprecationCheckerRegistry) Get(name string) DeprecationChecker {
	for _, c := range r.checkers {
		if c.Name() == name && c.Available() {
			return c
		}
	}
	return nil
}

// All returns all available checkers.
func (r *DeprecationCheckerRegistry) All() []DeprecationChecker {
	available := make([]DeprecationChecker, 0, len(r.checkers))
	for _, c := range r.checkers {
		if c.Available() {
			available = append(available, c)
		}
	}
	return available
}

// Names returns the names of all registered checkers.
func (r *DeprecationCheckerRegistry) Names() []string {
	names := make([]string, len(r.checkers))
	for i, c := range r.checkers {
		names[i] = c.Name()
	}
	return names
}
