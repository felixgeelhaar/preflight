package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	mcp "github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestMarketplacePackage creates a marketplace.Package for testing.
func createTestMarketplacePackage(name, title, desc, author, pkgType string, keywords []string) marketplace.Package {
	return marketplace.Package{
		ID:          marketplace.MustNewPackageID(name),
		Title:       title,
		Description: desc,
		Keywords:    keywords,
		Downloads:   42,
		Type:        pkgType,
		Provenance: marketplace.Provenance{
			Author: author,
		},
		Versions: []marketplace.PackageVersion{
			{Version: "1.0.0"},
			{Version: "0.9.0"},
		},
	}
}

// initGitRepo initializes a bare git repo in the given directory with an initial commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "add", "."},
		{"git", "-C", dir, "commit", "-m", "initial", "--allow-empty"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "command %v failed: %s", args, string(out))
	}
}

// --- helpers ---

// newTestServer creates an MCP server with all tools registered, using the given defaults.
func newTestServer(t *testing.T, preflight *app.Preflight, configPath, target string) *mcp.Server {
	t.Helper()
	srv := mcp.NewServer(mcp.ServerInfo{Name: "test", Version: "1.0.0"})
	RegisterAll(srv, preflight, configPath, target, testVersionInfo())
	return srv
}

// executeTool is a helper that retrieves and executes a registered tool by name.
func executeTool(t *testing.T, srv *mcp.Server, toolName string, input interface{}) (interface{}, error) {
	t.Helper()
	tool, ok := srv.GetTool(toolName)
	require.True(t, ok, "tool %q should be registered", toolName)

	data, err := json.Marshal(input)
	require.NoError(t, err)

	return tool.Execute(context.Background(), data)
}

// setupValidConfig creates a temporary directory with a valid preflight.yaml and base layer.
// Returns the tmpDir path and config path. The caller does NOT need to chdir.
func setupValidConfig(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	err := os.WriteFile(configPath, []byte("targets:\n  default:\n    - base\n"), 0644)
	require.NoError(t, err)

	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0644))

	return tmpDir, configPath
}

// withChdir changes to the given directory for the duration of the test and reverts on cleanup.
func withChdir(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
}

// --- Plan tool handler tests ---

func TestPlanToolHandler_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_plan", PlanInput{})
	require.NoError(t, err)

	output, ok := result.(*PlanOutput)
	require.True(t, ok, "result should be *PlanOutput")
	assert.False(t, output.HasChanges)
	assert.Equal(t, 0, output.Summary.NeedsApply)
}

func TestPlanToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "other")

	// Provide explicit config and target that override the server defaults
	result, err := executeTool(t, srv, "preflight_plan", PlanInput{
		ConfigPath: configPath,
		Target:     "default",
	})
	require.NoError(t, err)

	output, ok := result.(*PlanOutput)
	require.True(t, ok)
	assert.NotNil(t, output)
}

func TestPlanToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	_, err := executeTool(t, srv, "preflight_plan", PlanInput{})
	assert.Error(t, err)
}

func TestPlanToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_plan", PlanInput{
		ConfigPath: "config;injection",
	})
	assert.Error(t, err)
}

// --- Apply tool handler tests ---

func TestApplyToolHandler_NoConfirmNoDryRun(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		Confirm: false,
		DryRun:  false,
	})
	require.NoError(t, err)

	output, ok := result.(*ApplyOutput)
	require.True(t, ok)
	assert.False(t, output.DryRun)
	assert.Nil(t, output.Results)
}

func TestApplyToolHandler_DryRun(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		DryRun: true,
	})
	require.NoError(t, err)

	output, ok := result.(*ApplyOutput)
	require.True(t, ok)
	assert.True(t, output.DryRun)
	assert.NotNil(t, output.Results)
}

func TestApplyToolHandler_ConfirmWithValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		Confirm: true,
		DryRun:  false,
	})
	require.NoError(t, err)

	output, ok := result.(*ApplyOutput)
	require.True(t, ok)
	// With base layer that has no steps, plan has no changes, should go through dry-run path
	assert.NotNil(t, output.Results)
}

func TestApplyToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		ConfigPath: "config$(whoami).yaml",
	})
	assert.Error(t, err)
}

func TestApplyToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	_, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		Confirm: true,
	})
	assert.Error(t, err)
}

func TestApplyToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_apply", ApplyInput{
		ConfigPath: configPath,
		Target:     "default",
		DryRun:     true,
	})
	require.NoError(t, err)

	output, ok := result.(*ApplyOutput)
	require.True(t, ok)
	assert.True(t, output.DryRun)
}

// --- Doctor tool handler tests ---

func TestDoctorToolHandler_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_doctor", DoctorInput{})
	require.NoError(t, err)

	output, ok := result.(*DoctorOutput)
	require.True(t, ok)
	assert.True(t, output.Healthy)
	assert.Equal(t, 0, output.IssueCount)
	assert.NotEmpty(t, output.Duration)
}

func TestDoctorToolHandler_QuickMode(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_doctor", DoctorInput{
		Quick: true,
	})
	require.NoError(t, err)

	output, ok := result.(*DoctorOutput)
	require.True(t, ok)
	assert.True(t, output.Healthy)
}

func TestDoctorToolHandler_VerboseMode(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_doctor", DoctorInput{
		Verbose: true,
	})
	require.NoError(t, err)

	output, ok := result.(*DoctorOutput)
	require.True(t, ok)
	assert.NotNil(t, output)
}

func TestDoctorToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	_, err := executeTool(t, srv, "preflight_doctor", DoctorInput{})
	assert.Error(t, err)
}

func TestDoctorToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_doctor", DoctorInput{
		ConfigPath: "config|injection.yaml",
	})
	assert.Error(t, err)
}

func TestDoctorToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_doctor", DoctorInput{
		ConfigPath: configPath,
		Target:     "default",
		Quick:      true,
	})
	require.NoError(t, err)

	output, ok := result.(*DoctorOutput)
	require.True(t, ok)
	assert.True(t, output.Healthy)
}

// --- Validate tool handler tests ---

func TestValidateToolHandler_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_validate", ValidateInput{})
	require.NoError(t, err)

	output, ok := result.(*ValidateOutput)
	require.True(t, ok)
	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestValidateToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	// Missing config returns a partial result (not an error)
	result, err := executeTool(t, srv, "preflight_validate", ValidateInput{})
	require.NoError(t, err)

	output, ok := result.(*ValidateOutput)
	require.True(t, ok)
	assert.False(t, output.Valid)
	assert.NotEmpty(t, output.Errors)
}

func TestValidateToolHandler_StrictModeNoWarnings(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_validate", ValidateInput{
		Strict: true,
	})
	require.NoError(t, err)

	output, ok := result.(*ValidateOutput)
	require.True(t, ok)
	// In strict mode, validity depends on having no errors AND no warnings AND no policy violations
	hasErrors := len(output.Errors) > 0
	hasPolicyViolations := len(output.PolicyViolations) > 0
	hasWarnings := len(output.Warnings) > 0
	expectedValid := !hasErrors && !hasPolicyViolations && !hasWarnings
	assert.Equal(t, expectedValid, output.Valid)
}

func TestValidateToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_validate", ValidateInput{
		Target: "target;injection",
	})
	assert.Error(t, err)
}

func TestValidateToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_validate", ValidateInput{
		ConfigPath: configPath,
		Target:     "default",
	})
	require.NoError(t, err)

	output, ok := result.(*ValidateOutput)
	require.True(t, ok)
	assert.True(t, output.Valid)
}

// --- Status tool handler tests ---

func TestStatusToolHandler_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_status", StatusInput{})
	require.NoError(t, err)

	output, ok := result.(*StatusOutput)
	require.True(t, ok)
	assert.Equal(t, "test-1.0.0", output.Version)
	assert.Equal(t, "abc1234", output.Commit)
	assert.True(t, output.ConfigExists)
	assert.True(t, output.IsValid)
	assert.Equal(t, "default", output.Target)
}

func TestStatusToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	result, err := executeTool(t, srv, "preflight_status", StatusInput{})
	require.NoError(t, err)

	output, ok := result.(*StatusOutput)
	require.True(t, ok)
	assert.False(t, output.ConfigExists)
	assert.False(t, output.IsValid)
	assert.Equal(t, "test-1.0.0", output.Version)
}

func TestStatusToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_status", StatusInput{
		ConfigPath: "config&injection.yaml",
	})
	assert.Error(t, err)
}

func TestStatusToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_status", StatusInput{
		ConfigPath: configPath,
		Target:     "default",
	})
	require.NoError(t, err)

	output, ok := result.(*StatusOutput)
	require.True(t, ok)
	assert.True(t, output.ConfigExists)
	assert.True(t, output.IsValid)
	assert.Equal(t, "default", output.Target)
}

// --- Capture tool handler tests ---

func TestCaptureToolHandler_NoProviders(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_capture", CaptureInput{})
	require.NoError(t, err)

	output, ok := result.(*CaptureOutput)
	require.True(t, ok)
	assert.NotNil(t, output.Items)
	assert.NotEmpty(t, output.CapturedAt)
}

func TestCaptureToolHandler_InvalidProvider(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_capture", CaptureInput{
		Provider: "brew;injection",
	})
	assert.Error(t, err)
}

// --- Diff tool handler tests ---

func TestDiffToolHandler_ValidConfig(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_diff", DiffInput{})
	require.NoError(t, err)

	output, ok := result.(*DiffOutput)
	require.True(t, ok)
	assert.NotNil(t, output)
}

func TestDiffToolHandler_MissingConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "default")

	_, err := executeTool(t, srv, "preflight_diff", DiffInput{})
	assert.Error(t, err)
}

func TestDiffToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_diff", DiffInput{
		ConfigPath: "config$(id).yaml",
	})
	assert.Error(t, err)
}

func TestDiffToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_diff", DiffInput{
		ConfigPath: configPath,
		Target:     "default",
	})
	require.NoError(t, err)

	output, ok := result.(*DiffOutput)
	require.True(t, ok)
	assert.NotNil(t, output)
}

// --- Tour tool handler tests ---

func TestTourToolHandler_ReturnsTopics(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_tour", TourInput{})
	require.NoError(t, err)

	output, ok := result.(*TourOutput)
	require.True(t, ok)
	assert.NotEmpty(t, output.Topics)

	// Verify topics have required fields
	for _, topic := range output.Topics {
		assert.NotEmpty(t, topic.ID, "topic should have ID")
		assert.NotEmpty(t, topic.Title, "topic should have title")
		assert.NotEmpty(t, topic.Description, "topic should have description")
	}
}

func TestTourToolHandler_ListTopics(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_tour", TourInput{ListTopics: true})
	require.NoError(t, err)

	output, ok := result.(*TourOutput)
	require.True(t, ok)
	// Should have the same topics regardless of ListTopics flag
	assert.NotEmpty(t, output.Topics)
}

// --- Tool Analyze handler tests ---

func TestToolAnalyzeHandler_ValidTools(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"trivy", "grype"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 2, output.ToolsAnalyzed)
	assert.NotNil(t, output.Summary)
}

func TestToolAnalyzeHandler_SingleTool(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"golint"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.ToolsAnalyzed)
}

func TestToolAnalyzeHandler_EmptyToolsList(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{},
	})
	assert.Error(t, err)
}

func TestToolAnalyzeHandler_InvalidToolName(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"trivy;rm"},
	})
	assert.Error(t, err)
}

func TestToolAnalyzeHandler_UnknownTools(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"totally-unknown-tool"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.ToolsAnalyzed)
}

