package mcp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAll(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "preflight.yaml", "default")

	// Verify all tools are registered
	tools := srv.Tools()
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["preflight_plan"], "preflight_plan should be registered")
	assert.True(t, toolNames["preflight_apply"], "preflight_apply should be registered")
	assert.True(t, toolNames["preflight_doctor"], "preflight_doctor should be registered")
	assert.True(t, toolNames["preflight_validate"], "preflight_validate should be registered")
	assert.True(t, toolNames["preflight_status"], "preflight_status should be registered")
}

func TestPlanTool_NoConfig(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "nonexistent.yaml", "default")

	// Find the tool in the tools list
	var planTool *struct{ Name, Description string }
	for _, tool := range srv.Tools() {
		if tool.Name == "preflight_plan" {
			planTool = &struct{ Name, Description string }{tool.Name, tool.Description}
			break
		}
	}
	require.NotNil(t, planTool, "preflight_plan tool should exist")
	assert.Equal(t, "preflight_plan", planTool.Name)
	assert.Contains(t, planTool.Description, "Show what changes")
}

func TestValidateTool_NoConfig(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "nonexistent.yaml", "default")

	// Find the tool in the tools list
	var validateTool *struct{ Name, Description string }
	for _, tool := range srv.Tools() {
		if tool.Name == "preflight_validate" {
			validateTool = &struct{ Name, Description string }{tool.Name, tool.Description}
			break
		}
	}
	require.NotNil(t, validateTool, "preflight_validate tool should exist")
	assert.Equal(t, "preflight_validate", validateTool.Name)
	assert.Contains(t, validateTool.Description, "Validate configuration")
}

func TestStatusTool_NoConfig(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "nonexistent.yaml", "default")

	// Find the tool in the tools list
	var statusTool *struct{ Name, Description string }
	for _, tool := range srv.Tools() {
		if tool.Name == "preflight_status" {
			statusTool = &struct{ Name, Description string }{tool.Name, tool.Description}
			break
		}
	}
	require.NotNil(t, statusTool, "preflight_status tool should exist")
	assert.Equal(t, "preflight_status", statusTool.Name)
	assert.Contains(t, statusTool.Description, "current preflight status")
}

func TestDoctorTool_Description(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "preflight.yaml", "default")

	// Find the tool in the tools list
	var doctorTool *struct{ Name, Description string }
	for _, tool := range srv.Tools() {
		if tool.Name == "preflight_doctor" {
			doctorTool = &struct{ Name, Description string }{tool.Name, tool.Description}
			break
		}
	}
	require.NotNil(t, doctorTool, "preflight_doctor tool should exist")
	assert.Contains(t, doctorTool.Description, "Verify system state")
}

func TestApplyTool_Description(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "preflight.yaml", "default")

	// Find the tool in the tools list
	var applyTool *struct{ Name, Description string }
	for _, tool := range srv.Tools() {
		if tool.Name == "preflight_apply" {
			applyTool = &struct{ Name, Description string }{tool.Name, tool.Description}
			break
		}
	}
	require.NotNil(t, applyTool, "preflight_apply tool should exist")
	assert.Contains(t, applyTool.Description, "Apply configuration changes")
	assert.Contains(t, applyTool.Description, "confirmation")
}

// TestPlanWithValidConfig tests the plan tool with a valid configuration.
func TestPlanWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	configContent := `targets:
  default:
    - base
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create an empty base layer
	layersDir := filepath.Join(tmpDir, "layers")
	err = os.MkdirAll(layersDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir for the test
	oldWd, _ := os.Getwd()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	preflight := app.New(bytes.NewBuffer(nil))
	ctx := context.Background()

	// Call Plan directly
	plan, err := preflight.Plan(ctx, configPath, "default")
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.False(t, plan.HasChanges())
}

// TestValidateWithValidConfig tests the validate tool with a valid configuration.
func TestValidateWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	configContent := `targets:
  default:
    - base
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create an empty base layer
	layersDir := filepath.Join(tmpDir, "layers")
	err = os.MkdirAll(layersDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir for the test
	oldWd, _ := os.Getwd()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	preflight := app.New(bytes.NewBuffer(nil))
	ctx := context.Background()

	// Call Validate directly
	result, err := preflight.Validate(ctx, configPath, "default")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Errors)
}

