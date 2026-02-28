package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// catalog.go: deriveCatalogName
// ===========================================================================

func TestBatch3_DeriveCatalogName_URL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "my-catalog", deriveCatalogName("https://example.com/my-catalog"))
}

func TestBatch3_DeriveCatalogName_TrailingSlash(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "my-catalog", deriveCatalogName("https://example.com/my-catalog/"))
}

func TestBatch3_DeriveCatalogName_Path(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "presets", deriveCatalogName("/home/user/presets"))
}

func TestBatch3_DeriveCatalogName_NoSlashes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "my-catalog", deriveCatalogName("my-catalog"))
}

func TestBatch3_DeriveCatalogName_Empty(t *testing.T) {
	t.Parallel()
	result := deriveCatalogName("")
	assert.Contains(t, result, "catalog-")
}

func TestBatch3_DeriveCatalogName_JustSlash(t *testing.T) {
	t.Parallel()
	// "/" after removing trailing slash becomes empty, which returns the empty string
	result := deriveCatalogName("/")
	// The function returns empty string since after removing trailing slash,
	// it finds no more slashes and returns the location as-is (empty)
	assert.Equal(t, "", result)
}

// ===========================================================================
// catalog.go: filterBySeverity
// ===========================================================================

func TestBatch3_FilterBySeverity_Matches(t *testing.T) {
	t.Parallel()
	findings := []catalog.AuditFinding{
		{Severity: catalog.AuditSeverityCritical, Message: "a"},
		{Severity: catalog.AuditSeverityHigh, Message: "b"},
		{Severity: catalog.AuditSeverityCritical, Message: "c"},
	}
	result := filterBySeverity(findings, catalog.AuditSeverityCritical)
	assert.Len(t, result, 2)
}

func TestBatch3_FilterBySeverity_NoMatch(t *testing.T) {
	t.Parallel()
	findings := []catalog.AuditFinding{
		{Severity: catalog.AuditSeverityHigh, Message: "b"},
	}
	result := filterBySeverity(findings, catalog.AuditSeverityLow)
	assert.Empty(t, result)
}

func TestBatch3_FilterBySeverity_Nil(t *testing.T) {
	t.Parallel()
	result := filterBySeverity(nil, catalog.AuditSeverityCritical)
	assert.Empty(t, result)
}

// ===========================================================================
// catalog.go: verifyCatalogSignatures
// ===========================================================================

func TestBatch3_VerifyCatalogSignatures_NoSignature(t *testing.T) {
	t.Parallel()
	result := verifyCatalogSignatures(nil, nil)
	assert.False(t, result.hasSignature)
	assert.False(t, result.verified)
}

// ===========================================================================
// catalog.go: getRegistry
// ===========================================================================

func TestBatch3_GetRegistry_ReturnsRegistry(t *testing.T) {
	reg, err := getRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	// Should have at least builtin catalog
	catalogs := reg.List()
	assert.NotEmpty(t, catalogs)
}

// ===========================================================================
// catalog.go: getCatalogAuditService
// ===========================================================================

func TestBatch3_GetCatalogAuditService_ReturnsService(t *testing.T) {
	svc := getCatalogAuditService()
	assert.NotNil(t, svc)
	_ = svc.Close()
}

// ===========================================================================
// catalog.go: runCatalogList
// ===========================================================================

