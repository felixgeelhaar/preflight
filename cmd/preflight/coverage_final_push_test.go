package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/felixgeelhaar/preflight/internal/domain/security"
	"github.com/felixgeelhaar/preflight/internal/domain/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// sync_conflicts.go: relationString
// ===========================================================================

func TestCovFinal_RelationString_Equal(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "equal (in sync)", relationString(sync.Equal))
}

func TestCovFinal_RelationString_Before(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "behind (pull needed)", relationString(sync.Before))
}

func TestCovFinal_RelationString_After(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ahead (push needed)", relationString(sync.After))
}

func TestCovFinal_RelationString_Concurrent(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "concurrent (merge needed)", relationString(sync.Concurrent))
}

func TestCovFinal_RelationString_Unknown(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", relationString(sync.CausalRelation(99)))
}

// ===========================================================================
// sync_conflicts.go: printJSONOutput
// ===========================================================================

func TestCovFinal_PrintJSONOutput(t *testing.T) {
	output := captureStdout(t, func() {
		_ = printJSONOutput(ConflictsOutputJSON{
			Relation:        "equal (in sync)",
			TotalConflicts:  2,
			AutoResolvable:  1,
			ManualConflicts: []ConflictJSON{{PackageKey: "brew:go", Type: "version_mismatch", LocalVersion: "1.21", RemoteVersion: "1.22", Resolvable: true}},
			NeedsMerge:      true,
		})
	})
	assert.Contains(t, output, "brew:go")
	assert.Contains(t, output, "equal (in sync)")
	assert.Contains(t, output, "version_mismatch")
}

// ===========================================================================
// marketplace.go: formatReason
// ===========================================================================

func TestCovFinal_FormatReason_AllCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		reason   marketplace.RecommendationReason
		expected string
	}{
		{marketplace.ReasonPopular, "popular"},
		{marketplace.ReasonTrending, "trending"},
		{marketplace.ReasonSimilarKeywords, "similar"},
		{marketplace.ReasonSameType, "same type"},
		{marketplace.ReasonSameAuthor, "same author"},
		{marketplace.ReasonComplementary, "complements"},
		{marketplace.ReasonRecentlyUpdated, "recent"},
		{marketplace.ReasonHighlyRated, "rated"},
		{marketplace.ReasonProviderMatch, "provider"},
		{marketplace.ReasonFeatured, "featured"},
		{marketplace.RecommendationReason("custom"), "custom"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatReason(tt.reason))
	}
}

// ===========================================================================
// marketplace.go: formatInstallAge
// ===========================================================================

func TestCovFinal_FormatInstallAge_AllBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		offset   time.Duration
		contains string
	}{
		{"just_now", 10 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 4 * 24 * time.Hour, "4d ago"},
		{"weeks", 14 * 24 * time.Hour, "2w ago"},
		{"old", 60 * 24 * time.Hour, "20"}, // date format like 2026-...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatInstallAge(time.Now().Add(-tt.offset))
			assert.Contains(t, result, tt.contains)
		})
	}
}

// ===========================================================================
// marketplace.go: outputRecommendations
// ===========================================================================

func TestCovFinal_OutputRecommendations(t *testing.T) {
	id, _ := marketplace.NewPackageID("test/pkg")
	recs := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:    id,
				Type:  "preset",
				Title: "Test Package",
			},
			Score:   0.85,
			Reasons: []marketplace.RecommendationReason{marketplace.ReasonPopular, marketplace.ReasonTrending},
		},
	}
	output := captureStdout(t, func() {
		outputRecommendations(recs)
	})
	assert.Contains(t, output, "preset")
	assert.Contains(t, output, "85.0%")
	assert.Contains(t, output, "popular")
}

// ===========================================================================
// marketplace.go: buildUserContext
// ===========================================================================

func TestCovFinal_BuildUserContext_WithKeywords(t *testing.T) {
	old := mpKeywords
	oldType := mpRecommendType
	defer func() {
		mpKeywords = old
		mpRecommendType = oldType
	}()

	mpKeywords = "nvim, go, zsh"
	mpRecommendType = "preset"

	svc := newMarketplaceService()
	ctx := buildUserContext(svc)

	assert.Contains(t, ctx.Keywords, "nvim")
	assert.Contains(t, ctx.Keywords, "go")
	assert.Contains(t, ctx.Keywords, "zsh")
	assert.Equal(t, []string{"preset"}, ctx.PreferredTypes)
}

func TestCovFinal_BuildUserContext_NoKeywords(t *testing.T) {
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
// marketplace.go: newMarketplaceService
// ===========================================================================

func TestCovFinal_NewMarketplaceService(t *testing.T) {
	t.Parallel()
	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}

// ===========================================================================
// env.go: extractEnvVars and extractEnvVarsMap
// ===========================================================================

func TestCovFinal_ExtractEnvVars_WithEnv(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"GOPATH":    "/home/user/go",
			"API_KEY":   "secret://vault/key",
			"NODE_PATH": "/usr/local/lib/node",
		},
	}
	vars := extractEnvVars(config)
	assert.Len(t, vars, 3)

	// Check secret detection
	secretFound := false
	for _, v := range vars {
		if v.Name == "API_KEY" {
			assert.True(t, v.Secret)
			secretFound = true
		}
	}
	assert.True(t, secretFound)
}

func TestCovFinal_ExtractEnvVars_NoEnv(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{"brew": map[string]interface{}{}}
	vars := extractEnvVars(config)
	assert.Empty(t, vars)
}