// TestPlanOutputTypes verifies the output types are correct.
func TestPlanOutputTypes(t *testing.T) {
	output := &PlanOutput{
		HasChanges: true,
		Summary: PlanSummary{
			Total:      10,
			NeedsApply: 3,
			Satisfied:  7,
			Failed:     0,
			Unknown:    0,
		},
		Steps: []PlanStep{
			{
				ID:          "brew:install:git",
				Provider:    "brew",
				Status:      "needs_apply",
				DiffSummary: "Install git",
			},
		},
	}

	assert.True(t, output.HasChanges)
	assert.Equal(t, 10, output.Summary.Total)
	assert.Len(t, output.Steps, 1)
	assert.Equal(t, "brew", output.Steps[0].Provider)
}

// TestApplyOutputTypes verifies the apply output types.
func TestApplyOutputTypes(t *testing.T) {
	output := &ApplyOutput{
		DryRun:    true,
		Succeeded: 5,
		Failed:    0,
		Skipped:   2,
		Results: []ApplyResult{
			{
				StepID: "brew:install:git",
				Status: "satisfied",
			},
		},
	}

	assert.True(t, output.DryRun)
	assert.Equal(t, 5, output.Succeeded)
	assert.Len(t, output.Results, 1)
}

// TestDoctorOutputTypes verifies the doctor output types.
func TestDoctorOutputTypes(t *testing.T) {
	output := &DoctorOutput{
		Healthy:      false,
		IssueCount:   2,
		FixableCount: 1,
		Duration:     "1.5s",
		Issues: []DoctorIssue{
			{
				Provider:   "brew",
				StepID:     "brew:install:missing-pkg",
				Severity:   "warning",
				Message:    "Package not installed",
				Fixable:    true,
				FixCommand: "preflight apply",
			},
		},
	}

	assert.False(t, output.Healthy)
	assert.Equal(t, 2, output.IssueCount)
	assert.Len(t, output.Issues, 1)
	assert.True(t, output.Issues[0].Fixable)
}

// TestValidateOutputTypes verifies the validate output types.
func TestValidateOutputTypes(t *testing.T) {
	output := &ValidateOutput{
		Valid:            false,
		Errors:           []string{"Missing required field"},
		Warnings:         []string{"Deprecated syntax"},
		PolicyViolations: []string{"Forbidden package"},
		Info:             []string{"Loaded 5 layers"},
	}

	assert.False(t, output.Valid)
	assert.Len(t, output.Errors, 1)
	assert.Len(t, output.Warnings, 1)
	assert.Len(t, output.PolicyViolations, 1)
	assert.Len(t, output.Info, 1)
}

// TestStatusOutputTypes verifies the status output types.
func TestStatusOutputTypes(t *testing.T) {
	output := &StatusOutput{
		ConfigExists: true,
		ConfigPath:   "preflight.yaml",
		Target:       "work",
		IsValid:      true,
		StepCount:    25,
		HasDrift:     true,
		DriftCount:   3,
		Repo: &RepoStatus{
			Initialized:      true,
			Branch:           "main",
			RemoteConfigured: true,
			IsSynced:         false,
			NeedsPush:        true,
			NeedsPull:        false,
			HasChanges:       false,
		},
	}

	assert.True(t, output.ConfigExists)
	assert.Equal(t, "work", output.Target)
	assert.Equal(t, 25, output.StepCount)
	assert.NotNil(t, output.Repo)
	assert.True(t, output.Repo.NeedsPush)
}

// Phase 2: Configuration Management Output Types