func TestBatch3_RunCatalogList_HasBuiltin(t *testing.T) {
	output := captureStdout(t, func() {
		err := runCatalogList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "builtin")
}

// ===========================================================================
// catalog.go: runCatalogVerify - signatures flag error
// ===========================================================================

func TestBatch3_RunCatalogVerify_SignaturesNotSupported(t *testing.T) {
	old := catalogVerifySigs
	defer func() { catalogVerifySigs = old }()
	catalogVerifySigs = true

	err := runCatalogVerify(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// ===========================================================================
// catalog.go: runCatalogVerify - specific catalog
// ===========================================================================

func TestBatch3_RunCatalogVerify_BuiltinByName(t *testing.T) {
	old := catalogVerifySigs
	defer func() { catalogVerifySigs = old }()
	catalogVerifySigs = false

	output := captureStdout(t, func() {
		err := runCatalogVerify(nil, []string{"builtin"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "builtin")
	assert.Contains(t, output, "Verified")
}

func TestBatch3_RunCatalogVerify_NotFound(t *testing.T) {
	old := catalogVerifySigs
	defer func() { catalogVerifySigs = old }()
	catalogVerifySigs = false

	err := runCatalogVerify(nil, []string{"nonexistent-catalog-xyz"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ===========================================================================
// catalog.go: runCatalogAudit
// ===========================================================================

func TestBatch3_RunCatalogAudit_BuiltinPasses(t *testing.T) {
	output := captureStdout(t, func() {
		err := runCatalogAudit(nil, []string{"builtin"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Audit")
}

func TestBatch3_RunCatalogAudit_NotFound(t *testing.T) {
	err := runCatalogAudit(nil, []string{"nonexistent-xyz"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ===========================================================================
// catalog.go: runCatalogRemove
// ===========================================================================

func TestBatch3_RunCatalogRemove_NotFound(t *testing.T) {
	err := runCatalogRemove(nil, []string{"nonexistent-xyz"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBatch3_RunCatalogRemove_BuiltinCannotRemove(t *testing.T) {
	old := catalogForce
	defer func() { catalogForce = old }()
	catalogForce = true

	err := runCatalogRemove(nil, []string{"builtin"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove builtin")
}

// ===========================================================================
// marketplace.go: formatInstallAge
// ===========================================================================

func TestBatch3_FormatInstallAge_JustNow(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now())
	assert.Equal(t, "just now", result)
}

func TestBatch3_FormatInstallAge_MinutesAgo(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now().Add(-5 * time.Minute))
	assert.Contains(t, result, "m ago")
}

func TestBatch3_FormatInstallAge_HoursAgo(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now().Add(-3 * time.Hour))
	assert.Contains(t, result, "h ago")
}

func TestBatch3_FormatInstallAge_DaysAgo(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now().Add(-3 * 24 * time.Hour))
	assert.Contains(t, result, "d ago")
}

func TestBatch3_FormatInstallAge_WeeksAgo(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now().Add(-14 * 24 * time.Hour))
	assert.Contains(t, result, "w ago")
}

func TestBatch3_FormatInstallAge_OlderThanMonth(t *testing.T) {
	t.Parallel()
	result := formatInstallAge(time.Now().Add(-60 * 24 * time.Hour))
	// Should format as date
	assert.Contains(t, result, "-")
}

// ===========================================================================
// marketplace.go: formatReason
// ===========================================================================

func TestBatch3_FormatReason_AllTypes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "popular", formatReason(marketplace.ReasonPopular))
	assert.Equal(t, "trending", formatReason(marketplace.ReasonTrending))
	assert.Equal(t, "similar", formatReason(marketplace.ReasonSimilarKeywords))
	assert.Equal(t, "same type", formatReason(marketplace.ReasonSameType))
	assert.Equal(t, "same author", formatReason(marketplace.ReasonSameAuthor))
}

// ===========================================================================
// plugin.go: outputValidationResult
// ===========================================================================

func TestBatch3_OutputValidationResult_Valid(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	result := ValidationResult{
		Valid:   true,
		Plugin:  "my-plugin",
		Version: "1.0.0",
		Path:    "/tmp/test",
	}
	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "validated")
	assert.Contains(t, output, "my-plugin")
}

func TestBatch3_OutputValidationResult_Invalid(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	result := ValidationResult{
		Valid:  false,
		Errors: []string{"missing name", "invalid version"},
		Path:   "/tmp/test",
	}
	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.Error(t, err)
	})
	assert.Contains(t, output, "Validation failed")
	assert.Contains(t, output, "missing name")
}

func TestBatch3_OutputValidationResult_WithWarnings(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	result := ValidationResult{
		Valid:    true,
		Plugin:   "my-plugin",
		Version:  "1.0.0",
		Warnings: []string{"missing description"},
		Path:     "/tmp/test",
	}
	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Warnings")
	assert.Contains(t, output, "missing description")
}

func TestBatch3_OutputValidationResult_JSON(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = true

	result := ValidationResult{
		Valid:   true,
		Plugin:  "my-plugin",
		Version: "1.0.0",
		Path:    "/tmp/test",
	}
	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "\"valid\": true")
	assert.Contains(t, output, "my-plugin")
}

// ===========================================================================
// plugin.go: runPluginValidate - path doesn't exist
// ===========================================================================

func TestBatch3_RunPluginValidate_PathNotExist(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	output := captureStdout(t, func() {
		err := runPluginValidate("/nonexistent/path/xyz")
		assert.Error(t, err)
	})
	assert.Contains(t, output, "does not exist")
}

func TestBatch3_RunPluginValidate_PathNotDir(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("hello"), 0o644))

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpFile)
		assert.Error(t, err)
	})
	assert.Contains(t, output, "must be a directory")
}

func TestBatch3_RunPluginValidate_NoManifest(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = false

	tmpDir := t.TempDir()
	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		assert.Error(t, err)
	})
	assert.Contains(t, output, "Validation failed")
}

func TestBatch3_RunPluginValidate_PathNotExist_JSON(t *testing.T) {
	old := pluginValidateJSON
	defer func() { pluginValidateJSON = old }()
	pluginValidateJSON = true

	output := captureStdout(t, func() {
		err := runPluginValidate("/nonexistent/path/xyz")
		// In JSON mode, outputValidationResult may return nil even for invalid
		// plugins since it writes JSON to stdout
		_ = err
	})
	assert.Contains(t, output, "\"valid\"")
	assert.Contains(t, output, "false")
}

// ===========================================================================
// plugin.go: getPluginAuditService
// ===========================================================================

func TestBatch3_GetPluginAuditService_Returns(t *testing.T) {
	svc := getPluginAuditService()
	assert.NotNil(t, svc)
	_ = svc.Close()
}

// ===========================================================================
// plugin.go: runPluginList - no plugins
// ===========================================================================

func TestBatch3_RunPluginList_NoPlugins(t *testing.T) {
	output := captureStdout(t, func() {
		err := runPluginList()
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No plugins installed")
}

// ===========================================================================
// marketplace.go: newMarketplaceService
// ===========================================================================

func TestBatch3_NewMarketplaceService_Default(t *testing.T) {
	old := mpOfflineMode
	defer func() { mpOfflineMode = old }()
	mpOfflineMode = true

	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}

// ===========================================================================
// cleanup.go: handleRemove - dry run
// ===========================================================================

func TestBatch3_HandleRemove_DryRunJSON(t *testing.T) {
	old := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = old
		cleanupJSON = oldJSON
	}()
	cleanupDryRun = true
	cleanupJSON = true

	output := captureStdout(t, func() {
		err := handleRemove(context.Background(), nil, []string{"pkg1", "pkg2"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "pkg1")
	assert.Contains(t, output, "dry_run")
}

func TestBatch3_HandleRemove_DryRunText(t *testing.T) {
	old := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = old
		cleanupJSON = oldJSON
	}()
	cleanupDryRun = true
	cleanupJSON = false

	output := captureStdout(t, func() {
		err := handleRemove(context.Background(), nil, []string{"pkg1"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "pkg1")
}

// ===========================================================================
// cleanup.go: handleCleanupAll - dry run
// ===========================================================================

func TestBatch3_HandleCleanupAll_DryRunJSON(t *testing.T) {
	old := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = old
		cleanupJSON = oldJSON
	}()
	cleanupDryRun = true
	cleanupJSON = true

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}},
		},
	}
	output := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), nil, result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "go@1.24")
	assert.Contains(t, output, "dry_run")
}

func TestBatch3_HandleCleanupAll_DryRunText(t *testing.T) {
	old := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = old
		cleanupJSON = oldJSON
	}()
	cleanupDryRun = true
	cleanupJSON = false

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}},
		},
	}
	output := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), nil, result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "go@1.24")
}

