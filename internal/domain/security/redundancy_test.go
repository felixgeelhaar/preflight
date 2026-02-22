package security

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedundancies_ByType(t *testing.T) {
	t.Parallel()

	redundancies := Redundancies{
		{Type: RedundancyDuplicate, Packages: []string{"go", "go@1.24"}},
		{Type: RedundancyOverlap, Packages: []string{"grype", "trivy"}},
		{Type: RedundancyDuplicate, Packages: []string{"python", "python@3.12"}},
		{Type: RedundancyOrphan, Packages: []string{"libpng", "zlib"}},
	}

	t.Run("filter duplicates", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyDuplicate)
		assert.Len(t, result, 2)
	})

	t.Run("filter overlaps", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyOverlap)
		assert.Len(t, result, 1)
	})

	t.Run("filter orphans", func(t *testing.T) {
		t.Parallel()
		result := redundancies.ByType(RedundancyOrphan)
		assert.Len(t, result, 1)
	})
}

func TestRedundancies_TotalRemovable(t *testing.T) {
	t.Parallel()

	redundancies := Redundancies{
		{Type: RedundancyDuplicate, Remove: []string{"go@1.24"}},
		{Type: RedundancyOverlap, Remove: []string{"trivy"}},
		{Type: RedundancyOrphan, Remove: []string{"libpng", "zlib", "jpeg"}},
	}

	assert.Equal(t, 5, redundancies.TotalRemovable())
}

func TestRedundancyResult_Summary(t *testing.T) {
	t.Parallel()

	result := &RedundancyResult{
		Redundancies: Redundancies{
			{Type: RedundancyDuplicate, Remove: []string{"go@1.24"}},
			{Type: RedundancyDuplicate, Remove: []string{"python@3.12"}},
			{Type: RedundancyOverlap, Remove: []string{"trivy"}},
			{Type: RedundancyOrphan, Remove: []string{"libpng", "zlib"}},
		},
	}

	summary := result.Summary()

	assert.Equal(t, 4, summary.Total)
	assert.Equal(t, 2, summary.Duplicates)
	assert.Equal(t, 1, summary.Overlaps)
	assert.Equal(t, 1, summary.Orphans)
	assert.Equal(t, 5, summary.Removable) // 1+1+1+2 = 5 removable packages
}

func TestBrewRedundancyChecker_Name(t *testing.T) {
	t.Parallel()
	checker := NewBrewRedundancyChecker()
	assert.Equal(t, "brew", checker.Name())
}

func TestBrewRedundancyChecker_detectDuplicates(t *testing.T) {
	t.Parallel()

	checker := NewBrewRedundancyChecker()

	tests := []struct {
		name       string
		packages   []string
		ignore     map[string]bool
		keep       map[string]bool
		expectLen  int
		expectPkgs []string
	}{
		{
			name:       "no duplicates",
			packages:   []string{"git", "curl", "jq"},
			ignore:     nil,
			keep:       nil,
			expectLen:  0,
			expectPkgs: nil,
		},
		{
			name:       "go duplicates",
			packages:   []string{"go", "go@1.24", "go@1.23"},
			ignore:     nil,
			keep:       nil,
			expectLen:  1,
			expectPkgs: []string{"go@1.24", "go@1.23"}, // Remove versioned
		},
		{
			name:       "python duplicates",
			packages:   []string{"python@3.12", "python@3.14"},
			ignore:     nil,
			keep:       nil,
			expectLen:  1,
			expectPkgs: []string{"python@3.12"}, // Keep higher version
		},
		{
			name:       "ignore package",
			packages:   []string{"go", "go@1.24"},
			ignore:     map[string]bool{"go@1.24": true},
			keep:       nil,
			expectLen:  0,
			expectPkgs: nil,
		},
		{
			name:       "keep specific version",
			packages:   []string{"go", "go@1.24"},
			ignore:     nil,
			keep:       map[string]bool{"go@1.24": true},
			expectLen:  1,
			expectPkgs: []string{"go"}, // Remove unversioned since versioned is kept
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ignore := tt.ignore
			if ignore == nil {
				ignore = make(map[string]bool)
			}
			keep := tt.keep
			if keep == nil {
				keep = make(map[string]bool)
			}

			result := checker.detectDuplicates(tt.packages, ignore, keep)
			assert.Len(t, result, tt.expectLen)

			if tt.expectLen > 0 && tt.expectPkgs != nil {
				assert.ElementsMatch(t, tt.expectPkgs, result[0].Remove)
			}
		})
	}
}