// TestCaptureOutputTypes verifies the capture output types.
func TestCaptureOutputTypes(t *testing.T) {
	output := &CaptureOutput{
		Items: []CapturedItem{
			{
				Name:     "git",
				Provider: "brew",
				Value:    "2.42.0",
				Source:   "brew list",
				Redacted: false,
			},
		},
		Providers:  []string{"brew", "git", "ssh"},
		CapturedAt: "2024-01-15T10:30:00Z",
		Warnings:   []string{"Some packages were skipped"},
	}

	assert.Len(t, output.Items, 1)
	assert.Equal(t, "git", output.Items[0].Name)
	assert.Equal(t, "brew", output.Items[0].Provider)
	assert.Len(t, output.Providers, 3)
	assert.NotEmpty(t, output.CapturedAt)
	assert.Len(t, output.Warnings, 1)
}

// TestDiffOutputTypes verifies the diff output types.
func TestDiffOutputTypes(t *testing.T) {
	output := &DiffOutput{
		HasDifferences: true,
		Differences: []DiffItem{
			{
				Provider: "brew",
				Path:     "formulae/git",
				Type:     "added",
				Expected: "2.42.0",
				Actual:   "",
			},
		},
	}

	assert.True(t, output.HasDifferences)
	assert.Len(t, output.Differences, 1)
	assert.Equal(t, "brew", output.Differences[0].Provider)
	assert.Equal(t, "added", output.Differences[0].Type)
}

// TestTourOutputTypes verifies the tour output types.
func TestTourOutputTypes(t *testing.T) {
	output := &TourOutput{
		Topics: []TourTopic{
			{
				ID:          "getting-started",
				Title:       "Getting Started",
				Description: "Learn the basics of preflight",
			},
		},
	}

	assert.Len(t, output.Topics, 1)
	assert.Equal(t, "getting-started", output.Topics[0].ID)
	assert.Equal(t, "Getting Started", output.Topics[0].Title)
}

// Phase 3: Advanced Features Output Types

// TestSecurityOutputTypes verifies the security output types.
func TestSecurityOutputTypes(t *testing.T) {
	output := &SecurityOutput{
		Scanner: "grype",
		Version: "0.70.0",
		Vulnerabilities: []Vulnerability{
			{
				ID:        "CVE-2024-1234",
				Package:   "openssl",
				Version:   "1.1.1",
				Severity:  "critical",
				CVSS:      9.8,
				FixedIn:   "1.1.2",
				Title:     "Buffer overflow vulnerability",
				Reference: "https://nvd.nist.gov/vuln/detail/CVE-2024-1234",
			},
		},
		Summary: &SecuritySummary{
			TotalVulnerabilities: 1,
			Critical:             1,
			High:                 0,
			Medium:               0,
			Low:                  0,
			PackagesScanned:      50,
			FixableCount:         1,
		},
	}

	assert.Equal(t, "grype", output.Scanner)
	assert.Len(t, output.Vulnerabilities, 1)
	assert.Equal(t, "CVE-2024-1234", output.Vulnerabilities[0].ID)
	assert.Equal(t, "critical", output.Vulnerabilities[0].Severity)
	assert.NotNil(t, output.Summary)
	assert.Equal(t, 1, output.Summary.Critical)
}

// TestOutdatedOutputTypes verifies the outdated output types.
func TestOutdatedOutputTypes(t *testing.T) {
	output := &OutdatedOutput{
		Packages: []OutdatedPackage{
			{
				Name:           "node",
				CurrentVersion: "18.0.0",
				LatestVersion:  "20.0.0",
				UpdateType:     "major",
			},
			{
				Name:           "go",
				CurrentVersion: "1.21.0",
				LatestVersion:  "1.22.0",
				UpdateType:     "minor",
			},
		},
		Summary: OutdatedSummary{
			Total: 2,
			Major: 1,
			Minor: 1,
			Patch: 0,
		},
	}

	assert.Len(t, output.Packages, 2)
	assert.Equal(t, "node", output.Packages[0].Name)
	assert.Equal(t, "major", output.Packages[0].UpdateType)
	assert.Equal(t, 2, output.Summary.Total)
	assert.Equal(t, 1, output.Summary.Major)
}