// ===========================================================================
// discover.go: getPatternIcon - comprehensive
// ===========================================================================

func TestBatch3_GetPatternIcon_Default(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "•", getPatternIcon("unknown_type"))
}

// ===========================================================================
// tour.go: runTour - list mode
// ===========================================================================

func TestBatch3_RunTour_ListMode(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = true

	output := captureStdout(t, func() {
		err := runTour(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Available tour topics")
}

func TestBatch3_RunTour_InvalidTopic(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = false

	err := runTour(nil, []string{"nonexistent_topic_xyz_abc"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic")
}

// ===========================================================================
// init.go: runInitNonInteractive
// ===========================================================================

func TestBatch3_RunInitNonInteractive_NoPreset(t *testing.T) {
	old := initPreset
	defer func() { initPreset = old }()
	initPreset = ""

	err := runInitNonInteractive("/tmp/nonexistent/preflight.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--preset is required")
}

func TestBatch3_RunInitNonInteractive_ValidPreset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")

	oldPreset := initPreset
	oldOutput := initOutputDir
	defer func() {
		initPreset = oldPreset
		initOutputDir = oldOutput
	}()
	initPreset = "balanced"
	initOutputDir = tmpDir

	output := captureStdout(t, func() {
		err := runInitNonInteractive(configPath)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Configuration created")

	// Verify files exist
	_, err := os.Stat(configPath)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, "layers", "base.yaml"))
	assert.NoError(t, err)
}

// ===========================================================================
// init.go: runInit - config already exists
// ===========================================================================

func TestBatch3_RunInit_ConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("test"), 0o644))

	old := initOutputDir
	defer func() { initOutputDir = old }()
	initOutputDir = tmpDir

	output := captureStdout(t, func() {
		err := runInit(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "already exists")
}

// ===========================================================================
// init.go: generateLayerForPreset - all branches
// ===========================================================================

func TestBatch3_GenerateLayerForPreset_AllCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		preset   string
		contains string
	}{
		{"nvim:minimal", "minimal"},
		{"nvim:balanced", "kickstart"},
		{"balanced", "kickstart"},
		{"nvim:maximal", "astronvim"},
		{"maximal", "astronvim"},
		{"shell:minimal", "zsh"},
		{"shell:balanced", "oh-my-zsh"},
		{"git:minimal", "editor: vim"},
		{"brew:minimal", "ripgrep"},
		{"unknown-preset", "Add your configuration"},
	}

	for _, tc := range cases {
		result := generateLayerForPreset(tc.preset)
		assert.Contains(t, result, tc.contains, "preset %s", tc.preset)
	}
}

// ===========================================================================
// init.go: detectAIProvider - no keys set
// ===========================================================================

func TestBatch3_DetectAIProvider_NoKeys(t *testing.T) {
	// Save and clear AI-related env vars
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	oldProvider := aiProvider
	defer func() { aiProvider = oldProvider }()
	aiProvider = ""

	result := detectAIProvider()
	assert.Nil(t, result)
}

// ===========================================================================
// init.go: getProviderByName
// ===========================================================================

func TestBatch3_GetProviderByName_UnknownProvider(t *testing.T) {
	result := getProviderByName("unknown")
	assert.Nil(t, result)
}

func TestBatch3_GetProviderByName_AnthropicNoKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	result := getProviderByName("anthropic")
	assert.Nil(t, result)
}