func TestBrewRedundancyChecker_detectOverlaps(t *testing.T) {
	t.Parallel()

	checker := NewBrewRedundancyChecker()

	tests := []struct {
		name      string
		packages  []string
		expectLen int
		category  string
	}{
		{
			name:      "no overlaps",
			packages:  []string{"git", "curl", "jq"},
			expectLen: 0,
		},
		{
			name:      "security scanners overlap",
			packages:  []string{"grype", "trivy", "git"},
			expectLen: 1,
			category:  "security_scanners",
		},
		{
			name:      "node package managers overlap",
			packages:  []string{"npm", "yarn", "pnpm"},
			expectLen: 1,
			category:  "node_package_managers",
		},
		{
			name:      "multiple overlaps",
			packages:  []string{"grype", "trivy", "npm", "yarn"},
			expectLen: 2,
		},
		{
			name:      "single tool in category",
			packages:  []string{"grype", "git"},
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := checker.detectOverlaps(tt.packages, make(map[string]bool))
			assert.Len(t, result, tt.expectLen)

			if tt.expectLen == 1 && tt.category != "" {
				assert.Equal(t, tt.category, result[0].Category)
			}
		})
	}
}

func TestDefaultToolCategories(t *testing.T) {
	t.Parallel()

	categories := DefaultToolCategories()

	assert.NotEmpty(t, categories)

	// Verify some expected categories exist
	categoryNames := make(map[string]bool)
	for _, cat := range categories {
		categoryNames[cat.Name] = true
		assert.NotEmpty(t, cat.Tools, "category %s should have tools", cat.Name)
		assert.NotEmpty(t, cat.Description, "category %s should have description", cat.Name)
	}

	assert.True(t, categoryNames["security_scanners"])
	assert.True(t, categoryNames["node_package_managers"])
	assert.True(t, categoryNames["container_runtimes"])
}

func TestRedundancyCheckerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("register and get", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()

		brewChecker := NewBrewRedundancyChecker()
		registry.Register(brewChecker)

		// Get should return checker if available
		checker := registry.Get("brew")
		if brewChecker.Available() {
			assert.NotNil(t, checker)
		}
	})

	t.Run("get nonexistent returns nil", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()
		assert.Nil(t, registry.Get("nonexistent"))
	})

	t.Run("all returns available only", func(t *testing.T) {
		t.Parallel()
		registry := NewRedundancyCheckerRegistry()
		registry.Register(NewBrewRedundancyChecker())

		all := registry.All()
		for _, c := range all {
			assert.True(t, c.Available())
		}
	})
}

// mockRedundancyChecker is a test double for RedundancyChecker.
type mockRedundancyChecker struct {
	name         string
	available    bool
	redundancies Redundancies
	err          error
}

func (m *mockRedundancyChecker) Name() string {
	return m.name
}

func (m *mockRedundancyChecker) Available() bool {
	return m.available
}

func (m *mockRedundancyChecker) Check(_ context.Context, _ RedundancyOptions) (*RedundancyResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &RedundancyResult{
		Checker:      m.name,
		Redundancies: m.redundancies,
	}, nil
}

func TestRedundancyCheckerRegistry_WithMock(t *testing.T) {
	t.Parallel()

	registry := NewRedundancyCheckerRegistry()

	mockAvailable := &mockRedundancyChecker{
		name:      "mock-available",
		available: true,
		redundancies: Redundancies{
			{Type: RedundancyDuplicate, Packages: []string{"go", "go@1.24"}},
		},
	}

	mockUnavailable := &mockRedundancyChecker{
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

	t.Run("check returns redundancies", func(t *testing.T) {
		t.Parallel()
		checker := registry.Get("mock-available")
		require.NotNil(t, checker)

		result, err := checker.Check(context.Background(), RedundancyOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Redundancies, 1)
	})
}

func TestVersionedPackageRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
		matches  bool
	}{
		{"go@1.24", "go", true},
		{"python@3.12", "python", true},
		{"node@20.10.0", "node", true},
		{"go", "", false},
		{"git", "", false},
		{"@types/node", "", false}, // npm-style package
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			matches := versionedPackageRegex.FindStringSubmatch(tt.input)
			if tt.matches {
				require.Len(t, matches, 2)
				assert.Equal(t, tt.expected, matches[1])
			} else {
				assert.Empty(t, matches)
			}
		})
	}
}

func TestRedundancyResult_ToJSON(t *testing.T) {
	t.Parallel()

	result := &RedundancyResult{
		Checker: "brew",
		Redundancies: Redundancies{
			{
				Type:           RedundancyDuplicate,
				Packages:       []string{"go", "go@1.24"},
				Category:       "go",
				Recommendation: "Keep go (tracks latest)",
				Keep:           []string{"go"},
				Remove:         []string{"go@1.24"},
			},
		},
	}

	jsonBytes, err := result.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "duplicate")
	assert.Contains(t, string(jsonBytes), "go@1.24")
}