func TestCovFinal_ExtractEnvVarsMap(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"env": map[string]interface{}{
			"FOO": "bar",
			"BAZ": 42,
		},
	}
	m := extractEnvVarsMap(config)
	assert.Equal(t, "bar", m["FOO"])
	assert.Equal(t, "42", m["BAZ"])
}

func TestCovFinal_ExtractEnvVarsMap_NoEnv(t *testing.T) {
	t.Parallel()
	m := extractEnvVarsMap(map[string]interface{}{})
	assert.Empty(t, m)
}

// ===========================================================================
// env.go: runEnvSet and runEnvUnset in temp dir
// ===========================================================================

func TestCovFinal_RunEnvSet_And_Unset(t *testing.T) {
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))

	// Write a base layer
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\nenv:\n  EXISTING: value\n"), 0o644))

	// Save and restore global vars
	oldConfigPath := envConfigPath
	oldLayer := envLayer
	defer func() {
		envConfigPath = oldConfigPath
		envLayer = oldLayer
	}()

	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "base"

	// Test set
	output := captureStdout(t, func() {
		err := runEnvSet(nil, []string{"MY_VAR", "hello"})
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Set MY_VAR=hello")

	// Test unset
	output = captureStdout(t, func() {
		err := runEnvUnset(nil, []string{"MY_VAR"})
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Removed MY_VAR")
}

func TestCovFinal_RunEnvUnset_LayerNotFound(t *testing.T) {
	old := envConfigPath
	oldLayer := envLayer
	defer func() {
		envConfigPath = old
		envLayer = oldLayer
	}()

	envConfigPath = "/nonexistent/preflight.yaml"
	envLayer = "base"

	err := runEnvUnset(nil, []string{"FOO"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "layer not found")
}

func TestCovFinal_RunEnvUnset_NoEnvSection(t *testing.T) {
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	oldLayer := envLayer
	defer func() {
		envConfigPath = old
		envLayer = oldLayer
	}()

	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "base"

	err := runEnvUnset(nil, []string{"FOO"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no env section")
}

func TestCovFinal_RunEnvUnset_VarNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\nenv:\n  OTHER: val\n"), 0o644))

	old := envConfigPath
	oldLayer := envLayer
	defer func() {
		envConfigPath = old
		envLayer = oldLayer
	}()

	envConfigPath = filepath.Join(tmpDir, "preflight.yaml")
	envLayer = "base"

	err := runEnvUnset(nil, []string{"NONEXISTENT"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in layer")
}

// ===========================================================================
// env.go: runEnvExport
// ===========================================================================

func TestCovFinal_RunEnvExport_UnsupportedShell(t *testing.T) {
	oldShell := envShell
	oldConfig := envConfigPath
	defer func() {
		envShell = oldShell
		envConfigPath = oldConfig
	}()

	// Create a minimal config that can be loaded
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\nenv:\n  FOO: bar\n"), 0o644))

	envConfigPath = configPath
	envShell = "powershell"

	err := runEnvExport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}

// ===========================================================================
// init.go: runInitNonInteractive
// ===========================================================================

func TestCovFinal_RunInitNonInteractive_MissingPreset(t *testing.T) {
	old := initPreset
	defer func() { initPreset = old }()
	initPreset = ""

	err := runInitNonInteractive("/tmp/test-init/preflight.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--preset is required")
}

func TestCovFinal_RunInitNonInteractive_ValidPreset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")

	oldPreset := initPreset
	oldOutputDir := initOutputDir
	defer func() {
		initPreset = oldPreset
		initOutputDir = oldOutputDir
	}()

	initPreset = "balanced"
	initOutputDir = tmpDir

	output := captureStdout(t, func() {
		err := runInitNonInteractive(configPath)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Configuration created")

	// Check files were created
	_, err := os.Stat(configPath)
	assert.NoError(t, err)
	layerPath := filepath.Join(tmpDir, "layers", "base.yaml")
	_, err = os.Stat(layerPath)
	assert.NoError(t, err)
}

// ===========================================================================
// init.go: runInit - config already exists
// ===========================================================================

func TestCovFinal_RunInit_ConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("test"), 0o644))

	oldOutputDir := initOutputDir
	defer func() { initOutputDir = oldOutputDir }()
	initOutputDir = tmpDir

	output := captureStdout(t, func() {
		err := runInit(nil, nil)
		require.NoError(t, err)
	})
	assert.Contains(t, output, "already exists")
}

// ===========================================================================
// init.go: generateLayerForPreset - all branches
// ===========================================================================

func TestCovFinal_GenerateLayerForPreset_AllPresets(t *testing.T) {
	t.Parallel()
	presets := []struct {
		name     string
		contains string
	}{
		{"nvim:minimal", "preset: minimal"},
		{"nvim:balanced", "preset: kickstart"},
		{"balanced", "preset: kickstart"},
		{"nvim:maximal", "preset: astronvim"},
		{"maximal", "preset: astronvim"},
		{"shell:minimal", "default: zsh"},
		{"shell:balanced", "oh-my-zsh"},
		{"git:minimal", "editor: vim"},
		{"brew:minimal", "ripgrep"},
		{"anything-else", "Add your configuration"},
	}

	for _, tt := range presets {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := generateLayerForPreset(tt.name)
			assert.Contains(t, result, tt.contains)
		})
	}
}

// ===========================================================================
// init.go: detectAIProvider
// ===========================================================================

func TestCovFinal_DetectAIProvider_NoEnv(t *testing.T) {
	// Save and clear all AI-related env vars
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	oldProvider := aiProvider
	defer func() {
		aiProvider = oldProvider
	}()
	aiProvider = ""

	provider := detectAIProvider()
	assert.Nil(t, provider)
}

// ===========================================================================
// doctor.go: printDoctorQuiet
// ===========================================================================

func TestCovFinal_PrintDoctorQuiet_NoIssues(t *testing.T) {
	report := &app.DoctorReport{}
	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "No issues found")
}

func TestCovFinal_PrintDoctorQuiet_WithIssues(t *testing.T) {
	report := &app.DoctorReport{
		Issues: []app.DoctorIssue{
			{
				Severity:   app.SeverityError,
				Message:    "Package missing",
				Provider:   "brew",
				Expected:   "installed",
				Actual:     "not installed",
				FixCommand: "brew install go",
			},
			{
				Severity: app.SeverityWarning,
				Message:  "Config drift detected",
			},
		},
	}
	output := captureStdout(t, func() {
		printDoctorQuiet(report)
	})
	assert.Contains(t, output, "2 issue(s)")
	assert.Contains(t, output, "Package missing")
	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "installed")
	assert.Contains(t, output, "brew install go")
	assert.Contains(t, output, "Config drift")
}

// ===========================================================================
// clean.go: more branch coverage
// ===========================================================================

func TestCovFinal_FindOrphans_AllProviders(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "rust"},
			"casks":    []interface{}{"firefox"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "rust", "htop", "curl"},
			"casks":    []interface{}{"firefox", "slack"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-python.python", "golang.go"},
		},
	}

	orphans := findOrphans(config, systemState, nil, nil)

	// Should find htop, curl (formulae), slack (cask), golang.go (vscode ext)
	assert.Len(t, orphans, 4)
}

