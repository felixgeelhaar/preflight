package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineUpdateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		current  string
		latest   string
		expected UpdateType
	}{
		{
			name:     "major update",
			current:  "1.2.3",
			latest:   "2.0.0",
			expected: UpdateMajor,
		},
		{
			name:     "minor update",
			current:  "1.2.3",
			latest:   "1.3.0",
			expected: UpdateMinor,
		},
		{
			name:     "patch update",
			current:  "1.2.3",
			latest:   "1.2.4",
			expected: UpdatePatch,
		},
		{
			name:     "major with v prefix",
			current:  "v1.2.3",
			latest:   "v2.0.0",
			expected: UpdateMajor,
		},
		{
			name:     "mixed prefix handling",
			current:  "1.2.3",
			latest:   "v2.0.0",
			expected: UpdateMajor,
		},
		{
			name:     "invalid current version",
			current:  "latest",
			latest:   "1.2.3",
			expected: UpdateUnknown,
		},
		{
			name:     "invalid latest version",
			current:  "1.2.3",
			latest:   "HEAD",
			expected: UpdateUnknown,
		},
		{
			name:     "both invalid",
			current:  "alpha",
			latest:   "beta",
			expected: UpdateUnknown,
		},
		{
			name:     "same version",
			current:  "1.2.3",
			latest:   "1.2.3",
			expected: UpdatePatch, // Same version, but classified as patch
		},
		{
			name:     "prerelease versions",
			current:  "1.2.3-beta",
			latest:   "1.3.0",
			expected: UpdateMinor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := DetermineUpdateType(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateType_IsAtLeast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		u        UpdateType
		other    UpdateType
		expected bool
	}{
		{"major >= major", UpdateMajor, UpdateMajor, true},
		{"major >= minor", UpdateMajor, UpdateMinor, true},
		{"major >= patch", UpdateMajor, UpdatePatch, true},
		{"minor >= major", UpdateMinor, UpdateMajor, false},
		{"minor >= minor", UpdateMinor, UpdateMinor, true},
		{"minor >= patch", UpdateMinor, UpdatePatch, true},
		{"patch >= major", UpdatePatch, UpdateMajor, false},
		{"patch >= minor", UpdatePatch, UpdateMinor, false},
		{"patch >= patch", UpdatePatch, UpdatePatch, true},
		{"unknown >= patch", UpdateUnknown, UpdatePatch, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.u.IsAtLeast(tt.other))
		})
	}
}

func TestUpdateType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		u        UpdateType
		expected string
	}{
		{UpdateMajor, "major"},
		{UpdateMinor, "minor"},
		{UpdatePatch, "patch"},
		{UpdateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.u.String())
		})
	}
}

func TestOutdatedPackages_ByUpdateType(t *testing.T) {
	t.Parallel()

	packages := OutdatedPackages{
		{Name: "pkg-major", UpdateType: UpdateMajor},
		{Name: "pkg-minor", UpdateType: UpdateMinor},
		{Name: "pkg-patch", UpdateType: UpdatePatch},
		{Name: "pkg-unknown", UpdateType: UpdateUnknown},
	}

	t.Run("filter to major only", func(t *testing.T) {
		t.Parallel()
		result := packages.ByUpdateType(UpdateMajor)
		require.Len(t, result, 1)
		assert.Equal(t, "pkg-major", result[0].Name)
	})

	t.Run("filter to minor and above", func(t *testing.T) {
		t.Parallel()
		result := packages.ByUpdateType(UpdateMinor)
		require.Len(t, result, 2)
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		t.Parallel()
		result := packages.ByUpdateType("")
		assert.Len(t, result, 4)
	})
}

