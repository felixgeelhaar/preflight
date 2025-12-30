package config

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConditionEvaluator(t *testing.T) {
	t.Parallel()

	e := NewConditionEvaluator()
	require.NotNil(t, e)
	assert.Equal(t, runtime.GOOS, e.osName)
	assert.Equal(t, runtime.GOARCH, e.arch)
	assert.NotEmpty(t, e.hostname)
}

func TestConditionEvaluator_Evaluate_Empty(t *testing.T) {
	t.Parallel()

	e := NewConditionEvaluator()
	// Empty condition should always be true
	assert.True(t, e.Evaluate(Condition{}))
}

func TestConditionEvaluator_Evaluate_OS(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	tests := []struct {
		name   string
		os     string
		expect bool
	}{
		{"exact match", "darwin", true},
		{"alias mac", "mac", true},
		{"alias macos", "macos", true},
		{"alias darwin", "Darwin", true},
		{"no match linux", "linux", false},
		{"no match windows", "windows", false},
		{"comma separated with match", "linux, darwin", true},
		{"comma separated without match", "linux, windows", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := e.Evaluate(Condition{OS: tt.os})
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_OS_Windows(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "windows", arch: "amd64", hostname: "winhost"}

	tests := []struct {
		os     string
		expect bool
	}{
		{"windows", true},
		{"win", true},
		{"Windows", true},
		{"darwin", false},
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, e.Evaluate(Condition{OS: tt.os}))
		})
	}
}

func TestConditionEvaluator_Evaluate_Arch(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "linux", arch: "amd64", hostname: "testhost"}

	tests := []struct {
		name   string
		arch   string
		expect bool
	}{
		{"exact match", "amd64", true},
		{"alias x64", "x64", true},
		{"alias x86_64", "x86_64", true},
		{"no match arm64", "arm64", false},
		{"comma separated with match", "arm64, amd64", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := e.Evaluate(Condition{Arch: tt.arch})
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_Arch_ARM(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "machost"}

	tests := []struct {
		arch   string
		expect bool
	}{
		{"arm64", true},
		{"arm", true},
		{"aarch64", true},
		{"amd64", false},
	}

	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, e.Evaluate(Condition{Arch: tt.arch}))
		})
	}
}

