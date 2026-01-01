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
		assert.False(t, opts.UpdateConfig)
		assert.False(t, opts.DryRun)
	})

	t.Run("with verbose", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work").WithVerbose(true)

		assert.True(t, opts.Verbose)
	})

	t.Run("with update config", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work").WithUpdateConfig(true)

		assert.True(t, opts.UpdateConfig)
	})

	t.Run("with dry run", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work").WithDryRun(true)

		assert.True(t, opts.DryRun)
	})

	t.Run("chained options", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("preflight.yaml", "work").
			WithVerbose(true).
			WithUpdateConfig(true).
			WithDryRun(true)

		assert.True(t, opts.Verbose)
		assert.True(t, opts.UpdateConfig)
		assert.True(t, opts.DryRun)
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

func TestFixResult(t *testing.T) {
	t.Parallel()

	t.Run("all fixed when no remaining issues", func(t *testing.T) {
		t.Parallel()
		result := FixResult{
			FixedIssues: []DoctorIssue{
				{StepID: "brew.git", Severity: SeverityWarning},
			},
			RemainingIssues: []DoctorIssue{},
		}

		assert.True(t, result.AllFixed())
		assert.Equal(t, 0, result.RemainingCount())
		assert.Equal(t, 1, result.FixedCount())
	})

	t.Run("not all fixed when issues remain", func(t *testing.T) {
		t.Parallel()
		result := FixResult{
			FixedIssues: []DoctorIssue{
				{StepID: "brew.git", Severity: SeverityWarning},
			},
			RemainingIssues: []DoctorIssue{
				{StepID: "brew.curl", Severity: SeverityError},
				{StepID: "files.dotfiles", Severity: SeverityWarning},
			},
		}

		assert.False(t, result.AllFixed())
		assert.Equal(t, 2, result.RemainingCount())
		assert.Equal(t, 1, result.FixedCount())
	})

	t.Run("all fixed with empty result", func(t *testing.T) {
		t.Parallel()
		result := FixResult{}

		assert.True(t, result.AllFixed())
		assert.Equal(t, 0, result.RemainingCount())
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

func TestConfigPatch(t *testing.T) {
	t.Parallel()

	t.Run("new patch", func(t *testing.T) {
		t.Parallel()
		patch := NewConfigPatch(
			"layers/base.yaml",
			"files.links[0].src",
			PatchOpModify,
			"old_value",
			"new_value",
			"drift",
		)

		assert.Equal(t, "layers/base.yaml", patch.LayerPath)
		assert.Equal(t, "files.links[0].src", patch.YAMLPath)
		assert.Equal(t, PatchOpModify, patch.Operation)
		assert.Equal(t, "old_value", patch.OldValue)
		assert.Equal(t, "new_value", patch.NewValue)
		assert.Equal(t, "drift", patch.Provenance)
	})

	t.Run("patch operations", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, PatchOpAdd, PatchOp("add"))
		assert.Equal(t, PatchOpModify, PatchOp("modify"))
		assert.Equal(t, PatchOpRemove, PatchOp("remove"))
	})

	t.Run("patch description", func(t *testing.T) {
		t.Parallel()

		addPatch := NewConfigPatch("layer.yaml", "key", PatchOpAdd, nil, "value", "drift")
		assert.Contains(t, addPatch.Description(), "Add")

		modPatch := NewConfigPatch("layer.yaml", "key", PatchOpModify, "old", "new", "drift")
		assert.Contains(t, modPatch.Description(), "Modify")

		rmPatch := NewConfigPatch("layer.yaml", "key", PatchOpRemove, "old", nil, "drift")
		assert.Contains(t, rmPatch.Description(), "Remove")

		unknownPatch := ConfigPatch{
			LayerPath: "layer.yaml",
			YAMLPath:  "key",
			Operation: PatchOp("unknown"),
		}
		assert.Contains(t, unknownPatch.Description(), "Unknown")
	})
}

func TestDoctorReportWithPatches(t *testing.T) {
	t.Parallel()

	t.Run("no patches", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{}

		assert.False(t, report.HasPatches())
		assert.Equal(t, 0, report.PatchCount())
	})

	t.Run("with patches", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			SuggestedPatches: []ConfigPatch{
				NewConfigPatch("layer.yaml", "key1", PatchOpAdd, nil, "value1", "drift"),
				NewConfigPatch("layer.yaml", "key2", PatchOpModify, "old", "new", "drift"),
			},
		}

		assert.True(t, report.HasPatches())
		assert.Equal(t, 2, report.PatchCount())
	})

	t.Run("patches by layer", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			SuggestedPatches: []ConfigPatch{
				NewConfigPatch("layers/base.yaml", "key1", PatchOpAdd, nil, "v1", "drift"),
				NewConfigPatch("layers/base.yaml", "key2", PatchOpModify, "o", "n", "drift"),
				NewConfigPatch("layers/work.yaml", "key3", PatchOpRemove, "v", nil, "drift"),
			},
		}

		byLayer := report.PatchesByLayer()
		assert.Len(t, byLayer["layers/base.yaml"], 2)
		assert.Len(t, byLayer["layers/work.yaml"], 1)
	})
}