func TestBatch3_GetProviderByName_GeminiNoKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")

	result := getProviderByName("gemini")
	assert.Nil(t, result)
}

func TestBatch3_GetProviderByName_OpenAINoKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	result := getProviderByName("openai")
	assert.Nil(t, result)
}

// ===========================================================================
// env.go: WriteEnvFile
// ===========================================================================

func TestBatch3_WriteEnvFile_CreatesFile(t *testing.T) {
	vars := []EnvVar{
		{Name: "FOO", Value: "bar"},
		{Name: "SECRET_KEY", Value: "secret://vault/key", Secret: true},
	}

	err := WriteEnvFile(vars)
	assert.NoError(t, err)

	// Verify file exists
	home, _ := os.UserHomeDir()
	envPath := filepath.Join(home, ".preflight", "env.sh")
	data, err := os.ReadFile(envPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "FOO")
}

// ===========================================================================
// env.go: extractEnvVars with secrets
// ===========================================================================

func TestBatch3_ExtractEnvVars_WithSecrets(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"EDITOR":    "nvim",
			"API_KEY":   "secret://vault/key",
			"DEBUG":     true,
			"THRESHOLD": 42,
		},
	}
	vars := extractEnvVars(config)
	assert.Len(t, vars, 4)

	secretCount := 0
	for _, v := range vars {
		if v.Secret {
			secretCount++
		}
	}
	assert.Equal(t, 1, secretCount)
}

func TestBatch3_ExtractEnvVars_NoEnvSection(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"git": map[string]interface{}{"name": "test"},
	}
	vars := extractEnvVars(config)
	assert.Empty(t, vars)
}

