package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RedundancyType indicates the type of redundancy detected.
type RedundancyType string

const (
	// RedundancyDuplicate indicates version duplicates (e.g., go + go@1.24).
	RedundancyDuplicate RedundancyType = "duplicate"
	// RedundancyOverlap indicates overlapping tools serving same purpose.
	RedundancyOverlap RedundancyType = "overlap"
	// RedundancyOrphan indicates orphaned dependencies no longer needed.
	RedundancyOrphan RedundancyType = "orphan"
)

// Redundancy represents a detected redundancy.
type Redundancy struct {
	Type           RedundancyType `json:"type"`
	Packages       []string       `json:"packages"`
	Category       string         `json:"category,omitempty"`
	Recommendation string         `json:"recommendation"`
	Action         string         `json:"action,omitempty"`
	Keep           []string       `json:"keep,omitempty"`
	Remove         []string       `json:"remove,omitempty"`
}

// Redundancies is a collection of redundancies.
type Redundancies []Redundancy

// ByType filters redundancies to only those of the given type.
func (r Redundancies) ByType(t RedundancyType) Redundancies {
	result := make(Redundancies, 0, len(r))
	for _, red := range r {
		if red.Type == t {
			result = append(result, red)
		}
	}
	return result
}

// TotalRemovable returns the count of packages that can be removed.
func (r Redundancies) TotalRemovable() int {
	count := 0
	for _, red := range r {
		count += len(red.Remove)
	}
	return count
}

// RedundancyResult contains the results of a redundancy check.
type RedundancyResult struct {
	Checker      string       `json:"checker"`
	CheckedAt    time.Time    `json:"checked_at"`
	Redundancies Redundancies `json:"redundancies"`
}

// Summary returns a summary of redundancies.
func (r *RedundancyResult) Summary() RedundancySummary {
	summary := RedundancySummary{
		Total: len(r.Redundancies),
	}

	for _, red := range r.Redundancies {
		switch red.Type {
		case RedundancyDuplicate:
			summary.Duplicates++
		case RedundancyOverlap:
			summary.Overlaps++
		case RedundancyOrphan:
			summary.Orphans++
		}
		summary.Removable += len(red.Remove)
	}

	return summary
}

// RedundancySummary provides aggregate information about redundancies.
type RedundancySummary struct {
	Total      int `json:"total"`
	Duplicates int `json:"duplicates"`
	Overlaps   int `json:"overlaps"`
	Orphans    int `json:"orphans"`
	Removable  int `json:"removable"`
}

// RedundancyChecker checks for redundant packages.
type RedundancyChecker interface {
	// Name returns the checker name.
	Name() string
	// Available returns true if the checker can run.
	Available() bool
	// Check returns detected redundancies.
	Check(ctx context.Context, opts RedundancyOptions) (*RedundancyResult, error)
}

// RedundancyOptions configures redundancy checking.
type RedundancyOptions struct {
	IgnorePackages  []string `json:"ignore_packages"`
	KeepPackages    []string `json:"keep_packages"`
	IncludeOrphans  bool     `json:"include_orphans"`
	IncludeOverlaps bool     `json:"include_overlaps"`
}

// ToolCategory defines a category of overlapping tools.
type ToolCategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	KeepAll     bool     `json:"keep_all,omitempty"` // If true, don't suggest removal
}

