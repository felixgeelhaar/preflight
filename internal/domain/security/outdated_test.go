package security

import (
	"context"
	"os/exec"
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

func TestUpgradeOptions(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		opts := UpgradeOptions{}
		assert.False(t, opts.DryRun)
		assert.False(t, opts.IncludeMajor)
	})

	t.Run("with options set", func(t *testing.T) {
		t.Parallel()
		opts := UpgradeOptions{
			DryRun:       true,
			IncludeMajor: true,
		}
		assert.True(t, opts.DryRun)
		assert.True(t, opts.IncludeMajor)
	})
}

func TestUpgradeResult(t *testing.T) {
	t.Parallel()

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()
		result := &UpgradeResult{
			Upgraded: []UpgradedPackage{},
			Skipped:  []SkippedPackage{},
			Failed:   []FailedPackage{},
		}
		assert.Empty(t, result.Upgraded)
		assert.Empty(t, result.Skipped)
		assert.Empty(t, result.Failed)
	})

	t.Run("with packages", func(t *testing.T) {
		t.Parallel()
		result := &UpgradeResult{
			Upgraded: []UpgradedPackage{
				{Name: "pkg1", FromVersion: "1.0.0", ToVersion: "1.1.0", Provider: "brew"},
			},
			Skipped: []SkippedPackage{
				{Name: "pkg2", Reason: "major update", UpdateType: UpdateMajor},
			},
			Failed: []FailedPackage{
				{Name: "pkg3", Error: "network error"},
			},
			DryRun: false,
		}
		assert.Len(t, result.Upgraded, 1)
		assert.Len(t, result.Skipped, 1)
		assert.Len(t, result.Failed, 1)
		assert.False(t, result.DryRun)
	})

	t.Run("dry run result", func(t *testing.T) {
		t.Parallel()
		result := &UpgradeResult{
			Upgraded: []UpgradedPackage{
				{Name: "pkg1", FromVersion: "1.0.0", ToVersion: "1.1.0"},
			},
			DryRun: true,
		}
		assert.True(t, result.DryRun)
	})
}

func TestUpgradedPackage(t *testing.T) {
	t.Parallel()

	pkg := UpgradedPackage{
		Name:        "test-pkg",
		FromVersion: "1.0.0",
		ToVersion:   "1.1.0",
		Provider:    "brew",
	}

	assert.Equal(t, "test-pkg", pkg.Name)
	assert.Equal(t, "1.0.0", pkg.FromVersion)
	assert.Equal(t, "1.1.0", pkg.ToVersion)
	assert.Equal(t, "brew", pkg.Provider)
}

func TestSkippedPackage(t *testing.T) {
	t.Parallel()

	pkg := SkippedPackage{
		Name:       "test-pkg",
		Reason:     "major update requires --major flag",
		UpdateType: UpdateMajor,
	}

	assert.Equal(t, "test-pkg", pkg.Name)
	assert.Equal(t, "major update requires --major flag", pkg.Reason)
	assert.Equal(t, UpdateMajor, pkg.UpdateType)
}

func TestFailedPackage(t *testing.T) {
	t.Parallel()

	pkg := FailedPackage{
		Name:  "test-pkg",
		Error: "permission denied",
	}

	assert.Equal(t, "test-pkg", pkg.Name)
	assert.Equal(t, "permission denied", pkg.Error)
}

func TestBrewOutdatedChecker_Check_WithMockExec(t *testing.T) {
	t.Parallel()

	brewOutdatedJSON := `{
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
			},
			{
				"name": "curl",
				"installed_versions": ["8.0.0"],
				"current_version": "8.0.1",
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
	}`

	t.Run("check returns outdated packages with default filters", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewOutdatedJSON)
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		// Default: IncludePatch=false, IncludePinned=false
		result, err := checker.Check(context.Background(), OutdatedOptions{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "brew", result.Checker)
		// go (minor), docker (minor) are included; python (pinned, excluded), curl (patch, excluded)
		assert.Len(t, result.Packages, 2)
	})

	t.Run("check with include patch", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewOutdatedJSON)
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), OutdatedOptions{
			IncludePatch: true,
		})
		require.NoError(t, err)
		// go (minor), curl (patch), docker (minor) - python pinned excluded
		assert.Len(t, result.Packages, 3)
	})

	t.Run("check with include pinned", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewOutdatedJSON)
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), OutdatedOptions{
			IncludePinned: true,
		})
		require.NoError(t, err)
		// go (minor), python (minor, pinned), docker (minor) - curl (patch, excluded)
		assert.Len(t, result.Packages, 3)
	})

	t.Run("check with ignore packages", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", brewOutdatedJSON)
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), OutdatedOptions{
			IgnorePackages: []string{"go"},
		})
		require.NoError(t, err)
		// Only docker (minor) - go (ignored), python (pinned excluded), curl (patch excluded)
		assert.Len(t, result.Packages, 1)
		assert.Equal(t, "docker", result.Packages[0].Name)
	})

	t.Run("not available returns error", func(t *testing.T) {
		t.Parallel()
		checker := NewBrewOutdatedChecker()
		if checker.Available() {
			t.Skip("brew is available, skipping not-available test")
		}

		_, err := checker.Check(context.Background(), OutdatedOptions{})
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	})
}