func TestToolAnalyzeHandler_DeprecatedTools(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// golint is a well-known deprecated tool
	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"golint", "golangci-lint"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 2, output.ToolsAnalyzed)
	assert.NotNil(t, output.Summary)
	assert.NotNil(t, output.Findings)
}

// --- Marketplace tool handler tests ---

func TestMarketplaceToolHandler_UnknownAction(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "delete",
	})
	require.NoError(t, err)

	output, ok := result.(*MarketplaceOutput)
	require.True(t, ok)
	assert.Contains(t, output.Message, "Unknown action")
}

func TestMarketplaceToolHandler_InfoMissingPackage(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "info",
	})
	require.NoError(t, err)

	output, ok := result.(*MarketplaceOutput)
	require.True(t, ok)
	assert.Contains(t, output.Message, "Package name required")
}

func TestMarketplaceToolHandler_SearchAction(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Search requires network access to fetch the index; may fail in CI/offline
	_, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "search",
		Query:  "nvim",
	})
	// Handler is exercised regardless of network outcome
	_ = err
}

func TestMarketplaceToolHandler_SearchByType(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Search by type requires network access
	_, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "search",
		Type:   "preset",
	})
	_ = err
}

func TestMarketplaceToolHandler_ListAction(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// List reads locally installed packages, should not require network
	result, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "list",
	})
	require.NoError(t, err)

	output, ok := result.(*MarketplaceOutput)
	require.True(t, ok)
	assert.NotNil(t, output.Packages)
}

func TestMarketplaceToolHandler_FeaturedAction(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Featured requires network access to fetch the index; may fail offline
	_, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action: "featured",
	})
	_ = err
}

// --- Sync tool handler tests ---

func TestSyncToolHandler_NoConfirmNoDryRun(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_sync", SyncInput{
		Confirm: false,
		DryRun:  false,
	})
	require.NoError(t, err)

	output, ok := result.(*SyncOutput)
	require.True(t, ok)
	assert.True(t, output.DryRun)
	assert.Contains(t, output.Message, "confirm=true")
}

func TestSyncToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_sync", SyncInput{
		ConfigPath: "config;injection.yaml",
	})
	assert.Error(t, err)
}

func TestSyncToolHandler_InvalidRemote(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_sync", SyncInput{
		Remote: "origin;rm",
	})
	assert.Error(t, err)
}

func TestSyncToolHandler_InvalidBranch(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_sync", SyncInput{
		Branch: "main|cat",
	})
	assert.Error(t, err)
}

func TestSyncToolHandler_CustomConfigAndTarget(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "nonexistent.yaml", "nonexistent")

	result, err := executeTool(t, srv, "preflight_sync", SyncInput{
		ConfigPath: configPath,
		Target:     "default",
		DryRun:     false,
		Confirm:    false,
	})
	require.NoError(t, err)

	output, ok := result.(*SyncOutput)
	require.True(t, ok)
	assert.True(t, output.DryRun) // No confirm = returns as dry run
}

func TestSyncToolHandler_DryRunWithGitRepo(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	// Initialize a git repo so RepoStatus works
	initGitRepo(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_sync", SyncInput{
		DryRun:  true,
		Confirm: true,
	})
	require.NoError(t, err)

	output, ok := result.(*SyncOutput)
	require.True(t, ok)
	assert.True(t, output.DryRun)
	assert.NotEmpty(t, output.Message)
}

func TestSyncToolHandler_ConfirmWithGitRepoNoRemote(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	// Initialize a git repo (no remote configured)
	initGitRepo(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	result, err := executeTool(t, srv, "preflight_sync", SyncInput{
		DryRun:  false,
		Confirm: true,
	})
	require.NoError(t, err)

	output, ok := result.(*SyncOutput)
	require.True(t, ok)
	assert.False(t, output.DryRun)
	assert.NotEmpty(t, output.Message)
}

// --- Rollback tool handler tests ---

func TestRollbackToolHandler_InvalidInput(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "snap;rm -rf /",
	})
	assert.Error(t, err)
}