// ===========================================================================
// marketplace.go: buildUserContext
// ===========================================================================

func TestBatch3_BuildUserContext_WithKeywords(t *testing.T) {
	old := mpKeywords
	oldType := mpRecommendType
	defer func() {
		mpKeywords = old
		mpRecommendType = oldType
	}()
	mpKeywords = "docker, kubernetes, go"
	mpRecommendType = "preset"

	svc := newMarketplaceService()
	ctx := buildUserContext(svc)
	assert.Len(t, ctx.Keywords, 3)
	assert.Equal(t, []string{"preset"}, ctx.PreferredTypes)
}

func TestBatch3_BuildUserContext_NoKeywords(t *testing.T) {
	old := mpKeywords
	oldType := mpRecommendType
	defer func() {
		mpKeywords = old
		mpRecommendType = oldType
	}()
	mpKeywords = ""
	mpRecommendType = ""

	svc := newMarketplaceService()
	ctx := buildUserContext(svc)
	assert.Empty(t, ctx.Keywords)
	assert.Empty(t, ctx.PreferredTypes)
}

// ===========================================================================
// sync.go helpers
// ===========================================================================

func TestBatch3_FindRepoRoot_InGitRepo(t *testing.T) {
	root, err := findRepoRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, root)
}

func TestBatch3_GetCurrentBranch(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)
	branch, err := getCurrentBranch(root)
	assert.NoError(t, err)
	assert.NotEmpty(t, branch)
}

// ===========================================================================
// compliance.go: outputComplianceJSON error path
// ===========================================================================

// compliance.go: outputComplianceJSON takes a *policy.ComplianceReport
// It calls os.Exit on error, so we can't easily test the error path.
// Skip direct testing of outputComplianceJSON.

// ===========================================================================
// marketplace.go: formatReason additional types
// ===========================================================================

func TestBatch3_FormatReason_MoreTypes(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "complements", formatReason(marketplace.ReasonComplementary))
	assert.Equal(t, "recent", formatReason(marketplace.ReasonRecentlyUpdated))
	assert.Equal(t, "rated", formatReason(marketplace.ReasonHighlyRated))
	assert.Equal(t, "provider", formatReason(marketplace.ReasonProviderMatch))
	assert.Equal(t, "featured", formatReason(marketplace.ReasonFeatured))
	// Unknown reason returns string representation
	assert.NotEmpty(t, formatReason("unknown_reason"))
}

// ===========================================================================
// trust.go: getTrustStore
// ===========================================================================

func TestBatch3_GetTrustStore_Returns(t *testing.T) {
	store, err := getTrustStore()
	// May error if trust.json doesn't exist, but should not panic
	if err == nil {
		assert.NotNil(t, store)
	}
}

// ===========================================================================
// doctor.go: printDoctorQuiet
// ===========================================================================

func TestBatch3_PrintDoctorQuiet_NoIssues(t *testing.T) {
	report := &app.DoctorReport{}
	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "No issues found")
}

func TestBatch3_PrintDoctorQuiet_WithIssues(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityError,
				Message:    "git config wrong",
				Provider:   "git",
				Expected:   "nvim",
				Actual:     "vim",
				FixCommand: "git config core.editor nvim",
			},
			{
				Severity: app.SeverityWarning,
				Message:  "missing plugin",
			},
		},
	}
	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "git config wrong")
	assert.Contains(t, output, "Provider: git")
	assert.Contains(t, output, "Expected: nvim")
	assert.Contains(t, output, "Actual: vim")
	assert.Contains(t, output, "Fix: git config")
	assert.Contains(t, output, "missing plugin")
}

// ===========================================================================
// rollback.go: listSnapshots
// ===========================================================================

func TestBatch3_ListSnapshots_WithSets(t *testing.T) {
	t.Parallel()
	sets := []snapshot.Set{
		{
			ID:        "abcdef1234567890",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Reason:    "apply",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.bashrc"},
				{Path: "/home/user/.gitconfig"},
			},
		},
		{
			ID:        "1234567890abcdef",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			Reason:    "",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.zshrc"},
			},
		},
	}

	output := captureStdout(t, func() {
		err := listSnapshots(context.Background(), nil, sets)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "abcdef12")
	assert.Contains(t, output, "12345678")
	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "2 files")
	assert.Contains(t, output, "1 files")
}
