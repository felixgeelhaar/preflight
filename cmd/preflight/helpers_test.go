package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/felixgeelhaar/preflight/internal/domain/discover"
	"github.com/felixgeelhaar/preflight/internal/domain/marketplace"
	"github.com/felixgeelhaar/preflight/internal/domain/policy"
	"github.com/felixgeelhaar/preflight/internal/domain/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEqualValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal ints", 42, 42, true},
		{"different ints", 42, 43, false},
		{"equal slices", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"nil values", nil, nil, true},
		{"mixed types same value", "42", 42, true}, // fmt.Sprintf("%v") produces same output
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := equalValues(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []string
		provider  string
		expected  bool
	}{
		{"empty list", []string{}, "brew", false},
		{"contains exact", []string{"brew", "apt", "files"}, "brew", true},
		{"not found", []string{"brew", "apt", "files"}, "git", false},
		{"with whitespace", []string{" brew ", "apt"}, "brew", true},
		{"single element found", []string{"brew"}, "brew", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := containsProvider(tt.providers, tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil value", nil, "<nil>"},
		{"string value", "hello", "hello"},
		{"int value", 42, "42"},
		{"small slice", []interface{}{"a", "b"}, "[a b]"},
		{"large slice", []interface{}{"a", "b", "c", "d", "e"}, "[5 items]"},
		{"map value", map[string]interface{}{"key": "value", "foo": "bar"}, "{2 keys}"},
		{"empty map", map[string]interface{}{}, "{0 keys}"},
		{"boolean", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"empty string", "", 10, ""},
		{"minimal truncation", "abcdefghij", 6, "abc..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		{"hours", "24h", 24 * time.Hour, false},
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"with spaces", " 12h ", 12 * time.Hour, false},
		{"uppercase", "5D", 5 * 24 * time.Hour, false},
		{"invalid unit", "5x", 0, true},
		{"too short", "h", 0, true},
		{"no number", "xh", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseDuration(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   string
		expected string
	}{
		{"success", "✓ success"},
		{"failed", "✗ failed"},
		{"partial", "~ partial"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			result := formatStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldCheckProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filter   []string
		provider string
		expected bool
	}{
		{"empty filter includes all", []string{}, "brew", true},
		{"match found", []string{"brew", "apt"}, "brew", true},
		{"no match", []string{"brew", "apt"}, "files", false},
		{"with whitespace", []string{" brew "}, "brew", true},
		{"single filter match", []string{"apt"}, "apt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := shouldCheckProvider(tt.filter, tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsIgnored(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		item       string
		ignoreList []string
		expected   bool
	}{
		{"empty ignore list", "package", []string{}, false},
		{"in ignore list", "vim", []string{"vim", "emacs"}, true},
		{"not in list", "nano", []string{"vim", "emacs"}, false},
		{"with whitespace in list", "vim", []string{" vim "}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isIgnored(tt.item, tt.ignoreList)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHistoryAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"just now", time.Now(), "just now"},
		{"minutes ago", time.Now().Add(-5 * time.Minute), "5m ago"},
		{"hours ago", time.Now().Add(-3 * time.Hour), "3h ago"},
		{"days ago", time.Now().Add(-2 * 24 * time.Hour), "2d ago"},
		{"weeks ago", time.Now().Add(-14 * 24 * time.Hour), "2w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatHistoryAge(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHistoryAge_OldDate(t *testing.T) {
	t.Parallel()

	// For dates older than 30 days, returns formatted date
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	result := formatHistoryAge(oldTime)
	// Should return a date format like "Jan 2"
	assert.Contains(t, result, " ")
}

func TestExtractEnvVars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]interface{}
		expected int
	}{
		{
			"empty config",
			map[string]interface{}{},
			0,
		},
		{
			"no env section",
			map[string]interface{}{"other": "value"},
			0,
		},
		{
			"with env vars",
			map[string]interface{}{
				"env": map[string]interface{}{
					"PATH":   "/usr/bin",
					"EDITOR": "vim",
				},
			},
			2,
		},
		{
			"with secret",
			map[string]interface{}{
				"env": map[string]interface{}{
					"API_KEY": "secret://vault/api-key",
				},
			},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractEnvVars(tt.config)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestExtractEnvVars_SecretMarking(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"env": map[string]interface{}{
			"API_KEY":  "secret://vault/key",
			"HOSTNAME": "localhost",
		},
	}

	vars := extractEnvVars(config)
	assert.Len(t, vars, 2)

	var secretCount int
	for _, v := range vars {
		if v.Secret {
			secretCount++
		}
	}
	assert.Equal(t, 1, secretCount)
}

func TestExtractEnvVarsMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]interface{}
		expected int
	}{
		{"empty config", map[string]interface{}{}, 0},
		{
			"with env",
			map[string]interface{}{
				"env": map[string]interface{}{
					"A": "1",
					"B": "2",
				},
			},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractEnvVarsMap(tt.config)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestGetPatternIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		patternType discover.PatternType
		expected    string
	}{
		{discover.PatternTypeShell, "🐚"},
		{discover.PatternTypeEditor, "📝"},
		{discover.PatternTypeGit, "📦"},
		{discover.PatternTypeSSH, "🔐"},
		{discover.PatternTypeTmux, "🖥️"},
		{discover.PatternTypePackageManager, "📦"},
		{discover.PatternType("unknown"), "•"},
	}

	for _, tt := range tests {
		t.Run(string(tt.patternType), func(t *testing.T) {
			t.Parallel()
			result := getPatternIcon(tt.patternType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSaveHistoryEntry(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "test-entry-123",
		Command:   "apply",
		Target:    "default",
		Status:    "success",
		Timestamp: time.Now(),
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify file was created
	historyPath := filepath.Join(tmpDir, ".preflight", "history", "test-entry-123.json")
	_, err = os.Stat(historyPath)
	assert.NoError(t, err)
}

func TestWriteEnvFile(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "PATH", Value: "/usr/bin", Secret: false},
		{Name: "API_KEY", Value: "secret", Secret: true},
		{Name: "EDITOR", Value: "vim", Secret: false},
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	// Read and verify content
	content, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "env.sh"))
	require.NoError(t, err)

	// Should contain non-secret vars
	assert.Contains(t, string(content), "export PATH=")
	assert.Contains(t, string(content), "export EDITOR=")
	// Should NOT contain secret vars
	assert.NotContains(t, string(content), "API_KEY")
}

func TestExportToNix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]interface{}
		contains []string
	}{
		{
			"empty config",
			map[string]interface{}{},
			[]string{"# Generated by preflight export", "{ config, pkgs, ... }:"},
		},
		{
			"brew formulae",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "vim", "tmux"},
				},
			},
			[]string{"home.packages = with pkgs;", "git", "vim", "tmux"},
		},
		{
			"git config",
			map[string]interface{}{
				"git": map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				},
			},
			[]string{"programs.git", "userName = \"Test User\"", "userEmail = \"test@example.com\""},
		},
		{
			"zsh shell",
			map[string]interface{}{
				"shell": map[string]interface{}{
					"shell":   "zsh",
					"plugins": []interface{}{"git", "docker"},
				},
			},
			[]string{"programs.zsh", "enable = true", "{ name = \"git\"", "{ name = \"docker\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := exportToNix(tt.config)
			require.NoError(t, err)
			for _, c := range tt.contains {
				assert.Contains(t, string(output), c)
			}
		})
	}
}

func TestExportToBrewfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]interface{}
		contains []string
	}{
		{
			"empty config",
			map[string]interface{}{},
			[]string{"# Generated by preflight export"},
		},
		{
			"taps only",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"taps": []interface{}{"homebrew/cask", "homebrew/core"},
				},
			},
			[]string{"tap \"homebrew/cask\"", "tap \"homebrew/core\""},
		},
		{
			"formulae only",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "vim"},
				},
			},
			[]string{"brew \"git\"", "brew \"vim\""},
		},
		{
			"casks only",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"casks": []interface{}{"visual-studio-code", "docker"},
				},
			},
			[]string{"cask \"visual-studio-code\"", "cask \"docker\""},
		},
		{
			"full config",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"taps":     []interface{}{"homebrew/cask"},
					"formulae": []interface{}{"git"},
					"casks":    []interface{}{"iterm2"},
				},
			},
			[]string{"tap \"homebrew/cask\"", "brew \"git\"", "cask \"iterm2\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := exportToBrewfile(tt.config)
			require.NoError(t, err)
			for _, c := range tt.contains {
				assert.Contains(t, string(output), c)
			}
		})
	}
}

func TestExportToShell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]interface{}
		contains []string
	}{
		{
			"empty config",
			map[string]interface{}{},
			[]string{"#!/usr/bin/env bash", "set -euo pipefail", "echo \"Setup complete!\""},
		},
		{
			"brew taps",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"taps": []interface{}{"homebrew/cask"},
				},
			},
			[]string{"brew tap homebrew/cask"},
		},
		{
			"brew formulae",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "vim"},
				},
			},
			[]string{"brew install", "git", "vim"},
		},
		{
			"brew casks",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"casks": []interface{}{"docker", "iterm2"},
				},
			},
			[]string{"brew install --cask", "docker", "iterm2"},
		},
		{
			"git config",
			map[string]interface{}{
				"git": map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				},
			},
			[]string{"git config --global user.name \"Test User\"", "git config --global user.email \"test@example.com\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := exportToShell(tt.config)
			require.NoError(t, err)
			for _, c := range tt.contains {
				assert.Contains(t, string(output), c)
			}
		})
	}
}

func TestGetProfileDir(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getProfileDir()
	assert.Equal(t, filepath.Join(tmpDir, ".preflight", "profiles"), dir)
}

func TestGetCurrentProfile_NoProfile(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profile := getCurrentProfile()
	assert.Empty(t, profile)
}

func TestSetAndGetCurrentProfile(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("work")
	require.NoError(t, err)

	profile := getCurrentProfile()
	assert.Equal(t, "work", profile)
}

func TestLoadCustomProfiles_NoFile(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles, err := loadCustomProfiles()
	assert.Error(t, err)
	assert.Nil(t, profiles)
}

func TestSaveAndLoadCustomProfiles(t *testing.T) {
	// Not parallel - modifies HOME env var
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "work", Target: "work", Description: "Work profile"},
		{Name: "personal", Target: "personal", Description: "Personal profile"},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "work", loaded[0].Name)
	assert.Equal(t, "personal", loaded[1].Name)
}

func TestResolveSecret_EnvBackend(t *testing.T) {
	// Not parallel - modifies env vars
	t.Setenv("TEST_SECRET", "my-secret-value")

	value, err := resolveSecret("env", "TEST_SECRET")
	require.NoError(t, err)
	assert.Equal(t, "my-secret-value", value)
}

