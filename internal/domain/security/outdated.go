package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// UpdateType indicates the type of version update available.
type UpdateType string

const (
	// UpdateMajor indicates a major version update (breaking changes likely).
	UpdateMajor UpdateType = "major"
	// UpdateMinor indicates a minor version update (new features).
	UpdateMinor UpdateType = "minor"
	// UpdatePatch indicates a patch version update (bug fixes).
	UpdatePatch UpdateType = "patch"
	// UpdateUnknown indicates the update type couldn't be determined.
	UpdateUnknown UpdateType = ""
)

// OutdatedPackage represents a package that has an available update.
type OutdatedPackage struct {
	Name           string        `json:"name"`
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version"`
	UpdateType     UpdateType    `json:"update_type"`
	Provider       string        `json:"provider"`
	Age            time.Duration `json:"age,omitempty"`
	Pinned         bool          `json:"pinned,omitempty"`
}

// OutdatedPackages is a collection of outdated packages.
type OutdatedPackages []OutdatedPackage

// ByUpdateType filters packages to only those with at least the given update type.
// Major > Minor > Patch.
func (o OutdatedPackages) ByUpdateType(minType UpdateType) OutdatedPackages {
	if minType == "" {
		return o
	}

	result := make(OutdatedPackages, 0, len(o))
	for _, pkg := range o {
		if pkg.UpdateType.IsAtLeast(minType) {
			result = append(result, pkg)
		}
	}
	return result
}

// ExcludeNames filters out packages with the given names.
func (o OutdatedPackages) ExcludeNames(names []string) OutdatedPackages {
	if len(names) == 0 {
		return o
	}

	exclude := make(map[string]bool)
	for _, name := range names {
		exclude[name] = true
	}

	result := make(OutdatedPackages, 0, len(o))
	for _, pkg := range o {
		if !exclude[pkg.Name] {
			result = append(result, pkg)
		}
	}
	return result
}

// ExcludePinned filters out pinned packages.
func (o OutdatedPackages) ExcludePinned() OutdatedPackages {
	result := make(OutdatedPackages, 0, len(o))
	for _, pkg := range o {
		if !pkg.Pinned {
			result = append(result, pkg)
		}
	}
	return result
}

// HasMajor returns true if there are any major updates.
func (o OutdatedPackages) HasMajor() bool {
	for _, pkg := range o {
		if pkg.UpdateType == UpdateMajor {
			return true
		}
	}
	return false
}

// IsAtLeast returns true if this update type is at least as significant as other.
func (u UpdateType) IsAtLeast(other UpdateType) bool {
	return u.order() >= other.order()
}

func (u UpdateType) order() int {
	switch u {
	case UpdateMajor:
		return 3
	case UpdateMinor:
		return 2
	case UpdatePatch:
		return 1
	default:
		return 0
	}
}

// String returns the string representation of the update type.
func (u UpdateType) String() string {
	if u == "" {
		return "unknown"
	}
	return string(u)
}

// OutdatedResult contains the results of an outdated check.
type OutdatedResult struct {
	Checker   string           `json:"checker"`
	CheckedAt time.Time        `json:"checked_at"`
	Packages  OutdatedPackages `json:"packages"`
}

// Summary returns a summary of outdated packages.
func (r *OutdatedResult) Summary() OutdatedSummary {
	summary := OutdatedSummary{
		Total:  len(r.Packages),
		Pinned: 0,
	}

	for _, pkg := range r.Packages {
		if pkg.Pinned {
			summary.Pinned++
		}
		switch pkg.UpdateType { //nolint:exhaustive // UpdateUnknown intentionally ignored
		case UpdateMajor:
			summary.Major++
		case UpdateMinor:
			summary.Minor++
		case UpdatePatch:
			summary.Patch++
		}
	}

	return summary
}

// OutdatedSummary provides aggregate information about outdated packages.
type OutdatedSummary struct {
	Total  int `json:"total"`
	Major  int `json:"major"`
	Minor  int `json:"minor"`
	Patch  int `json:"patch"`
	Pinned int `json:"pinned"`
}

// OutdatedChecker checks for outdated packages.
type OutdatedChecker interface {
	// Name returns the checker name.
	Name() string
	// Available returns true if the checker can run.
	Available() bool
	// Check returns outdated packages.
	Check(ctx context.Context, opts OutdatedOptions) (*OutdatedResult, error)
}

// OutdatedOptions configures outdated checking.
type OutdatedOptions struct {
	IncludePatch   bool          `json:"include_patch"`
	IncludePinned  bool          `json:"include_pinned"`
	IgnorePackages []string      `json:"ignore_packages"`
	MaxAge         time.Duration `json:"max_age"` // Only report if update is older than this
}

// DetermineUpdateType determines the update type between two semver versions.
func DetermineUpdateType(current, latest string) UpdateType {
	// Normalize versions to have v prefix
	if current != "" && !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if latest != "" && !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}

	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return UpdateUnknown
	}

	// Compare major versions
	currentMajor := semver.Major(current)
	latestMajor := semver.Major(latest)
	if currentMajor != latestMajor {
		return UpdateMajor
	}

	// Extract minor versions (semver package doesn't have a Minor function)
	currentMinor := extractMinor(current)
	latestMinor := extractMinor(latest)
	if currentMinor != latestMinor {
		return UpdateMinor
	}

	return UpdatePatch
}