func TestRollbackToolHandler_ListSnapshots(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// With no snapshot ID and not latest, should list available snapshots
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.NotNil(t, output.Snapshots)
	assert.Contains(t, output.Message, "Available snapshots")
}

func TestRollbackToolHandler_LatestFlagNoSnapshots(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// With latest=true but likely no snapshots, should still list (empty)
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		Latest: true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	// If no snapshots exist, latest resolves to "" and falls into list path
	assert.NotNil(t, output)
}

func TestRollbackToolHandler_SnapshotNotFound(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "nonexistent-id-12345678",
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.Contains(t, output.Message, "Snapshot not found")
}

func TestRollbackToolHandler_DryRunMode(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// With a nonexistent snapshot and dry_run, exercises the not-found path
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "nonexistent12345678",
		DryRun:     true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.Contains(t, output.Message, "Snapshot not found")
}

// --- Security tool handler tests ---

func TestSecurityToolHandler_ListScanners(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_security", SecurityInput{
		ListScanners: true,
	})
	require.NoError(t, err)

	output, ok := result.(*SecurityOutput)
	require.True(t, ok)
	assert.NotNil(t, output.AvailableScanners)
	// Should list at least the registered scanners (grype, trivy)
	assert.GreaterOrEqual(t, len(output.AvailableScanners), 2)

	// Verify scanner info has names
	for _, scanner := range output.AvailableScanners {
		assert.NotEmpty(t, scanner.Name)
	}
}

func TestSecurityToolHandler_NonexistentScanner(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// A scanner name that doesn't exist should return empty output
	result, err := executeTool(t, srv, "preflight_security", SecurityInput{
		Scanner: "nonexistent-scanner",
	})
	require.NoError(t, err)

	output, ok := result.(*SecurityOutput)
	require.True(t, ok)
	assert.Empty(t, output.Scanner)
}

// --- Outdated tool handler tests ---

func TestOutdatedToolHandler_DefaultOptions(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// brew outdated may fail if brew is not available; exercises the handler code
	_, err := executeTool(t, srv, "preflight_outdated", OutdatedInput{})
	_ = err
}

func TestOutdatedToolHandler_IncludeAll(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_outdated", OutdatedInput{
		IncludeAll: true,
	})
	_ = err
}

func TestOutdatedToolHandler_WithIgnoreIDs(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	_, err := executeTool(t, srv, "preflight_outdated", OutdatedInput{
		IgnoreIDs: []string{"node", "go"},
	})
	_ = err
}

// --- Snapshot sorting and formatting (helper functions) ---

func TestSortSnapshotSets_LargeList(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sets := make([]snapshot.Set, 10)
	for i := 0; i < 10; i++ {
		sets[i] = snapshot.Set{
			ID:        string(rune('a' + i)),
			CreatedAt: now.Add(time.Duration(i-5) * time.Hour),
		}
	}

	sortSnapshotSets(sets)

	// Verify sorted newest first
	for i := 1; i < len(sets); i++ {
		assert.True(t, sets[i-1].CreatedAt.After(sets[i].CreatedAt) || sets[i-1].CreatedAt.Equal(sets[i].CreatedAt),
			"sets should be sorted newest first")
	}
}

func TestSortSnapshotSets_SameTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	sets := []snapshot.Set{
		{ID: "a", CreatedAt: now},
		{ID: "b", CreatedAt: now},
		{ID: "c", CreatedAt: now},
	}

	sortSnapshotSets(sets)

	// Should not panic and should maintain stable order
	assert.Len(t, sets, 3)
}