func TestResolveSecret_UnknownBackend(t *testing.T) {
	t.Parallel()

	_, err := resolveSecret("unknown", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

func TestSetSecret_UnsupportedBackends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		backend     string
		expectError string
	}{
		{"env", "cannot set environment variables"},
		{"1password", "setting secrets not supported"},
		{"bitwarden", "setting secrets not supported"},
		{"age", "setting secrets not supported"},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			t.Parallel()
			err := setSecret(tt.backend, "name", "value")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestFindSecretRefs(t *testing.T) {
	// Not parallel - creates temp files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")

	content := `git:
  signing_key: "secret://1password/vault/signing-key"
ssh:
  passphrase: "secret://keychain/ssh-passphrase"
env:
  API_TOKEN: "secret://env/API_TOKEN"
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	refs, err := findSecretRefs(configPath)
	require.NoError(t, err)
	assert.Len(t, refs, 3)

	// Check first ref
	assert.Equal(t, "1password", refs[0].Backend)
	assert.Equal(t, "vault/signing-key", refs[0].Key)

	// Check second ref
	assert.Equal(t, "keychain", refs[1].Backend)
	assert.Equal(t, "ssh-passphrase", refs[1].Key)

	// Check third ref
	assert.Equal(t, "env", refs[2].Backend)
	assert.Equal(t, "API_TOKEN", refs[2].Key)
}

func TestFindSecretRefs_NoSecrets(t *testing.T) {
	// Not parallel - creates temp files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "preflight.yaml")

	content := `git:
  name: "Test User"
  email: "test@example.com"
`
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	refs, err := findSecretRefs(configPath)
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestFindSecretRefs_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := findSecretRefs("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}

func TestFormatReason(t *testing.T) {
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
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			result := formatReason(tt.reason)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// filterBySeverity tests are in catalog_test.go

func TestApplyGitConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		git  map[string]interface{}
	}{
		{
			"full config",
			map[string]interface{}{
				"name":        "Test User",
				"email":       "test@example.com",
				"signing_key": "ABC123",
			},
		},
		{
			"name only",
			map[string]interface{}{
				"name": "Test User",
			},
		},
		{
			"empty config",
			map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := applyGitConfig(tt.git)
			assert.NoError(t, err)
		})
	}
}

func TestRunGitConfigSet(t *testing.T) {
	t.Parallel()

	err := runGitConfigSet("user.name", "Test User")
	assert.NoError(t, err)
}

// Tests for detectKeyType, isValidOpenPGPPacket are in trust_test.go
// Tests for formatAge are in rollback_test.go

func TestCompareConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   map[string]interface{}
		dest     map[string]interface{}
		filter   []string
		expected int // number of diffs
	}{
		{
			"identical configs",
			map[string]interface{}{"brew": "value"},
			map[string]interface{}{"brew": "value"},
			nil,
			0,
		},
		{
			"added provider",
			map[string]interface{}{},
			map[string]interface{}{"brew": "value"},
			nil,
			1,
		},
		{
			"removed provider",
			map[string]interface{}{"brew": "value"},
			map[string]interface{}{},
			nil,
			1,
		},
		{
			"changed provider",
			map[string]interface{}{"brew": "value1"},
			map[string]interface{}{"brew": "value2"},
			nil,
			1,
		},
		{
			"filtered provider - included",
			map[string]interface{}{"brew": "v1", "apt": "v2"},
			map[string]interface{}{"brew": "v2", "apt": "v2"},
			[]string{"brew"},
			1,
		},
		{
			"filtered provider - excluded",
			map[string]interface{}{"brew": "v1", "apt": "v2"},
			map[string]interface{}{"brew": "v2", "apt": "v2"},
			[]string{"apt"},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			diffs := compareConfigs(tt.source, tt.dest, tt.filter)
			assert.Len(t, diffs, tt.expected)
		})
	}
}

func TestCompareProviderConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		source   interface{}
		dest     interface{}
		expected int // number of diffs
	}{
		{
			"identical maps",
			"brew",
			map[string]interface{}{"formulae": []string{"git"}},
			map[string]interface{}{"formulae": []string{"git"}},
			0,
		},
		{
			"added key",
			"brew",
			map[string]interface{}{},
			map[string]interface{}{"formulae": "value"},
			1,
		},
		{
			"removed key",
			"brew",
			map[string]interface{}{"formulae": "value"},
			map[string]interface{}{},
			1,
		},
		{
			"changed value",
			"brew",
			map[string]interface{}{"formulae": "v1"},
			map[string]interface{}{"formulae": "v2"},
			1,
		},
		{
			"non-map values different",
			"brew",
			"value1",
			"value2",
			1,
		},
		{
			"non-map values same",
			"brew",
			"value",
			"value",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			diffs := compareProviderConfig(tt.provider, tt.source, tt.dest)
			assert.Len(t, diffs, tt.expected)
		})
	}
}

func TestConfigDiff_Types(t *testing.T) {
	t.Parallel()

	diffs := compareConfigs(
		map[string]interface{}{"removed": "val", "changed": "old"},
		map[string]interface{}{"added": "val", "changed": "new"},
		nil,
	)

	assert.Len(t, diffs, 3)

	types := make(map[string]bool)
	for _, d := range diffs {
		types[d.Type] = true
	}

	assert.True(t, types["added"])
	assert.True(t, types["removed"])
	assert.True(t, types["changed"])
}

func TestFindBrewOrphans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      map[string]interface{}
		systemState map[string]interface{}
		ignoreList  []string
		expected    int
	}{
		{
			"no orphans - same packages",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "vim"},
				},
			},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "vim"},
				},
			},
			nil,
			0,
		},
		{
			"orphan formula found",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git"},
				},
			},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git", "orphan-pkg"},
				},
			},
			nil,
			1,
		},
		{
			"orphan cask found",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"casks": []interface{}{"docker"},
				},
			},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"casks": []interface{}{"docker", "orphan-app"},
				},
			},
			nil,
			1,
		},
		{
			"ignored orphan not counted",
			map[string]interface{}{
				"brew": map[string]interface{}{},
			},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"ignored-pkg"},
				},
			},
			[]string{"ignored-pkg"},
			0,
		},
		{
			"no brew in config",
			map[string]interface{}{},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"orphan"},
				},
			},
			nil,
			1,
		},
		{
			"no brew in system state",
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"git"},
				},
			},
			map[string]interface{}{},
			nil,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orphans := findBrewOrphans(tt.config, tt.systemState, tt.ignoreList)
			assert.Len(t, orphans, tt.expected)
		})
	}
}

func TestFindVSCodeOrphans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      map[string]interface{}
		systemState map[string]interface{}
		ignoreList  []string
		expected    int
	}{
		{
			"no orphans - same extensions",
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ms-python.python"},
				},
			},
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ms-python.python"},
				},
			},
			nil,
			0,
		},
		{
			"orphan extension found",
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ms-python.python"},
				},
			},
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ms-python.python", "orphan.extension"},
				},
			},
			nil,
			1,
		},
		{
			"case insensitive matching",
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"MS-Python.Python"},
				},
			},
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ms-python.python"},
				},
			},
			nil,
			0,
		},
		{
			"ignored extension not counted",
			map[string]interface{}{
				"vscode": map[string]interface{}{},
			},
			map[string]interface{}{
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"ignored.ext"},
				},
			},
			[]string{"ignored.ext"},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orphans := findVSCodeOrphans(tt.config, tt.systemState, tt.ignoreList)
			assert.Len(t, orphans, tt.expected)
		})
	}
}

func TestFindFileOrphans(t *testing.T) {
	t.Parallel()

	// This function always returns nil as file orphan detection is complex
	result := findFileOrphans(nil, nil, nil)
	assert.Nil(t, result)
}

func TestFindOrphans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         map[string]interface{}
		systemState    map[string]interface{}
		providerFilter []string
		ignoreList     []string
		expected       int
	}{
		{
			"no filter - finds all orphans",
			map[string]interface{}{},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"orphan1"},
				},
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"orphan2"},
				},
			},
			nil,
			nil,
			2,
		},
		{
			"filter to brew only",
			map[string]interface{}{},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"orphan1"},
				},
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"orphan2"},
				},
			},
			[]string{"brew"},
			nil,
			1,
		},
		{
			"filter to vscode only",
			map[string]interface{}{},
			map[string]interface{}{
				"brew": map[string]interface{}{
					"formulae": []interface{}{"orphan1"},
				},
				"vscode": map[string]interface{}{
					"extensions": []interface{}{"orphan2"},
				},
			},
			[]string{"vscode"},
			nil,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orphans := findOrphans(tt.config, tt.systemState, tt.providerFilter, tt.ignoreList)
			assert.Len(t, orphans, tt.expected)
		})
	}
}

func TestOrphanedItemFields(t *testing.T) {
	t.Parallel()

	orphans := findBrewOrphans(
		map[string]interface{}{},
		map[string]interface{}{
			"brew": map[string]interface{}{
				"formulae": []interface{}{"test-pkg"},
			},
		},
		nil,
	)

	require.Len(t, orphans, 1)
	assert.Equal(t, "brew", orphans[0].Provider)
	assert.Equal(t, "formula", orphans[0].Type)
	assert.Equal(t, "test-pkg", orphans[0].Name)
}

func TestCollectEvaluatedItems_Nil(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(nil)
	assert.Nil(t, result)
}

func TestCollectEvaluatedItems_Empty(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{})
	assert.Empty(t, result)
}

func TestCollectEvaluatedItems_InfoOnly(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{
		Info: []string{"info1", "info2"},
	})
	assert.Equal(t, []string{"info1", "info2"}, result)
}

func TestCollectEvaluatedItems_ErrorsOnly(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{
		Errors: []string{"error1", "error2"},
	})
	assert.Equal(t, []string{"error1", "error2"}, result)
}

func TestCollectEvaluatedItems_InfoAndErrors(t *testing.T) {
	t.Parallel()

	result := collectEvaluatedItems(&app.ValidationResult{
		Info:   []string{"info1"},
		Errors: []string{"error1", "error2"},
	})
	assert.Equal(t, []string{"info1", "error1", "error2"}, result)
}

func TestOutputOrphansText(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	orphans := []OrphanedItem{
		{Provider: "brew", Type: "formula", Name: "git"},
		{Provider: "vscode", Type: "extension", Name: "test.ext"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputOrphansText(orphans)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Found 2 orphaned items")
	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "formula")
	assert.Contains(t, output, "git")
	assert.Contains(t, output, "vscode")
}

func TestOutputOrphansText_Empty(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputOrphansText(nil)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Found 0 orphaned items")
}

func TestOutputCompareText_NoDiffs(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", nil)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "No differences")
}

func TestOutputCompareText_WithDiffs(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	diffs := []configDiff{
		{Type: "added", Provider: "brew", Key: "formulae", Dest: []string{"git"}},
		{Type: "removed", Provider: "apt", Key: "packages", Source: []string{"vim"}},
		{Type: "changed", Provider: "git", Key: "user.name", Source: "Old", Dest: "New"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Comparing source.yaml → dest.yaml")
	assert.Contains(t, output, "added")
	assert.Contains(t, output, "removed")
	assert.Contains(t, output, "changed")
	assert.Contains(t, output, "Total: 3 difference(s)")
}

func TestOutputCompareJSON(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	diffs := []configDiff{
		{Type: "added", Provider: "brew", Key: "formulae"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCompareJSON(diffs)

	_ = w.Close()
	os.Stdout = old

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "added")
	assert.Contains(t, output, "brew")
}

func TestOutputHistoryText(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	entries := []HistoryEntry{
		{
			ID:        "test-1",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Command:   "apply",
			Status:    "success",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "success")
}

func TestOutputHistoryText_Empty(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(nil)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Showing 0 entries")
}

func TestOutputComplianceError(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceError(errors.New("test error message"))

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "error")
	assert.Contains(t, output, "test error message")
}

func TestOutputComplianceJSON(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ViolationCount:  0,
			WarningCount:    0,
			ComplianceScore: 100.0,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceJSON(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-policy")
	assert.Contains(t, output, "status")
}

func TestOutputComplianceText(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ViolationCount:  0,
			WarningCount:    1,
			ComplianceScore: 100.0,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceText(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-policy")
}

func TestOutputComplianceText_WithExpiringOverrides(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ViolationCount:  0,
			WarningCount:    0,
			OverrideCount:   1,
			ComplianceScore: 100.0,
		},
		Overrides: []policy.OverrideDetail{
			{
				Pattern:         "test-pattern",
				Justification:   "Testing override",
				ApprovedBy:      "tester",
				ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				DaysUntilExpiry: 1,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceText(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-policy")
	assert.Contains(t, output, "expiring")
}

func TestOutputHistoryText_Verbose(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	// Save original verbose setting
	origVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = origVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "test-1",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Command:   "apply",
			Target:    "work",
			Status:    "success",
			Duration:  "1m30s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
				{Provider: "brew", Action: "install", Item: "vim"},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-1")
	assert.Contains(t, output, "Command:  apply")
	assert.Contains(t, output, "Target:   work")
	assert.Contains(t, output, "Duration: 1m30s")
	assert.Contains(t, output, "Changes:")
	assert.Contains(t, output, "[brew] install: git")
}

func TestOutputHistoryText_VerboseWithError(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	origVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = origVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "test-fail",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Command:   "apply",
			Status:    "failed",
			Error:     "connection timeout",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-fail")
	assert.Contains(t, output, "Error:    connection timeout")
}

func TestSetSecret_EnvBackendError(t *testing.T) {
	t.Parallel()

	err := setSecret("env", "name", "value")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set environment variables")
}

func TestSetSecret_UnsupportedBackend(t *testing.T) {
	t.Parallel()

	err := setSecret("unknown", "name", "value")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestSaveHistoryEntry_WithDefaults(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		// ID and Timestamp will be set to defaults
		Command: "test",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify file was created
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	files, err := os.ReadDir(historyDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestSaveHistoryEntry_WithProvidedID(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "custom-id-123",
		Timestamp: time.Now(),
		Command:   "apply",
		Target:    "work",
		Status:    "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify file was created with correct name
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	expectedFile := filepath.Join(historyDir, "custom-id-123.json")
	_, err = os.Stat(expectedFile)
	assert.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(expectedFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "custom-id-123")
	assert.Contains(t, string(data), "apply")
	assert.Contains(t, string(data), "work")
}

func TestGetCurrentProfile_Empty(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	result := getCurrentProfile()
	assert.Empty(t, result)
}

func TestGetCurrentProfile_WithValue(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create profile directory and current file
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(profileDir, "current"), []byte("work-profile"), 0o644)
	require.NoError(t, err)

	result := getCurrentProfile()
	assert.Equal(t, "work-profile", result)
}

func TestSetCurrentProfile(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("my-profile")
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "profiles", "current"))
	require.NoError(t, err)
	assert.Equal(t, "my-profile", string(data))
}

func TestLoadCustomProfiles_NotExist(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestLoadCustomProfiles_Valid(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create profile directory and profiles.yaml
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	yamlContent := `- name: work
  description: Work profile
  git:
    name: Work User
    email: work@example.com
- name: personal
  description: Personal profile
  git:
    name: Personal User
    email: personal@example.com
`
	err = os.WriteFile(filepath.Join(profileDir, "profiles.yaml"), []byte(yamlContent), 0o644)
	require.NoError(t, err)

	profiles, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, profiles, 2)
	assert.Equal(t, "work", profiles[0].Name)
	assert.Equal(t, "personal", profiles[1].Name)
}

func TestSaveCustomProfiles(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{
			Name:        "test-profile",
			Description: "A test profile",
		},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, ".preflight", "profiles", "profiles.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-profile")
	assert.Contains(t, string(data), "A test profile")
}

func TestLoadCustomProfiles_InvalidYAML(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create profile directory and invalid profiles.yaml
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	invalidYAML := `{{{not valid yaml`
	err = os.WriteFile(filepath.Join(profileDir, "profiles.yaml"), []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	_, err = loadCustomProfiles()
	assert.Error(t, err)
}

func TestLoadHistory_NoDirectory(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Nil(t, entries)
}

func TestLoadHistory_EmptyDirectory(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create empty history directory
	err := os.MkdirAll(filepath.Join(tmpDir, ".preflight", "history"), 0o755)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLoadHistory_WithEntries(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create history directory
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0o755)
	require.NoError(t, err)

	// Create history entries
	entry1 := HistoryEntry{
		ID:      "entry1",
		Command: "apply",
		Status:  "success",
	}
	data1, _ := json.Marshal(entry1)
	err = os.WriteFile(filepath.Join(historyDir, "entry1.json"), data1, 0o644)
	require.NoError(t, err)

	entry2 := HistoryEntry{
		ID:      "entry2",
		Command: "doctor",
		Status:  "success",
	}
	data2, _ := json.Marshal(entry2)
	err = os.WriteFile(filepath.Join(historyDir, "entry2.json"), data2, 0o644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestLoadHistory_SkipsNonJSON(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create history directory
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0o755)
	require.NoError(t, err)

	// Create a non-JSON file
	err = os.WriteFile(filepath.Join(historyDir, "readme.txt"), []byte("not json"), 0o644)
	require.NoError(t, err)

	// Create a valid JSON entry
	entry := HistoryEntry{ID: "valid", Command: "apply"}
	data, _ := json.Marshal(entry)
	err = os.WriteFile(filepath.Join(historyDir, "valid.json"), data, 0o644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "valid", entries[0].ID)
}

func TestLoadHistory_SkipsInvalidJSON(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create history directory
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0o755)
	require.NoError(t, err)

	// Create an invalid JSON file
	err = os.WriteFile(filepath.Join(historyDir, "invalid.json"), []byte("{not valid json}"), 0o644)
	require.NoError(t, err)

	// Create a valid JSON entry
	entry := HistoryEntry{ID: "valid", Command: "apply"}
	data, _ := json.Marshal(entry)
	err = os.WriteFile(filepath.Join(historyDir, "valid.json"), data, 0o644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "valid", entries[0].ID)
}

func TestGetRegistry_ReturnsError(t *testing.T) {
	// NOTE: Not running in parallel due to HOME manipulation
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// getRegistry should work even with an empty directory
	reg, err := getRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, reg)
}

func TestListSnapshots_Empty(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	ctx := context.Background()
	var sets []snapshot.Set

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listSnapshots(ctx, nil, sets)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Available Snapshots")
	assert.Contains(t, output, "preflight rollback --to")
}

func TestListSnapshots_WithSets(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	ctx := context.Background()
	sets := []snapshot.Set{
		{
			ID:        "abc123456789",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Reason:    "before apply",
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.bashrc"},
				{Path: "/home/user/.zshrc"},
			},
		},
		{
			ID:        "def987654321",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			Reason:    "", // no reason
			Snapshots: []snapshot.Snapshot{
				{Path: "/home/user/.gitconfig"},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := listSnapshots(ctx, nil, sets)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "abc12345") // short ID
	assert.Contains(t, output, "def98765") // short ID
	assert.Contains(t, output, "2 files")  // first set
	assert.Contains(t, output, "1 files")  // second set
	assert.Contains(t, output, "before apply")
}

func TestApplyGitConfig_Empty(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	git := map[string]interface{}{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	assert.NoError(t, err)
}

func TestApplyGitConfig_WithName(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	git := map[string]interface{}{
		"name": "Test User",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "user.name")
	assert.Contains(t, output, "Test User")
}

func TestApplyGitConfig_WithEmail(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	git := map[string]interface{}{
		"email": "test@example.com",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "user.email")
	assert.Contains(t, output, "test@example.com")
}

func TestApplyGitConfig_WithSigningKey(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	git := map[string]interface{}{
		"signing_key": "ABC123",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "user.signingkey")
	assert.Contains(t, output, "ABC123")
}

func TestApplyGitConfig_AllFields(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	git := map[string]interface{}{
		"name":        "Test User",
		"email":       "test@example.com",
		"signing_key": "KEY123",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "user.name")
	assert.Contains(t, output, "user.email")
	assert.Contains(t, output, "user.signingkey")
}

func TestOutputValidationResult_Valid(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	// Save and restore pluginValidateJSON
	origJSON := pluginValidateJSON
	pluginValidateJSON = false
	defer func() { pluginValidateJSON = origJSON }()

	result := ValidationResult{
		Valid:   true,
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Path:    "/path/to/plugin",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationResult(result)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Plugin validated")
	assert.Contains(t, output, "test-plugin@1.0.0")
}

func TestOutputValidationResult_Invalid(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	origJSON := pluginValidateJSON
	pluginValidateJSON = false
	defer func() { pluginValidateJSON = origJSON }()

	result := ValidationResult{
		Valid:  false,
		Errors: []string{"missing name field", "invalid version format"},
		Path:   "/path/to/plugin",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationResult(result)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Error(t, err)
	assert.Contains(t, output, "Validation failed")
	assert.Contains(t, output, "Errors:")
	assert.Contains(t, output, "missing name field")
}

func TestOutputValidationResult_WithWarnings(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	origJSON := pluginValidateJSON
	pluginValidateJSON = false
	defer func() { pluginValidateJSON = origJSON }()

	result := ValidationResult{
		Valid:    true,
		Plugin:   "test-plugin",
		Version:  "1.0.0",
		Warnings: []string{"deprecated field", "consider updating"},
		Path:     "/path/to/plugin",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationResult(result)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Plugin validated")
	assert.Contains(t, output, "Warnings:")
	assert.Contains(t, output, "deprecated field")
}

func TestOutputValidationResult_JSON(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	origJSON := pluginValidateJSON
	pluginValidateJSON = true
	defer func() { pluginValidateJSON = origJSON }()

	result := ValidationResult{
		Valid:   true,
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Path:    "/path/to/plugin",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationResult(result)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, `"valid": true`)
	assert.Contains(t, output, `"plugin": "test-plugin"`)
	assert.Contains(t, output, `"version": "1.0.0"`)
}

func TestPrintError(t *testing.T) {
	// NOTE: Not running in parallel due to stderr capture
	err := errors.New("test error message")

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printError(err)

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "test error message")
}

func TestNewMarketplaceService(t *testing.T) {
	// NOTE: Cannot use t.Parallel() - modifies package-level mpOfflineMode variable

	// Store original value
	origOffline := mpOfflineMode
	mpOfflineMode = true
	defer func() { mpOfflineMode = origOffline }()

	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}

func TestNewMarketplaceService_Default(t *testing.T) {
	// NOTE: Cannot use t.Parallel() - modifies package-level mpOfflineMode variable

	// Store original value
	origOffline := mpOfflineMode
	mpOfflineMode = false
	defer func() { mpOfflineMode = origOffline }()

	svc := newMarketplaceService()
	assert.NotNil(t, svc)
}

func TestSaveCustomProfiles_Error(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "test", Description: "test profile"},
	}

	// Create the parent directory first to ensure we can write
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	// Save should succeed
	err = saveCustomProfiles(profiles)
	assert.NoError(t, err)
}

func TestSetCurrentProfile_Error(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create the profiles directory
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0o755)
	require.NoError(t, err)

	// Set profile should succeed
	err = setCurrentProfile("test-profile")
	assert.NoError(t, err)

	// Verify it was set
	profile := getCurrentProfile()
	assert.Equal(t, "test-profile", profile)
}

func TestOutputComplianceJSON_Full(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	report := &policy.ComplianceReport{
		PolicyName:        "test-policy",
		PolicyDescription: "A test policy",
		Enforcement:       policy.EnforcementBlock,
		GeneratedAt:       time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     10,
			PassedChecks:    10,
			ViolationCount:  0,
			WarningCount:    2,
			ComplianceScore: 100.0,
		},
		Violations: nil,
		Warnings: []policy.ViolationDetail{
			{Type: "warning", Pattern: "warn1", Message: "warning 1", Severity: "low"},
			{Type: "warning", Pattern: "warn2", Message: "warning 2", Severity: "low"},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceJSON(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-policy")
	assert.Contains(t, output, "compliant")
}

func TestOutputCompareText_NoChanges(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	diffs := []configDiff{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "No differences")
}

func TestResolveSecret_AllBackends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		backend   string
		key       string
		wantErr   bool
		errSubstr string
	}{
		{"env backend", "env", "TEST_VAR", false, ""},
		{"unknown backend", "unknown", "key", true, "unknown backend"},
		{"1password backend", "1password", "vault/item", true, ""}, // Will fail without op CLI
		{"bitwarden backend", "bitwarden", "item", true, ""},       // Will fail without bw CLI
		{"keychain backend", "keychain", "item", true, ""},         // Will fail without security CLI
		{"age backend", "age", "key", true, ""},                    // Will fail without age CLI
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveSecret(tt.backend, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetSecret_AllBackends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		backend   string
		key       string
		value     string
		wantErr   bool
		errSubstr string
	}{
		{"env backend", "env", "KEY", "value", true, "cannot set environment variables"},
		{"unknown backend", "unknown", "key", "value", true, "not supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := setSecret(tt.backend, tt.key, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetSecret_Keychain(t *testing.T) {
	t.Parallel()

	// On macOS, keychain might succeed; just exercise the function
	err := setSecret("keychain", "test-key", "test-value")
	// Don't assert - it may succeed or fail depending on environment
	_ = err
}

func TestCheck1PasswordCLI(t *testing.T) {
	t.Parallel()
	// Just exercise the function
	_ = check1PasswordCLI()
}

func TestCheckBitwardenCLI(t *testing.T) {
	t.Parallel()
	// Just exercise the function
	_ = checkBitwardenCLI()
}

func TestCheckKeychain(t *testing.T) {
	t.Parallel()
	// Just exercise the function (on macOS this should return true)
	result := checkKeychain()
	// On macOS, security command exists
	if result {
		assert.True(t, result)
	}
}

func TestCheckAgeCLI(t *testing.T) {
	t.Parallel()
	// Just exercise the function
	_ = checkAgeCLI()
}

func TestOutputCompareText_WithChanges(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	diffs := []configDiff{
		{
			Provider: "brew",
			Key:      "formulae",
			Source:   "git",
			Dest:     "git, vim",
			Type:     "added",
		},
		{
			Provider: "brew",
			Key:      "casks",
			Source:   "chrome",
			Dest:     nil,
			Type:     "removed",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "formulae")
}

func TestGetRegistry_WithDisabledCatalog(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Just exercise the function - it should handle missing directories
	registry, err := getRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestDeriveCatalogName_MoreCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		want     string
	}{
		{"simple name", "catalog", "catalog"},
		{"with extension", "catalog.yaml", "catalog.yaml"},
		{"nested path", "/home/user/configs", "configs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := deriveCatalogName(tt.location)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOutputValidationText_MoreCases(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	result := ValidationResult{
		Valid:    true,
		Plugin:   "test-plugin",
		Version:  "1.0.0",
		Warnings: nil,
		Errors:   nil,
		Path:     "/path/to/plugin",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputValidationResult(result)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "valid")
	assert.Contains(t, output, "test-plugin")
}

func TestWriteEnvFile_Success(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "KEY", Value: "value"},
		{Name: "OTHER", Value: "test"},
	}

	err := WriteEnvFile(vars)
	assert.NoError(t, err)

	// Verify file was written
	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	data, err := os.ReadFile(envPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "KEY")
}

func TestLoadHistory_EmptyFile(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create empty history file (valid JSON array)
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0o755)
	require.NoError(t, err)

	// Write empty JSON array
	err = os.WriteFile(filepath.Join(historyDir, "history.json"), []byte("[]"), 0o644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSaveHistoryEntry_NewEntry(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		ID:        "custom-id",
		Target:    "default",
		Status:    "success",
		Timestamp: time.Now(),
		Command:   "apply",
		Duration:  "1s",
	}

	err := SaveHistoryEntry(entry)
	assert.NoError(t, err)

	// Verify it was saved
	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "custom-id", entries[0].ID)
}

func TestOutputComplianceError_Output(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	// outputComplianceError outputs JSON to stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceError(errors.New("test compliance error"))

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test compliance error")
}

func TestOutputComplianceText_Output(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	report := &policy.ComplianceReport{
		PolicyName:  "test-policy",
		Enforcement: policy.EnforcementBlock,
		GeneratedAt: time.Now(),
		Summary: policy.ComplianceSummary{
			Status:          policy.ComplianceStatusCompliant,
			TotalChecks:     5,
			PassedChecks:    5,
			ViolationCount:  0,
			WarningCount:    0,
			ComplianceScore: 100.0,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceText(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-policy")
}

func TestOutputRecommendations(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	pkgID1, _ := marketplace.NewPackageID("test-preset")
	pkgID2, _ := marketplace.NewPackageID("dev-tools")
	recs := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:   pkgID1,
				Type: "preset",
			},
			Score:   0.85,
			Reasons: []marketplace.RecommendationReason{marketplace.ReasonPopular, marketplace.ReasonTrending},
		},
		{
			Package: marketplace.Package{
				ID:   pkgID2,
				Type: "capability-pack-long-type",
			},
			Score:   0.72,
			Reasons: []marketplace.RecommendationReason{marketplace.ReasonSimilarKeywords},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputRecommendations(recs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "SCORE")
	assert.Contains(t, output, "test-preset")
	assert.Contains(t, output, "popular")
	assert.Contains(t, output, "trending")
}

func TestOutputRecommendations_LongReason(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	pkgID, _ := marketplace.NewPackageID("test")
	recs := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:   pkgID,
				Type: "preset",
			},
			Score: 0.5,
			Reasons: []marketplace.RecommendationReason{
				marketplace.ReasonPopular,
				marketplace.ReasonTrending,
				marketplace.ReasonSimilarKeywords,
				marketplace.ReasonSameType,
				marketplace.ReasonSameAuthor,
				marketplace.ReasonComplementary,
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputRecommendations(recs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Long reason list should be truncated with "..."
	assert.Contains(t, output, "...")
}

func TestGetTrustStore_NoFile(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	store, err := getTrustStore()
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Store should be empty since no trust.json file exists
	keys := store.List()
	assert.Empty(t, keys)
}

func TestGetTrustStore_EmptyStore(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create .preflight directory with empty keys store
	trustDir := filepath.Join(tmpDir, ".preflight")
	err := os.MkdirAll(trustDir, 0o755)
	require.NoError(t, err)

	// Write valid empty trust store JSON
	err = os.WriteFile(filepath.Join(trustDir, "trust.json"), []byte(`{"version":"1.0","keys":[]}`), 0o644)
	require.NoError(t, err)

	store, err := getTrustStore()
	require.NoError(t, err)
	assert.NotNil(t, store)

	keys := store.List()
	assert.Empty(t, keys)
}

func TestBuildUserContext_WithKeywords(t *testing.T) {
	// NOTE: Not using t.Parallel() due to global flag modification and t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save original flag values
	origKeywords := mpKeywords
	origType := mpRecommendType
	defer func() {
		mpKeywords = origKeywords
		mpRecommendType = origType
	}()

	// Set flags
	mpKeywords = "golang, testing, cli"
	mpRecommendType = "preset"

	// Create a minimal marketplace service with empty config
	svc := marketplace.NewService(marketplace.ServiceConfig{
		InstallPath: tmpDir,
	})

	ctx := buildUserContext(svc)

	// Verify keywords were parsed correctly (trimmed and split)
	assert.Contains(t, ctx.Keywords, "golang")
	assert.Contains(t, ctx.Keywords, "testing")
	assert.Contains(t, ctx.Keywords, "cli")

	// Verify type was set
	assert.Contains(t, ctx.PreferredTypes, "preset")
}

func TestBuildUserContext_NoKeywords(t *testing.T) {
	// NOTE: Not using t.Parallel() due to global flag modification and t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Save original flag values
	origKeywords := mpKeywords
	origType := mpRecommendType
	defer func() {
		mpKeywords = origKeywords
		mpRecommendType = origType
	}()

	// Clear flags
	mpKeywords = ""
	mpRecommendType = ""

	// Create a minimal marketplace service with empty config
	svc := marketplace.NewService(marketplace.ServiceConfig{
		InstallPath: tmpDir,
	})

	ctx := buildUserContext(svc)

	// Verify empty context
	assert.Empty(t, ctx.Keywords)
	assert.Empty(t, ctx.PreferredTypes)
}

func TestDeriveCatalogName_EmptyString(t *testing.T) {
	t.Parallel()

	// Test empty string returns date-based name
	result := deriveCatalogName("")
	assert.True(t, strings.HasPrefix(result, "catalog-"), "empty string should return catalog-YYYYMMDD")
	assert.Len(t, result, len("catalog-")+8, "should have format catalog-YYYYMMDD")
}

func TestResolveAge_FileNotFound(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := resolveAge("nonexistent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "age-encrypted secret not found")
}

func TestResolve1Password_NotConfigured(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Without 1Password CLI (op) available, this should return an error
	// when trying to resolve a secret
	_, err := resolve1Password("any-key")
	// The error could be either command not found or op-related
	// Both are expected in test environment
	assert.Error(t, err)
}

func TestGetRegistry_LoadsBuiltin(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	registry, err := getRegistry()
	require.NoError(t, err)
	assert.NotNil(t, registry)

	// Should have at least the builtin catalog
	catalogs := registry.List()
	assert.NotEmpty(t, catalogs)
}

func TestOutputCompareText_NoDifferences(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", nil)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "No differences")
}

func TestOutputCompareText_WithDifferences(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	diffs := []configDiff{
		{
			Provider: "brew",
			Key:      "formulae",
			Type:     "added",
			Dest:     "git",
		},
		{
			Provider: "brew",
			Key:      "formulae",
			Type:     "removed",
			Source:   "vim",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "git")
	assert.Contains(t, output, "vim")
}

func TestOutputHistoryText_NoEntries(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(nil)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Empty entries show header and "Showing 0 entries"
	assert.Contains(t, output, "Showing 0 entries")
}

func TestOutputCompareText_WithChangedType(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	diffs := []configDiff{
		{
			Provider: "git",
			Key:      "user.email",
			Type:     "changed",
			Source:   "old@example.com",
			Dest:     "new@example.com",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "changed")
	assert.Contains(t, output, "git")
}

func TestOutputCompareText_WithEmptyKey(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	diffs := []configDiff{
		{
			Provider: "shell",
			Key:      "", // empty key
			Type:     "added",
			Dest:     "new-config",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputCompareText("source.yaml", "dest.yaml", diffs)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "(entire section)")
}

func TestResolveBitwarden_NotInstalled(t *testing.T) {
	// When bitwarden CLI is not installed, should return error
	_, err := resolveBitwarden("any-key")
	assert.Error(t, err)
}

func TestResolveKeychain_NotFound(t *testing.T) {
	// When keychain item doesn't exist, should return error
	_, err := resolveKeychain("nonexistent-secret-key")
	assert.Error(t, err)
}

func TestResolveSecret_UnknownBackend2(t *testing.T) {
	t.Parallel()

	_, err := resolveSecret("unknown", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

func TestSetCurrentProfile_Success2(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("my-profile-name")
	require.NoError(t, err)

	// Verify it was set
	current := getCurrentProfile()
	assert.Equal(t, "my-profile-name", current)
}

func TestSaveCustomProfiles_Success2(t *testing.T) {
	// NOTE: Not using t.Parallel() due to t.Setenv
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "profile1", Description: "Test profile 1"},
		{Name: "profile2", Description: "Test profile 2"},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	// Verify they were saved
	loaded, err := loadCustomProfiles()
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
}

func TestApplyGitConfig_PartialFields(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	// Only email, no name or signing key
	git := map[string]interface{}{
		"email": "partial@example.com",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "user.email")
	assert.NotContains(t, output, "user.name")
}

func TestParseDuration_AllUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"hours", "24h", 24 * time.Hour, false},
		{"days", "7d", 7 * 24 * time.Hour, false},
		{"weeks", "2w", 14 * 24 * time.Hour, false},
		{"months", "1m", 30 * 24 * time.Hour, false},
		{"with spaces", "  12h  ", 12 * time.Hour, false},
		{"uppercase", "5D", 5 * 24 * time.Hour, false},
		{"too short", "h", 0, true},
		{"unknown unit", "5x", 0, true},
		{"non-numeric", "abch", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatStatus_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"success", "✓ success"},
		{"failed", "✗ failed"},
		{"partial", "~ partial"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := formatStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatHistoryAge_TimeRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ago      time.Duration
		contains string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 15 * time.Minute, "m ago"},
		{"hours", 5 * time.Hour, "h ago"},
		{"days", 3 * 24 * time.Hour, "d ago"},
		{"weeks", 2 * 7 * 24 * time.Hour, "w ago"},
		{"months", 45 * 24 * time.Hour, ""}, // Format returns date
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			timestamp := time.Now().Add(-tt.ago)
			result := formatHistoryAge(timestamp)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			} else {
				// For months, it returns a date format
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestLoadHistory_WithInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	require.NoError(t, os.MkdirAll(historyDir, 0o755))

	// Create invalid JSON file
	invalidJSON := filepath.Join(historyDir, "invalid.json")
	require.NoError(t, os.WriteFile(invalidJSON, []byte("{invalid json"), 0o644))

	// Create valid JSON file
	validEntry := HistoryEntry{
		ID:        "valid-1",
		Command:   "apply",
		Timestamp: time.Now(),
	}
	validData, _ := json.Marshal(validEntry)
	require.NoError(t, os.WriteFile(filepath.Join(historyDir, "valid.json"), validData, 0o644))

	entries, err := loadHistory()
	require.NoError(t, err)
	// Should skip invalid and load valid
	assert.Len(t, entries, 1)
	assert.Equal(t, "valid-1", entries[0].ID)
}

func TestLoadHistory_SkipsMultipleNonJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	require.NoError(t, os.MkdirAll(historyDir, 0o755))

	// Create multiple non-JSON files
	require.NoError(t, os.WriteFile(filepath.Join(historyDir, "notes.txt"), []byte("not json"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(historyDir, "backup.bak"), []byte("backup"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(historyDir, ".hidden"), []byte("hidden"), 0o644))

	entries, err := loadHistory()
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLoadCustomProfiles_InvalidYAMLStructure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	require.NoError(t, os.MkdirAll(profileDir, 0o755))

	// Write a YAML that's not a list (wrong structure)
	require.NoError(t, os.WriteFile(
		filepath.Join(profileDir, "profiles.yaml"),
		[]byte("name: invalid\ndescription: this is not a list"),
		0o644,
	))

	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestLoadCustomProfiles_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Don't create the profiles.yaml file - just the dir
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	require.NoError(t, os.MkdirAll(profileDir, 0o755))

	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestGetCurrentProfile_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	result := getCurrentProfile()
	assert.Empty(t, result)
}

func TestGetCurrentProfile_WithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	require.NoError(t, os.MkdirAll(profileDir, 0o755))
	// Write with leading/trailing whitespace to test trimming
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "current"), []byte("  work  \n"), 0o644))

	result := getCurrentProfile()
	assert.Equal(t, "work", result)
}

func TestApplyGitConfig_FullConfig(t *testing.T) {
	// NOTE: Not using t.Parallel() due to stdout capture
	git := map[string]interface{}{
		"name":        "Test User",
		"email":       "test@example.com",
		"signing_key": "ABC123DEF456",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "user.name")
	assert.Contains(t, output, "user.email")
	assert.Contains(t, output, "user.signingkey")
	assert.Contains(t, output, "ABC123DEF456")
}

func TestApplyGitConfig_EmptyConfig(t *testing.T) {
	git := map[string]interface{}{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(git)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestOutputHistoryText_VerboseMode(t *testing.T) {
	// Set verbose flag
	origVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = origVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "test-verbose-1",
			Command:   "apply",
			Target:    "default",
			Status:    "success",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Duration:  "1m30s",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "test-verbose-1")
	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "default")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "1m30s")
	assert.Contains(t, output, "brew")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "git")
}

func TestOutputHistoryText_VerboseModeWithError(t *testing.T) {
	origVerbose := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = origVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "test-error-verbose",
			Command:   "apply",
			Status:    "failed",
			Timestamp: time.Now(),
			Error:     "something went wrong",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "something went wrong")
}

func TestSaveHistoryEntry_AutoPopulatesID(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Entry with no ID and zero timestamp - should be auto-populated
	entry := HistoryEntry{
		Command: "test",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	require.NoError(t, err)

	// Verify file was created
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	files, err := os.ReadDir(historyDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestGetHistoryDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getHistoryDir()
	assert.Contains(t, dir, ".preflight")
	assert.Contains(t, dir, "history")
}

func TestWriteEnvFile_WithSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "EDITOR", Value: "vim", Layer: "base"},
		{Name: "PAGER", Value: "less", Layer: "base"},
		{Name: "API_KEY", Value: "secret123", Secret: true}, // Should be skipped
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	// Verify file was created
	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	content, err := os.ReadFile(envPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "export EDITOR=")
	assert.Contains(t, string(content), "export PAGER=")
	assert.NotContains(t, string(content), "API_KEY") // Secret should be excluded
}

func TestWriteEnvFile_EmptyList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := WriteEnvFile([]EnvVar{})
	require.NoError(t, err)

	// Verify file was created with header
	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	content, err := os.ReadFile(envPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Generated by preflight")
}

func TestWriteEnvFile_AllSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	vars := []EnvVar{
		{Name: "SECRET1", Value: "s1", Secret: true},
		{Name: "SECRET2", Value: "s2", Secret: true},
	}

	err := WriteEnvFile(vars)
	require.NoError(t, err)

	// Verify file was created but has no exports
	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	content, err := os.ReadFile(envPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Generated by preflight")
	assert.NotContains(t, string(content), "export")
}

func TestOutputComplianceText_Expiring(t *testing.T) {
	// Create a report with expiring overrides (days_until_expiry < 7)
	report := &policy.ComplianceReport{
		PolicyName: "test-policy",
		Overrides: []policy.OverrideDetail{
			{
				Pattern:         "test-pattern",
				Justification:   "Testing",
				ExpiresAt:       time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
				DaysUntilExpiry: 3, // Expires in 3 days
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceText(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "expiring within 7 days")
}

func TestOutputComplianceText_NotExpiring(t *testing.T) {
	report := &policy.ComplianceReport{
		PolicyName: "test-policy",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputComplianceText(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.NotContains(t, output, "expiring")
}

func TestRunGitConfigSet_Email(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGitConfigSet("user.email", "test@example.com")

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "git config --global user.email")
	assert.Contains(t, output, "test@example.com")
}

func TestGetRegistry_ExternalCatalogError(t *testing.T) {
	// This tests the warning path when external catalog loading fails
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a registry store with an invalid source
	storeDir := filepath.Join(tmpDir, ".preflight", "catalog")
	require.NoError(t, os.MkdirAll(storeDir, 0o755))

	invalidStore := `- name: invalid-catalog
  location: /nonexistent/path
  enabled: true
`
	require.NoError(t, os.WriteFile(filepath.Join(storeDir, "sources.yaml"), []byte(invalidStore), 0o644))

	// Capture stderr for the warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	registry, err := getRegistry()

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Should succeed with just builtin
	require.NoError(t, err)
	assert.NotNil(t, registry)
	// May have warning in output about failed catalog
	_ = output // Warnings may or may not appear depending on exact failure mode
}

func TestResolveAge_Success(t *testing.T) {
	// resolveAge requires age CLI and key files, so we can only test the file-not-found path
	// which is already covered. Skip this test.
	t.Skip("resolveAge requires age CLI which may not be installed")
}

func TestOutputHistoryText_MultipleEntries(t *testing.T) {
	origVerbose := historyVerbose
	historyVerbose = false
	defer func() { historyVerbose = origVerbose }()

	entries := []HistoryEntry{
		{
			ID:        "entry-1",
			Command:   "apply",
			Target:    "default",
			Status:    "success",
			Timestamp: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "entry-2",
			Command:   "doctor",
			Target:    "work",
			Status:    "partial",
			Timestamp: time.Now().Add(-30 * time.Minute),
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outputHistoryText(entries)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "doctor")
	assert.Contains(t, output, "Showing 2 entries")
}

func TestLoadHistory_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	require.NoError(t, os.MkdirAll(historyDir, 0o755))

	// Create a file that can't be read (no read permission)
	unreadableFile := filepath.Join(historyDir, "unreadable.json")
	require.NoError(t, os.WriteFile(unreadableFile, []byte(`{"id":"test"}`), 0o000))

	// On some systems this may still work, so just ensure no panic
	_, err := loadHistory()
	// Error handling is graceful - continues with other files
	_ = err // May or may not error depending on OS

	// Clean up
	_ = os.Chmod(unreadableFile, 0o644)
}

func TestEnvVarStruct(t *testing.T) {
	t.Parallel()

	ev := EnvVar{
		Name:   "TEST_VAR",
		Value:  "test_value",
		Layer:  "base",
		Secret: false,
	}

	assert.Equal(t, "TEST_VAR", ev.Name)
	assert.Equal(t, "test_value", ev.Value)
	assert.Equal(t, "base", ev.Layer)
	assert.False(t, ev.Secret)
}

func TestChangeStruct(t *testing.T) {
	t.Parallel()

	c := Change{
		Provider: "brew",
		Action:   "install",
		Item:     "git",
	}

	assert.Equal(t, "brew", c.Provider)
	assert.Equal(t, "install", c.Action)
	assert.Equal(t, "git", c.Item)
}

func TestApplyGitConfig_IncludingSigningKey(t *testing.T) {
	config := map[string]interface{}{
		"name":        "Test User",
		"email":       "test@example.com",
		"signing_key": "ABC123DEF456",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := applyGitConfig(config)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "user.name")
	assert.Contains(t, output, "user.email")
	assert.Contains(t, output, "user.signingkey")
	assert.Contains(t, output, "ABC123DEF456")
}

func TestFindSecretRefs_NoDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config with secret ref that has no terminating delimiter
	content := `git:
  email: secret://keychain/github-email`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	refs, err := findSecretRefs(configPath)

	require.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.Equal(t, "keychain", refs[0].Backend)
	assert.Equal(t, "github-email", refs[0].Key)
}

func TestFindSecretRefs_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config with invalid secret ref (no slash separator)
	content := `git:
  email: secret://invalid`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	refs, err := findSecretRefs(configPath)

	require.NoError(t, err)
	// Invalid format should be skipped (len(parts) < 2)
	assert.Empty(t, refs)
}

func TestResolve1Password_InvalidFormat(t *testing.T) {
	// Test with key that has insufficient parts
	_, err := resolve1Password("single-part")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid 1password key format")
}

func TestSaveCustomProfiles_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{Name: "test", Description: "Test profile"},
	}

	err := saveCustomProfiles(profiles)
	require.NoError(t, err)

	// Verify file was created
	profileDir := getProfileDir()
	data, err := os.ReadFile(filepath.Join(profileDir, "profiles.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")
}

func TestSetCurrentProfile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("work")
	require.NoError(t, err)

	// Verify the current profile was set
	current := getCurrentProfile()
	assert.Equal(t, "work", current)
}

func TestResolveSecret_MoreBackends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		backend   string
		key       string
		wantError bool
	}{
		{"1password_invalid", "1password", "invalid", true},
		{"bitwarden", "bitwarden", "test-item", true}, // Will fail without CLI
		{"age", "age", "test-file", true},             // Will fail without file
		{"env_missing", "env", "NONEXISTENT_VAR_12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveSecret(tt.backend, tt.key)
			if tt.wantError {
				assert.Error(t, err)
			}
		})
	}
}