func TestBrewRedundancyChecker_getInstalledPackages(t *testing.T) {
	t.Parallel()

	t.Run("successful list", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "go\ncurl\njq\ngit\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		packages, err := checker.getInstalledPackages(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []string{"go", "curl", "jq", "git"}, packages)
	})

	t.Run("empty output", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "")
			},
			toolCategories: DefaultToolCategories(),
		}

		packages, err := checker.getInstalledPackages(context.Background())
		require.NoError(t, err)
		assert.Empty(t, packages)
	})

	t.Run("command error", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("false")
			},
			toolCategories: DefaultToolCategories(),
		}

		_, err := checker.getInstalledPackages(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list brew packages")
	})
}

func TestBrewRedundancyChecker_detectOrphans(t *testing.T) {
	t.Parallel()

	t.Run("orphans found", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "==> Would remove:\nlibpng zlib jpeg\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		result := checker.detectOrphans(context.Background(), make(map[string]bool))
		assert.Equal(t, RedundancyOrphan, result.Type)
		assert.ElementsMatch(t, []string{"libpng", "zlib", "jpeg"}, result.Packages)
		assert.ElementsMatch(t, []string{"libpng", "zlib", "jpeg"}, result.Remove)
		assert.Contains(t, result.Recommendation, "3 orphaned dependencies")
	})

	t.Run("no orphans", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "Nothing to remove.\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		result := checker.detectOrphans(context.Background(), make(map[string]bool))
		assert.Empty(t, result.Packages)
	})

	t.Run("orphans with ignore list", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "libpng zlib jpeg\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		ignore := map[string]bool{"zlib": true}
		result := checker.detectOrphans(context.Background(), ignore)
		assert.ElementsMatch(t, []string{"libpng", "jpeg"}, result.Packages)
	})

	t.Run("command failure returns empty", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("false")
			},
			toolCategories: DefaultToolCategories(),
		}

		result := checker.detectOrphans(context.Background(), make(map[string]bool))
		assert.Empty(t, result.Packages)
	})

	t.Run("empty lines and headers filtered", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "==> Header\n\nWould remove something\npkg1 pkg2\n\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		result := checker.detectOrphans(context.Background(), make(map[string]bool))
		assert.ElementsMatch(t, []string{"pkg1", "pkg2"}, result.Packages)
	})
}

func TestBrewRedundancyChecker_Check_WithMockExec(t *testing.T) {
	t.Parallel()

	t.Run("check with duplicates only", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "list" {
					return exec.Command("printf", "%s", "go\ngo@1.24\ncurl\n")
				}
				return exec.Command("echo", "")
			},
			toolCategories: DefaultToolCategories(),
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), RedundancyOptions{})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "brew", result.Checker)
		// Should find go/go@1.24 duplicate
		assert.NotEmpty(t, result.Redundancies)
	})

	t.Run("check with overlaps enabled", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "list" {
					return exec.Command("printf", "%s", "grype\ntrivy\ncurl\n")
				}
				return exec.Command("echo", "")
			},
			toolCategories: DefaultToolCategories(),
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), RedundancyOptions{
			IncludeOverlaps: true,
		})
		require.NoError(t, err)
		// Should find grype/trivy overlap in security_scanners
		found := false
		for _, r := range result.Redundancies {
			if r.Type == RedundancyOverlap && r.Category == "security_scanners" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected security_scanners overlap")
	})

	t.Run("check with orphans enabled", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "list" {
					return exec.Command("printf", "%s", "curl\ngit\n")
				}
				if name == "brew" && len(args) > 0 && args[0] == "autoremove" {
					return exec.Command("printf", "%s", "libpng zlib\n")
				}
				return exec.Command("echo", "")
			},
			toolCategories: DefaultToolCategories(),
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), RedundancyOptions{
			IncludeOrphans: true,
		})
		require.NoError(t, err)
		// Should find orphan entries
		found := false
		for _, r := range result.Redundancies {
			if r.Type == RedundancyOrphan {
				found = true
				break
			}
		}
		assert.True(t, found, "expected orphan redundancy")
	})

	t.Run("check with ignore and keep lists", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
				if name == "brew" && len(args) > 0 && args[0] == "list" {
					return exec.Command("printf", "%s", "go\ngo@1.24\npython\npython@3.12\n")
				}
				return exec.Command("echo", "")
			},
			toolCategories: DefaultToolCategories(),
		}

		if !checker.Available() {
			t.Skip("brew not available")
		}

		result, err := checker.Check(context.Background(), RedundancyOptions{
			IgnorePackages: []string{"python", "python@3.12"},
			KeepPackages:   []string{"go@1.24"},
		})
		require.NoError(t, err)
		// Should only find go duplicate, with go@1.24 kept and go removed
		require.Len(t, result.Redundancies, 1)
		assert.Contains(t, result.Redundancies[0].Keep, "go@1.24")
		assert.Contains(t, result.Redundancies[0].Remove, "go")
	})

	t.Run("not available returns error", func(t *testing.T) {
		t.Parallel()
		checker := NewBrewRedundancyChecker()
		if checker.Available() {
			t.Skip("brew is available")
		}

		_, err := checker.Check(context.Background(), RedundancyOptions{})
		assert.ErrorIs(t, err, ErrScannerNotAvailable)
	})
}