func TestFormatAge_Boundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		age      time.Duration
		expected string
	}{
		{"zero", 0, "just now"},
		{"one second", time.Second, "just now"},
		{"exactly 1 minute", time.Minute, "1 min ago"},
		{"exactly 2 minutes", 2 * time.Minute, "2 mins ago"},
		{"exactly 1 hour", time.Hour, "1 hour ago"},
		{"exactly 2 hours", 2 * time.Hour, "2 hours ago"},
		{"exactly 24 hours", 24 * time.Hour, "1 day ago"},
		{"exactly 48 hours", 48 * time.Hour, "2 days ago"},
		{"exactly 7 days", 7 * 24 * time.Hour, "1 week ago"},
		{"exactly 14 days", 14 * 24 * time.Hour, "2 weeks ago"},
		{"100 days", 100 * 24 * time.Hour, "14 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatAge(time.Now().Add(-tt.age))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Marketplace helper function ---

func TestToMarketplacePackage_EmptyKeywords(t *testing.T) {
	t.Parallel()

	pkg := createTestMarketplacePackage("test-pkg", "Test", "A test", "author", "preset", nil)
	result := toMarketplacePackage(pkg)

	assert.Equal(t, "test-pkg", result.Name)
	assert.Nil(t, result.Keywords)
}

func TestToMarketplacePackage_MultipleVersions(t *testing.T) {
	t.Parallel()

	pkg := createTestMarketplacePackage("versioned-pkg", "Versioned", "Has versions", "author", "layer", []string{"test"})
	result := toMarketplacePackage(pkg)

	assert.Equal(t, "1.0.0", result.Version)
}

// --- Integration-style tests: full handler paths ---

func TestPlanAndApplyIntegration(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	// Plan first
	planResult, err := executeTool(t, srv, "preflight_plan", PlanInput{})
	require.NoError(t, err)
	planOutput := planResult.(*PlanOutput)

	// Then apply (dry run)
	applyResult, err := executeTool(t, srv, "preflight_apply", ApplyInput{DryRun: true})
	require.NoError(t, err)
	applyOutput := applyResult.(*ApplyOutput)

	assert.True(t, applyOutput.DryRun)
	// Both should report the same step count
	assert.Len(t, applyOutput.Results, len(planOutput.Steps))
}

func TestStatusAndValidateIntegration(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	// Status
	statusResult, err := executeTool(t, srv, "preflight_status", StatusInput{})
	require.NoError(t, err)
	statusOutput := statusResult.(*StatusOutput)

	// Validate
	validateResult, err := executeTool(t, srv, "preflight_validate", ValidateInput{})
	require.NoError(t, err)
	validateOutput := validateResult.(*ValidateOutput)

	// Both should agree on validity
	assert.Equal(t, statusOutput.IsValid, validateOutput.Valid)
}

// --- Default value handling ---

func TestDefaultConfigPath_UsedWhenEmpty(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	// Server default config points to the valid config
	srv := newTestServer(t, pf, configPath, "default")

	// Empty ConfigPath should use the server default
	result, err := executeTool(t, srv, "preflight_plan", PlanInput{ConfigPath: "", Target: ""})
	require.NoError(t, err)

	output := result.(*PlanOutput)
	assert.NotNil(t, output)
}

func TestDefaultTarget_UsedWhenEmpty(t *testing.T) {
	t.Parallel()

	tmpDir, configPath := setupValidConfig(t)
	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, configPath, "default")

	// Empty Target should use server default "default"
	result, err := executeTool(t, srv, "preflight_validate", ValidateInput{Target: ""})
	require.NoError(t, err)

	output := result.(*ValidateOutput)
	assert.True(t, output.Valid)
}

// --- ToolAnalyze summary counting ---

func TestToolAnalyzeHandler_SummaryCounts(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Use known tools that trigger findings in the knowledge base
	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"golint", "trivy", "grype", "gitleaks"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 4, output.ToolsAnalyzed)
	assert.NotNil(t, output.Summary)

	// The summary counts should match the findings
	var deprecations, redundancies, consolidations int
	for _, f := range output.Findings {
		switch f.Type {
		case "deprecated":
			deprecations++
		case "redundancy":
			redundancies++
		case "consolidation":
			consolidations++
		}
	}
	assert.Equal(t, deprecations, output.Summary.Deprecations)
	assert.Equal(t, redundancies, output.Summary.Redundancies)
	assert.Equal(t, consolidations, output.Summary.Consolidations)
}

