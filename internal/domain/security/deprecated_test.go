package security

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeprecationReason_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason   DeprecationReason
		expected string
	}{
		{ReasonDeprecated, "deprecated"},
		{ReasonDisabled, "disabled"},
		{ReasonEOL, "end-of-life"},
		{ReasonUnmaintained, "unmaintained"},
		{DeprecationReason(""), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.reason.String())
		})
	}
}

func TestDeprecatedPackages_ExcludeNames(t *testing.T) {
	t.Parallel()

	packages := DeprecatedPackages{
		{Name: "keep1"},
		{Name: "exclude1"},
		{Name: "keep2"},
		{Name: "exclude2"},
	}

	t.Run("exclude some", func(t *testing.T) {
		t.Parallel()
		result := packages.ExcludeNames([]string{"exclude1", "exclude2"})
		require.Len(t, result, 2)
		assert.Equal(t, "keep1", result[0].Name)
		assert.Equal(t, "keep2", result[1].Name)
	})

	t.Run("empty exclude list returns all", func(t *testing.T) {
		t.Parallel()
		result := packages.ExcludeNames(nil)
		assert.Len(t, result, 4)
	})
}

func TestDeprecatedPackages_ByReason(t *testing.T) {
	t.Parallel()

	packages := DeprecatedPackages{
		{Name: "pkg1", Reason: ReasonDeprecated},
		{Name: "pkg2", Reason: ReasonDisabled},
		{Name: "pkg3", Reason: ReasonDeprecated},
		{Name: "pkg4", Reason: ReasonEOL},
	}

	t.Run("filter deprecated", func(t *testing.T) {
		t.Parallel()
		result := packages.ByReason(ReasonDeprecated)
		require.Len(t, result, 2)
		assert.Equal(t, "pkg1", result[0].Name)
		assert.Equal(t, "pkg3", result[1].Name)
	})

	t.Run("filter disabled", func(t *testing.T) {
		t.Parallel()
		result := packages.ByReason(ReasonDisabled)
		require.Len(t, result, 1)
		assert.Equal(t, "pkg2", result[0].Name)
	})

	t.Run("filter eol", func(t *testing.T) {
		t.Parallel()
		result := packages.ByReason(ReasonEOL)
		require.Len(t, result, 1)
		assert.Equal(t, "pkg4", result[0].Name)
	})

	t.Run("filter unmaintained returns empty", func(t *testing.T) {
		t.Parallel()
		result := packages.ByReason(ReasonUnmaintained)
		assert.Empty(t, result)
	})
}

func TestDeprecatedResult_Summary(t *testing.T) {
	t.Parallel()

	result := &DeprecatedResult{
		Packages: DeprecatedPackages{
			{Name: "pkg1", Reason: ReasonDeprecated},
			{Name: "pkg2", Reason: ReasonDeprecated},
			{Name: "pkg3", Reason: ReasonDisabled},
			{Name: "pkg4", Reason: ReasonEOL},
			{Name: "pkg5", Reason: ReasonUnmaintained},
		},
	}

	summary := result.Summary()

	assert.Equal(t, 5, summary.Total)
	assert.Equal(t, 2, summary.Deprecated)
	assert.Equal(t, 1, summary.Disabled)
	assert.Equal(t, 1, summary.EOL)
	assert.Equal(t, 1, summary.Unmaintained)
}

func TestBrewDeprecationChecker_Name(t *testing.T) {
	t.Parallel()
	checker := NewBrewDeprecationChecker()
	assert.Equal(t, "brew", checker.Name())
}