func TestDoctorOptions_SecurityOptions(t *testing.T) {
	t.Parallel()

	t.Run("with security enabled", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithSecurity(true)
		assert.True(t, opts.SecurityEnabled)
	})

	t.Run("with security scanner", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithSecurityScanner("grype")
		assert.Equal(t, "grype", opts.SecurityScanner)
	})

	t.Run("with security severity", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithSecuritySeverity("high")
		assert.Equal(t, "high", opts.SecuritySeverity)
	})

	t.Run("with security ignore", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithSecurityIgnore([]string{"CVE-2024-1234"})
		assert.Equal(t, []string{"CVE-2024-1234"}, opts.SecurityIgnore)
	})

	t.Run("with security fail on", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithSecurityFailOn("critical")
		assert.Equal(t, "critical", opts.SecurityFailOn)
	})

	t.Run("with outdated enabled", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithOutdated(true)
		assert.True(t, opts.OutdatedEnabled)
	})

	t.Run("with outdated max age", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithOutdatedMaxAge(90 * 24 * time.Hour)
		assert.Equal(t, 90*24*time.Hour, opts.OutdatedMaxAge)
	})

	t.Run("with deprecated enabled", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").WithDeprecated(true)
		assert.True(t, opts.DeprecatedEnabled)
	})

	t.Run("with quick mode", func(t *testing.T) {
		t.Parallel()
		opts := NewDoctorOptions("", "").
			WithSecurity(true).
			WithOutdated(true).
			WithQuick(true)
		assert.True(t, opts.Quick)
		assert.False(t, opts.SecurityEnabled)
		assert.False(t, opts.OutdatedEnabled)
	})
}

func TestDoctorReport_SecurityHelpers(t *testing.T) {
	t.Parallel()

	t.Run("no security issues by default", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{}
		assert.False(t, report.HasSecurityIssues())
		assert.Equal(t, 0, report.SecurityVulnerabilityCount())
		assert.False(t, report.HasCriticalVulnerabilities())
	})

	t.Run("no outdated packages by default", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{}
		assert.False(t, report.HasOutdatedPackages())
		assert.Equal(t, 0, report.OutdatedCount())
	})

	t.Run("no deprecated packages by default", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{}
		assert.False(t, report.HasDeprecatedPackages())
		assert.Equal(t, 0, report.DeprecatedCount())
	})

	t.Run("total health issues empty report", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{}
		assert.Equal(t, 0, report.TotalHealthIssues())
	})

	t.Run("has outdated packages", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			OutdatedPackages: []OutdatedPackage{
				{Name: "git", CurrentVersion: "2.40.0", LatestVersion: "2.43.0"},
			},
		}
		assert.True(t, report.HasOutdatedPackages())
		assert.Equal(t, 1, report.OutdatedCount())
	})

	t.Run("has deprecated packages", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			DeprecatedPackages: []DeprecatedPackage{
				{Name: "python@2", Reason: "deprecated"},
			},
		}
		assert.True(t, report.HasDeprecatedPackages())
		assert.Equal(t, 1, report.DeprecatedCount())
	})

	t.Run("total health issues with all types", func(t *testing.T) {
		t.Parallel()
		report := DoctorReport{
			Issues: []DoctorIssue{
				{Severity: SeverityError, Message: "test"},
			},
			OutdatedPackages: []OutdatedPackage{
				{Name: "git"},
				{Name: "curl"},
			},
			DeprecatedPackages: []DeprecatedPackage{
				{Name: "python@2"},
			},
		}
		// 1 issue + 2 outdated + 1 deprecated = 4
		assert.Equal(t, 4, report.TotalHealthIssues())
	})
}