func TestConditionEvaluator_Evaluate_Hostname(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "linux", arch: "amd64", hostname: "myworkstation"}

	tests := []struct {
		name    string
		pattern string
		expect  bool
	}{
		{"exact match", "myworkstation", true},
		{"case insensitive", "MyWorkstation", true},
		{"glob star", "my*", true},
		{"glob question", "myworkstatio?", true},
		{"glob middle", "*work*", true},
		{"no match", "otherhost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := e.Evaluate(Condition{Hostname: tt.pattern})
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_EnvVar(t *testing.T) {
	// Not parallel - modifies environment
	t.Setenv("TEST_VAR", "test_value")
	t.Setenv("TEST_EMPTY", "")

	e := NewConditionEvaluator()

	tests := []struct {
		name   string
		expr   string
		expect bool
	}{
		{"var exists with value", "TEST_VAR", true},
		{"var exists empty", "TEST_EMPTY", false},
		{"var not exists", "NONEXISTENT_VAR", false},
		{"exact value match", "TEST_VAR=test_value", true},
		{"value mismatch", "TEST_VAR=wrong_value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(Condition{EnvVar: tt.expr})
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_EnvSet(t *testing.T) {
	t.Setenv("TEST_SET_VAR", "value")

	e := NewConditionEvaluator()

	assert.True(t, e.Evaluate(Condition{EnvSet: "TEST_SET_VAR"}))
	assert.False(t, e.Evaluate(Condition{EnvUnset: "TEST_SET_VAR"}))
	assert.False(t, e.Evaluate(Condition{EnvSet: "NONEXISTENT_VAR_12345"}))
	assert.True(t, e.Evaluate(Condition{EnvUnset: "NONEXISTENT_VAR_12345"}))
}

func TestConditionEvaluator_Evaluate_Command(t *testing.T) {
	t.Parallel()

	e := NewConditionEvaluator()

	// "go" should exist on the PATH in a Go dev environment
	// Use a command that's likely to exist
	assert.True(t, e.Evaluate(Condition{Command: "go"}))
	assert.False(t, e.Evaluate(Condition{Command: "nonexistent_command_xyz123"}))
}

func TestConditionEvaluator_Evaluate_File(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	e := NewConditionEvaluator()

	assert.True(t, e.Evaluate(Condition{File: testFile}))
	assert.True(t, e.Evaluate(Condition{File: tmpDir}))
	assert.False(t, e.Evaluate(Condition{File: filepath.Join(tmpDir, "nonexistent")}))
}

func TestConditionEvaluator_Evaluate_File_HomeExpansion(t *testing.T) {
	t.Parallel()

	e := NewConditionEvaluator()

	// Test home directory expansion (~/path should be expanded)
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// ~/.bashrc or similar commonly exists, but safer to test with home itself
	// The matchFile only expands "~/" prefix, not standalone "~"
	// So we test with a path that exists under home
	assert.True(t, e.Evaluate(Condition{File: home}))

	// Test tilde expansion with trailing path separator
	// This verifies the ~/path expansion logic works
	assert.True(t, e.Evaluate(Condition{File: "~/."}))
}

func TestConditionEvaluator_Evaluate_Not(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	// NOT darwin should be false (since we are darwin)
	assert.False(t, e.Evaluate(Condition{Not: &Condition{OS: "darwin"}}))

	// NOT linux should be true (since we are not linux)
	assert.True(t, e.Evaluate(Condition{Not: &Condition{OS: "linux"}}))
}

func TestConditionEvaluator_Evaluate_All(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	// All conditions must match
	assert.True(t, e.Evaluate(Condition{
		All: []Condition{
			{OS: "darwin"},
			{Arch: "arm64"},
		},
	}))

	// One condition fails
	assert.False(t, e.Evaluate(Condition{
		All: []Condition{
			{OS: "darwin"},
			{Arch: "amd64"}, // This fails
		},
	}))
}

func TestConditionEvaluator_Evaluate_Any(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	// Any condition matches
	assert.True(t, e.Evaluate(Condition{
		Any: []Condition{
			{OS: "linux"},
			{OS: "darwin"}, // This matches
		},
	}))

	// No condition matches
	assert.False(t, e.Evaluate(Condition{
		Any: []Condition{
			{OS: "linux"},
			{OS: "windows"},
		},
	}))
}

func TestConditionEvaluator_Evaluate_Combined(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	// Complex condition: darwin AND (arm64 OR amd64)
	assert.True(t, e.Evaluate(Condition{
		OS: "darwin",
		Any: []Condition{
			{Arch: "arm64"},
			{Arch: "amd64"},
		},
	}))

	// darwin AND (amd64 only) - fails
	assert.False(t, e.Evaluate(Condition{
		OS:   "darwin",
		Arch: "amd64",
	}))
}

func TestConditionEvaluator_ShouldApplyLayer(t *testing.T) {
	t.Parallel()

	e := &ConditionEvaluator{osName: "darwin", arch: "arm64", hostname: "testhost"}

	tests := []struct {
		name   string
		layer  ConditionalLayer
		expect bool
	}{
		{
			name:   "no conditions",
			layer:  ConditionalLayer{Name: "base"},
			expect: true,
		},
		{
			name:   "when matches",
			layer:  ConditionalLayer{Name: "mac", When: Condition{OS: "darwin"}},
			expect: true,
		},
		{
			name:   "when does not match",
			layer:  ConditionalLayer{Name: "linux", When: Condition{OS: "linux"}},
			expect: false,
		},
		{
			name:   "unless matches (should not apply)",
			layer:  ConditionalLayer{Name: "notmac", Unless: Condition{OS: "darwin"}},
			expect: false,
		},
		{
			name:   "unless does not match (should apply)",
			layer:  ConditionalLayer{Name: "notlinux", Unless: Condition{OS: "linux"}},
			expect: true,
		},
		{
			name: "when and unless both specified",
			layer: ConditionalLayer{
				Name:   "complex",
				When:   Condition{OS: "darwin"},
				Unless: Condition{Arch: "amd64"},
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := e.ShouldApplyLayer(tt.layer)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestConditionEvaluator_isEmpty(t *testing.T) {
	t.Parallel()

	e := NewConditionEvaluator()

	assert.True(t, e.isEmpty(Condition{}))
	assert.False(t, e.isEmpty(Condition{OS: "darwin"}))
	assert.False(t, e.isEmpty(Condition{Arch: "amd64"}))
	assert.False(t, e.isEmpty(Condition{Not: &Condition{}}))
	assert.False(t, e.isEmpty(Condition{All: []Condition{{}}}))
	assert.False(t, e.isEmpty(Condition{Any: []Condition{{}}}))
}

func TestGlobToRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		glob   string
		input  string
		expect bool
	}{
		{"*", "anything", true},
		{"test*", "testing", true},
		{"test*", "test", true},
		{"test*", "tes", false},
		{"*.txt", "file.txt", true},
		{"*.txt", "file.md", false},
		{"test?", "tests", true},
		{"test?", "test", false},
		{"test.go", "test.go", true},
		{"test.go", "testXgo", false},
	}

	for _, tt := range tests {
		t.Run(tt.glob+"_"+tt.input, func(t *testing.T) {
			t.Parallel()
			pattern := globToRegex(tt.glob)
			matched, _ := regexp.MatchString("^"+pattern+"$", tt.input)
			assert.Equal(t, tt.expect, matched)
		})
	}
}