// DefaultToolCategories returns the default overlapping tool categories.
func DefaultToolCategories() []ToolCategory {
	return []ToolCategory{
		{
			Name:        "security_scanners",
			Description: "Vulnerability scanners",
			Tools:       []string{"grype", "trivy", "snyk", "clair"},
			KeepAll:     false,
		},
		{
			Name:        "node_package_managers",
			Description: "Node.js package managers",
			Tools:       []string{"npm", "yarn", "pnpm", "bun"},
			KeepAll:     false,
		},
		{
			Name:        "python_env_managers",
			Description: "Python environment managers",
			Tools:       []string{"pyenv", "conda", "miniconda", "miniforge", "anaconda"},
			KeepAll:     false,
		},
		{
			Name:        "version_managers",
			Description: "Runtime version managers",
			Tools:       []string{"asdf", "mise", "rtx", "nvm", "rbenv", "pyenv"},
			KeepAll:     false,
		},
		{
			Name:        "container_runtimes",
			Description: "Container runtimes",
			Tools:       []string{"docker", "podman", "colima", "lima", "orbstack"},
			KeepAll:     false,
		},
		{
			Name:        "terminal_emulators",
			Description: "Terminal emulators",
			Tools:       []string{"iterm2", "alacritty", "kitty", "wezterm", "warp"},
			KeepAll:     true, // Users often have preferences
		},
		{
			Name:        "shell_prompts",
			Description: "Shell prompt customization",
			Tools:       []string{"starship", "powerlevel10k", "oh-my-posh"},
			KeepAll:     false,
		},
		{
			Name:        "git_clients",
			Description: "Git GUI clients",
			Tools:       []string{"lazygit", "tig", "gitui", "git-delta", "diff-so-fancy"},
			KeepAll:     true, // Different purposes
		},
		{
			Name:        "editors",
			Description: "Terminal editors",
			Tools:       []string{"vim", "neovim", "emacs", "nano", "micro"},
			KeepAll:     true, // Personal preference
		},
	}
}

// BrewRedundancyChecker checks for redundant Homebrew packages.
type BrewRedundancyChecker struct {
	execCommand    func(ctx context.Context, name string, args ...string) *exec.Cmd
	toolCategories []ToolCategory
}

// NewBrewRedundancyChecker creates a new Homebrew redundancy checker.
func NewBrewRedundancyChecker() *BrewRedundancyChecker {
	return &BrewRedundancyChecker{
		execCommand:    exec.CommandContext,
		toolCategories: DefaultToolCategories(),
	}
}

// Name returns the checker name.
func (b *BrewRedundancyChecker) Name() string {
	return "brew"
}

// Available returns true if brew is installed.
func (b *BrewRedundancyChecker) Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

// Check returns detected redundancies in Homebrew packages.
func (b *BrewRedundancyChecker) Check(ctx context.Context, opts RedundancyOptions) (*RedundancyResult, error) {
	if !b.Available() {
		return nil, ErrScannerNotAvailable
	}

	result := &RedundancyResult{
		Checker:      b.Name(),
		CheckedAt:    time.Now(),
		Redundancies: make(Redundancies, 0),
	}

	// Get installed packages
	installed, err := b.getInstalledPackages(ctx)
	if err != nil {
		return nil, err
	}

	// Build ignore/keep maps
	ignoreMap := make(map[string]bool)
	for _, pkg := range opts.IgnorePackages {
		ignoreMap[pkg] = true
	}
	keepMap := make(map[string]bool)
	for _, pkg := range opts.KeepPackages {
		keepMap[pkg] = true
	}

	// Detect version duplicates
	duplicates := b.detectDuplicates(installed, ignoreMap, keepMap)
	result.Redundancies = append(result.Redundancies, duplicates...)

	// Detect overlapping tools
	if opts.IncludeOverlaps {
		overlaps := b.detectOverlaps(installed, ignoreMap)
		result.Redundancies = append(result.Redundancies, overlaps...)
	}

	// Detect orphaned dependencies
	if opts.IncludeOrphans {
		orphans := b.detectOrphans(ctx, ignoreMap)
		if len(orphans.Packages) > 0 {
			result.Redundancies = append(result.Redundancies, orphans)
		}
	}

	return result, nil
}

// getInstalledPackages returns list of installed brew packages.
func (b *BrewRedundancyChecker) getInstalledPackages(ctx context.Context) ([]string, error) {
	cmd := b.execCommand(ctx, "brew", "list", "--formula", "-1")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list brew packages: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	packages := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			packages = append(packages, line)
		}
	}

	return packages, nil
}

// versionedPackageRegex matches packages with version suffix like go@1.24
var versionedPackageRegex = regexp.MustCompile(`^(.+)@[\d.]+$`)

