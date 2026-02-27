package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/advisor"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 1. runEnvList, runEnvGet, runEnvExport, runEnvDiff (env.go)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestRunEnvList_MissingConfig(t *testing.T) {
	orig := envConfigPath
	defer func() { envConfigPath = orig }()
	envConfigPath = "/nonexistent/config.yaml"

	err := runEnvList(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

//nolint:tparallel // modifies global flags
func TestRunEnvGet_MissingConfig(t *testing.T) {
	orig := envConfigPath
	defer func() { envConfigPath = orig }()
	envConfigPath = "/nonexistent/config.yaml"

	err := runEnvGet(nil, []string{"MY_VAR"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

//nolint:tparallel // modifies global flags
func TestRunEnvExport_MissingConfig(t *testing.T) {
	orig := envConfigPath
	defer func() { envConfigPath = orig }()
	envConfigPath = "/nonexistent/config.yaml"

	err := runEnvExport(nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

//nolint:tparallel // modifies global flags
func TestRunEnvDiff_MissingConfig(t *testing.T) {
	orig := envConfigPath
	defer func() { envConfigPath = orig }()
	envConfigPath = "/nonexistent/config.yaml"

	err := runEnvDiff(nil, []string{"target1", "target2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load")
}

// ---------------------------------------------------------------------------
// 2. runAuditShow, runAuditSummary, runAuditSecurity (audit.go)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags and HOME
func TestRunAuditShow_NoMatchingEvents(t *testing.T) {
	// Redirect HOME so getAuditService creates a fresh, empty audit log
	t.Setenv("HOME", t.TempDir())

	output := captureStdout(t, func() {
		err := runAuditShow(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No audit events found")
}

//nolint:tparallel // modifies global flags and HOME
func TestRunAuditSummary_EmptyLog(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origJSON := auditOutputJSON
	defer func() { auditOutputJSON = origJSON }()
	auditOutputJSON = false

	output := captureStdout(t, func() {
		err := runAuditSummary(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Audit Log Summary")
}

//nolint:tparallel // modifies global flags and HOME
func TestRunAuditSummary_JSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origJSON := auditOutputJSON
	defer func() { auditOutputJSON = origJSON }()
	auditOutputJSON = true

	output := captureStdout(t, func() {
		err := runAuditSummary(nil, nil)
		assert.NoError(t, err)
	})
	// Should be valid JSON containing summary fields
	assert.Contains(t, output, "total_events")
}

//nolint:tparallel // modifies global flags and HOME
func TestRunAuditSecurity_NoEvents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	origJSON := auditOutputJSON
	origDays := auditDays
	defer func() {
		auditOutputJSON = origJSON
		auditDays = origDays
	}()
	auditOutputJSON = false
	auditDays = 1

	output := captureStdout(t, func() {
		err := runAuditSecurity(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No security events")
}

// ---------------------------------------------------------------------------
// 3. Git functions in sync.go
// ---------------------------------------------------------------------------

func TestGitFetch_InvalidRemote(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	err := gitFetch(tmpDir, "nonexistent-remote")
	assert.Error(t, err)
}

func TestGetCommitDiff_InvalidRemote(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "init")
	require.NoError(t, cmd.Run())

	_, _, err := getCommitDiff(tmpDir, "nonexistent-remote", "main")
	assert.Error(t, err)
}

func TestGitPull_InvalidRemote(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	err := gitPull(tmpDir, "nonexistent-remote", "main")
	assert.Error(t, err)
}

func TestGitPush_InvalidRemote(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init", tmpDir)
	require.NoError(t, cmd.Run())

	err := gitPush(tmpDir, "nonexistent-remote", "main")
	assert.Error(t, err)
}

func TestCheckLockfileConflicts_NoLockfile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	ctx := context.Background()

	_, err := checkLockfileConflicts(ctx, tmpDir, filepath.Join(tmpDir, "preflight.lock"), "origin", "main")
	assert.Error(t, err) // lockfile doesn't exist
}

// ---------------------------------------------------------------------------
// 4. getProviderByName (init.go)
// ---------------------------------------------------------------------------

func TestGetProviderByName_UnknownProvider(t *testing.T) {
	t.Parallel()
	result := getProviderByName("unknown-provider")
	assert.Nil(t, result)
}

func TestGetProviderByName_AnthropicNoKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	result := getProviderByName("anthropic")
	assert.Nil(t, result)
}

func TestGetProviderByName_OpenAINoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	result := getProviderByName("openai")
	assert.Nil(t, result)
}

func TestGetProviderByName_GeminiNoKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	result := getProviderByName("gemini")
	assert.Nil(t, result)
}

func TestGetProviderByName_AnthropicWithKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	result := getProviderByName("anthropic")
	assert.NotNil(t, result)
}

func TestGetProviderByName_OpenAIWithKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-key")
	result := getProviderByName("openai")
	assert.NotNil(t, result)
}

func TestGetProviderByName_GeminiWithKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	result := getProviderByName("gemini")
	assert.NotNil(t, result)
}

func TestGetProviderByName_GeminiWithGoogleKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "test-key")
	result := getProviderByName("gemini")
	assert.NotNil(t, result)
}

// ---------------------------------------------------------------------------
// 5. runMarketplaceList (marketplace.go)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestRunMarketplaceList_NoPackagesInstalled(t *testing.T) {
	output := captureStdout(t, func() {
		err := runMarketplaceList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No packages installed")
}

// ---------------------------------------------------------------------------
// 6. enhanceWithAI (analyze.go)
// ---------------------------------------------------------------------------

// mockAIProvider implements advisor.AIProvider for testing.
type mockAIProvider struct {
	name      string
	available bool
	err       error
	response  advisor.Response
}

func (m *mockAIProvider) Name() string { return m.name }

func (m *mockAIProvider) Complete(_ context.Context, _ advisor.Prompt) (advisor.Response, error) {
	if m.err != nil {
		return advisor.Response{}, m.err
	}
	return m.response, nil
}

func (m *mockAIProvider) Available() bool { return m.available }

func TestEnhanceWithAI_ProviderError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result := &security.ToolAnalysisResult{
		Findings:      []security.ToolFinding{},
		ToolsAnalyzed: 1,
	}
	provider := &mockAIProvider{
		name:      "test",
		available: true,
		err:       fmt.Errorf("api error"),
	}

	enhanced := enhanceWithAI(ctx, result, []string{"git"}, provider)
	// Should return the original result unchanged when AI fails
	assert.Equal(t, result, enhanced)
	assert.Empty(t, enhanced.Findings)
}

func TestEnhanceWithAI_ValidResponse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result := &security.ToolAnalysisResult{
		Findings:      []security.ToolFinding{},
		ToolsAnalyzed: 2,
	}

	jsonResp := `{
		"insights": [
			{
				"type": "recommendation",
				"severity": "info",
				"tools": ["git"],
				"message": "Consider enabling signed commits",
				"suggestion": "Use GPG signing"
			}
		]
	}`
	provider := &mockAIProvider{
		name:      "test",
		available: true,
		response:  advisor.NewResponse(jsonResp, 100, "test-model"),
	}

	enhanced := enhanceWithAI(ctx, result, []string{"git", "vim"}, provider)
	assert.NotEmpty(t, enhanced.Findings)
}

func TestEnhanceWithAI_EmptyAIResponse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	result := &security.ToolAnalysisResult{
		Findings:      []security.ToolFinding{},
		ToolsAnalyzed: 1,
	}
	provider := &mockAIProvider{
		name:      "test",
		available: true,
		response:  advisor.NewResponse("no json here", 50, "test-model"),
	}

	enhanced := enhanceWithAI(ctx, result, []string{"git"}, provider)
	// No parseable JSON means no extra findings
	assert.Empty(t, enhanced.Findings)
}

// ---------------------------------------------------------------------------
// 7. captureDotfiles (capture.go)
// ---------------------------------------------------------------------------

//nolint:tparallel // modifies global flags
func TestCaptureDotfiles_DefaultOutput(t *testing.T) {
	origOutput := captureOutput
	origTarget := captureTarget
	defer func() {
		captureOutput = origOutput
		captureTarget = origTarget
	}()
	captureOutput = t.TempDir()
	captureTarget = "default"

	// captureDotfiles scans the real home directory; it may find files or not,
	// but it should not error out.
	result, err := captureDotfiles()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

//nolint:tparallel // modifies global flags
func TestCaptureDotfiles_WithTarget(t *testing.T) {
	origOutput := captureOutput
	origTarget := captureTarget
	defer func() {
		captureOutput = origOutput
		captureTarget = origTarget
	}()
	captureOutput = t.TempDir()
	captureTarget = "work"

	result, err := captureDotfiles()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------------------------------------------------------------------------
// 8. runRepoClone (repo.go)
// ---------------------------------------------------------------------------

func TestRunRepoClone_InvalidURL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	// Use a clearly invalid URL that will fail to clone
	err := runRepoClone(repoCloneCmd, []string{"not-a-valid-url", filepath.Join(tmpDir, "dest")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clone failed")
}