func TestBrewRedundancyChecker_Check_GetInstalledError(t *testing.T) {
	t.Parallel()

	checker := &BrewRedundancyChecker{
		execCommand: func(_ context.Context, name string, args ...string) *exec.Cmd {
			if name == "brew" && len(args) > 0 && args[0] == "list" {
				return exec.Command("false")
			}
			return exec.Command("echo", "")
		},
		toolCategories: DefaultToolCategories(),
	}

	if !checker.Available() {
		t.Skip("brew not available")
	}

	_, err := checker.Check(context.Background(), RedundancyOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list brew packages")
}

func TestBrewRedundancyChecker_Cleanup(t *testing.T) {
	t.Parallel()

	t.Run("cleanup empty list", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("true")
			},
			toolCategories: DefaultToolCategories(),
		}

		err := checker.Cleanup(context.Background(), nil, false)
		assert.NoError(t, err)
	})

	t.Run("cleanup success", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, args ...string) *exec.Cmd {
				capturedArgs = args
				return exec.Command("true")
			},
			toolCategories: DefaultToolCategories(),
		}

		err := checker.Cleanup(context.Background(), []string{"pkg1", "pkg2"}, false)
		assert.NoError(t, err)
		assert.Equal(t, []string{"uninstall", "pkg1", "pkg2"}, capturedArgs)
	})

	t.Run("cleanup dry run", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, args ...string) *exec.Cmd {
				capturedArgs = args
				return exec.Command("true")
			},
			toolCategories: DefaultToolCategories(),
		}

		err := checker.Cleanup(context.Background(), []string{"pkg1"}, true)
		assert.NoError(t, err)
		assert.Equal(t, []string{"uninstall", "--dry-run", "pkg1"}, capturedArgs)
	})

	t.Run("cleanup error", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("false")
			},
			toolCategories: DefaultToolCategories(),
		}

		err := checker.Cleanup(context.Background(), []string{"pkg1"}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to uninstall packages")
	})
}

func TestBrewRedundancyChecker_Autoremove(t *testing.T) {
	t.Parallel()

	t.Run("autoremove success with packages", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "==> Autoremoving\nlibpng zlib\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		removed, err := checker.Autoremove(context.Background(), false)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"libpng", "zlib"}, removed)
	})

	t.Run("autoremove dry run", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, args ...string) *exec.Cmd {
				capturedArgs = args
				return exec.Command("printf", "%s", "pkg1 pkg2\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		removed, err := checker.Autoremove(context.Background(), true)
		require.NoError(t, err)
		assert.Contains(t, capturedArgs, "--dry-run")
		assert.ElementsMatch(t, []string{"pkg1", "pkg2"}, removed)
	})

	t.Run("autoremove nothing to remove", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("printf", "%s", "Nothing to do.\n")
			},
			toolCategories: DefaultToolCategories(),
		}

		removed, err := checker.Autoremove(context.Background(), false)
		require.NoError(t, err)
		assert.Empty(t, removed)
	})

	t.Run("autoremove error", func(t *testing.T) {
		t.Parallel()
		checker := &BrewRedundancyChecker{
			execCommand: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
				return exec.Command("false")
			},
			toolCategories: DefaultToolCategories(),
		}

		_, err := checker.Autoremove(context.Background(), false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to autoremove")
	})
}

func TestBrewRedundancyChecker_detectOverlaps_KeepAll(t *testing.T) {
	t.Parallel()

	checker := NewBrewRedundancyChecker()

	// Terminal emulators have KeepAll=true
	packages := []string{"alacritty", "kitty", "git"}
	result := checker.detectOverlaps(packages, make(map[string]bool))
	require.Len(t, result, 1)

	assert.Equal(t, RedundancyOverlap, result[0].Type)
	assert.Equal(t, "terminal_emulators", result[0].Category)
	// KeepAll=true means Keep should contain all installed, Remove should be empty
	assert.ElementsMatch(t, []string{"alacritty", "kitty"}, result[0].Keep)
	assert.Empty(t, result[0].Remove)
	assert.Contains(t, result[0].Recommendation, "typically used together")
}