// detectDuplicates finds version duplicates (e.g., go + go@1.24).
func (b *BrewRedundancyChecker) detectDuplicates(packages []string, ignore, keep map[string]bool) Redundancies {
	// Group packages by base name
	groups := make(map[string][]string)

	for _, pkg := range packages {
		if ignore[pkg] {
			continue
		}

		baseName := pkg
		if matches := versionedPackageRegex.FindStringSubmatch(pkg); len(matches) > 1 {
			baseName = matches[1]
		}

		groups[baseName] = append(groups[baseName], pkg)
	}

	// Find groups with duplicates
	redundancies := make(Redundancies, 0)

	for baseName, pkgs := range groups {
		if len(pkgs) <= 1 {
			continue
		}

		// Sort packages: unversioned first, then by version descending
		sort.Slice(pkgs, func(i, j int) bool {
			iVersioned := versionedPackageRegex.MatchString(pkgs[i])
			jVersioned := versionedPackageRegex.MatchString(pkgs[j])

			if iVersioned != jVersioned {
				return !iVersioned // Unversioned first
			}
			return pkgs[i] > pkgs[j] // Higher version first
		})

		// Determine keep/remove
		var keepPkgs, removePkgs []string

		for _, pkg := range pkgs {
			if keep[pkg] {
				keepPkgs = append(keepPkgs, pkg)
			}
		}

		// If nothing explicitly kept, keep the first (unversioned or latest)
		if len(keepPkgs) == 0 {
			keepPkgs = []string{pkgs[0]}
		}

		keepSet := make(map[string]bool)
		for _, k := range keepPkgs {
			keepSet[k] = true
		}

		for _, pkg := range pkgs {
			if !keepSet[pkg] {
				removePkgs = append(removePkgs, pkg)
			}
		}

		if len(removePkgs) == 0 {
			continue
		}

		red := Redundancy{
			Type:           RedundancyDuplicate,
			Packages:       pkgs,
			Category:       baseName,
			Recommendation: fmt.Sprintf("Keep %s (tracks latest)", keepPkgs[0]),
			Action:         fmt.Sprintf("preflight cleanup --remove %s", strings.Join(removePkgs, " ")),
			Keep:           keepPkgs,
			Remove:         removePkgs,
		}

		redundancies = append(redundancies, red)
	}

	return redundancies
}

// detectOverlaps finds overlapping tools in the same category.
func (b *BrewRedundancyChecker) detectOverlaps(packages []string, ignore map[string]bool) Redundancies {
	packageSet := make(map[string]bool)
	for _, pkg := range packages {
		if !ignore[pkg] {
			packageSet[pkg] = true
		}
	}

	redundancies := make(Redundancies, 0)

	for _, category := range b.toolCategories {
		// Find installed tools in this category
		var installed []string
		for _, tool := range category.Tools {
			if packageSet[tool] {
				installed = append(installed, tool)
			}
		}

		if len(installed) <= 1 {
			continue
		}

		red := Redundancy{
			Type:     RedundancyOverlap,
			Packages: installed,
			Category: category.Name,
		}

		if category.KeepAll {
			red.Recommendation = fmt.Sprintf("%s - typically used together", category.Description)
			red.Keep = installed
		} else {
			red.Recommendation = fmt.Sprintf("%s - consider keeping only one", category.Description)
			red.Keep = []string{installed[0]}
			if len(installed) > 1 {
				red.Remove = installed[1:]
			}
		}

		redundancies = append(redundancies, red)
	}

	return redundancies
}