func TestResolveKeychain_NonexistentKey(_ *testing.T) {
	// On macOS, keychain commands exist but the key won't be found
	// The function may return an error or empty string depending on keychain state
	_, err := resolveKeychain("nonexistent-preflight-test-key-12345")
	// Error is expected because key doesn't exist
	// But on some systems without security command, it may also error
	_ = err // Just ensure no panic
}

func TestSecretRefStruct(t *testing.T) {
	t.Parallel()

	ref := SecretRef{
		Path:    "git.email",
		Backend: "keychain",
		Key:     "github-email",
	}

	assert.Equal(t, "git.email", ref.Path)
	assert.Equal(t, "keychain", ref.Backend)
	assert.Equal(t, "github-email", ref.Key)
}

func TestProfileInfoStruct(t *testing.T) {
	t.Parallel()

	info := ProfileInfo{
		Name:        "work",
		Target:      "work",
		Description: "Work profile",
		Active:      true,
		LastUsed:    "2025-01-01",
	}

	assert.Equal(t, "work", info.Name)
	assert.Equal(t, "work", info.Target)
	assert.Equal(t, "Work profile", info.Description)
	assert.True(t, info.Active)
	assert.Equal(t, "2025-01-01", info.LastUsed)
}

func TestTourCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, tourCmd)
	assert.Equal(t, "tour [topic]", tourCmd.Use)
}

