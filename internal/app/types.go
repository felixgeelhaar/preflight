package app

import (
	"time"
)

// CaptureOptions configures the capture operation.
type CaptureOptions struct {
	// Providers to capture from (empty means all)
	Providers []string
	// IncludeSecrets controls whether to include sensitive data
	IncludeSecrets bool
	// HomeDir overrides the home directory for capture
	HomeDir string
}

// NewCaptureOptions creates default capture options.
func NewCaptureOptions() CaptureOptions {
	return CaptureOptions{
		Providers:      []string{},
		IncludeSecrets: false,
	}
}

// WithProviders sets the providers to capture.
func (o CaptureOptions) WithProviders(providers ...string) CaptureOptions {
	o.Providers = providers
	return o
}

// WithSecrets enables secret capture.
func (o CaptureOptions) WithSecrets(include bool) CaptureOptions {
	o.IncludeSecrets = include
	return o
}

// CapturedItem represents a single captured configuration item.
type CapturedItem struct {
	Provider   string
	Name       string
	Value      interface{}
	Source     string // e.g., "~/.gitconfig", "brew list"
	Redacted   bool
	CapturedAt time.Time
}

// CaptureFindings holds the results of a capture operation.
type CaptureFindings struct {
	Items      []CapturedItem
	Providers  []string
	CapturedAt time.Time
	HomeDir    string
	Warnings   []string
}

// ItemCount returns the total number of captured items.
func (f CaptureFindings) ItemCount() int {
	return len(f.Items)
}

// ItemsByProvider returns items grouped by provider.
func (f CaptureFindings) ItemsByProvider() map[string][]CapturedItem {
	result := make(map[string][]CapturedItem)
	for _, item := range f.Items {
		result[item.Provider] = append(result[item.Provider], item)
	}
	return result
}

// DoctorOptions configures the doctor operation.
type DoctorOptions struct {
	// ConfigPath is the path to the config file
	ConfigPath string
	// Target is the target to check
	Target string
	// Verbose enables detailed output
	Verbose bool
}

// NewDoctorOptions creates default doctor options.
func NewDoctorOptions(configPath, target string) DoctorOptions {
	return DoctorOptions{
		ConfigPath: configPath,
		Target:     target,
		Verbose:    false,
	}
}

// WithVerbose enables verbose output.
func (o DoctorOptions) WithVerbose(verbose bool) DoctorOptions {
	o.Verbose = verbose
	return o
}

// IssueSeverity indicates the severity of a doctor issue.
type IssueSeverity string

// IssueSeverity constants.
const (
	SeverityInfo    IssueSeverity = "info"
	SeverityWarning IssueSeverity = "warning"
	SeverityError   IssueSeverity = "error"
)

// DoctorIssue represents a single issue found by doctor.
type DoctorIssue struct {
	Provider   string
	StepID     string
	Severity   IssueSeverity
	Message    string
	Expected   string
	Actual     string
	Fixable    bool
	FixCommand string
}

// DoctorReport holds the results of a doctor check.
type DoctorReport struct {
	ConfigPath string
	Target     string
	Issues     []DoctorIssue
	CheckedAt  time.Time
	Duration   time.Duration
}

// IssueCount returns the total number of issues.
func (r DoctorReport) IssueCount() int {
	return len(r.Issues)
}

// HasIssues returns true if there are any issues.
func (r DoctorReport) HasIssues() bool {
	return len(r.Issues) > 0
}

// FixableCount returns the number of fixable issues.
func (r DoctorReport) FixableCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Fixable {
			count++
		}
	}
	return count
}

// IssuesBySeverity returns issues grouped by severity.
func (r DoctorReport) IssuesBySeverity() map[IssueSeverity][]DoctorIssue {
	result := make(map[IssueSeverity][]DoctorIssue)
	for _, issue := range r.Issues {
		result[issue.Severity] = append(result[issue.Severity], issue)
	}
	return result
}

// ErrorCount returns the number of error-severity issues.
func (r DoctorReport) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-severity issues.
func (r DoctorReport) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// FixResult holds the results of a fix operation.
type FixResult struct {
	// FixedIssues are issues that were successfully fixed
	FixedIssues []DoctorIssue
	// RemainingIssues are issues that could not be fixed
	RemainingIssues []DoctorIssue
	// VerificationReport is the doctor report after fixing
	VerificationReport *DoctorReport
}

// FixedCount returns the number of fixed issues.
func (r FixResult) FixedCount() int {
	return len(r.FixedIssues)
}

// RemainingCount returns the number of remaining issues.
func (r FixResult) RemainingCount() int {
	return len(r.RemainingIssues)
}

// AllFixed returns true if all fixable issues were resolved.
func (r FixResult) AllFixed() bool {
	return len(r.RemainingIssues) == 0
}

// DiffEntry represents a single difference between config and system.
type DiffEntry struct {
	Provider string
	Path     string
	Type     DiffType
	Expected string
	Actual   string
}

// DiffType indicates the type of difference.
type DiffType string

// DiffType constants.
const (
	DiffTypeAdded    DiffType = "added"
	DiffTypeRemoved  DiffType = "removed"
	DiffTypeModified DiffType = "modified"
)

// DiffResult holds the results of a diff operation.
type DiffResult struct {
	ConfigPath string
	Target     string
	Entries    []DiffEntry
	DiffedAt   time.Time
}

// HasDifferences returns true if there are any differences.
func (r DiffResult) HasDifferences() bool {
	return len(r.Entries) > 0
}

// EntriesByProvider returns entries grouped by provider.
func (r DiffResult) EntriesByProvider() map[string][]DiffEntry {
	result := make(map[string][]DiffEntry)
	for _, entry := range r.Entries {
		result[entry.Provider] = append(result[entry.Provider], entry)
	}
	return result
}

// RepoOptions configures repository operations.
type RepoOptions struct {
	// Path is the repository path
	Path string
	// Remote is the git remote URL
	Remote string
	// Branch is the git branch
	Branch string
}

// NewRepoOptions creates default repo options.
func NewRepoOptions(path string) RepoOptions {
	return RepoOptions{
		Path:   path,
		Branch: "main",
	}
}

// WithRemote sets the remote URL.
func (o RepoOptions) WithRemote(remote string) RepoOptions {
	o.Remote = remote
	return o
}

// WithBranch sets the branch.
func (o RepoOptions) WithBranch(branch string) RepoOptions {
	o.Branch = branch
	return o
}

// RepoStatus holds the status of a configuration repository.
type RepoStatus struct {
	Path         string
	Initialized  bool
	Branch       string
	Remote       string
	HasChanges   bool
	Ahead        int
	Behind       int
	LastCommit   string
	LastCommitAt time.Time
}

// IsSynced returns true if the repo is in sync with remote.
func (s RepoStatus) IsSynced() bool {
	return s.Ahead == 0 && s.Behind == 0 && !s.HasChanges
}

// NeedsPush returns true if local commits need to be pushed.
func (s RepoStatus) NeedsPush() bool {
	return s.Ahead > 0
}

// NeedsPull returns true if remote commits need to be pulled.
func (s RepoStatus) NeedsPull() bool {
	return s.Behind > 0
}