func TestCovFinal_FindOrphans_WithFilter(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "htop"},
		},
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"golang.go"},
		},
	}

	// Only check brew
	orphans := findOrphans(config, systemState, []string{"brew"}, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "htop", orphans[0].Name)
}

func TestCovFinal_FindOrphans_WithIgnoreList(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go"},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"go", "htop", "curl"},
		},
	}

	orphans := findOrphans(config, systemState, nil, []string{"htop"})
	assert.Len(t, orphans, 1)
	assert.Equal(t, "curl", orphans[0].Name)
}

func TestCovFinal_RemoveOrphans(t *testing.T) {
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "htop"},
		{Provider: "brew", Type: "cask", Name: "slack"},
		{Provider: "vscode", Type: "extension", Name: "golang.go"},
	}

	output := captureStdout(t, func() {
		removed, failed := removeOrphans(context.Background(), orphans)
		assert.Equal(t, 3, removed)
		assert.Equal(t, 0, failed)
	})
	assert.Contains(t, output, "brew uninstall htop")
	assert.Contains(t, output, "brew uninstall --cask slack")
	assert.Contains(t, output, "Removed vscode")
}

// ===========================================================================
// export.go: all export formats
// ===========================================================================

func TestCovFinal_ExportToNix_Full(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"ripgrep", "fzf"},
		},
		"git": map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
		},
		"shell": map[string]interface{}{
			"shell":   "zsh",
			"plugins": []interface{}{"git", "docker"},
		},
	}
	output, err := exportToNix(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "home.packages")
	assert.Contains(t, s, "ripgrep")
	assert.Contains(t, s, "programs.git")
	assert.Contains(t, s, `userName = "John"`)
	assert.Contains(t, s, "programs.zsh")
	assert.Contains(t, s, `name = "git"`)
}

func TestCovFinal_ExportToBrewfile_Full(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"homebrew/cask"},
			"formulae": []interface{}{"go", "rust"},
			"casks":    []interface{}{"firefox"},
		},
	}
	output, err := exportToBrewfile(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, `tap "homebrew/cask"`)
	assert.Contains(t, s, `brew "go"`)
	assert.Contains(t, s, `brew "rust"`)
	assert.Contains(t, s, `cask "firefox"`)
}

func TestCovFinal_ExportToShell_Full(t *testing.T) {
	t.Parallel()
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"taps":     []interface{}{"custom/tap"},
			"formulae": []interface{}{"go", "rust"},
			"casks":    []interface{}{"firefox"},
		},
		"git": map[string]interface{}{
			"name":  "User",
			"email": "user@example.com",
		},
	}
	output, err := exportToShell(config)
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	assert.Contains(t, s, "brew tap custom/tap")
	assert.Contains(t, s, "brew install")
	assert.Contains(t, s, "brew install --cask")
	assert.Contains(t, s, `git config --global user.name "User"`)
	assert.Contains(t, s, `git config --global user.email "user@example.com"`)
}

func TestCovFinal_ExportToShell_EmptyConfig(t *testing.T) {
	t.Parallel()
	output, err := exportToShell(map[string]interface{}{})
	require.NoError(t, err)
	s := string(output)
	assert.Contains(t, s, "#!/usr/bin/env bash")
	assert.Contains(t, s, "Setup complete!")
}

// ===========================================================================
// security.go: helper functions
// ===========================================================================

func TestCovFinal_OutputSecurityJSON_Error(t *testing.T) {
	output := captureStdout(t, func() {
		outputSecurityJSON(nil, assert.AnError)
	})
	assert.Contains(t, output, "error")
}

func TestCovFinal_OutputSecurityJSON_Result(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-2024-1", Package: "openssl", Version: "1.1.1", Severity: security.SeverityCritical, CVSS: 9.8, FixedIn: "1.1.2", Title: "RCE"},
		},
	}
	output := captureStdout(t, func() {
		outputSecurityJSON(result, nil)
	})
	assert.Contains(t, output, "CVE-2024-1")
	assert.Contains(t, output, "grype")
	assert.Contains(t, output, "critical")
}