func TestOutdatedPackages_ExcludeNames(t *testing.T) {
	t.Parallel()

	packages := OutdatedPackages{
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

func TestOutdatedPackages_ExcludePinned(t *testing.T) {
	t.Parallel()

	packages := OutdatedPackages{
		{Name: "unpinned1", Pinned: false},
		{Name: "pinned1", Pinned: true},
		{Name: "unpinned2", Pinned: false},
	}

	result := packages.ExcludePinned()
	require.Len(t, result, 2)
	assert.Equal(t, "unpinned1", result[0].Name)
	assert.Equal(t, "unpinned2", result[1].Name)
}

func TestOutdatedPackages_HasMajor(t *testing.T) {
	t.Parallel()

	t.Run("has major", func(t *testing.T) {
		t.Parallel()
		packages := OutdatedPackages{
			{Name: "pkg1", UpdateType: UpdatePatch},
			{Name: "pkg2", UpdateType: UpdateMajor},
		}
		assert.True(t, packages.HasMajor())
	})

	t.Run("no major", func(t *testing.T) {
		t.Parallel()
		packages := OutdatedPackages{
			{Name: "pkg1", UpdateType: UpdatePatch},
			{Name: "pkg2", UpdateType: UpdateMinor},
		}
		assert.False(t, packages.HasMajor())
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		packages := OutdatedPackages{}
		assert.False(t, packages.HasMajor())
	})
}

func TestOutdatedResult_Summary(t *testing.T) {
	t.Parallel()

	result := &OutdatedResult{
		Packages: OutdatedPackages{
			{Name: "pkg1", UpdateType: UpdateMajor},
			{Name: "pkg2", UpdateType: UpdateMinor},
			{Name: "pkg3", UpdateType: UpdateMinor, Pinned: true},
			{Name: "pkg4", UpdateType: UpdatePatch},
			{Name: "pkg5", UpdateType: UpdatePatch},
		},
	}

	summary := result.Summary()

	assert.Equal(t, 5, summary.Total)
	assert.Equal(t, 1, summary.Major)
	assert.Equal(t, 2, summary.Minor)
	assert.Equal(t, 2, summary.Patch)
	assert.Equal(t, 1, summary.Pinned)
}

func TestBrewOutdatedChecker_Name(t *testing.T) {
	t.Parallel()
	checker := NewBrewOutdatedChecker()
	assert.Equal(t, "brew", checker.Name())
}

func TestBrewOutdatedChecker_parseOutput(t *testing.T) {
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
			name: "formulae only",
			input: `{
				"formulae": [
					{
						"name": "go",
						"installed_versions": ["1.21.0"],
						"current_version": "1.22.0",
						"pinned": false
					}
				],
				"casks": []
			}`,
			expectLen: 1,
		},
		{
			name: "casks only",
			input: `{
				"formulae": [],
				"casks": [
					{
						"name": "visual-studio-code",
						"installed_versions": ["1.85.0"],
						"current_version": "1.86.0"
					}
				]
			}`,
			expectLen: 1,
		},
		{
			name: "mixed formulae and casks",
			input: `{
				"formulae": [
					{
						"name": "go",
						"installed_versions": ["1.21.0"],
						"current_version": "1.22.0",
						"pinned": false
					}
				],
				"casks": [
					{
						"name": "docker",
						"installed_versions": ["4.25.0"],
						"current_version": "4.26.0"
					}
				]
			}`,
			expectLen: 2,
		},
		{
			name:        "invalid json",
			input:       "not json",
			expectError: true,
		},
		{
			name: "pinned formula",
			input: `{
				"formulae": [
					{
						"name": "python",
						"installed_versions": ["3.11.0"],
						"current_version": "3.12.0",
						"pinned": true,
						"pinned_version": "3.11.0"
					}
				],
				"casks": []
			}`,
			expectLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			checker := NewBrewOutdatedChecker()
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

func TestBrewOutdatedChecker_parseOutput_Details(t *testing.T) {
	t.Parallel()

	input := `{
		"formulae": [
			{
				"name": "go",
				"installed_versions": ["1.21.0"],
				"current_version": "1.22.0",
				"pinned": false
			},
			{
				"name": "python",
				"installed_versions": ["3.11.0"],
				"current_version": "3.12.0",
				"pinned": true
			}
		],
		"casks": [
			{
				"name": "docker",
				"installed_versions": ["4.25.0"],
				"current_version": "4.26.0"
			}
		]
	}`

	checker := NewBrewOutdatedChecker()
	result, err := checker.parseOutput([]byte(input))
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Check Go formula
	goPkg := result[0]
	assert.Equal(t, "go", goPkg.Name)
	assert.Equal(t, "1.21.0", goPkg.CurrentVersion)
	assert.Equal(t, "1.22.0", goPkg.LatestVersion)
	assert.Equal(t, UpdateMinor, goPkg.UpdateType)
	assert.Equal(t, "brew", goPkg.Provider)
	assert.False(t, goPkg.Pinned)

	// Check Python formula (pinned)
	pythonPkg := result[1]
	assert.Equal(t, "python", pythonPkg.Name)
	assert.True(t, pythonPkg.Pinned)
	assert.Equal(t, UpdateMinor, pythonPkg.UpdateType) // 3.11 -> 3.12 is minor per semver

	// Check Docker cask
	dockerPkg := result[2]
	assert.Equal(t, "docker", dockerPkg.Name)
	assert.Equal(t, "cask", dockerPkg.Provider)
}

func TestOutdatedCheckerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("register and get", func(t *testing.T) {
		t.Parallel()
		registry := NewOutdatedCheckerRegistry()

		// Brew checker should be available on macOS
		brewChecker := NewBrewOutdatedChecker()
		registry.Register(brewChecker)

		names := registry.Names()
		assert.Contains(t, names, "brew")
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		t.Parallel()
		registry := NewOutdatedCheckerRegistry()
		assert.Nil(t, registry.Get("nonexistent"))
	})

	t.Run("all returns available only", func(t *testing.T) {
		t.Parallel()
		registry := NewOutdatedCheckerRegistry()
		registry.Register(NewBrewOutdatedChecker())

		all := registry.All()
		// Number depends on whether brew is installed
		for _, c := range all {
			assert.True(t, c.Available())
		}
	})
}

// mockOutdatedChecker is a test double for OutdatedChecker.
type mockOutdatedChecker struct {
	name      string
	available bool
	packages  OutdatedPackages
	err       error
}

func (m *mockOutdatedChecker) Name() string {
	return m.name
}

func (m *mockOutdatedChecker) Available() bool {
	return m.available
}

func (m *mockOutdatedChecker) Check(_ context.Context, _ OutdatedOptions) (*OutdatedResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &OutdatedResult{
		Checker:  m.name,
		Packages: m.packages,
	}, nil
}

func TestOutdatedCheckerRegistry_WithMock(t *testing.T) {
	t.Parallel()

	registry := NewOutdatedCheckerRegistry()

	mockAvailable := &mockOutdatedChecker{
		name:      "mock-available",
		available: true,
		packages: OutdatedPackages{
			{Name: "test-pkg", UpdateType: UpdateMinor},
		},
	}

	mockUnavailable := &mockOutdatedChecker{
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