func TestBrewDeprecationChecker_parseOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectLen   int
		expectError bool
	}{
		{
			name:      "empty input",
			input:     "",
			expectLen: 0,
		},
		{
			name: "no deprecated packages",
			input: `[
				{
					"name": "go",
					"full_name": "go",
					"deprecated": false,
					"disabled": false,
					"installed": [{"version": "1.22.0"}]
				}
			]`,
			expectLen: 0,
		},
		{
			name: "deprecated package",
			input: `[
				{
					"name": "python@2",
					"full_name": "python@2",
					"deprecated": true,
					"deprecation_date": "2020-04-01",
					"deprecation_reason": "uses Python 2",
					"disabled": false,
					"installed": [{"version": "2.7.18"}]
				}
			]`,
			expectLen: 1,
		},
		{
			name: "disabled package",
			input: `[
				{
					"name": "old-tool",
					"full_name": "old-tool",
					"deprecated": false,
					"disabled": true,
					"disable_date": "2023-01-15",
					"disable_reason": "no longer maintained",
					"installed": [{"version": "1.0.0"}]
				}
			]`,
			expectLen: 1,
		},
		{
			name: "mixed packages",
			input: `[
				{
					"name": "active",
					"full_name": "active",
					"deprecated": false,
					"disabled": false,
					"installed": [{"version": "1.0.0"}]
				},
				{
					"name": "deprecated-pkg",
					"full_name": "deprecated-pkg",
					"deprecated": true,
					"deprecation_reason": "use new-pkg instead",
					"disabled": false,
					"installed": [{"version": "2.0.0"}]
				},
				{
					"name": "disabled-pkg",
					"full_name": "disabled-pkg",
					"deprecated": false,
					"disabled": true,
					"disable_reason": "security vulnerability",
					"installed": [{"version": "3.0.0"}]
				}
			]`,
			expectLen: 2,
		},
		{
			name:        "invalid json",
			input:       "not json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checker := NewBrewDeprecationChecker()
			result, err := checker.parseOutput([]byte(tt.input))

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectLen)
		})
	}
}

func TestBrewDeprecationChecker_parseOutput_Details(t *testing.T) {
	t.Parallel()

	input := `[
		{
			"name": "python@2",
			"full_name": "python@2",
			"deprecated": true,
			"deprecation_date": "2020-04-01",
			"deprecation_reason": "uses Python 2",
			"disabled": false,
			"installed": [{"version": "2.7.18"}]
		},
		{
			"name": "old-tool",
			"full_name": "old-tool",
			"deprecated": false,
			"disabled": true,
			"disable_date": "2023-01-15",
			"disable_reason": "no longer maintained",
			"installed": [{"version": "1.0.0"}]
		}
	]`

	checker := NewBrewDeprecationChecker()
	result, err := checker.parseOutput([]byte(input))
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Check deprecated package
	deprecatedPkg := result[0]
	assert.Equal(t, "python@2", deprecatedPkg.Name)
	assert.Equal(t, "2.7.18", deprecatedPkg.Version)
	assert.Equal(t, "brew", deprecatedPkg.Provider)
	assert.Equal(t, ReasonDeprecated, deprecatedPkg.Reason)
	assert.Equal(t, "uses Python 2", deprecatedPkg.Message)
	require.NotNil(t, deprecatedPkg.Date)
	assert.Equal(t, 2020, deprecatedPkg.Date.Year())
	assert.Equal(t, time.April, deprecatedPkg.Date.Month())

	// Check disabled package
	disabledPkg := result[1]
	assert.Equal(t, "old-tool", disabledPkg.Name)
	assert.Equal(t, ReasonDisabled, disabledPkg.Reason)
	assert.Equal(t, "no longer maintained", disabledPkg.Message)
	require.NotNil(t, disabledPkg.Date)
	assert.Equal(t, 2023, disabledPkg.Date.Year())
}

func TestDeprecationCheckerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("register and get", func(t *testing.T) {
		t.Parallel()
		registry := NewDeprecationCheckerRegistry()

		brewChecker := NewBrewDeprecationChecker()
		registry.Register(brewChecker)

		names := registry.Names()
		assert.Contains(t, names, "brew")
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		t.Parallel()
		registry := NewDeprecationCheckerRegistry()
		assert.Nil(t, registry.Get("nonexistent"))
	})

	t.Run("all returns available only", func(t *testing.T) {
		t.Parallel()
		registry := NewDeprecationCheckerRegistry()
		registry.Register(NewBrewDeprecationChecker())

		all := registry.All()
		for _, c := range all {
			assert.True(t, c.Available())
		}
	})
}