func TestCovFinal_OutputSecurityText_NoVulns(t *testing.T) {
	result := &security.ScanResult{Scanner: "grype", Version: "1.0"}
	opts := security.ScanOptions{}
	output := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, output, "No vulnerabilities found")
}

func TestCovFinal_OutputSecurityText_WithVulns(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Package: "openssl", Version: "1.0", Severity: security.SeverityCritical, FixedIn: "1.1"},
			{ID: "CVE-2", Package: "curl", Version: "7.0", Severity: security.SeverityHigh},
		},
	}
	opts := security.ScanOptions{}
	output := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, output, "2 vulnerabilities")
	assert.Contains(t, output, "CVE-1")
	assert.Contains(t, output, "CRITICAL")
}

func TestCovFinal_OutputSecurityText_Quiet(t *testing.T) {
	result := &security.ScanResult{
		Scanner: "grype",
		Version: "1.0",
		Vulnerabilities: security.Vulnerabilities{
			{ID: "CVE-1", Package: "openssl", Version: "1.0", Severity: security.SeverityCritical},
		},
	}
	opts := security.ScanOptions{Quiet: true}
	output := captureStdout(t, func() {
		outputSecurityText(result, opts)
	})
	assert.Contains(t, output, "1 vulnerabilities")
	assert.NotContains(t, output, "CVE-1") // table not shown in quiet mode
}

func TestCovFinal_ShouldFail_Yes(t *testing.T) {
	t.Parallel()
	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{
			{Severity: security.SeverityCritical},
		},
	}
	assert.True(t, shouldFail(result, security.SeverityHigh))
}

func TestCovFinal_ShouldFail_No(t *testing.T) {
	t.Parallel()
	result := &security.ScanResult{
		Vulnerabilities: security.Vulnerabilities{
			{Severity: security.SeverityLow},
		},
	}
	assert.False(t, shouldFail(result, security.SeverityHigh))
}

func TestCovFinal_ShouldFail_Empty(t *testing.T) {
	t.Parallel()
	result := &security.ScanResult{}
	assert.False(t, shouldFail(result, security.SeverityHigh))
}

func TestCovFinal_GetScanner_AutoNoScanners(t *testing.T) {
	t.Parallel()
	registry := security.NewScannerRegistry()
	_, err := getScanner(registry, "auto")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scanners available")
}

func TestCovFinal_GetScanner_NamedNotFound(t *testing.T) {
	t.Parallel()
	registry := security.NewScannerRegistry()
	_, err := getScanner(registry, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestCovFinal_ListScanners(t *testing.T) {
	registry := security.NewScannerRegistry()
	registry.Register(security.NewGrypeScanner())
	registry.Register(security.NewTrivyScanner())

	output := captureStdout(t, func() {
		_ = listScanners(registry)
	})
	assert.Contains(t, output, "Available security scanners")
}

func TestCovFinal_FormatSeverity_All(t *testing.T) {
	t.Parallel()
	tests := []struct {
		sev      security.Severity
		contains string
	}{
		{security.SeverityCritical, "CRITICAL"},
		{security.SeverityHigh, "HIGH"},
		{security.SeverityMedium, "MEDIUM"},
		{security.SeverityLow, "LOW"},
		{security.SeverityNegligible, "NEGLIGIBLE"},
		{security.Severity("custom"), "custom"},
	}
	for _, tt := range tests {
		result := formatSeverity(tt.sev)
		assert.Contains(t, result, tt.contains)
	}
}

// ===========================================================================
// outdated.go: helper functions
// ===========================================================================

func TestCovFinal_ParseUpdateType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, security.UpdateMajor, parseUpdateType("major"))
	assert.Equal(t, security.UpdateMinor, parseUpdateType("minor"))
	assert.Equal(t, security.UpdatePatch, parseUpdateType("patch"))
	assert.Equal(t, security.UpdateMinor, parseUpdateType("unknown"))
	assert.Equal(t, security.UpdateMajor, parseUpdateType("MAJOR"))
}

func TestCovFinal_ShouldFailOutdated(t *testing.T) {
	t.Parallel()
	result := &security.OutdatedResult{
		Packages: security.OutdatedPackages{
			{Name: "go", UpdateType: security.UpdateMajor},
		},
	}
	assert.True(t, shouldFailOutdated(result, security.UpdateMajor))
	assert.True(t, shouldFailOutdated(result, security.UpdateMinor))
	assert.False(t, shouldFailOutdated(&security.OutdatedResult{}, security.UpdateMajor))
}

func TestCovFinal_FormatUpdateType(t *testing.T) {
	t.Parallel()
	assert.Contains(t, formatUpdateType(security.UpdateMajor), "MAJOR")
	assert.Contains(t, formatUpdateType(security.UpdateMinor), "MINOR")
	assert.Contains(t, formatUpdateType(security.UpdatePatch), "PATCH")
	assert.Contains(t, formatUpdateType(security.UpdateType("x")), "x")
}

func TestCovFinal_OutputOutdatedText_WithPackages(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", CurrentVersion: "1.20", LatestVersion: "1.22", UpdateType: security.UpdateMajor, Provider: "brew"},
			{Name: "rust", CurrentVersion: "1.70", LatestVersion: "1.71", UpdateType: security.UpdateMinor, Provider: "brew"},
		},
	}
	output := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, output, "2 packages")
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "rust")
}

func TestCovFinal_OutputOutdatedText_Quiet(t *testing.T) {
	result := &security.OutdatedResult{
		Checker: "brew",
		Packages: security.OutdatedPackages{
			{Name: "go", UpdateType: security.UpdateMajor},
		},
	}
	output := captureStdout(t, func() {
		outputOutdatedText(result, true)
	})
	assert.Contains(t, output, "1 packages")
	assert.NotContains(t, output, "CURRENT") // table header not shown
}