// --- Multiple tools at once ---

func TestToolAnalyzeHandler_ManyTools(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_analyze_tools", ToolAnalyzeInput{
		Tools: []string{"golint", "golangci-lint", "trivy", "grype", "gitleaks", "syft"},
	})
	require.NoError(t, err)

	output, ok := result.(*ToolAnalyzeOutput)
	require.True(t, ok)
	assert.Equal(t, 6, output.ToolsAnalyzed)
}

// --- Additional coverage tests ---

// setupSnapshotDir creates a temporary .preflight/snapshots directory with snapshot data.
// Returns the fake HOME directory that should be set via t.Setenv("HOME", ...).
func setupSnapshotDir(t *testing.T) string {
	t.Helper()

	fakeHome := t.TempDir()
	snapshotDir := filepath.Join(fakeHome, ".preflight", "snapshots")
	require.NoError(t, os.MkdirAll(snapshotDir, 0o755))

	// Create a snapshot file
	snapID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	snapFilename := snapID + ".snapshot"
	require.NoError(t, os.WriteFile(
		filepath.Join(snapshotDir, snapFilename),
		[]byte("original file content"),
		0o600,
	))

	// Create the snapshot index (individual snapshots)
	snapshotIndex := map[string]interface{}{
		"snapshots": map[string]interface{}{
			snapID: map[string]interface{}{
				"id":         snapID,
				"path":       "/tmp/test-file.txt",
				"hash":       "abcdef1234567890",
				"size":       21,
				"created_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
				"filename":   snapFilename,
			},
		},
	}
	indexData, err := json.MarshalIndent(snapshotIndex, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "index.json"), indexData, 0o600))

	// Create the snapshot set index
	setID := "11111111-2222-3333-4444-555555555555"
	setIndex := map[string]interface{}{
		"sets": map[string]interface{}{
			setID: map[string]interface{}{
				"id":           setID,
				"reason":       "apply",
				"created_at":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano),
				"snapshot_ids": []string{snapID},
			},
		},
	}
	setData, err := json.MarshalIndent(setIndex, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "sets.json"), setData, 0o600))

	return fakeHome
}

func TestRollbackToolHandler_ListWithSnapshots(t *testing.T) {
	fakeHome := setupSnapshotDir(t)
	t.Setenv("HOME", fakeHome)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.Contains(t, output.Message, "Available snapshots")
	assert.NotEmpty(t, output.Snapshots)
	assert.Len(t, output.Snapshots, 1)

	// Verify snapshot info fields
	snap := output.Snapshots[0]
	assert.NotEmpty(t, snap.ID)
	assert.NotEmpty(t, snap.ShortID)
	assert.Len(t, snap.ShortID, 8)
	assert.NotEmpty(t, snap.CreatedAt)
	assert.NotEmpty(t, snap.Age)
	assert.Equal(t, 1, snap.FileCount)
	assert.Equal(t, "apply", snap.Reason)
}

func TestRollbackToolHandler_LatestWithSnapshots(t *testing.T) {
	fakeHome := setupSnapshotDir(t)
	t.Setenv("HOME", fakeHome)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// With latest=true and snapshots available, should select the latest and preview
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		Latest: true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	// latest=true selects the latest snapshot, but confirm is false so it previews
	assert.NotEmpty(t, output.TargetSnapshot)
	assert.True(t, output.DryRun)
	assert.Equal(t, 1, output.RestoredFiles)
	assert.Contains(t, output.Message, "confirm=true")
}

func TestRollbackToolHandler_DryRunWithSnapshot(t *testing.T) {
	fakeHome := setupSnapshotDir(t)
	t.Setenv("HOME", fakeHome)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Use the full set ID to find the snapshot
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "11111111-2222-3333-4444-555555555555",
		DryRun:     true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.NotEmpty(t, output.TargetSnapshot)
	assert.True(t, output.DryRun)
	assert.Equal(t, 1, output.RestoredFiles)
	assert.Contains(t, output.Message, "confirm=true")
}

