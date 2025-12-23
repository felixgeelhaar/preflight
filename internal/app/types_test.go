package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCaptureOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		opts := NewCaptureOptions()

		assert.Empty(t, opts.Providers)
		assert.False(t, opts.IncludeSecrets)
	})

	t.Run("with providers", func(t *testing.T) {
		t.Parallel()
		opts := NewCaptureOptions().WithProviders("brew", "git")

		assert.Equal(t, []string{"brew", "git"}, opts.Providers)
	})

	t.Run("with secrets", func(t *testing.T) {
		t.Parallel()
		opts := NewCaptureOptions().WithSecrets(true)

		assert.True(t, opts.IncludeSecrets)
	})
}

func TestCaptureFindings(t *testing.T) {
	t.Parallel()

	t.Run("item count", func(t *testing.T) {
		t.Parallel()
		findings := CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git"},
				{Provider: "brew", Name: "curl"},
				{Provider: "git", Name: "user.name"},
			},
		}

		assert.Equal(t, 3, findings.ItemCount())
	})

	t.Run("items by provider", func(t *testing.T) {
		t.Parallel()
		findings := CaptureFindings{
			Items: []CapturedItem{
				{Provider: "brew", Name: "git"},
				{Provider: "brew", Name: "curl"},
				{Provider: "git", Name: "user.name"},
			},
		}

		byProvider := findings.ItemsByProvider()

		assert.Len(t, byProvider["brew"], 2)
		assert.Len(t, byProvider["git"], 1)
	})

	t.Run("empty findings", func(t *testing.T) {
		t.Parallel()
		findings := CaptureFindings{}

		assert.Equal(t, 0, findings.ItemCount())
		assert.Empty(t, findings.ItemsByProvider())
	})
}

func TestDoctorOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work")

		assert.Equal(t, "preflight.yaml", opts.ConfigPath)
		assert.Equal(t, "work", opts.Target)
		assert.False(t, opts.Verbose)
	})

	t.Run("with verbose", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work").WithVerbose(true)

		assert.True(t, opts.Verbose)
	})
}

func TestDoctorReport(t *testing.T) {
	t.Parallel()

	t.Run("no issues", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			ConfigPath: "preflight.yaml",
			Target:     "work",
		}

		assert.False(t, report.HasIssues())
		assert.Equal(t, 0, report.IssueCount())
		assert.Equal(t, 0, report.FixableCount())
	})

	t.Run("with issues", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			Issues: []DoctorIssue{
				{Severity: SeverityError, Fixable: true},
				{Severity: SeverityError, Fixable: false},
				{Severity: SeverityWarning, Fixable: true},
				{Severity: SeverityInfo, Fixable: false},
			},
		}

		assert.True(t, report.HasIssues())
		assert.Equal(t, 4, report.IssueCount())
		assert.Equal(t, 2, report.FixableCount())
		assert.Equal(t, 2, report.ErrorCount())
		assert.Equal(t, 1, report.WarningCount())
	})

	t.Run("issues by severity", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			Issues: []DoctorIssue{
				{Severity: SeverityError},
				{Severity: SeverityError},
				{Severity: SeverityWarning},
			},
		}

		bySeverity := report.IssuesBySeverity()

		assert.Len(t, bySeverity[SeverityError], 2)
		assert.Len(t, bySeverity[SeverityWarning], 1)
		assert.Empty(t, bySeverity[SeverityInfo])
	})

	t.Run("no binary issues", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			BinaryChecks: []BinaryCheckResult{
				{Name: "nvim", Found: true, MeetsMin: true, Required: true},
				{Name: "rg", Found: true, MeetsMin: true, Required: false},
			},
		}

		assert.False(t, report.HasBinaryIssues())
		assert.Equal(t, 0, report.BinaryIssueCount())
	})

	t.Run("required binary missing", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			BinaryChecks: []BinaryCheckResult{
				{Name: "nvim", Found: false, MeetsMin: false, Required: true},
				{Name: "rg", Found: false, MeetsMin: false, Required: false},
			},
		}

		assert.True(t, report.HasBinaryIssues())
		assert.Equal(t, 1, report.BinaryIssueCount())
	})

	t.Run("required binary version too low", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			BinaryChecks: []BinaryCheckResult{
				{Name: "nvim", Found: true, MeetsMin: false, Required: true, Version: "0.8.0", MinVersion: "0.9.0"},
			},
		}

		assert.True(t, report.HasBinaryIssues())
		assert.Equal(t, 1, report.BinaryIssueCount())
	})

	t.Run("optional binary missing no issue", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			BinaryChecks: []BinaryCheckResult{
				{Name: "nvim", Found: true, MeetsMin: true, Required: true},
				{Name: "rg", Found: false, MeetsMin: false, Required: false},
				{Name: "fd", Found: false, MeetsMin: false, Required: false},
			},
		}

		assert.False(t, report.HasBinaryIssues())
		assert.Equal(t, 0, report.BinaryIssueCount())
	})
}