func TestCovFinal_OutputOutdatedText_NoPackages(t *testing.T) {
	result := &security.OutdatedResult{Checker: "brew"}
	output := captureStdout(t, func() {
		outputOutdatedText(result, false)
	})
	assert.Contains(t, output, "All packages are up to date")
}

func TestCovFinal_OutputUpgradeText_DryRun(t *testing.T) {
	result := &security.UpgradeResult{
		DryRun: true,
		Upgraded: []security.UpgradedPackage{
			{Name: "go", FromVersion: "1.20", ToVersion: "1.22"},
		},
	}
	output := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, output, "would upgrade")
	assert.Contains(t, output, "Would upgrade 1")
}

func TestCovFinal_OutputUpgradeText_WithSkippedAndFailed(t *testing.T) {
	old := outdatedMajor
	defer func() { outdatedMajor = old }()
	outdatedMajor = false

	result := &security.UpgradeResult{
		Upgraded: []security.UpgradedPackage{{Name: "rust", FromVersion: "1.70", ToVersion: "1.71"}},
		Skipped:  []security.SkippedPackage{{Name: "go", Reason: "major update"}},
		Failed:   []security.FailedPackage{{Name: "node", Error: "permission denied"}},
	}
	output := captureStdout(t, func() {
		outputUpgradeText(result)
	})
	assert.Contains(t, output, "rust")
	assert.Contains(t, output, "go: skipped")
	assert.Contains(t, output, "node: permission denied")
	assert.Contains(t, output, "1 skipped")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "--major")
}

// ===========================================================================
// deprecated.go: helper functions
// ===========================================================================

func TestCovFinal_OutputDeprecatedJSON_Error(t *testing.T) {
	output := captureStdout(t, func() {
		outputDeprecatedJSON(nil, assert.AnError)
	})
	assert.Contains(t, output, "error")
}

func TestCovFinal_OutputDeprecatedJSON_Result(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "oldpkg", Version: "1.0", Provider: "brew", Reason: security.ReasonDeprecated, Message: "use newpkg"},
		},
	}
	output := captureStdout(t, func() {
		outputDeprecatedJSON(result, nil)
	})
	assert.Contains(t, output, "oldpkg")
	assert.Contains(t, output, "brew")
}

func TestCovFinal_OutputDeprecatedText_WithPackages(t *testing.T) {
	result := &security.DeprecatedResult{
		Checker: "brew",
		Packages: security.DeprecatedPackages{
			{Name: "old1", Version: "1.0", Provider: "brew", Reason: security.ReasonDisabled, Message: "disabled"},
			{Name: "old2", Provider: "brew", Reason: security.ReasonDeprecated},
		},
	}
	output := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})
	assert.Contains(t, output, "2 packages")
	assert.Contains(t, output, "old1")
	assert.Contains(t, output, "old2")
	assert.Contains(t, output, "DISABLED")
}

func TestCovFinal_OutputDeprecatedText_NoPackages(t *testing.T) {
	result := &security.DeprecatedResult{Checker: "brew"}
	output := captureStdout(t, func() {
		outputDeprecatedText(result, false)
	})
	assert.Contains(t, output, "No deprecated packages")
}

func TestCovFinal_FormatDeprecationStatus_All(t *testing.T) {
	t.Parallel()
	assert.Contains(t, formatDeprecationStatus(security.ReasonDisabled), "DISABLED")
	assert.Contains(t, formatDeprecationStatus(security.ReasonDeprecated), "DEPRECATED")
	assert.Contains(t, formatDeprecationStatus(security.ReasonEOL), "EOL")
	assert.Contains(t, formatDeprecationStatus(security.ReasonUnmaintained), "UNMAINTAINED")
	assert.Contains(t, formatDeprecationStatus(security.DeprecationReason("other")), "other")
}

// ===========================================================================
// cleanup.go: output functions
// ===========================================================================

func TestCovFinal_OutputCleanupJSON_Error(t *testing.T) {
	output := captureStdout(t, func() {
		outputCleanupJSON(nil, nil, assert.AnError)
	})
	assert.Contains(t, output, "error")
}

func TestCovFinal_OutputCleanupJSON_Result(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Type: security.RedundancyDuplicate, Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}},
		},
	}
	output := captureStdout(t, func() {
		outputCleanupJSON(result, nil, nil)
	})
	assert.Contains(t, output, "go@1.24")
}

func TestCovFinal_OutputCleanupJSON_Cleanup(t *testing.T) {
	cleanup := &security.CleanupResult{
		Removed: []string{"go@1.24"},
		DryRun:  true,
	}
	output := captureStdout(t, func() {
		outputCleanupJSON(nil, cleanup, nil)
	})
	assert.Contains(t, output, "go@1.24")
}

func TestCovFinal_OutputCleanupText_NoRedundancies(t *testing.T) {
	result := &security.RedundancyResult{Checker: "brew"}
	output := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, output, "No redundancies detected")
}

func TestCovFinal_OutputCleanupText_WithRedundancies(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Type: security.RedundancyDuplicate, Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}, Recommendation: "remove versioned"},
			{Type: security.RedundancyOverlap, Packages: []string{"vim", "nvim"}, Category: "text_editor", Keep: []string{"nvim"}, Remove: []string{"vim"}, Recommendation: "use nvim"},
			{Type: security.RedundancyOrphan, Packages: []string{"libfoo"}, Action: "brew autoremove"},
		},
	}
	output := captureStdout(t, func() {
		outputCleanupText(result, false)
	})
	assert.Contains(t, output, "3 redundancies")
	assert.Contains(t, output, "go + go@1.24")
	assert.Contains(t, output, "Text Editor")
	assert.Contains(t, output, "libfoo")
}