// mockDeprecationChecker is a test double for DeprecationChecker.
type mockDeprecationChecker struct {
	name      string
	available bool
	packages  DeprecatedPackages
	err       error
}

func (m *mockDeprecationChecker) Name() string {
	return m.name
}

func (m *mockDeprecationChecker) Available() bool {
	return m.available
}

func (m *mockDeprecationChecker) Check(_ context.Context, _ DeprecationOptions) (*DeprecatedResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &DeprecatedResult{
		Checker:  m.name,
		Packages: m.packages,
	}, nil
}

func TestBrewDeprecationChecker_Check_WithMockExec(t *testing.T) {
	t.Parallel()

	brewInfoJSON := `[
		{
			"name": "python@2",
			"full_name": "python@2",
			"deprecated": true,
			"deprecation_date": "2020-04-01",
			"deprecation_reason": "uses Python 2",
			"disabled": false,
			"installed": [{"version": "2.7.18"}]
		},
		{
			"name": "go",
			"full_name": "go",
			"deprecated": false,
			"disabled": false,
			"installed": [{"version": "1.22.0"}]
		},
		{
			"name": "old-tool",
			"full_name": "old-tool",
			"deprecated": false,
			"disabled": true,
			"disable_date": "2023-01-15",
			"disable_reason": "no longer maintained",
			"installed": [{"version": "1.0.0"}]
		}
	]`

	t.Run("successful check returns deprecated packages", func(t *testing.T) {
		t.Parallel()
		checker := &BrewDeprecationChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewInfoJSON)
			},
		}

		// Available() uses exec.LookPath - skip if brew not available
		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), DeprecationOptions{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "brew", result.Checker)
		assert.Len(t, result.Packages, 2) // python@2 (deprecated) + old-tool (disabled)
	})

	t.Run("check with ignore filter", func(t *testing.T) {
		t.Parallel()
		checker := &BrewDeprecationChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewInfoJSON)
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		opts := DeprecationOptions{
			IgnorePackages: []string{"python@2"},
		}

		result, err := checker.Check(context.Background(), opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Packages, 1)
		assert.Equal(t, "old-tool", result.Packages[0].Name)
	})

	t.Run("not available returns error", func(t *testing.T) {
		t.Parallel()
		checker := NewBrewDeprecationChecker()
		if checker.Available() {
			t.Skip("brew is available, skipping not-available test")
		}

		_, err := checker.Check(context.Background(), DeprecationOptions{})
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	})
}

func TestBrewDeprecationChecker_Check_ExecError(t *testing.T) {
	t.Parallel()

	checker := &BrewDeprecationChecker{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
	}

	if !checker.Available() {
		t.Skip("brew not available")
	}

	_, err := checker.Check(context.Background(), DeprecationOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run brew info")
}

func TestDeprecationCheckerRegistry_WithMock(t *testing.T) {
	t.Parallel()

	registry := NewDeprecationCheckerRegistry()

	mockAvailable := &mockDeprecationChecker{
		name:      "mock-available",
		available: true,
		packages: DeprecatedPackages{
			{Name: "test-pkg", Reason: ReasonDeprecated},
		},
	}

	mockUnavailable := &mockDeprecationChecker{
		name:      "mock-unavailable",
		available: false,
	}

	registry.Register(mockAvailable)
	registry.Register(mockUnavailable)

	t.Run("get available returns checker", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-available")
		require.NotNil(t, checker)
		assert.Equal(t, "mock-available", checker.Name())
	})

	t.Run("get unavailable returns nil", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-unavailable")
		assert.Nil(t, checker)
	})

	t.Run("all returns only available", func(t *testing.T) {
		t.Parallel()
		all := registry.All()
		require.Len(t, all, 1)
		assert.Equal(t, "mock-available", all[0].Name())
	})

	t.Run("names returns all registered", func(t *testing.T) {
		t.Parallel()
		names := registry.Names()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "mock-available")
		assert.Contains(t, names, "mock-unavailable")
	})
}