func TestBrewOutdatedChecker_Check_ExecError(t *testing.T) {
	t.Parallel()

	checker := &BrewOutdatedChecker{
		execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return exec.Command("false")
		},
	}

	if !checker.Available() {
		t.Skip("brew not available")
	}

	_, err := checker.Check(context.Background(), OutdatedOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run brew outdated")
}

func TestBrewOutdatedChecker_Upgrade_WithMockExec(t *testing.T) {
	t.Parallel()

	brewOutdatedJSON := `{
		"formulae": [
			{
				"name": "go",
				"installed_versions": ["1.21.0"],
				"current_version": "1.22.0",
				"pinned": false
			},
			{
				"name": "node",
				"installed_versions": ["18.0.0"],
				"current_version": "20.0.0",
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
	}`

	t.Run("dry run upgrade", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				// brew outdated --json call
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				return exec.Command("echo", "ok")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"go"}, UpgradeOptions{
			DryRun: true,
		})
		require.NoError(t, err)
		assert.True(t, result.DryRun)
		assert.Len(t, result.Upgraded, 1)
		assert.Equal(t, "go", result.Upgraded[0].Name)
		assert.Equal(t, "1.21.0", result.Upgraded[0].FromVersion)
		assert.Equal(t, "1.22.0", result.Upgraded[0].ToVersion)
	})

	t.Run("skip major without flag", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				return exec.Command("echo", "ok")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"node"}, UpgradeOptions{
			DryRun: true,
		})
		require.NoError(t, err)
		assert.Len(t, result.Skipped, 1)
		assert.Equal(t, "node", result.Skipped[0].Name)
		assert.Equal(t, UpdateMajor, result.Skipped[0].UpdateType)
	})

	t.Run("include major upgrade", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				return exec.Command("echo", "ok")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"node"}, UpgradeOptions{
			DryRun:       true,
			IncludeMajor: true,
		})
		require.NoError(t, err)
		assert.Len(t, result.Upgraded, 1)
		assert.Equal(t, "node", result.Upgraded[0].Name)
	})

	t.Run("upgrade all when no packages specified", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				return exec.Command("echo", "ok")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), nil, UpgradeOptions{
			DryRun:       true,
			IncludeMajor: true,
		})
		require.NoError(t, err)
		// All packages should be upgraded (go, node, docker)
		assert.Len(t, result.Upgraded, 3)
	})

	t.Run("actual upgrade success", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				// brew upgrade <pkg> succeeds
				return exec.Command("true")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"go"}, UpgradeOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Upgraded, 1)
		assert.False(t, result.DryRun)
	})

	t.Run("actual upgrade failure", func(t *testing.T) {
		t.Parallel()
		callNum := 0
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				callNum++
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				// brew upgrade <pkg> fails
				return exec.Command("false")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"go"}, UpgradeOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Failed, 1)
		assert.Equal(t, "go", result.Failed[0].Name)
	})

	t.Run("cask upgrade uses --cask flag", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				// Capture the upgrade command args
				capturedArgs = args
				return exec.Command("true")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"docker"}, UpgradeOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Upgraded, 1)
		assert.Contains(t, capturedArgs, "--cask")
	})

	t.Run("not available returns error", func(t *testing.T) {
		t.Parallel()
		checker := NewBrewOutdatedChecker()
		if checker.Available() {
			t.Skip("brew is available")
		}

		_, err := checker.Upgrade(context.Background(), nil, UpgradeOptions{})
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	})

	t.Run("package not in outdated list is skipped", func(t *testing.T) {
		t.Parallel()
		checker := &BrewOutdatedChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "outdated" {
					return exec.Command("printf", "%s", brewOutdatedJSON)
				}
				return exec.Command("true")
			},
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Upgrade(context.Background(), []string{"nonexistent-pkg"}, UpgradeOptions{DryRun: true})
		require.NoError(t, err)
		assert.Empty(t, result.Upgraded)
		assert.Empty(t, result.Failed)
		assert.Empty(t, result.Skipped)
	})
}

func TestExtractMinor_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal version",
			input:    "v1.2.3",
			expected: "2",
		},
		{
			name:     "with prerelease in minor",
			input:    "v1.2-beta",
			expected: "2",
		},
		{
			name:     "single part version",
			input:    "v1",
			expected: "",
		},
		{
			name:     "just major",
			input:    "1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractMinor(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