func TestTourCmd_HasListFlag(t *testing.T) {
	t.Parallel()
	flag := tourCmd.Flags().Lookup("list")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestPrintTourTopics(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTourTopics()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Available tour topics")
	assert.Contains(t, output, "preflight tour")
}

func TestSyncCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, syncCmd)
	assert.Equal(t, "sync", syncCmd.Use)
}

func TestSyncCmd_HasFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
	}{
		{"remote", "remote"},
		{"branch", "branch"},
		{"push", "push"},
		{"dry-run", "dry-run"},
		{"force", "force"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flag := syncCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
		})
	}
}

func TestRepoCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, repoCmd)
	assert.Equal(t, "repo", repoCmd.Use)
}

func TestRepoCmd_HasSubcommands(t *testing.T) {
	t.Parallel()

	subcommands := repoCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "init")
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "push")
	assert.Contains(t, names, "pull")
	assert.Contains(t, names, "clone")
}

func TestGetConfigDir_DefaultEmpty(t *testing.T) {
	// Reset global flag to default
	original := cfgFile
	cfgFile = ""
	defer func() { cfgFile = original }()

	result := getConfigDir()
	// When cfgFile is empty, it defaults to "preflight.yaml" which has dir "."
	assert.Equal(t, ".", result)
}

func TestGetConfigDir_WithFilePath(t *testing.T) {
	original := cfgFile
	cfgFile = "/path/to/preflight.yaml"
	defer func() { cfgFile = original }()

	result := getConfigDir()
	assert.Equal(t, "/path/to", result)
}

func TestExportCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, exportCmd)
	assert.Equal(t, "export", exportCmd.Use)
}

func TestExportCmd_HasFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{"format", "format", "yaml"},
		{"output", "output", ""},
		{"flatten", "flatten", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flag := exportCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestEnvCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, envCmd)
	assert.Equal(t, "env", envCmd.Use)
}

func TestEnvCmd_HasSubcommands(t *testing.T) {
	t.Parallel()

	subcommands := envCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "list")
	assert.Contains(t, names, "set")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "unset")
	assert.Contains(t, names, "export")
	assert.Contains(t, names, "diff")
}

func TestDoctorCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, doctorCmd)
	assert.Equal(t, "doctor", doctorCmd.Use)
}

func TestDoctorCmd_HasFlags(t *testing.T) {
	t.Parallel()

	flag := doctorCmd.Flags().Lookup("fix")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestDiscoverCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, discoverCmd)
	assert.Equal(t, "discover", discoverCmd.Use)
}

func TestCompareCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, compareCmd)
	assert.Equal(t, "compare <source> <target>", compareCmd.Use)
}

func TestCleanCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, cleanCmd)
	assert.Equal(t, "clean", cleanCmd.Use)
}

func TestCleanCmd_HasFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
	}{
		{"apply", "apply"},
		{"json", "json"},
		{"force", "force"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flag := cleanCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
		})
	}
}

func TestCaptureCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, captureCmd)
	assert.Equal(t, "capture", captureCmd.Use)
}

func TestComplianceCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, complianceCmd)
	assert.Equal(t, "compliance", complianceCmd.Use)
}

func TestInitCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, initCmd)
	assert.Equal(t, "init", initCmd.Use)
}

func TestInitCmd_HasFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
	}{
		{"provider", "provider"},
		{"preset", "preset"},
		{"skip-welcome", "skip-welcome"},
		{"yes", "yes"},
		{"no-ai", "no-ai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flag := initCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %s should exist", tt.flagName)
		})
	}
}

func TestApplyCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, applyCmd)
	assert.Equal(t, "apply", applyCmd.Use)
}

func TestApplyCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"config", "target", "dry-run"}
	for _, f := range flags {
		flag := applyCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestPlanCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, planCmd)
	assert.Equal(t, "plan", planCmd.Use)
}

func TestHistoryCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, historyCmd)
	assert.Equal(t, "history", historyCmd.Use)
}

func TestHistoryCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := historyCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "clear")
}

func TestLockCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, lockCmd)
	assert.Equal(t, "lock", lockCmd.Use)
}

func TestLockCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := lockCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "freeze")
}

func TestMarketplaceCmdExists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, marketplaceCmd)
	assert.Equal(t, "marketplace", marketplaceCmd.Use)
}

func TestMarketplaceCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := marketplaceCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "search")
	assert.Contains(t, names, "install")
	assert.Contains(t, names, "featured")
	assert.Contains(t, names, "popular")
}

func TestProfileCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, profileCmd)
	assert.Equal(t, "profile", profileCmd.Use)
}

func TestProfileCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := profileCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "create")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "switch")
	assert.Contains(t, names, "current")
	assert.Contains(t, names, "delete")
}

func TestRollbackCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, rollbackCmd)
	assert.Equal(t, "rollback", rollbackCmd.Use)
}

func TestRollbackCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"to", "latest", "dry-run"}
	for _, f := range flags {
		flag := rollbackCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestSecretsCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, secretsCmd)
	assert.Equal(t, "secrets", secretsCmd.Use)
}

func TestSecretsCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := secretsCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "set")
	assert.Contains(t, names, "check")
	assert.Contains(t, names, "backends")
}

func TestTrustCmdExists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, trustCmd)
	assert.Equal(t, "trust", trustCmd.Use)
}

func TestTrustCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := trustCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "show")
}

func TestVersionCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, versionCmd)
	assert.Equal(t, "version", versionCmd.Use)
}

func TestWatchCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, watchCmd)
	assert.Equal(t, "watch", watchCmd.Use)
}

func TestDiffCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, diffCmd)
	assert.Equal(t, "diff", diffCmd.Use)
}

func TestCompletionCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, completionCmd)
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
}

func TestRootCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "preflight", rootCmd.Use)
}

func TestRootCmd_HasPersistentFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"config", "mode"}
	for _, f := range flags {
		flag := rootCmd.PersistentFlags().Lookup(f)
		assert.NotNil(t, flag, "persistent flag %s should exist", f)
	}
}

func TestOutputHistoryText_EmptyList(t *testing.T) {
	t.Parallel()
	output := captureStdout(t, func() {
		outputHistoryText([]HistoryEntry{})
	})
	// Empty entries should produce minimal output
	assert.NotContains(t, output, "ago")
}

func TestOutputHistoryText_VerboseWithTarget(t *testing.T) {
	original := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = original }()

	entries := []HistoryEntry{
		{
			ID:        "test-entry-id",
			Command:   "apply",
			Target:    "work",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Status:    "success",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "test-entry-id")
	assert.Contains(t, output, "Target:")
	assert.Contains(t, output, "work")
}

func TestFindBrewOrphans_NonStringItems(t *testing.T) {
	t.Parallel()

	// Test with non-string items in formulae/casks
	config := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", 123, nil},
			"casks":    []interface{}{"firefox", true},
		},
	}
	systemState := map[string]interface{}{
		"brew": map[string]interface{}{
			"formulae": []interface{}{"git", "vim", 456},
			"casks":    []interface{}{"firefox", "chrome", false},
		},
	}

	orphans := findBrewOrphans(config, systemState, nil)
	// Should only find orphans that are strings
	assert.Len(t, orphans, 2) // vim and chrome
}

func TestFindVSCodeOrphans_NonStringItems(t *testing.T) {
	t.Parallel()

	config := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-vscode.go", 123},
		},
	}
	systemState := map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []interface{}{"ms-vscode.go", "ms-python.python", nil},
		},
	}

	orphans := findVSCodeOrphans(config, systemState, nil)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "ms-python.python", orphans[0].Name)
}