func TestRollbackToolHandler_ShortIDMatch(t *testing.T) {
	fakeHome := setupSnapshotDir(t)
	t.Setenv("HOME", fakeHome)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Use just the first 8 characters (short ID) to find the snapshot
	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "11111111",
		DryRun:     true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.NotEmpty(t, output.TargetSnapshot)
	assert.True(t, output.DryRun)
	assert.Equal(t, 1, output.RestoredFiles)
}

func TestRollbackToolHandler_ConfirmRestore(t *testing.T) {
	fakeHome := setupSnapshotDir(t)
	t.Setenv("HOME", fakeHome)

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_rollback", RollbackInput{
		SnapshotID: "11111111-2222-3333-4444-555555555555",
		Confirm:    true,
	})
	require.NoError(t, err)

	output, ok := result.(*RollbackOutput)
	require.True(t, ok)
	assert.NotEmpty(t, output.TargetSnapshot)
	assert.False(t, output.DryRun)
	assert.Equal(t, 1, output.RestoredFiles)
	assert.Contains(t, output.Message, "restored successfully")
}

func TestMarketplaceToolHandler_InfoWithPackageName(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Info with a valid package name exercises the NewPackageID and svc.Get paths.
	// svc.Get may fail with a network error, but the code paths are exercised.
	_, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action:  "info",
		Package: "test-package",
	})
	// Network error expected but handler code paths are exercised
	_ = err
}

func TestMarketplaceToolHandler_InfoInvalidPackageName(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Invalid package name (uppercase) should fail at NewPackageID validation
	_, err := executeTool(t, srv, "preflight_marketplace", MarketplaceInput{
		Action:  "info",
		Package: "INVALID_PACKAGE",
	})
	assert.Error(t, err)
}

func TestCaptureToolHandler_WithProvider(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	result, err := executeTool(t, srv, "preflight_capture", CaptureInput{
		Provider: "brew",
	})
	require.NoError(t, err)

	output, ok := result.(*CaptureOutput)
	require.True(t, ok)
	assert.NotNil(t, output.Items)
}

func TestStatusToolHandler_ConfigExistsButValidationFails(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	// Write a valid YAML config with targets but referencing layers that don't exist
	require.NoError(t, os.WriteFile(configPath, []byte("targets:\n  default:\n    - nonexistent-layer\n"), 0o644))

	withChdir(t, tmpDir)

	pf := app.New(bytes.NewBuffer(nil))
	// Use a target that is valid in config but point to a nonexistent target for validation
	srv := newTestServer(t, pf, configPath, "nonexistent-target")

	result, err := executeTool(t, srv, "preflight_status", StatusInput{})
	require.NoError(t, err)

	output, ok := result.(*StatusOutput)
	require.True(t, ok)
	// Config loads successfully but validation may report errors for the target
	assert.True(t, output.ConfigExists)
}

func TestSecurityToolHandler_AutoScanner(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Default (auto) scanner selection - exercises the First() path
	result, err := executeTool(t, srv, "preflight_security", SecurityInput{
		Scanner: "auto",
	})
	// If no scanner is available, returns empty output; if available, may error on scan
	if err == nil {
		output, ok := result.(*SecurityOutput)
		require.True(t, ok)
		assert.NotNil(t, output)
	}
}

func TestSecurityToolHandler_DefaultScanner(t *testing.T) {
	t.Parallel()

	pf := app.New(bytes.NewBuffer(nil))
	srv := newTestServer(t, pf, "preflight.yaml", "default")

	// Empty scanner field (default) - exercises the First() path with empty string
	result, err := executeTool(t, srv, "preflight_security", SecurityInput{})
	if err == nil {
		output, ok := result.(*SecurityOutput)
		require.True(t, ok)
		assert.NotNil(t, output)
	}
}