// extractMinor extracts the minor version from a semver string.
func extractMinor(version string) string {
	// Remove v prefix and split
	v := strings.TrimPrefix(version, "v")
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		// Handle prerelease (e.g., "1.2.3-beta")
		minor := parts[1]
		if idx := strings.IndexAny(minor, "-+"); idx >= 0 {
			minor = minor[:idx]
		}
		return minor
	}
	return ""
}

// BrewOutdatedChecker checks for outdated Homebrew packages.
type BrewOutdatedChecker struct {
	execCommand func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// NewBrewOutdatedChecker creates a new Homebrew outdated checker.
func NewBrewOutdatedChecker() *BrewOutdatedChecker {
	return &BrewOutdatedChecker{
		execCommand: exec.CommandContext,
	}
}

// Name returns the checker name.
func (b *BrewOutdatedChecker) Name() string {
	return "brew"
}

// Available returns true if brew is installed.
func (b *BrewOutdatedChecker) Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// Check returns outdated Homebrew packages.
func (b *BrewOutdatedChecker) Check(ctx context.Context, opts OutdatedOptions) (*OutdatedResult, error) {
	if !b.Available() {
		return nil, ErrScannerNotAvailable
	}

	result := &OutdatedResult{
		Checker:   b.Name(),
		CheckedAt: time.Now(),
		Packages:  make(OutdatedPackages, 0),
	}

	cmd := b.execCommand(ctx, "brew", "outdated", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run brew outdated: %w", err)
	}

	packages, err := b.parseOutput(stdout.Bytes())
	if err != nil {
		return nil, err
	}

	result.Packages = packages

	// Apply filters
	if !opts.IncludePatch {
		result.Packages = result.Packages.ByUpdateType(UpdateMinor)
	}

	if !opts.IncludePinned {
		result.Packages = result.Packages.ExcludePinned()
	}

	if len(opts.IgnorePackages) > 0 {
		result.Packages = result.Packages.ExcludeNames(opts.IgnorePackages)
	}

	return result, nil
}

// brewOutdatedOutput represents the JSON output from brew outdated --json.
type brewOutdatedOutput struct {
	Formulae []brewOutdatedFormula `json:"formulae"`
	Casks    []brewOutdatedCask    `json:"casks"`
}

type brewOutdatedFormula struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	PinnedVersion     string   `json:"pinned_version,omitempty"`
	Pinned            bool     `json:"pinned"`
}

type brewOutdatedCask struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

// parseOutput parses brew outdated --json output.
func (b *BrewOutdatedChecker) parseOutput(data []byte) (OutdatedPackages, error) {
	if len(data) == 0 {
		return OutdatedPackages{}, nil
	}

	var output brewOutdatedOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse brew outdated output: %w", err)
	}

	packages := make(OutdatedPackages, 0, len(output.Formulae)+len(output.Casks))

	// Process formulae
	for _, f := range output.Formulae {
		current := ""
		if len(f.InstalledVersions) > 0 {
			current = f.InstalledVersions[0]
		}

		pkg := OutdatedPackage{
			Name:           f.Name,
			CurrentVersion: current,
			LatestVersion:  f.CurrentVersion,
			UpdateType:     DetermineUpdateType(current, f.CurrentVersion),
			Provider:       "brew",
			Pinned:         f.Pinned,
		}
		packages = append(packages, pkg)
	}

	// Process casks
	for _, c := range output.Casks {
		current := ""
		if len(c.InstalledVersions) > 0 {
			current = c.InstalledVersions[0]
		}

		pkg := OutdatedPackage{
			Name:           c.Name,
			CurrentVersion: current,
			LatestVersion:  c.CurrentVersion,
			UpdateType:     DetermineUpdateType(current, c.CurrentVersion),
			Provider:       "cask",
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// OutdatedCheckerRegistry manages available outdated checkers.
type OutdatedCheckerRegistry struct {
	checkers []OutdatedChecker
}

// NewOutdatedCheckerRegistry creates a new registry.
func NewOutdatedCheckerRegistry() *OutdatedCheckerRegistry {
	return &OutdatedCheckerRegistry{
		checkers: make([]OutdatedChecker, 0),
	}
}

// Register adds a checker to the registry.
func (r *OutdatedCheckerRegistry) Register(checker OutdatedChecker) {
	r.checkers = append(r.checkers, checker)
}

// Get returns a checker by name, or nil if not found or not available.
func (r *OutdatedCheckerRegistry) Get(name string) OutdatedChecker {
	for _, c := range r.checkers {
		if c.Name() == name && c.Available() {
			return c
		}
	}
	return nil
}

// All returns all available checkers.
func (r *OutdatedCheckerRegistry) All() []OutdatedChecker {
	available := make([]OutdatedChecker, 0, len(r.checkers))
	for _, c := range r.checkers {
		if c.Available() {
			available = append(available, c)
		}
	}
	return available
}

// Names returns the names of all registered checkers.
func (r *OutdatedCheckerRegistry) Names() []string {
	names := make([]string, len(r.checkers))
	for i, c := range r.checkers {
		names[i] = c.Name()
	}
	return names
}
