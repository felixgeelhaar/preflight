package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/felixgeelhaar/preflight/internal/domain/lock"
)

// Capture discovers current machine configuration.
func (p *Preflight) Capture(ctx context.Context, opts CaptureOptions) (*CaptureFindings, error) {
	findings := &CaptureFindings{
		CapturedAt: time.Now(),
		HomeDir:    opts.HomeDir,
		Items:      make([]CapturedItem, 0),
		Providers:  make([]string, 0),
	}

	if findings.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		findings.HomeDir = home
	}

	// Determine which providers to capture
	providers := opts.Providers
	if len(providers) == 0 {
		providers = []string{"brew", "git", "ssh", "shell"}
	}

	for _, provider := range providers {
		findings.Providers = append(findings.Providers, provider)

		items, err := p.captureProvider(ctx, provider, findings.HomeDir, opts.IncludeSecrets)
		if err != nil {
			findings.Warnings = append(findings.Warnings, fmt.Sprintf("%s: %v", provider, err))
			continue
		}

		findings.Items = append(findings.Items, items...)
	}

	return findings, nil
}

func (p *Preflight) captureProvider(ctx context.Context, provider, homeDir string, includeSecrets bool) ([]CapturedItem, error) {
	now := time.Now()
	var items []CapturedItem

	switch provider {
	case "brew":
		items = p.captureBrewFormulae(ctx, now)
	case "git":
		items = p.captureGitConfig(homeDir, now)
	case "ssh":
		items = p.captureSSHConfig(homeDir, now, includeSecrets)
	case "shell":
		items = p.captureShellConfig(homeDir, now)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	return items, nil
}

func (p *Preflight) captureBrewFormulae(_ context.Context, capturedAt time.Time) []CapturedItem {
	// Try to list installed formulae
	cmd := exec.Command("brew", "list", "--formula", "-1")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	formulae := strings.Split(strings.TrimSpace(string(output)), "\n")
	items := make([]CapturedItem, 0, len(formulae))
	for _, f := range formulae {
		if f == "" {
			continue
		}
		items = append(items, CapturedItem{
			Provider:   "brew",
			Name:       f,
			Value:      f,
			Source:     "brew list --formula",
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureGitConfig(homeDir string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	gitconfigPath := filepath.Join(homeDir, ".gitconfig")
	if _, err := os.Stat(gitconfigPath); err == nil {
		// Read key config values
		keys := []string{"user.name", "user.email", "core.editor", "init.defaultBranch"}
		for _, key := range keys {
			cmd := exec.Command("git", "config", "--global", key)
			output, err := cmd.Output()
			if err == nil {
				items = append(items, CapturedItem{
					Provider:   "git",
					Name:       key,
					Value:      strings.TrimSpace(string(output)),
					Source:     gitconfigPath,
					CapturedAt: capturedAt,
				})
			}
		}
	}

	return items
}

func (p *Preflight) captureSSHConfig(homeDir string, capturedAt time.Time, _ bool) []CapturedItem {
	var items []CapturedItem

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	if _, err := os.Stat(sshConfigPath); err == nil {
		items = append(items, CapturedItem{
			Provider:   "ssh",
			Name:       "config",
			Value:      sshConfigPath,
			Source:     sshConfigPath,
			CapturedAt: capturedAt,
		})
	}

	return items
}

func (p *Preflight) captureShellConfig(homeDir string, capturedAt time.Time) []CapturedItem {
	var items []CapturedItem

	shellFiles := []string{".zshrc", ".bashrc", ".bash_profile"}
	for _, file := range shellFiles {
		path := filepath.Join(homeDir, file)
		if _, err := os.Stat(path); err == nil {
			items = append(items, CapturedItem{
				Provider:   "shell",
				Name:       file,
				Value:      path,
				Source:     path,
				CapturedAt: capturedAt,
			})
		}
	}

	return items
}

// Doctor checks system state against configuration and reports issues.
func (p *Preflight) Doctor(ctx context.Context, opts DoctorOptions) (*DoctorReport, error) {
	startTime := time.Now()

	report := &DoctorReport{
		ConfigPath: opts.ConfigPath,
		Target:     opts.Target,
		Issues:     make([]DoctorIssue, 0),
		CheckedAt:  startTime,
	}

	// Load and compile configuration
	plan, err := p.Plan(ctx, opts.ConfigPath, opts.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Check each step for drift
	for _, entry := range plan.Entries() {
		status := entry.Status()
		step := entry.Step()

		switch status {
		case compiler.StatusNeedsApply:
			diff := entry.Diff()
			report.Issues = append(report.Issues, DoctorIssue{
				Provider:   step.ID().Provider(),
				StepID:     step.ID().String(),
				Severity:   SeverityWarning,
				Message:    "Configuration drift detected",
				Expected:   diff.Summary(),
				Actual:     "current state differs",
				Fixable:    true,
				FixCommand: "preflight apply",
			})

		case compiler.StatusFailed:
			report.Issues = append(report.Issues, DoctorIssue{
				Provider: step.ID().Provider(),
				StepID:   step.ID().String(),
				Severity: SeverityError,
				Message:  "Step check failed",
				Fixable:  false,
			})

		case compiler.StatusUnknown:
			report.Issues = append(report.Issues, DoctorIssue{
				Provider: step.ID().Provider(),
				StepID:   step.ID().String(),
				Severity: SeverityInfo,
				Message:  "Unable to determine step status",
				Fixable:  false,
			})

		case compiler.StatusSatisfied, compiler.StatusSkipped:
			// No issues for satisfied or skipped steps
		}
	}

	report.Duration = time.Since(startTime)
	return report, nil
}

// Fix applies fixes for issues found by Doctor.
func (p *Preflight) Fix(ctx context.Context, report *DoctorReport) ([]DoctorIssue, error) {
	if report == nil || !report.HasIssues() {
		return nil, nil
	}

	// Re-run plan and apply
	plan, err := p.Plan(ctx, report.ConfigPath, report.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create fix plan: %w", err)
	}

	_, err = p.Apply(ctx, plan, false)
	if err != nil {
		return nil, fmt.Errorf("failed to apply fixes: %w", err)
	}

	// Return list of issues that were fixed
	var fixed []DoctorIssue
	for _, issue := range report.Issues {
		if issue.Fixable {
			fixed = append(fixed, issue)
		}
	}

	return fixed, nil
}

// Diff shows differences between configuration and current system state.
func (p *Preflight) Diff(ctx context.Context, configPath, target string) (*DiffResult, error) {
	result := &DiffResult{
		ConfigPath: configPath,
		Target:     target,
		Entries:    make([]DiffEntry, 0),
		DiffedAt:   time.Now(),
	}

	// Create plan to see differences
	plan, err := p.Plan(ctx, configPath, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Convert plan entries to diff entries
	for _, entry := range plan.Entries() {
		if entry.Status() != compiler.StatusNeedsApply {
			continue
		}

		step := entry.Step()
		diff := entry.Diff()

		result.Entries = append(result.Entries, DiffEntry{
			Provider: step.ID().Provider(),
			Path:     step.ID().String(),
			Type:     DiffTypeModified,
			Expected: diff.Summary(),
			Actual:   "current state",
		})
	}

	return result, nil
}

// LockUpdate updates the lockfile with current versions.
func (p *Preflight) LockUpdate(ctx context.Context, configPath string) error {
	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"

	// Check if lockfile repository is configured
	if p.lockRepo == nil {
		return fmt.Errorf("lockfile repository not configured")
	}

	// Load current lockfile or create new one
	lockfile, err := p.lockRepo.Load(ctx, lockPath)
	if err != nil {
		// Create new lockfile if it doesn't exist
		machineInfo := lock.MachineInfoFromSystem()
		lockfile = lock.NewLockfile(config.ModeIntent, machineInfo)
	}

	// Update mode to allow updates
	lockfile = lockfile.WithMode(config.ModeIntent)

	// Save the updated lockfile
	if err := p.lockRepo.Save(ctx, lockPath, lockfile); err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	p.printf("Lockfile updated: %s\n", lockPath)
	return nil
}

// LockFreeze freezes the lockfile to prevent version changes.
func (p *Preflight) LockFreeze(ctx context.Context, configPath string) error {
	lockPath := strings.TrimSuffix(configPath, filepath.Ext(configPath)) + ".lock"

	// Check if lockfile repository is configured
	if p.lockRepo == nil {
		return fmt.Errorf("lockfile repository not configured")
	}

	lockfile, err := p.lockRepo.Load(ctx, lockPath)
	if err != nil {
		return fmt.Errorf("lockfile not found: %w", err)
	}

	// Change mode to frozen
	lockfile = lockfile.WithMode(config.ModeFrozen)

	// Save the frozen lockfile
	if err := p.lockRepo.Save(ctx, lockPath, lockfile); err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	p.printf("Lockfile frozen: %s\n", lockPath)
	return nil
}

// RepoInit initializes a configuration repository.
func (p *Preflight) RepoInit(ctx context.Context, opts RepoOptions) error {
	// Check if already initialized
	gitDir := filepath.Join(opts.Path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("repository already initialized at %s", opts.Path)
	}

	// Initialize git repository
	cmd := exec.CommandContext(ctx, "git", "init", opts.Path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Set up remote if provided
	if opts.Remote != "" {
		cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "remote", "add", "origin", opts.Remote)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	// Create initial branch
	cmd = exec.CommandContext(ctx, "git", "-C", opts.Path, "checkout", "-b", opts.Branch)
	_ = cmd.Run() // Ignore error if branch already exists

	p.printf("Repository initialized: %s\n", opts.Path)
	return nil
}

// RepoStatus returns the status of a configuration repository.
func (p *Preflight) RepoStatus(ctx context.Context, path string) (*RepoStatus, error) {
	status := &RepoStatus{
		Path: path,
	}

	// Check if git repo exists
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		status.Initialized = false
		return status, nil
	}
	status.Initialized = true

	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := cmd.Output(); err == nil {
		status.Branch = strings.TrimSpace(string(output))
	}

	// Get remote
	cmd = exec.CommandContext(ctx, "git", "-C", path, "remote", "get-url", "origin")
	if output, err := cmd.Output(); err == nil {
		status.Remote = strings.TrimSpace(string(output))
	}

	// Check for uncommitted changes
	cmd = exec.CommandContext(ctx, "git", "-C", path, "status", "--porcelain")
	if output, err := cmd.Output(); err == nil {
		status.HasChanges = len(strings.TrimSpace(string(output))) > 0
	}

	// Get ahead/behind counts
	cmd = exec.CommandContext(ctx, "git", "-C", path, "rev-list", "--count", "--left-right", "@{upstream}...HEAD")
	if output, err := cmd.Output(); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(output)))
		if len(parts) == 2 {
			_, _ = fmt.Sscanf(parts[0], "%d", &status.Behind)
			_, _ = fmt.Sscanf(parts[1], "%d", &status.Ahead)
		}
	}

	// Get last commit
	cmd = exec.CommandContext(ctx, "git", "-C", path, "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		status.LastCommit = strings.TrimSpace(string(output))
	}

	// Get last commit time
	cmd = exec.CommandContext(ctx, "git", "-C", path, "log", "-1", "--format=%ct")
	if output, err := cmd.Output(); err == nil {
		var timestamp int64
		if n, _ := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &timestamp); n == 1 {
			status.LastCommitAt = time.Unix(timestamp, 0)
		}
	}

	return status, nil
}

// PrintDoctorReport outputs a human-readable doctor report.
func (p *Preflight) PrintDoctorReport(report *DoctorReport) {
	p.printf("\nDoctor Report\n")
	p.printf("=============\n\n")

	if !report.HasIssues() {
		p.printf("✓ No issues found. Your system matches the configuration.\n")
		return
	}

	p.printf("Found %d issue(s):\n\n", report.IssueCount())

	bySeverity := report.IssuesBySeverity()

	// Print errors first
	for _, issue := range bySeverity[SeverityError] {
		p.printf("  ✗ [ERROR] %s: %s\n", issue.StepID, issue.Message)
	}

	// Then warnings
	for _, issue := range bySeverity[SeverityWarning] {
		p.printf("  ⚠ [WARNING] %s: %s\n", issue.StepID, issue.Message)
		if issue.Fixable {
			p.printf("      Fix: %s\n", issue.FixCommand)
		}
	}

	// Then info
	for _, issue := range bySeverity[SeverityInfo] {
		p.printf("  ℹ [INFO] %s: %s\n", issue.StepID, issue.Message)
	}

	p.printf("\nSummary: %d errors, %d warnings, %d fixable\n",
		report.ErrorCount(), report.WarningCount(), report.FixableCount())
}

// PrintCaptureFindings outputs captured configuration.
func (p *Preflight) PrintCaptureFindings(findings *CaptureFindings) {
	p.printf("\nCapture Results\n")
	p.printf("===============\n\n")

	p.printf("Captured %d items from %d providers\n\n",
		findings.ItemCount(), len(findings.Providers))

	byProvider := findings.ItemsByProvider()
	for provider, items := range byProvider {
		p.printf("%s (%d items):\n", provider, len(items))
		for _, item := range items {
			p.printf("  - %s\n", item.Name)
		}
		p.printf("\n")
	}

	if len(findings.Warnings) > 0 {
		p.printf("Warnings:\n")
		for _, w := range findings.Warnings {
			p.printf("  ⚠ %s\n", w)
		}
	}
}

// PrintDiff outputs differences in unified format.
func (p *Preflight) PrintDiff(result *DiffResult) {
	p.printf("\nConfiguration Diff\n")
	p.printf("==================\n\n")

	if !result.HasDifferences() {
		p.printf("No differences. Configuration matches system state.\n")
		return
	}

	p.printf("Found %d difference(s):\n\n", len(result.Entries))

	byProvider := result.EntriesByProvider()
	for provider, entries := range byProvider {
		p.printf("%s:\n", provider)
		for _, entry := range entries {
			symbol := "~"
			switch entry.Type {
			case DiffTypeAdded:
				symbol = "+"
			case DiffTypeRemoved:
				symbol = "-"
			case DiffTypeModified:
				symbol = "~"
			}
			p.printf("  %s %s\n", symbol, entry.Path)
			if entry.Expected != "" {
				p.printf("      expected: %s\n", entry.Expected)
			}
		}
		p.printf("\n")
	}
}

// PrintRepoStatus outputs repository status.
func (p *Preflight) PrintRepoStatus(status *RepoStatus) {
	p.printf("\nRepository Status\n")
	p.printf("=================\n\n")

	if !status.Initialized {
		p.printf("Not a git repository. Run 'preflight repo init' to initialize.\n")
		return
	}

	p.printf("Path:   %s\n", status.Path)
	p.printf("Branch: %s\n", status.Branch)

	if status.Remote != "" {
		p.printf("Remote: %s\n", status.Remote)
	}

	if status.IsSynced() {
		p.printf("Status: ✓ Up to date\n")
	} else {
		if status.HasChanges {
			p.printf("Status: ⚠ Uncommitted changes\n")
		}
		if status.NeedsPush() {
			p.printf("Status: ↑ %d commit(s) ahead\n", status.Ahead)
		}
		if status.NeedsPull() {
			p.printf("Status: ↓ %d commit(s) behind\n", status.Behind)
		}
	}

	if status.LastCommit != "" {
		p.printf("Last commit: %s (%s)\n", status.LastCommit,
			status.LastCommitAt.Format("2006-01-02 15:04"))
	}
}