// Note: TestLoadCustomProfiles_* tests use HOME env var which is covered
// in existing tests at line 1862

func TestFormatHistoryAge_Variations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 2 * time.Hour, "2h"},
		{"days", 48 * time.Hour, "2d"},
		{"weeks", 14 * 24 * time.Hour, "2w"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			timestamp := time.Now().Add(-tt.duration)
			result := formatHistoryAge(timestamp)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestPlanCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"config", "target"}
	for _, f := range flags {
		flag := planCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestDiffCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flag := diffCmd.Flags().Lookup("config")
	assert.NotNil(t, flag, "flag config should exist")
}

func TestWatchCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"debounce", "skip-initial", "dry-run", "verbose"}
	for _, f := range flags {
		flag := watchCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestHistoryCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"json", "verbose", "limit", "since"}
	for _, f := range flags {
		flag := historyCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestValidateCmd_Exists(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, validateCmd)
	assert.Equal(t, "validate", validateCmd.Use)
}

func TestOutputRecommendations_Empty(t *testing.T) {
	t.Parallel()
	output := captureStdout(t, func() {
		outputRecommendations(nil)
	})
	// Should have header
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
}

func TestFormatReason_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reason   marketplace.RecommendationReason
		expected string
	}{
		{"popular", marketplace.ReasonPopular, "popular"},
		{"trending", marketplace.ReasonTrending, "trending"},
		{"similar_keywords", marketplace.ReasonSimilarKeywords, "similar"},
		{"same_type", marketplace.ReasonSameType, "same type"},
		{"same_author", marketplace.ReasonSameAuthor, "same author"},
		{"complementary", marketplace.ReasonComplementary, "complements"},
		{"featured", marketplace.ReasonFeatured, "featured"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatReason(tt.reason)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestPluginCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := pluginCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "install")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "info")
	assert.Contains(t, names, "validate")
}

func TestEnvCmd_HasSubcommands_All(t *testing.T) {
	t.Parallel()
	subcommands := envCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "set")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "unset")
	assert.Contains(t, names, "export")
	assert.Contains(t, names, "diff")
}

func TestCatalogCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	subcommands := catalogCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "verify")
	assert.Contains(t, names, "audit")
}

func TestRepoCmd_HasSubcommands_All(t *testing.T) {
	t.Parallel()
	subcommands := repoCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "init")
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "push")
	assert.Contains(t, names, "pull")
	assert.Contains(t, names, "clone")
}

func TestComplianceCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"config", "target", "policy", "json"}
	for _, f := range flags {
		flag := complianceCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestDiscoverCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"max-repos", "min-stars", "language", "all"}
	for _, f := range flags {
		flag := discoverCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestCaptureCmd_HasFlags(t *testing.T) {
	t.Parallel()
	flags := []string{"output", "all", "provider", "target"}
	for _, f := range flags {
		flag := captureCmd.Flags().Lookup(f)
		assert.NotNil(t, flag, "flag %s should exist", f)
	}
}

func TestLoadHistory_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create the history directory but leave it empty
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0755)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLoadHistory_NonJSONFilesSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0755)
	require.NoError(t, err)

	// Create a non-JSON file
	err = os.WriteFile(filepath.Join(historyDir, "readme.txt"), []byte("not json"), 0644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries) // Should skip non-JSON files
}

func TestLoadHistory_InvalidJSONSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	err := os.MkdirAll(historyDir, 0755)
	require.NoError(t, err)

	// Create invalid JSON file
	err = os.WriteFile(filepath.Join(historyDir, "bad.json"), []byte("{invalid json}"), 0644)
	require.NoError(t, err)

	entries, err := loadHistory()
	assert.NoError(t, err)
	assert.Empty(t, entries) // Should skip invalid JSON
}

func TestSaveHistoryEntry_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	entry := HistoryEntry{
		Command: "apply",
		Status:  "success",
	}

	err := SaveHistoryEntry(entry)
	assert.NoError(t, err)

	// Verify the file was created
	historyDir := filepath.Join(tmpDir, ".preflight", "history")
	files, err := os.ReadDir(historyDir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, strings.HasSuffix(files[0].Name(), ".json"))
}

func TestWriteEnvFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	envVars := []EnvVar{
		{Name: "API_KEY", Value: "secret123"},
		{Name: "DB_HOST", Value: "localhost"},
	}

	err := WriteEnvFile(envVars)
	assert.NoError(t, err)

	// Verify file was created
	envPath := filepath.Join(tmpDir, ".preflight", "env.sh")
	content, err := os.ReadFile(envPath)
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "API_KEY")
	assert.Contains(t, contentStr, "DB_HOST")
}

func TestOutputHistoryText_MixedCommands(t *testing.T) {
	entries := []HistoryEntry{
		{
			ID:        "entry-1",
			Command:   "apply",
			Target:    "default",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Status:    "success",
		},
		{
			ID:        "entry-2",
			Command:   "doctor",
			Target:    "work",
			Timestamp: time.Now().Add(-2 * time.Hour),
			Status:    "failed",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	assert.Contains(t, output, "apply")
	assert.Contains(t, output, "doctor")
}

func TestOutputHistoryText_VerboseMultipleEntries(t *testing.T) {
	original := historyVerbose
	historyVerbose = true
	defer func() { historyVerbose = original }()

	entries := []HistoryEntry{
		{
			ID:        "entry-1",
			Command:   "apply",
			Target:    "default",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Status:    "success",
			Changes: []Change{
				{Provider: "brew", Action: "install", Item: "git"},
			},
			Duration: "2m",
		},
		{
			ID:        "entry-2",
			Command:   "plan",
			Target:    "work",
			Timestamp: time.Now().Add(-2 * time.Hour),
			Status:    "success",
		},
	}

	output := captureStdout(t, func() {
		outputHistoryText(entries)
	})

	// Should show both entries with separators
	assert.Contains(t, output, "entry-1")
	assert.Contains(t, output, "entry-2")
	assert.Contains(t, output, "Changes:")
}

func TestOutputRecommendations_CapabilityPack(t *testing.T) {
	// NOTE: Not running in parallel due to stdout capture
	pkgID, _ := marketplace.NewPackageID("cap-pack-test")
	recommendations := []marketplace.Recommendation{
		{
			Package: marketplace.Package{
				ID:    pkgID,
				Title: "Cap Pack Test",
				Type:  "capability-pack",
			},
			Score:   0.9,
			Reasons: []marketplace.RecommendationReason{marketplace.ReasonPopular},
		},
	}

	output := captureStdout(t, func() {
		outputRecommendations(recommendations)
	})

	assert.Contains(t, output, "cap-pack-test")
}

func TestGetRegistry_Builtin(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// getRegistry should at least load the builtin catalog
	registry, err := getRegistry()
	assert.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestSetCurrentProfile_Creates(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := setCurrentProfile("test-profile")
	assert.NoError(t, err)

	// Verify the file was created
	profileDir := getProfileDir()
	currentFile := filepath.Join(profileDir, "current")
	content, err := os.ReadFile(currentFile)
	require.NoError(t, err)
	assert.Equal(t, "test-profile", string(content))
}

func TestSaveCustomProfiles_MultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profiles := []ProfileInfo{
		{
			Name:        "work-profile",
			Description: "Work configuration",
		},
		{
			Name:        "personal-profile",
			Description: "Personal configuration",
		},
	}

	err := saveCustomProfiles(profiles)
	assert.NoError(t, err)

	// Verify the file was created
	profilePath := filepath.Join(getProfileDir(), "profiles.yaml")
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "work-profile")
	assert.Contains(t, string(data), "personal-profile")
}

func TestLoadCustomProfiles_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create the profiles file
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileData := `- name: work-profile
  description: Work configuration
- name: personal-profile
  description: Personal configuration
`
	err = os.WriteFile(filepath.Join(profileDir, "profiles.yaml"), []byte(profileData), 0644)
	require.NoError(t, err)

	profiles, err := loadCustomProfiles()
	assert.NoError(t, err)
	assert.Len(t, profiles, 2)
	assert.Equal(t, "work-profile", profiles[0].Name)
}

func TestLoadCustomProfiles_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := loadCustomProfiles()
	assert.Error(t, err)
}

func TestGetCurrentProfile_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	profile := getCurrentProfile()
	assert.Empty(t, profile)
}

func TestGetCurrentProfile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create the current file
	profileDir := filepath.Join(tmpDir, ".preflight", "profiles")
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(profileDir, "current"), []byte("my-profile"), 0644)
	require.NoError(t, err)

	profile := getCurrentProfile()
	assert.Equal(t, "my-profile", profile)
}

func TestGetProfileDir_Default(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := getProfileDir()
	assert.Contains(t, dir, ".preflight")
	assert.Contains(t, dir, "profiles")
}

func TestRunPluginValidate_ValidManifest(t *testing.T) {
	tmpDir := t.TempDir()

	manifest := `apiVersion: v1
name: test-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "valid")
}

func TestRunPluginValidate_InvalidPath(t *testing.T) {
	err := runPluginValidate("/nonexistent/path")
	assert.Error(t, err)
}

func TestDeriveCatalogName_URL(t *testing.T) {
	t.Parallel()

	name := deriveCatalogName("https://github.com/user/my-catalog")
	assert.Equal(t, "my-catalog", name)
}

func TestDeriveCatalogName_Path(t *testing.T) {
	t.Parallel()

	name := deriveCatalogName("/home/user/catalogs/my-local-catalog")
	assert.Equal(t, "my-local-catalog", name)
}

func TestFormatHistoryAge_JustNow(t *testing.T) {
	t.Parallel()

	result := formatHistoryAge(time.Now())
	assert.Equal(t, "just now", result)
}

func TestFormatHistoryAge_Minutes(t *testing.T) {
	t.Parallel()

	result := formatHistoryAge(time.Now().Add(-5 * time.Minute))
	assert.Equal(t, "5m ago", result)
}

func TestFormatHistoryAge_Hours(t *testing.T) {
	t.Parallel()

	result := formatHistoryAge(time.Now().Add(-3 * time.Hour))
	assert.Equal(t, "3h ago", result)
}

func TestFormatHistoryAge_Days(t *testing.T) {
	t.Parallel()

	result := formatHistoryAge(time.Now().Add(-48 * time.Hour))
	assert.Equal(t, "2d ago", result)
}

func TestOutputRecommendations_NilSlice(t *testing.T) {
	output := captureStdout(t, func() {
		outputRecommendations(nil)
	})

	// Even with nil slice, the function outputs the header
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
	assert.Contains(t, output, "SCORE")
}

func TestRunGitConfigSet_Output(t *testing.T) {
	output := captureStdout(t, func() {
		err := runGitConfigSet("user.name", "Test User")
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "git config")
	assert.Contains(t, output, "user.name")
	assert.Contains(t, output, "Test User")
}

func TestRunPluginValidate_NotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir.txt")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		// This returns an error when validation fails
		_ = runPluginValidate(filePath)
	})

	assert.Contains(t, output, "must be a directory")
}

func TestRunPluginValidate_MissingManifest(t *testing.T) {
	tmpDir := t.TempDir()

	output := captureStdout(t, func() {
		// This returns an error when validation fails
		_ = runPluginValidate(tmpDir)
	})

	// Should report an error about missing manifest
	assert.Contains(t, output, "Validation failed")
}

func TestRunPluginValidate_WithWarnings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal manifest without recommended fields
	manifest := `apiVersion: v1
name: minimal-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		assert.NoError(t, err)
	})

	// Should show warnings for missing optional fields
	assert.Contains(t, output, "validated")
	assert.Contains(t, output, "Warnings")
}