// detectOrphans finds orphaned dependencies.
func (b *BrewRedundancyChecker) detectOrphans(ctx context.Context, ignore map[string]bool) Redundancy {
	// Get packages that are installed but not required by others
	cmd := b.execCommand(ctx, "brew", "autoremove", "--dry-run")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// autoremove returns exit code 0 even with packages to remove
	_ = cmd.Run()

	output := stdout.String()
	var orphans []string

	// Parse "Would remove: pkg1, pkg2, ..." or line-by-line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "==>") {
			continue
		}
		// Skip informational lines
		if strings.Contains(line, "Would") || strings.Contains(line, "Nothing") {
			continue
		}
		// Extract package names
		for _, pkg := range strings.Fields(line) {
			pkg = strings.TrimSuffix(pkg, ",")
			if pkg != "" && !ignore[pkg] {
				orphans = append(orphans, pkg)
			}
		}
	}

	if len(orphans) == 0 {
		return Redundancy{}
	}

	return Redundancy{
		Type:           RedundancyOrphan,
		Packages:       orphans,
		Category:       "orphaned_dependencies",
		Recommendation: fmt.Sprintf("%d orphaned dependencies can be removed", len(orphans)),
		Action:         "preflight cleanup --autoremove",
		Remove:         orphans,
	}
}

// Cleanup removes specified packages.
func (b *BrewRedundancyChecker) Cleanup(ctx context.Context, packages []string, dryRun bool) error {
	if len(packages) == 0 {
		return nil
	}

	args := []string{"uninstall"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, packages...)

	cmd := b.execCommand(ctx, "brew", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall packages: %w: %s", err, stderr.String())
	}

	return nil
}

// Autoremove removes orphaned dependencies.
func (b *BrewRedundancyChecker) Autoremove(ctx context.Context, dryRun bool) ([]string, error) {
	args := []string{"autoremove"}
	if dryRun {
		args = append(args, "--dry-run")
	}

	cmd := b.execCommand(ctx, "brew", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to autoremove: %w: %s", err, stderr.String())
	}

	// Parse removed packages from output
	var removed []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") && !strings.Contains(line, "Nothing") {
			for _, pkg := range strings.Fields(line) {
				pkg = strings.TrimSuffix(pkg, ",")
				if pkg != "" {
					removed = append(removed, pkg)
				}
			}
		}
	}

	return removed, nil
}

// RedundancyCheckerRegistry manages available redundancy checkers.
type RedundancyCheckerRegistry struct {
	checkers []RedundancyChecker
}

// NewRedundancyCheckerRegistry creates a new registry.
func NewRedundancyCheckerRegistry() *RedundancyCheckerRegistry {
	return &RedundancyCheckerRegistry{
		checkers: make([]RedundancyChecker, 0),
	}
}

// Register adds a checker to the registry.
func (r *RedundancyCheckerRegistry) Register(checker RedundancyChecker) {
	r.checkers = append(r.checkers, checker)
}

// Get returns a checker by name, or nil if not found or not available.
func (r *RedundancyCheckerRegistry) Get(name string) RedundancyChecker {
	for _, c := range r.checkers {
		if c.Name() == name && c.Available() {
			return c
		}
	}
	return nil
}

// All returns all available checkers.
func (r *RedundancyCheckerRegistry) All() []RedundancyChecker {
	available := make([]RedundancyChecker, 0, len(r.checkers))
	for _, c := range r.checkers {
		if c.Available() {
			available = append(available, c)
		}
	}
	return available
}

// CleanupResult contains the result of a cleanup operation.
type CleanupResult struct {
	Removed []string `json:"removed"`
	DryRun  bool     `json:"dry_run"`
	Error   string   `json:"error,omitempty"`
}

// CleanupResultJSON is the JSON output format for cleanup.
type CleanupResultJSON struct {
	Redundancies Redundancies       `json:"redundancies,omitempty"`
	Cleanup      *CleanupResult     `json:"cleanup,omitempty"`
	Summary      *RedundancySummary `json:"summary,omitempty"`
	Error        string             `json:"error,omitempty"`
}

// ToJSON converts the result to JSON bytes.
func (r *RedundancyResult) ToJSON() ([]byte, error) {
	output := CleanupResultJSON{
		Redundancies: r.Redundancies,
	}
	summary := r.Summary()
	output.Summary = &summary
	return json.MarshalIndent(output, "", "  ")
}