func TestCovFinal_OutputCleanupText_Quiet(t *testing.T) {
	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Type: security.RedundancyDuplicate, Packages: []string{"go", "go@1.24"}, Remove: []string{"go@1.24"}},
		},
	}
	output := captureStdout(t, func() {
		outputCleanupText(result, true)
	})
	assert.Contains(t, output, "1 redundancies")
	assert.Contains(t, output, "preflight cleanup --all")
}

func TestCovFinal_FormatCategory(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Text Editor", formatCategory("text_editor"))
	assert.Equal(t, "Git", formatCategory("git"))
	assert.Equal(t, "", formatCategory(""))
	assert.Equal(t, "Version Duplicate", formatCategory("version_duplicate"))
}

// ===========================================================================
// compliance.go: collectEvaluatedItems
// ===========================================================================

func TestCovFinal_CollectEvaluatedItems_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, collectEvaluatedItems(nil))
}

func TestCovFinal_CollectEvaluatedItems_WithData(t *testing.T) {
	t.Parallel()
	result := &app.ValidationResult{
		Info:   []string{"brew: 5 packages", "git: configured"},
		Errors: []string{"ssh: missing"},
	}
	items := collectEvaluatedItems(result)
	assert.Len(t, items, 3)
	assert.Contains(t, items, "brew: 5 packages")
	assert.Contains(t, items, "ssh: missing")
}

func TestCovFinal_OutputComplianceError(t *testing.T) {
	output := captureStdout(t, func() {
		outputComplianceError(assert.AnError)
	})
	assert.Contains(t, output, "error")
}

// ===========================================================================
// rollback.go: formatAge
// ===========================================================================

func TestCovFinal_FormatAge_AllBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		offset   time.Duration
		expected string
	}{
		{"just_now", 10 * time.Second, "just now"},
		{"1_min", 1 * time.Minute, "1 min ago"},
		{"5_mins", 5 * time.Minute, "5 mins ago"},
		{"1_hour", 1 * time.Hour, "1 hour ago"},
		{"3_hours", 3 * time.Hour, "3 hours ago"},
		{"1_day", 25 * time.Hour, "1 day ago"},
		{"3_days", 73 * time.Hour, "3 days ago"},
		{"1_week", 8 * 24 * time.Hour, "1 week ago"},
		{"3_weeks", 22 * 24 * time.Hour, "3 weeks ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatAge(time.Now().Add(-tt.offset))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ===========================================================================
// trust.go: detectKeyType
// ===========================================================================

func TestCovFinal_DetectKeyType_SSH(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ssh", string(detectKeyType([]byte("ssh-ed25519 AAAA..."))))
	assert.Equal(t, "ssh", string(detectKeyType([]byte("ssh-rsa AAAA..."))))
	assert.Equal(t, "ssh", string(detectKeyType([]byte("ecdsa-sha2-nistp256 AAAA..."))))
}

func TestCovFinal_DetectKeyType_GPGArmored(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "gpg", string(detectKeyType([]byte("-----BEGIN PGP PUBLIC KEY BLOCK-----"))))
}

func TestCovFinal_DetectKeyType_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", string(detectKeyType(nil)))
	assert.Equal(t, "", string(detectKeyType([]byte{})))
}

func TestCovFinal_DetectKeyType_Unknown(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", string(detectKeyType([]byte("some random data"))))
}

// ===========================================================================
// trust.go: isValidOpenPGPPacket
// ===========================================================================

func TestCovFinal_IsValidOpenPGPPacket(t *testing.T) {
	t.Parallel()
	// Too short
	assert.False(t, isValidOpenPGPPacket([]byte{0x00}))
	// Bit 7 not set
	assert.False(t, isValidOpenPGPPacket([]byte{0x00, 0x00}))
	// Old format with public key tag (tag 6): 0x80 | (6 << 2) | length_type = 0x98 | 0x01
	assert.True(t, isValidOpenPGPPacket([]byte{0x98, 0x01}))
	// New format with public key tag (tag 6): 0xC0 | 6 = 0xC6
	assert.True(t, isValidOpenPGPPacket([]byte{0xC6, 0x01}))
}

// ===========================================================================
// discover.go: getPatternIcon
// ===========================================================================

func TestCovFinal_GetPatternIcon_All(t *testing.T) {
	t.Parallel()
	// The icon function uses discover.PatternType values
	// Test the default case
	assert.NotEmpty(t, getPatternIcon("shell"))
	assert.NotEmpty(t, getPatternIcon("editor"))
	assert.NotEmpty(t, getPatternIcon("git"))
	assert.NotEmpty(t, getPatternIcon("ssh"))
	assert.NotEmpty(t, getPatternIcon("tmux"))
	assert.NotEmpty(t, getPatternIcon("package_manager"))
	assert.Equal(t, "•", getPatternIcon("other"))
}

// ===========================================================================
// handleRemove and handleCleanupAll dry-run paths (cleanup.go)
// ===========================================================================