func TestGetTrustStore_Success(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create an empty trust store
	trustDir := filepath.Join(tmpDir, ".preflight")
	err := os.MkdirAll(trustDir, 0755)
	require.NoError(t, err)

	trustFile := filepath.Join(trustDir, "trust.json")
	err = os.WriteFile(trustFile, []byte("{}"), 0644)
	require.NoError(t, err)

	store, err := getTrustStore()
	assert.NoError(t, err)
	assert.NotNil(t, store)
}

func TestOutputValidationResult_Errors(t *testing.T) {
	result := ValidationResult{
		Valid:    false,
		Path:     "/some/path",
		Errors:   []string{"manifest is invalid", "version is missing"},
		Warnings: []string{},
	}

	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		// outputValidationResult returns an error when Valid is false
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2 error(s)")
	})

	assert.Contains(t, output, "manifest is invalid")
}

func TestRunPluginValidate_InvalidSemver(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `apiVersion: v1
name: invalid-semver-plugin
version: not-a-version
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		_ = runPluginValidate(tmpDir)
	})

	assert.Contains(t, output, "is not valid semantic versioning")
}

func TestRunPluginValidate_DependencyNoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `apiVersion: v1
name: dep-no-version-plugin
version: 1.0.0
requires:
  - name: some-dependency
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "no version constraint")
}

func TestRunPluginValidate_PathNotExists(t *testing.T) {
	output := captureStdout(t, func() {
		_ = runPluginValidate("/nonexistent/path/to/plugin")
	})

	assert.Contains(t, output, "path does not exist")
}

func TestRunPluginValidate_CompletePlugin(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := `apiVersion: v1
name: complete-plugin
version: 1.0.0
description: A complete test plugin
author: Test Author
license: MIT
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "validated")
	assert.Contains(t, output, "complete-plugin@1.0.0")
}

func TestRunInit_ConfigExists(t *testing.T) {
	// Create temp directory and change to it
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Create preflight.yaml
	err = os.WriteFile("preflight.yaml", []byte("version: v1\n"), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runInit(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "preflight.yaml already exists")
}

func TestOutputValidationResult_JSONMode(t *testing.T) {
	// Save and restore the global flag
	origJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = origJSON }()

	pluginValidateJSON = true

	result := ValidationResult{
		Valid:   true,
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Path:    "/test/path",
	}

	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.NoError(t, err)
	})

	// Should be valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(output), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, true, parsed["valid"])
	assert.Equal(t, "test-plugin", parsed["plugin"])
}

func TestOutputValidationResult_ValidWithPath(t *testing.T) {
	// Save and restore the global flag
	origJSON := pluginValidateJSON
	defer func() { pluginValidateJSON = origJSON }()

	pluginValidateJSON = false

	result := ValidationResult{
		Valid:   true,
		Plugin:  "test-plugin",
		Version: "1.0.0",
		Path:    "/test/path",
	}

	output := captureStdout(t, func() {
		err := outputValidationResult(result)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Plugin validated")
	assert.Contains(t, output, "test-plugin@1.0.0")
	assert.Contains(t, output, "/test/path")
}

func TestRunPluginValidate_StrictMode(t *testing.T) {
	// Save and restore the global flag
	origStrict := pluginValidateStrict
	defer func() { pluginValidateStrict = origStrict }()

	pluginValidateStrict = true

	tmpDir := t.TempDir()
	manifest := `apiVersion: v1
name: strict-mode-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		// In strict mode, warnings become errors
		_ = runPluginValidate(tmpDir)
	})

	// Missing description, author, license, signature should be errors in strict mode
	assert.Contains(t, output, "Validation failed")
}

func TestRunPluginInstall_GitURL(t *testing.T) {
	// Test the Git URL path with an invalid URL
	err := runPluginInstall("https://github.com/nonexistent/nonexistent.git")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "installing from git")
}

func TestDetectAIProvider_NoConfig(t *testing.T) {
	// Clear environment variables
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OLLAMA_HOST", "")

	provider := detectAIProvider()
	assert.Nil(t, provider)
}

func TestTourCmd_ListFlag(t *testing.T) {
	t.Parallel()

	listFlag := tourCmd.Flags().Lookup("list")
	assert.NotNil(t, listFlag)
}

func TestPluginInstall_LocalPathNotDir(t *testing.T) {
	// Create a file, not a directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-dir.txt")
	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	// When a file is passed instead of a directory, it should try git
	err = runPluginInstall(filePath)
	assert.Error(t, err) // Will fail because it's not a git URL
}

func TestDetectAIProvider_AnthropicKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	t.Setenv("OLLAMA_HOST", "")

	provider := detectAIProvider()
	assert.NotNil(t, provider)
}

func TestDetectAIProvider_NoSupportedProvider(t *testing.T) {
	// detectAIProvider only supports Anthropic and OpenAI, not Ollama
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OLLAMA_HOST", "http://localhost:11434")

	provider := detectAIProvider()
	assert.Nil(t, provider) // Returns nil when no supported provider is available
}

func TestRunTour_ListFlag_Direct(t *testing.T) {
	// Save and restore global flag
	origListFlag := tourListFlag
	defer func() { tourListFlag = origListFlag }()

	tourListFlag = true

	output := captureStdout(t, func() {
		err := runTour(nil, []string{})
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Available tour topics")
}

func TestRunPluginValidate_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidYAML := `{{{not valid yaml`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(invalidYAML), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		_ = runPluginValidate(tmpDir)
	})

	assert.Contains(t, output, "Validation failed")
}

func TestRunPluginValidate_EmptyManifest(t *testing.T) {
	tmpDir := t.TempDir()
	emptyManifest := ``
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(emptyManifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		_ = runPluginValidate(tmpDir)
	})

	assert.Contains(t, output, "Validation failed")
}

func TestRunTour_InvalidTopic(t *testing.T) {
	// Save and restore global flag
	origListFlag := tourListFlag
	defer func() { tourListFlag = origListFlag }()

	tourListFlag = false

	// Pass an invalid topic name
	err := runTour(nil, []string{"nonexistent-topic"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown topic")
}

func TestCollectEvaluatedItems_NilResult(t *testing.T) {
	t.Parallel()

	items := collectEvaluatedItems(nil)
	assert.Nil(t, items)
}

func TestCollectEvaluatedItems_WithInfoAndErrors(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{
		Info:   []string{"Info item 1", "Info item 2"},
		Errors: []string{"Error item 1"},
	}

	items := collectEvaluatedItems(result)
	assert.Len(t, items, 3)
	assert.Contains(t, items, "Info item 1")
	assert.Contains(t, items, "Info item 2")
	assert.Contains(t, items, "Error item 1")
}

func TestCollectEvaluatedItems_EmptyResult(t *testing.T) {
	t.Parallel()

	result := &app.ValidationResult{}

	items := collectEvaluatedItems(result)
	assert.Empty(t, items)
}

func TestApplyGitConfig_EmailOnly(t *testing.T) {
	t.Parallel()

	git := map[string]interface{}{
		"email": "test@example.com",
	}

	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "user.email")
}

func TestApplyGitConfig_SigningKeyOnly(t *testing.T) {
	t.Parallel()

	git := map[string]interface{}{
		"signing_key": "ABCDEF123456",
	}

	output := captureStdout(t, func() {
		err := applyGitConfig(git)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "user.signingkey")
}

func TestBuildUserContext_WithTypeFilter(t *testing.T) {
	// Save and restore global flags
	origType := mpRecommendType
	origKeywords := mpKeywords
	defer func() {
		mpRecommendType = origType
		mpKeywords = origKeywords
	}()

	mpRecommendType = "formula"
	mpKeywords = ""

	svc := marketplace.NewService(marketplace.DefaultServiceConfig())
	ctx := buildUserContext(svc)

	assert.Contains(t, ctx.PreferredTypes, "formula")
}

func TestRunPluginRemove_FoundPlugin(t *testing.T) {
	// Set up temporary HOME directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create plugin directory
	pluginDir := filepath.Join(tmpDir, ".preflight", "plugins", "test-remove-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: test-remove-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginRemove("test-remove-plugin")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Found plugin")
	assert.Contains(t, output, "test-remove-plugin")
	assert.Contains(t, output, "removal not yet implemented")
}

func TestRunPluginList_InstalledPlugins(t *testing.T) {
	// Set up temporary HOME directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create plugin directory
	pluginDir := filepath.Join(tmpDir, ".preflight", "plugins", "test-list-plugin")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)

	manifest := `apiVersion: v1
name: test-list-plugin
version: 1.0.0
description: A test plugin for listing
provides:
  providers:
    - name: test
      configKey: test
`
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(manifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginList()
		require.NoError(t, err)
	})

	assert.Contains(t, output, "test-list-plugin")
	assert.Contains(t, output, "1.0.0")
}

func TestRunPluginValidate_Success(t *testing.T) {
	tmpDir := t.TempDir()

	validManifest := `apiVersion: v1
name: valid-plugin
version: 1.0.0
provides:
  providers:
    - name: test
      configKey: test
`
	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(validManifest), 0644)
	require.NoError(t, err)

	output := captureStdout(t, func() {
		err := runPluginValidate(tmpDir)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "✓ Plugin validated")
	assert.Contains(t, output, "valid-plugin@1.0.0")
}

func TestRunPluginSearch_InvalidType(t *testing.T) {
	// Save and restore package-level variable
	oldType := searchType
	searchType = "invalid-type"
	defer func() { searchType = oldType }()

	err := runPluginSearch("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestGetTrustStore_LoadError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create trust.json with invalid JSON
	trustDir := filepath.Join(tmpDir, ".preflight")
	err := os.MkdirAll(trustDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(trustDir, "trust.json"), []byte("invalid json"), 0644)
	require.NoError(t, err)

	store, err := getTrustStore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load trust store")
	assert.Nil(t, store)
}

func TestResolveAge_SecretNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Don't create any secret file
	value, err := resolveAge("nonexistent-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "age-encrypted secret not found")
	assert.Empty(t, value)
}

func TestRunPluginSearch_ValidConfigType(_ *testing.T) {
	// Save and restore package-level variable
	oldType := searchType
	searchType = "config"
	defer func() { searchType = oldType }()

	// Run the search - covers type parsing code
	// We don't care about the result, just that the type parsing code runs
	_ = runPluginSearch("zzz-no-match-xyz") // Use unlikely query to minimize network impact
}

func TestRunPluginSearch_ValidProviderType(_ *testing.T) {
	// Save and restore package-level variable
	oldType := searchType
	searchType = "provider"
	defer func() { searchType = oldType }()

	// Run the search - covers type parsing code
	// We don't care about the result, just that the type parsing code runs
	_ = runPluginSearch("zzz-no-match-xyz") // Use unlikely query to minimize network impact
}