// TestRollbackOutputTypes verifies the rollback output types.
func TestRollbackOutputTypes(t *testing.T) {
	output := &RollbackOutput{
		Snapshots: []SnapshotInfo{
			{
				ID:        "abc12345-def6-7890",
				ShortID:   "abc12345",
				CreatedAt: "2024-01-15T10:30:00Z",
				Age:       "2 hours ago",
				FileCount: 5,
				Reason:    "before apply",
			},
		},
		RestoredFiles:  0,
		TargetSnapshot: "",
		DryRun:         false,
		Message:        "Available snapshots listed. Use snapshot_id to restore.",
	}

	assert.Len(t, output.Snapshots, 1)
	assert.Equal(t, "abc12345", output.Snapshots[0].ShortID)
	assert.Equal(t, 5, output.Snapshots[0].FileCount)
	assert.NotEmpty(t, output.Message)
}

// TestSyncOutputTypes verifies the sync output types.
func TestSyncOutputTypes(t *testing.T) {
	output := &SyncOutput{
		DryRun:       false,
		Branch:       "main",
		Behind:       0,
		Ahead:        2,
		Pulled:       true,
		Pushed:       true,
		AppliedSteps: 5,
		Message:      "Sync completed",
	}

	assert.True(t, output.Pushed)
	assert.True(t, output.Pulled)
	assert.Equal(t, 5, output.AppliedSteps)
	assert.Equal(t, "main", output.Branch)
	assert.Equal(t, "Sync completed", output.Message)
}

// TestMarketplaceOutputTypes verifies the marketplace output types.
func TestMarketplaceOutputTypes(t *testing.T) {
	output := &MarketplaceOutput{
		Packages: []MarketplacePackage{
			{
				Name:        "nvim-kickstart",
				Title:       "Neovim Kickstart",
				Description: "A minimal Neovim configuration",
				Author:      "community",
				Type:        "preset",
				Version:     "1.0.0",
				Downloads:   1000,
				Keywords:    []string{"neovim", "editor"},
				Featured:    true,
			},
		},
	}

	assert.Len(t, output.Packages, 1)
	assert.Equal(t, "nvim-kickstart", output.Packages[0].Name)
	assert.Equal(t, "preset", output.Packages[0].Type)
	assert.True(t, output.Packages[0].Featured)
	assert.Len(t, output.Packages[0].Keywords, 2)
}

// TestRegisterAllNewTools verifies all Phase 2 and Phase 3 tools are registered.
func TestRegisterAllNewTools(t *testing.T) {
	preflight := app.New(bytes.NewBuffer(nil))
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "preflight-test",
		Version: "1.0.0",
	})

	RegisterAll(srv, preflight, "preflight.yaml", "default")

	// Verify all tools are registered
	tools := srv.Tools()
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	// Phase 1 tools
	assert.True(t, toolNames["preflight_plan"], "preflight_plan should be registered")
	assert.True(t, toolNames["preflight_apply"], "preflight_apply should be registered")
	assert.True(t, toolNames["preflight_doctor"], "preflight_doctor should be registered")
	assert.True(t, toolNames["preflight_validate"], "preflight_validate should be registered")
	assert.True(t, toolNames["preflight_status"], "preflight_status should be registered")

	// Phase 2 tools
	assert.True(t, toolNames["preflight_capture"], "preflight_capture should be registered")
	assert.True(t, toolNames["preflight_diff"], "preflight_diff should be registered")
	assert.True(t, toolNames["preflight_tour"], "preflight_tour should be registered")

	// Phase 3 tools
	assert.True(t, toolNames["preflight_security"], "preflight_security should be registered")
	assert.True(t, toolNames["preflight_outdated"], "preflight_outdated should be registered")
	assert.True(t, toolNames["preflight_rollback"], "preflight_rollback should be registered")
	assert.True(t, toolNames["preflight_sync"], "preflight_sync should be registered")
	assert.True(t, toolNames["preflight_marketplace"], "preflight_marketplace should be registered")
}