func TestCovFinal_HandleRemove_DryRun_Text(t *testing.T) {
	oldDryRun := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = oldDryRun
		cleanupJSON = oldJSON
	}()

	cleanupDryRun = true
	cleanupJSON = false

	output := captureStdout(t, func() {
		err := handleRemove(context.Background(), nil, []string{"go@1.24", "node@18"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "go@1.24")
	assert.Contains(t, output, "node@18")
}

func TestCovFinal_HandleRemove_DryRun_JSON(t *testing.T) {
	oldDryRun := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = oldDryRun
		cleanupJSON = oldJSON
	}()

	cleanupDryRun = true
	cleanupJSON = true

	output := captureStdout(t, func() {
		err := handleRemove(context.Background(), nil, []string{"go@1.24"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "go@1.24")
	assert.Contains(t, output, "dry_run")
}

func TestCovFinal_HandleCleanupAll_DryRun(t *testing.T) {
	oldDryRun := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = oldDryRun
		cleanupJSON = oldJSON
	}()

	cleanupDryRun = true
	cleanupJSON = false

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Remove: []string{"go@1.24"}},
			{Remove: []string{"node@18"}},
		},
	}
	output := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), nil, result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Would remove 2 package(s)")
}

func TestCovFinal_HandleCleanupAll_EmptyRemove(t *testing.T) {
	oldDryRun := cleanupDryRun
	oldJSON := cleanupJSON
	defer func() {
		cleanupDryRun = oldDryRun
		cleanupJSON = oldJSON
	}()

	cleanupDryRun = false
	cleanupJSON = false

	result := &security.RedundancyResult{
		Checker: "brew",
		Redundancies: security.Redundancies{
			{Packages: []string{"foo"}, Remove: nil},
		},
	}
	output := captureStdout(t, func() {
		err := handleCleanupAll(context.Background(), nil, result)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Nothing to clean up")
}

// ===========================================================================
// marketplace.go: runMarketplaceSearch - empty results
// ===========================================================================

func TestCovFinal_RunMarketplaceSearch_NoResults(t *testing.T) {
	old := mpOfflineMode
	defer func() { mpOfflineMode = old }()
	mpOfflineMode = true

	captureStdout(t, func() {
		err := runMarketplaceSearch(nil, []string{"xyznonexistent"})
		// In offline mode, may return error about no cached index
		_ = err
	})
}

func TestCovFinal_RunMarketplaceSearch_ByType(t *testing.T) {
	old := mpOfflineMode
	oldType := mpSearchType
	defer func() {
		mpOfflineMode = old
		mpSearchType = oldType
	}()

	mpOfflineMode = true
	mpSearchType = "preset"

	output := captureStdout(t, func() {
		err := runMarketplaceSearch(nil, nil)
		// May return error or no results, both are fine
		_ = err
	})
	_ = output
}

// ===========================================================================
// marketplace.go: runMarketplaceFeatured
// ===========================================================================

func TestCovFinal_RunMarketplaceFeatured_Empty(t *testing.T) {
	old := mpOfflineMode
	defer func() { mpOfflineMode = old }()
	mpOfflineMode = true

	captureStdout(t, func() {
		err := runMarketplaceFeatured(nil, nil)
		// In offline mode, may return error about no cached index
		_ = err
	})
}

// ===========================================================================
// marketplace.go: runMarketplacePopular
// ===========================================================================

func TestCovFinal_RunMarketplacePopular_Empty(t *testing.T) {
	old := mpOfflineMode
	defer func() { mpOfflineMode = old }()
	mpOfflineMode = true

	captureStdout(t, func() {
		err := runMarketplacePopular(nil, nil)
		// In offline mode, may return error about no cached index
		_ = err
	})
}

// ===========================================================================
// marketplace.go: runMarketplaceList
// ===========================================================================

func TestCovFinal_RunMarketplaceList_Empty(t *testing.T) {
	old := mpOfflineMode
	defer func() { mpOfflineMode = old }()
	mpOfflineMode = true

	captureStdout(t, func() {
		err := runMarketplaceList(nil, nil)
		// In offline mode, may return error or show empty list
		_ = err
	})
}

// ===========================================================================
// marketplace.go: runMarketplaceRecommend
// ===========================================================================

func TestCovFinal_RunMarketplaceRecommend_Empty(t *testing.T) {
	old := mpOfflineMode
	oldKw := mpKeywords
	oldSimilar := mpSimilarTo
	defer func() {
		mpOfflineMode = old
		mpKeywords = oldKw
		mpSimilarTo = oldSimilar
	}()

	mpOfflineMode = true
	mpKeywords = ""
	mpSimilarTo = ""

	captureStdout(t, func() {
		err := runMarketplaceRecommend(nil, nil)
		// In offline mode, may return error about no cached index
		_ = err
	})
}

// ===========================================================================
// sync.go: findRepoRoot, getCurrentBranch, hasUncommittedChanges
// ===========================================================================

func TestCovFinal_FindRepoRoot(t *testing.T) {
	// We're in a git repo, so this should work
	root, err := findRepoRoot()
	assert.NoError(t, err)
	assert.NotEmpty(t, root)
}

func TestCovFinal_GetCurrentBranch(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)
	branch, err := getCurrentBranch(root)
	assert.NoError(t, err)
	assert.NotEmpty(t, branch)
}

func TestCovFinal_HasUncommittedChanges(t *testing.T) {
	root, err := findRepoRoot()
	require.NoError(t, err)
	// Just test it doesn't error
	_, err = hasUncommittedChanges(root)
	assert.NoError(t, err)
}

// ===========================================================================
// sync.go: runSync error paths
// ===========================================================================

func TestCovFinal_RunSync_InvalidRemote(t *testing.T) {
	oldRemote := syncRemote
	defer func() { syncRemote = oldRemote }()
	syncRemote = "invalid/remote/name"

	err := runSync(syncCmd, nil)
	assert.Error(t, err)
}

func TestCovFinal_RunSync_InvalidBranch(t *testing.T) {
	oldBranch := syncBranch
	defer func() { syncBranch = oldBranch }()
	syncBranch = "invalid..branch"

	err := runSync(syncCmd, nil)
	assert.Error(t, err)
}

// ===========================================================================
// clean.go: runClean error path
// ===========================================================================

func TestCovFinal_RunClean_NonexistentConfig(t *testing.T) {
	old := cleanConfigPath
	defer func() { cleanConfigPath = old }()
	cleanConfigPath = "/nonexistent/path/preflight.yaml"

	err := runClean(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

// ===========================================================================
// doctor.go: runDoctor error path with quiet mode
// ===========================================================================

func TestCovFinal_RunDoctor_QuietMode_NonexistentConfig(t *testing.T) {
	old := cfgFile
	oldQuiet := doctorQuiet
	defer func() {
		cfgFile = old
		doctorQuiet = oldQuiet
	}()
	cfgFile = "/nonexistent/preflight.yaml"
	doctorQuiet = true

	err := runDoctor(nil, nil)
	assert.Error(t, err)
}

// ===========================================================================
// export.go: runExport unsupported format
// ===========================================================================

func TestCovFinal_RunExport_UnsupportedFormat(t *testing.T) {
	old := exportFormat
	oldConfig := exportConfigPath
	defer func() {
		exportFormat = old
		exportConfigPath = oldConfig
	}()

	// Create a loadable config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	exportConfigPath = configPath
	exportFormat = "invalid_format"

	err := runExport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestCovFinal_RunExport_NonexistentConfig(t *testing.T) {
	old := exportConfigPath
	defer func() { exportConfigPath = old }()
	exportConfigPath = "/nonexistent/preflight.yaml"

	err := runExport(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

// ===========================================================================
// tour.go: more branches
// ===========================================================================

func TestCovFinal_RunTour_ListFlag(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = true

	output := captureStdout(t, func() {
		err := runTour(nil, nil)
		assert.NoError(t, err)
	})
	assert.NotEmpty(t, output)
}

func TestCovFinal_RunTour_InvalidTopic(t *testing.T) {
	old := tourListFlag
	defer func() { tourListFlag = old }()
	tourListFlag = false

	err := runTour(nil, []string{"nonexistent_topic_xyz"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic")
}

// ===========================================================================
// catalog.go: runCatalogVerify more branches
// ===========================================================================

func TestCovFinal_RunCatalogVerify_BuiltinCatalog(t *testing.T) {
	output := captureStdout(t, func() {
		err := runCatalogVerify(nil, []string{"builtin"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "builtin")
	assert.Contains(t, output, "Verified")
}

// ===========================================================================
// secrets.go: resolveAge
// ===========================================================================

func TestCovFinal_ResolveAge_FileNotFound(t *testing.T) {
	_, err := resolveAge("nonexistent-key-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ===========================================================================
// WriteEnvFile test
// ===========================================================================

func TestCovFinal_WriteEnvFile(t *testing.T) {
	vars := []EnvVar{
		{Name: "FOO", Value: "bar"},
		{Name: "SECRET", Value: "secret://vault/key", Secret: true},
	}

	// This writes to user home, which should be fine in tests
	err := WriteEnvFile(vars)
	assert.NoError(t, err)
}

// ===========================================================================
// runEnvList with valid config
// ===========================================================================

func TestCovFinal_RunEnvList_NoVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	defer func() { envConfigPath = old }()
	envConfigPath = configPath

	output := captureStdout(t, func() {
		err := runEnvList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No environment variables")
}

func TestCovFinal_RunEnvList_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	oldJSON := envJSON
	defer func() {
		envConfigPath = old
		envJSON = oldJSON
	}()
	envConfigPath = configPath
	envJSON = false

	output := captureStdout(t, func() {
		err := runEnvList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No environment variables")
}

func TestCovFinal_RunEnvList_JSON_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	oldJSON := envJSON
	defer func() {
		envConfigPath = old
		envJSON = oldJSON
	}()
	envConfigPath = configPath
	envJSON = true

	output := captureStdout(t, func() {
		err := runEnvList(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No environment variables")
}

// ===========================================================================
// runEnvExport with valid config
// ===========================================================================

func TestCovFinal_RunEnvExport_Bash_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	oldShell := envShell
	defer func() {
		envConfigPath = old
		envShell = oldShell
	}()
	envConfigPath = configPath
	envShell = "bash"

	output := captureStdout(t, func() {
		err := runEnvExport(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Generated by preflight")
}

func TestCovFinal_RunEnvExport_Fish_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	oldShell := envShell
	defer func() {
		envConfigPath = old
		envShell = oldShell
	}()
	envConfigPath = configPath
	envShell = "fish"

	output := captureStdout(t, func() {
		err := runEnvExport(nil, nil)
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Generated by preflight")
}

// ===========================================================================
// runEnvDiff
// ===========================================================================

func TestCovFinal_RunEnvDiff_NoDiffs(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n  work:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\nenv:\n  FOO: bar\n"), 0o644))

	old := envConfigPath
	defer func() { envConfigPath = old }()
	envConfigPath = configPath

	output := captureStdout(t, func() {
		err := runEnvDiff(nil, []string{"default", "work"})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "No differences")
}

// ===========================================================================
// runEnvGet
// ===========================================================================

func TestCovFinal_RunEnvGet_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaults:\n  mode: intent\ntargets:\n  default:\n    - base\n"), 0o644))
	layersDir := filepath.Join(tmpDir, "layers")
	require.NoError(t, os.MkdirAll(layersDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layersDir, "base.yaml"), []byte("name: base\n"), 0o644))

	old := envConfigPath
	defer func() { envConfigPath = old }()
	envConfigPath = configPath

	err := runEnvGet(nil, []string{"NONEXISTENT"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