func TestIssueSeverity(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SeverityInfo, IssueSeverity("info"))
	assert.Equal(t, SeverityWarning, IssueSeverity("warning"))
	assert.Equal(t, SeverityError, IssueSeverity("error"))
}

func TestDiffResult(t *testing.T) {
	t.Parallel()

	t.Run("no differences", func(t *testing.T) {
		t.Parallel()
		result := DiffResult{}

		assert.False(t, result.HasDifferences())
	})

	t.Run("with differences", func(t *testing.T) {
		t.Parallel()
		result := DiffResult{
			Entries: []DiffEntry{
				{Provider: "brew", Type: DiffTypeAdded},
				{Provider: "brew", Type: DiffTypeRemoved},
				{Provider: "git", Type: DiffTypeModified},
			},
		}

		assert.True(t, result.HasDifferences())

		byProvider := result.EntriesByProvider()
		assert.Len(t, byProvider["brew"], 2)
		assert.Len(t, byProvider["git"], 1)
	})
}

func TestDiffType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DiffTypeAdded, DiffType("added"))
	assert.Equal(t, DiffTypeRemoved, DiffType("removed"))
	assert.Equal(t, DiffTypeModified, DiffType("modified"))
}

func TestRepoOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		opts := NewRepoOptions("/path/to/config")

		assert.Equal(t, "/path/to/config", opts.Path)
		assert.Equal(t, "main", opts.Branch)
		assert.Empty(t, opts.Remote)
	})

	t.Run("with remote", func(t *testing.T) {
		t.Parallel()
		opts := NewRepoOptions("/path").WithRemote("git@github.com:user/config.git")

		assert.Equal(t, "git@github.com:user/config.git", opts.Remote)
	})

	t.Run("with branch", func(t *testing.T) {
		t.Parallel()
		opts := NewRepoOptions("/path").WithBranch("develop")

		assert.Equal(t, "develop", opts.Branch)
	})
}

func TestRepoStatus(t *testing.T) {
	t.Parallel()

	t.Run("synced repo", func(t *testing.T) {
		t.Parallel()
		status := RepoStatus{
			Initialized: true,
			Ahead:       0,
			Behind:      0,
			HasChanges:  false,
		}

		assert.True(t, status.IsSynced())
		assert.False(t, status.NeedsPush())
		assert.False(t, status.NeedsPull())
	})

	t.Run("needs push", func(t *testing.T) {
		t.Parallel()
		status := RepoStatus{
			Ahead:  2,
			Behind: 0,
		}

		assert.False(t, status.IsSynced())
		assert.True(t, status.NeedsPush())
		assert.False(t, status.NeedsPull())
	})

	t.Run("needs pull", func(t *testing.T) {
		t.Parallel()
		status := RepoStatus{
			Ahead:  0,
			Behind: 3,
		}

		assert.False(t, status.IsSynced())
		assert.False(t, status.NeedsPush())
		assert.True(t, status.NeedsPull())
	})

	t.Run("has uncommitted changes", func(t *testing.T) {
		t.Parallel()
		status := RepoStatus{
			Ahead:      0,
			Behind:     0,
			HasChanges: true,
		}

		assert.False(t, status.IsSynced())
	})

	t.Run("full status", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		status := RepoStatus{
			Path:         "/config",
			Initialized:  true,
			Branch:       "main",
			Remote:       "origin",
			HasChanges:   false,
			Ahead:        1,
			Behind:       2,
			LastCommit:   "abc123",
			LastCommitAt: now,
		}

		assert.Equal(t, "/config", status.Path)
		assert.True(t, status.Initialized)
		assert.Equal(t, "main", status.Branch)
		assert.Equal(t, "abc123", status.LastCommit)
	})
}
